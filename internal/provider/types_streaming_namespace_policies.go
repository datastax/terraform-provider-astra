package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/datastax/pulsar-admin-client-go/src/pulsaradmin"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	// String constants to help avoid typos.  These are used to define the Terraform
	// schema for Pulsar namespace policies, therefore they must use snake_case.
	// The capitalization rules may not always match the capitalization of the JSON
	// policy coming from Pulsar which sometimes uses camelCase.
	policyAllowAutoTopicCreation                = "allow_auto_topic_creation"
	policyAutoTopicCreationOverride             = "auto_topic_creation_override"
	policyAutoTopicCreationType                 = "topic_type"
	policyAutoTopicCreationDefaultNumPartitions = "default_num_partitions"
	policyBacklogQuotaMap                       = "backlog_quota_map"
	policyBacklogQuotaLimit                     = "limit"
	policyBacklogQuotaLimitSize                 = "limit_size"
	policyBacklogQuotaLimitTime                 = "limit_time"
	policyBacklogQuotaLimitPolicy               = "policy"
	policyIsAllowAutoUpdateSchema               = "is_allow_auto_update_schema"
	policyMessageTTLInSeconds                   = "message_ttl_in_seconds"
	policyRetentionPolicies                     = "retention_policies"
	policyRetentionTimeInMinutes                = "retention_time_in_minutes"
	policyRetentionSizeInMB                     = "retention_size_in_mb"
	policySchemaAutoUpdateCompatibilityStrategy = "schema_auto_update_compatibility_strategy"
	policySchemaCompatibilityStrategy           = "schema_compatibility_strategy"
	policySchemaValidationEnforced              = "schema_validation_enforced"


	policySetOffloadThreshold = "set_offload_threshold"

	policyInactiveTopicPolicies                   = "inactive_topic_policies"
	policyInactiveTopicDeleteWhileInactive        = "delete_while_inactive"
	policyInactiveTopicMaxInactiveDurationSeconds = "max_inactive_duration_seconds"
	policyInactiveTopicDeleteMode                 = "delete_mode"

	policySubscriptionExpirationTimeMinutes = "subscription_expiration_time_minutes"
)

type PulsarNamespacePolicies struct {
	IsAllowAutoUpdateSchema               *bool   `tfsdk:"is_allow_auto_update_schema" json:"is_allow_auto_update_schema,omitempty"`
	MessageTTLInSeconds                   *int32  `tfsdk:"message_ttl_in_seconds" json:"message_ttl_in_seconds,omitempty"`
	SchemaAutoUpdateCompatibilityStrategy *string `tfsdk:"schema_auto_update_compatibility_strategy" json:"schema_auto_update_compatibility_strategy,omitempty"`
	SchemaCompatibilityStrategy           *string `tfsdk:"schema_compatibility_strategy" json:"schema_compatibility_strategy,omitempty"`
	SchemaValidationEnforced              *bool   `tfsdk:"schema_validation_enforced" json:"schema_validation_enforced,omitempty"`


	AutoTopicCreationOverride *PulsarNamespaceAutoTopicCreationOverride `tfsdk:"auto_topic_creation_override" json:"autoTopicCreationOverride,omitempty"`
	BacklogQuota              map[string]*PulsarNamespaceBacklogQuota   `tfsdk:"backlog_quota_map" json:"backlog_quota_map,omitempty"`
	RetentionPolicies         *PulsarNamespaceRetentionPolicies         `tfsdk:"retention_policies" json:"retention_policies,omitempty"`
	SetOffloadThreshold       *string                                   `tfsdk:"set_offload_threshold" json:"set_offload_threshold,omitempty"`

	AutoTopicCreationOverride         *PulsarNamespaceAutoTopicCreationOverride `tfsdk:"auto_topic_creation_override" json:"autoTopicCreationOverride,omitempty"`
	BacklogQuota                      map[string]*PulsarNamespaceBacklogQuota   `tfsdk:"backlog_quota_map" json:"backlog_quota_map,omitempty"`
	RetentionPolicies                 *PulsarNamespaceRetentionPolicies         `tfsdk:"retention_policies" json:"retention_policies,omitempty"`
	InactiveTopicPolicies             *PulsarNamespaceInactiveTopicPolicies     `tfsdk:"inactive_topic_policies" json:"inactive_topic_policies,omitempty"`
	SubscriptionExpirationTimeMinutes *int64                                    `tfsdk:"subscription_expiration_time_minutes" json:"subscription_expiration_time_minutes,omitempty"`

}

type PulsarNamespaceRetentionPolicies struct {
	RetentionTimeInMinutes *int32 `tfsdk:"retention_time_in_minutes" json:"retentionTimeInMinutes,omitempty"`
	RetentionSizeInMB      *int64 `tfsdk:"retention_size_in_mb" json:"retentionSizeInMB,omitempty"`
}

