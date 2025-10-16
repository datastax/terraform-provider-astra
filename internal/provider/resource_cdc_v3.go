package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type CDCResource struct {
	clients *astraClients2
}

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &CDCResource{}
	_ resource.ResourceWithConfigure   = &CDCResource{}
	_ resource.ResourceWithImportState = &CDCResource{}
)

func NewAstraCDCv3Resource() resource.Resource {
	return &CDCResource{}
}

// Configure adds the provider configured client to the data source.
func (r *CDCResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.clients = req.ProviderData.(*astraClients2)
}

// CDCResourceModel represents data used to configure CDC
type CDCResourceModel struct {
	DatabaseID   types.String               `tfsdk:"database_id"`
	DatabaseName types.String               `tfsdk:"database_name"`
	Tables       []KeyspaceTable            `tfsdk:"tables"`
	Regions      []DatacenterToStreamingMap `tfsdk:"regions"`
	DataTopics   types.Map                  `tfsdk:"data_topics"`
}

type KeyspaceTable struct {
	Keyspace types.String `tfsdk:"keyspace"`
	Table    types.String `tfsdk:"table"`
}

type DatacenterToStreamingMap struct {
	Region           types.String `tfsdk:"region"`
	DatacenterID     types.String `tfsdk:"datacenter_id"`
	StreamingCluster types.String `tfsdk:"streaming_cluster"`
	StreamingTenant  types.String `tfsdk:"streaming_tenant"`
}

func (r *CDCResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "astra_cdc_v3"
}

func (r *CDCResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "`astra_cdc_v3` enables CDC between Astra Serverless database and Astra Streaming.",
		Attributes: map[string]schema.Attribute{
			"database_id": schema.StringAttribute{
				Description: "Astra database to create the keyspace.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"database_name": schema.StringAttribute{
				Description: "Astra database name.",
				Required:    true,
			},
			"tables": schema.SetNestedAttribute{
				Description: "List of tables to enable CDC.  Must include at least 1.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"keyspace": schema.StringAttribute{
							Required: true,
						},
						"table": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},

			"regions": schema.SetNestedAttribute{
				Description: "Mapping between datacenter regions and streaming tenants.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"region": schema.StringAttribute{
							Description: "Cloud provider region",
							Required:    true,
						},
						"datacenter_id": schema.StringAttribute{
							Description: "Astra Datacenter ID",
							Required:    true,
						},
						"streaming_cluster": schema.StringAttribute{
							Description: "Name of Pulsar cluster hosting the streaming tenant.",
							Required:    true,
						},
						"streaming_tenant": schema.StringAttribute{
							Description: "Name of the streaming tenant",
							Required:    true,
						},
					},
				},
			},
			"data_topics": schema.MapAttribute{
				Description: "Map of CDC data topics for each table in each region. " +
					"Use the region as the first key and the keyspace.table as the second key. " +
					"For example, astra_cdc.mycdc.data_topics['us-east1']['ks1.table1'].",
				Computed: true,
				ElementType: types.MapType{
					ElemType: types.StringType,
				},
			},
		},
	}
}