type PulsarNamespaceAutoTopicCreationOverride struct {
	AllowAutoTopicCreation *bool   `tfsdk:"allow_auto_topic_creation" json:"allowAutoTopicCreation,omitempty"`
	TopicType              *string `tfsdk:"topic_type" json:"topicType,omitempty"`
	DefaultNumPartitions   *int64  `tfsdk:"default_num_partitions" json:"defaultNumPartitions,omitempty"`
}

type PulsarNamespaceBacklogQuota struct {
	Limit     *int64  `tfsdk:"limit" json:"limit,omitempty"`
	LimitSize *int64  `tfsdk:"limit_size" json:"limitSize,omitempty"`
	LimitTime *int64  `tfsdk:"limit_time" json:"limitTime,omitempty"`
	Policy    *string `tfsdk:"policy" json:"policy,omitempty"`
}

type PulsarNamespaceInactiveTopicPolicies struct {
	DeleteWhileInactive        *bool   `tfsdk:"delete_while_inactive" json:"deleteWhileInactive,omitempty"`
	DeleteMode                 *string `tfsdk:"delete_mode" json:"inactiveTopicDeleteMode,omitempty"`
	MaxInactiveDurationSeconds *int64  `tfsdk:"max_inactive_duration_seconds" json:"maxInactiveDurationSeconds,omitempty"`
}

var (
	boolPulsarNamespacePolicyAttribute = schema.BoolAttribute{
		Optional: true,
		Computed: true,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	}
	int64PulsarNamespacePolicyAttribute = schema.Int64Attribute{
		Optional: true,
		Computed: true,
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
	stringPulsarNamespacePolicyAttribute = schema.StringAttribute{
		Optional: true,
		Computed: true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}

	// Schema definition for Pulsar namespace policies
	pulsarNamespacePoliciesSchema = schema.SingleNestedAttribute{
		Description: "Policies to be applied to the Pulsar namespace. For more details related to valid policy configuration, " +
			"refer to the Pulsar namespace policies documentation (https://pulsar.apache.org/docs/3.0.x/admin-api-namespaces/).",
		Optional: true,
		Computed: true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			policyIsAllowAutoUpdateSchema:               boolPulsarNamespacePolicyAttribute,
			policySchemaAutoUpdateCompatibilityStrategy: stringPulsarNamespacePolicyAttribute,
			policySchemaCompatibilityStrategy:           stringPulsarNamespacePolicyAttribute,
			policySchemaValidationEnforced:              boolPulsarNamespacePolicyAttribute,
			policyMessageTTLInSeconds:                   int64PulsarNamespacePolicyAttribute,
			policyAutoTopicCreationOverride: schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
					AutoTopicCreationOverridePlanModifier{},
				},
				Attributes: map[string]schema.Attribute{
					policyAllowAutoTopicCreation:                boolPulsarNamespacePolicyAttribute,
					policyAutoTopicCreationType:                 stringPulsarNamespacePolicyAttribute,
					policyAutoTopicCreationDefaultNumPartitions: int64PulsarNamespacePolicyAttribute,
				},
			},
			policyBacklogQuotaMap: schema.MapNestedAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						policyBacklogQuotaLimit:     int64PulsarNamespacePolicyAttribute,
						policyBacklogQuotaLimitSize: int64PulsarNamespacePolicyAttribute,
						policyBacklogQuotaLimitTime: int64PulsarNamespacePolicyAttribute,
						policyBacklogQuotaLimitPolicy: schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
							Validators: []validator.String{
								stringvalidator.OneOf([]string{
									"producer_request_hold", "producer_exception", "consumer_backlog_eviction"}...,
								),
							},
						},
					},
				},
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.OneOf([]string{
							"destination_storage", "message_age"}...,
						),
					),
				},
			},
			policyRetentionPolicies: schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					policyRetentionTimeInMinutes: int64PulsarNamespacePolicyAttribute,
					policyRetentionSizeInMB:      int64PulsarNamespacePolicyAttribute,
				},
			},

			

			policyInactiveTopicPolicies: schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					policyInactiveTopicDeleteWhileInactive:        boolPulsarNamespacePolicyAttribute,
					policyInactiveTopicDeleteMode:                 stringPulsarNamespacePolicyAttribute,
					policyInactiveTopicMaxInactiveDurationSeconds: int64PulsarNamespacePolicyAttribute,
				},
			},
			policySubscriptionExpirationTimeMinutes: int64PulsarNamespacePolicyAttribute,
      policySetOffloadThreshold: stringPulsarNamespacePolicyAttribute,

		},
	}
)

func pulsarAdminPoliciesStructToPulsarNamespacePoliciesStruct(p *pulsaradmin.Policies) (*PulsarNamespacePolicies, error) {
	if p == nil {
		return nil, nil
	}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Pulsar admin client namespace policies: %w", err)
	}

	policies := &PulsarNamespacePolicies{}
	err = json.Unmarshal(data, policies)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal Terraform Pulsar namespace policies: %w", err)
	}
	return policies, nil
}

func pulsarNamespacePoliciesStructToPulsarAdminPoliciesStruct(p *PulsarNamespacePolicies) (*pulsaradmin.Policies, error) {
	if p == nil {
		return nil, nil
	}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Terraform Pulsar namespace policies: %w", err)
	}

	policies := &pulsaradmin.Policies{}
	err = json.Unmarshal(data, policies)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal Pulsar admin client namespace policies: %w", err)
	}
	return policies, nil
}

// pulsarNamespacePoliciesObjectToStruct converts from a Terraform object into a pulsar admin namespace policies struct
func pulsarNamespacePoliciesObjectToStruct(ctx context.Context, policyObj types.Object) (*pulsaradmin.Policies, diag.Diagnostics) {

	diags := diag.Diagnostics{}

	objOptions := basetypes.ObjectAsOptions{
		UnhandledUnknownAsEmpty: true,
	}
	policies := &PulsarNamespacePolicies{}
	diags.Append(policyObj.As(ctx, policies, objOptions)...)

	adminClientPolicies, err := pulsarNamespacePoliciesStructToPulsarAdminPoliciesStruct(policies)
	if err != nil {
		diags.AddError("Unable to convert from Terraform Namespace Policies Object to Pulsar admin client struct", err.Error())
		return nil, diags
	}
	return adminClientPolicies, diags
}

func pulsarNamespacePolicyError(policy string) string {
	return fmt.Sprintf("Error setting namespace policy '%v', default values will be used.  Perform a `terraform apply --refresh-only` view the diff", policy)
}

func getPulsarNamespacePolicies(ctx context.Context, pulsarAdminClient *pulsaradmin.ClientWithResponses, plan StreamingNamespaceResourceModel, requestEditors ...pulsaradmin.RequestEditorFn) (types.Object, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	policiesAttrTypes := plan.Policies.AttributeTypes(ctx)

	resp, err := pulsarAdminClient.NamespacesGetPoliciesWithResponse(ctx, plan.Tenant.ValueString(), plan.Namespace.ValueString(), requestEditors...)
	diags.Append(HTTPResponseDiagErrWithBody(resp.StatusCode(), resp.Body, err, "failed to get namespace policies")...)
	if diags.HasError() {
		return types.ObjectNull(policiesAttrTypes), diags
	}

	policiesStruct, err := pulsarAdminPoliciesStructToPulsarNamespacePoliciesStruct(resp.JSON200)
	if err != nil {
		diags.AddError("Failed to convert from Terraform Namespace Policies Object to Pulsar admin client struct", err.Error())
		return types.ObjectNull(policiesAttrTypes), diags
	}
	policiesTerraformObj, objDiags := types.ObjectValueFrom(ctx, policiesAttrTypes, policiesStruct)
	diags.Append(objDiags...)
	return policiesTerraformObj, diags
}