func (r *CDCResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CDCResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraClient := r.clients.astraClient

	cdcRequestBody := createEnableCDCRequestBody(&plan)
	cdcResponse, err := astraClient.EnableCDCWithResponse(ctx, cdcRequestBody.DatabaseID, cdcRequestBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to enable CDC",
			err.Error())
		return
	} else if cdcResponse.StatusCode() != http.StatusCreated {
		errString := fmt.Sprintf("failed to enable CDC for DB '%s' with status code '%v', message: '%s'",
			plan.DatabaseID.ValueString(), cdcResponse.StatusCode(), string(cdcResponse.Body))
		resp.Diagnostics.AddError("failed to enable CDC", errString)
		return
	}

	// wait for the database to be active before after CDC
	if err := waitForDatabaseActive(ctx, astraClient, plan.DatabaseID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to wait for database '%s' to be active", plan.DatabaseID.ValueString()),
			err.Error())
		return
	}

	plan.setDataTopics()

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *CDCResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CDCResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraClient := r.clients.astraClient

	cdcResponse, err := astraClient.GetCDCWithResponse(ctx, state.DatabaseID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to get CDC status",
			err.Error())
		return
	} else if cdcResponse.StatusCode() != http.StatusOK {
		errString := fmt.Sprintf("failed to get CDC status for DB '%s' with status code '%v', message: '%s'",
			state.DatabaseID.ValueString(), cdcResponse.StatusCode(), string(cdcResponse.Body))
		resp.Diagnostics.AddError("failed to get CDC status", errString)
		return
	}

	copyResponseDataToResourceState(&state, cdcResponse.JSON200)
	state.setDataTopics()

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *CDCResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CDCResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraClient := r.clients.astraClient

	// wait for the database to be active before updating CDC
	if err := waitForDatabaseActive(ctx, astraClient, plan.DatabaseID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to wait for database '%s' to be active", plan.DatabaseID.ValueString()),
			err.Error())
		return
	}

	cdcRequestBody := createUpdateCDCRequestBody(&plan)
	cdcResponse, err := astraClient.UpdateCDCWithResponse(ctx, cdcRequestBody.DatabaseID, cdcRequestBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to enable CDC",
			err.Error())
		return
	} else if cdcResponse.StatusCode() != http.StatusNoContent {
		errString := fmt.Sprintf("failed to update CDC for DB '%s' with status code '%v', message: '%s'",
			plan.DatabaseID.ValueString(), cdcResponse.StatusCode(), string(cdcResponse.Body))
		resp.Diagnostics.AddError("failed to update CDC", errString)
		return
	}

	// wait for the database to be active before after CDC
	if err := waitForDatabaseActive(ctx, astraClient, plan.DatabaseID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to wait for database '%s' to be active", plan.DatabaseID.ValueString()),
			err.Error())
		return
	}

	plan.setDataTopics()

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *CDCResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CDCResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraClient := r.clients.astraClient

	cdcRequestBody := createDeleteCDCRequestBody(&state)

	cdcResponse, err := astraClient.DeleteCDCWithResponse(ctx, state.DatabaseID.ValueString(), cdcRequestBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to delete CDC",
			err.Error())
		return
	} else if cdcResponse.StatusCode() != http.StatusNoContent {
		errString := fmt.Sprintf("failed to delete CDC for DB '%s' with status code '%v', message: '%s'",
			state.DatabaseID.ValueString(), cdcResponse.StatusCode(), string(cdcResponse.Body))
		resp.Diagnostics.AddError("failed to delete CDC", errString)
		return
	}

	// Remove the resource from state
	resp.State.RemoveResource(ctx)
}

// ImportState just reads the ID from the CLI and then calls Read() to get the state of the object
func (r *CDCResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), req.ID)...)
}

// Regions List of regions where CDC will be enabled
type RegionJSON struct {
	// DatacenterID Unique identifier for the data center
	DatacenterID string `json:"datacenterID"`

	// DatacenterRegion Cloud region where the data center is located
	DatacenterRegion string `json:"datacenterRegion"`

	// StreamingClusterName Name of the streaming cluster
	StreamingClusterName string `json:"streamingClusterName"`

	// StreamingTenantName Name of the streaming tenant
	StreamingTenantName string `json:"streamingTenantName"`
}

// Tables List of tables for which CDC needs to be enabled
type TableJSON struct {
	// KeyspaceName Name of the keyspace
	KeyspaceName string `json:"keyspaceName"`

	// TableName Name of the table
	TableName string `json:"tableName"`
}

func createEnableCDCRequestBody(tfData *CDCResourceModel) astra.EnableCDCJSONRequestBody {
	reqData := astra.EnableCDCJSONRequestBody{
		DatabaseID:   tfData.DatabaseID.ValueString(),
		DatabaseName: tfData.DatabaseName.ValueString(),
	}
	for _, table := range tfData.Tables {
		nextTable := TableJSON{
			KeyspaceName: table.Keyspace.ValueString(),
			TableName:    table.Table.ValueString(),
		}
		reqData.Tables = append(reqData.Tables, nextTable)
	}
	for _, region := range tfData.Regions {
		nextRegion := RegionJSON{
			DatacenterRegion:     region.Region.ValueString(),
			DatacenterID:         region.DatacenterID.ValueString(),
			StreamingClusterName: region.StreamingCluster.ValueString(),
			StreamingTenantName:  region.StreamingTenant.ValueString(),
		}
		reqData.Regions = append(reqData.Regions, nextRegion)
	}
	return reqData
}

func createUpdateCDCRequestBody(tfData *CDCResourceModel) astra.UpdateCDCJSONRequestBody {
	reqData := astra.UpdateCDCJSONRequestBody{
		DatabaseID:   tfData.DatabaseID.ValueString(),
		DatabaseName: tfData.DatabaseName.ValueString(),
	}
	for _, table := range tfData.Tables {
		nextTable := TableJSON{
			KeyspaceName: table.Keyspace.ValueString(),
			TableName:    table.Table.ValueString(),
		}
		reqData.Tables = append(reqData.Tables, nextTable)
	}
	for _, region := range tfData.Regions {
		nextRegion := RegionJSON{
			DatacenterRegion:     region.Region.ValueString(),
			DatacenterID:         region.DatacenterID.ValueString(),
			StreamingClusterName: region.StreamingCluster.ValueString(),
			StreamingTenantName:  region.StreamingTenant.ValueString(),
		}
		reqData.Regions = append(reqData.Regions, nextRegion)
	}
	return reqData
}