// setNamespacePolicies calls the endpoints for the various policy values to update
// TODO: can this be made more generic?
func setNamespacePolicies(ctx context.Context, client *pulsaradmin.ClientWithResponses, plan StreamingNamespaceResourceModel, requestEditors ...pulsaradmin.RequestEditorFn) diag.Diagnostics {

	tenant := plan.Tenant.ValueString()
	namespace := plan.Namespace.ValueString()

	diags := diag.Diagnostics{}
	if plan.Policies.IsNull() || plan.Policies.IsUnknown() {
		return diags
	}
	policies, policyDiags := pulsarNamespacePoliciesObjectToStruct(ctx, plan.Policies)
	diags.Append(policyDiags...)

	// Individual attributes
	if policies.IsAllowAutoUpdateSchema != nil {
		resp, err := client.NamespacesSetIsAllowAutoUpdateSchema(ctx, tenant, namespace, *policies.IsAllowAutoUpdateSchema, requestEditors...)
		diags.Append(HTTPResponseDiagWarn(resp, err, pulsarNamespacePolicyError(policySchemaValidationEnforced))...)
	}
	if policies.MessageTtlInSeconds != nil {
		resp, err := client.NamespacesSetNamespaceMessageTTL(ctx, tenant, namespace, *policies.MessageTtlInSeconds, requestEditors...)
		diags.Append(HTTPResponseDiagWarn(resp, err, pulsarNamespacePolicyError(policyMessageTTLInSeconds))...)
	}
	if policies.SchemaAutoUpdateCompatibilityStrategy != nil {
		resp, err := client.NamespacesSetSchemaAutoUpdateCompatibilityStrategy(ctx, tenant, namespace, string(*policies.SchemaAutoUpdateCompatibilityStrategy), requestEditors...)
		diags.Append(HTTPResponseDiagWarn(resp, err, pulsarNamespacePolicyError(policySchemaAutoUpdateCompatibilityStrategy))...)
	}
	if policies.SchemaCompatibilityStrategy != nil {
		resp, err := client.NamespacesSetSchemaCompatibilityStrategy(ctx, tenant, namespace, string(*policies.SchemaCompatibilityStrategy), requestEditors...)
		diags.Append(HTTPResponseDiagWarn(resp, err, pulsarNamespacePolicyError(policySchemaCompatibilityStrategy))...)
	}
	if policies.SchemaValidationEnforced != nil {
		resp, err := client.NamespacesSetSchemaValidationEnforced(ctx, tenant, namespace, *policies.SchemaValidationEnforced, requestEditors...)
		diags.Append(HTTPResponseDiagWarn(resp, err, pulsarNamespacePolicyError(policySchemaValidationEnforced))...)
	}

	// Nested objects
	if policies.AutoTopicCreationOverride != nil {
		resp, err := client.NamespacesSetAutoTopicCreation(ctx, tenant, namespace, *policies.AutoTopicCreationOverride, requestEditors...)
		diags.Append(HTTPResponseDiagWarn(resp, err, pulsarNamespacePolicyError(policyAutoTopicCreationOverride))...)
	}
	if policies.BacklogQuotaMap != nil {
		for quotaTypeName, quota := range *policies.BacklogQuotaMap {
			quotaType := (pulsaradmin.NamespacesSetBacklogQuotaParamsBacklogQuotaType)(quotaTypeName)
			params := pulsaradmin.NamespacesSetBacklogQuotaParams{BacklogQuotaType: &quotaType}
			resp, err := client.NamespacesSetBacklogQuota(ctx, tenant, namespace, &params, quota, requestEditors...)
			diags.Append(HTTPResponseDiagWarn(resp, err, pulsarNamespacePolicyError(policyBacklogQuotaMap))...)
		}
	}
	if policies.RetentionPolicies != nil {
		resp, err := client.NamespacesSetRetention(ctx, tenant, namespace, *policies.RetentionPolicies, requestEditors...)
		diags.Append(HTTPResponseDiagWarn(resp, err, pulsarNamespacePolicyError(policyRetentionPolicies))...)
	}

	if policies.OffloadThreshold != nil {
		// Set offload threshold
		resp, err := client.NamespacesSetOffloadThreshold(ctx, tenant, namespace, *policies.OffloadThreshold, requestEditors...)
		diags.Append(HTTPResponseDiagWarn(resp, err, pulsarNamespacePolicyError(policySetOffloadThreshold))...)


	if policies.InactiveTopicPolicies != nil {
		resp, err := client.NamespacesSetInactiveTopicPolicies(ctx, tenant, namespace, *policies.InactiveTopicPolicies, requestEditors...)
		diags.Append(HTTPResponseDiagWarn(resp, err, pulsarNamespacePolicyError(policyInactiveTopicPolicies))...)
	}
	if policies.SubscriptionExpirationTimeMinutes != nil {
		resp, err := client.NamespacesSetSubscriptionExpirationTime(ctx, tenant, namespace, *policies.SubscriptionExpirationTimeMinutes, requestEditors...)
		diags.Append(HTTPResponseDiagWarn(resp, err, pulsarNamespacePolicyError(policySubscriptionExpirationTimeMinutes))...)

	}
	return diags
}

// AutoTopicCreationOverridePlanModifier handles special cases related to auto topic creation, such as ensuring that
// the number of partitions is set to null if the topic_type is set to 'non-partitioned'
type AutoTopicCreationOverridePlanModifier struct{}

func (m AutoTopicCreationOverridePlanModifier) Description(_ context.Context) string {
	return "Adjust plan for certain cases in auto topic creation override settings."
}

func (m AutoTopicCreationOverridePlanModifier) MarkdownDescription(_ context.Context) string {
	return "Adjust plan for certain cases in auto topic creation override settings."
}

func (m AutoTopicCreationOverridePlanModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	// Do nothing if there is no state value.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	// If topic type is non-partitioned, the number of partitions must be nil
	if topicType, ok := req.PlanValue.Attributes()[policyAutoTopicCreationType]; ok && CompareTerraformAttrToString(topicType, "non-partitioned") {
		newPlan, diags := UpdateTerraformObjectWithAttr(ctx, req.PlanValue, policyAutoTopicCreationDefaultNumPartitions, types.Int64Null())
		resp.Diagnostics.Append(diags...)
		resp.PlanValue = newPlan
	}
}