// copyResponseDataToResourceState copies the data from the REST endpoing response to the Terraform resource state model.
func copyResponseDataToResourceState(tfData *CDCResourceModel, respData *astra.ListCDCResponse) {
	tfData.DatabaseID = types.StringValue(respData.DatabaseID)
	tfData.DatabaseName = types.StringValue(respData.DatabaseName)
	var tables []KeyspaceTable
	for _, table := range respData.Tables {
		tables = append(tables, KeyspaceTable{
			Keyspace: types.StringValue(table.KeyspaceName),
			Table:    types.StringValue(table.TableName),
		})
	}
	tfData.Tables = tables

	var regions []DatacenterToStreamingMap
	for _, region := range respData.Regions {
		regions = append(regions, DatacenterToStreamingMap{
			Region:           types.StringValue(region.DatacenterRegion),
			DatacenterID:     types.StringValue(region.DatacenterID),
			StreamingCluster: types.StringValue(region.StreamingClusterName),
			StreamingTenant:  types.StringValue(region.StreamingTenantName),
		})
	}
	tfData.Regions = regions
}

func createDeleteCDCRequestBody(tfData *CDCResourceModel) astra.DeleteCDCJSONRequestBody {
	reqData := astra.DeleteCDCJSONRequestBody{
		DatabaseID: tfData.DatabaseID.ValueString(),
	}
	for _, table := range tfData.Tables {
		nextTable := TableJSON{
			KeyspaceName: table.Keyspace.ValueString(),
			TableName:    table.Table.ValueString(),
		}
		reqData.Tables = append(reqData.Tables, nextTable)
	}
	return reqData
}

const AstraCDCPulsarNamespace = "astracdc"

// calculateCDCDataTopicName constructs the expected CDC data topic name based on the database ID and streaming tenant.
// For example 'persistent://terraform-support1/astracdc/data-0d509b0f-d38a-4c8e-9680-fa7c752189b7-ks1.table1'
func calculateCDCDataTopicName(streamingTenant, databaseID, keyspace, tableName string) string {
	return fmt.Sprintf("persistent://%s/%s/data-%s-%s.%s", streamingTenant, AstraCDCPulsarNamespace, databaseID, keyspace, tableName)
}

// getDataTopicsList uses the region and table config to create the two dimensional (region and table) map of data topics.
func (m *CDCResourceModel) setDataTopics() {

	dataTopicsMap := map[string]attr.Value{}

	for _, region := range m.Regions {
		regionDataTopics := make(map[string]attr.Value)

		for _, table := range m.Tables {
			keyspaceTable := fmt.Sprintf("%s.%s", table.Keyspace.ValueString(), table.Table.ValueString())
			topicFQDN := calculateCDCDataTopicName(region.StreamingTenant.ValueString(), m.DatabaseID.ValueString(), table.Keyspace.ValueString(), table.Table.ValueString())
			regionDataTopics[keyspaceTable] = types.StringValue(topicFQDN)
		}
		dataTopicsMap[region.Region.ValueString()] = types.MapValueMust(types.StringType, regionDataTopics)
	}

	m.DataTopics = types.MapValueMust(types.MapType{ElemType: types.StringType}, dataTopicsMap)
}

var (
	cdcUpdateTimeout = time.Duration(2 * time.Minute)
)

// waitForDatabaseActive waits for the database to reach the ACTIVE state.  Will return an error if the database reaches a terminal state (ERROR, TERMINATED, or TERMINATING),
// or if the request returns an unexpected HTTP status code, or if the request times out.
func waitForDatabaseActive(ctx context.Context, client *astra.ClientWithResponses, databaseID string) error {
	return retry.RetryContext(ctx, cdcUpdateTimeout, func() *retry.RetryError {
		res, err := client.GetDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID))
		// Errors sending request should be retried and are assumed to be transient
		if err != nil || res.StatusCode() >= http.StatusInternalServerError {
			return retry.RetryableError(fmt.Errorf("error getting database status: %s", string(res.Body)))
		}

		// don't retry on unexpected HTTP errors
		if res.StatusCode() == http.StatusUnauthorized {
			return retry.NonRetryableError(fmt.Errorf("user not authorized. Effective role must have 'View DB' permission on the database (or on all DBs in the current org)"))
		} else if res.StatusCode() > http.StatusOK || res.JSON200 == nil {
			return retry.NonRetryableError(fmt.Errorf("unexpected response fetching database, status code: %d, message %s", res.StatusCode(), string(res.Body)))
		}

		// Success fetching database
		dbStatus := res.JSON200.Status
		switch dbStatus {
		//case astra.ERROR, astra.TERMINATED, astra.TERMINATING: TODO uncomment
		//	// If the database reached a terminal state it will never become active
		//	return retry.NonRetryableError(fmt.Errorf("database failed to reach active status: status='%s'", dbStatus))
		//case astra.ACTIVE:
		//	return nil
		default:
			return retry.RetryableError(fmt.Errorf("waiting database to be active but is '%s'", dbStatus))
		}
	})
}
