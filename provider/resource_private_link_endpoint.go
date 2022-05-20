package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourcePrivateLinkEndpoint() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_private_link_endpoint` completes the creation of a private link endpoint by associating it with your endpoint.",
		CreateContext: resourcePrivateLinkEndpointCreate,
		ReadContext:   resourcePrivateLinkEndpointRead,
		DeleteContext: resourcePrivateLinkEndpointDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Required
			"database_id": {
				Description:  "The ID of the Astra database.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				ForceNew: true,
			},
			"datacenter_id": {
				Description:  "Astra datacenter in the region where the private link will be created.",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
			},
			"endpoint_id": {
				Description:      "Endpoint created in your cloud provider account example: \"vpce-svc-1148ea04af8675309\"",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
			},
            // Computed
			"astra_endpoint_id": {
                Description:  "Endpoint ID for referencing within Astra. May be different than the endpoint_id of this resource.",
				Type:         schema.TypeString,
				Computed:     true,
			},
		},
	}
}

func resourcePrivateLinkEndpointCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	databaseID := d.Get("database_id").(string)
	datacenterID := d.Get("datacenter_id").(string)
	endpointID := d.Get("endpoint_id").(string)

	resp, err := client.AcceptEndpointToServiceWithResponse(ctx,
		databaseID,
		datacenterID,
		astra.AcceptEndpointToServiceJSONRequestBody{
			EndpointID: &endpointID,
		},
	)

	if err != nil {
		return diag.FromErr(err)
	} else if resp.StatusCode() >= 400 {
		return diag.Errorf("error adding private link to database: %s", string(resp.Body))
	}

	if err := setPrivateLinkEndpointData(d, databaseID, datacenterID, endpointID, *resp.JSON200.EndpointID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourcePrivateLinkEndpointDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	id := d.Id()
	astraEndpointID := d.Get("astra_endpoint_id")

	databaseID, datacenterID, endpointID, err := parsePrivateLinkEndpointID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	var astraEndpointIDStr string
	if astraEndpointID == nil {
		// set it to the endpointID
		astraEndpointIDStr = endpointID
	} else {
		astraEndpointIDStr = astraEndpointID.(string)
	}

	resp, err := client.RejectEndpoint(ctx, databaseID, datacenterID, astraEndpointIDStr)

	if err != nil {
		return diag.FromErr(err)
	} else if resp.StatusCode >= 400 {
		return diag.Errorf("error removing private link from database: %s", resp.Body)
	}

	return nil
}

func resourcePrivateLinkEndpointRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	id := d.Id()
	astraEndpointID := d.Get("astra_endpoint_id")

	databaseID, datacenterID, endpointID, err := parsePrivateLinkEndpointID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	var astraEndpointIDStr string
	if astraEndpointID == nil {
		// set it to the endpointID
		astraEndpointIDStr = endpointID
	} else {
		astraEndpointIDStr = astraEndpointID.(string)
	}

	privateLinks, err := listPrivateLinkEndpoints(ctx, client, databaseID, datacenterID, astraEndpointIDStr)
	if err != nil {
		return diag.FromErr(err)
	}

	if privateLinks == nil {
        return diag.Errorf("privateLinks was nil. DatabaseID: %s, DatacenterID: %s, endpointID: %s, astraEndpointID: %s", databaseID, datacenterID, endpointID, astraEndpointIDStr)
	}

	if string(*privateLinks.EndpointID) == astraEndpointIDStr {
		if err := setPrivateLinkEndpointData(d, databaseID, datacenterID, endpointID, astraEndpointIDStr); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}

	// Private Link not found. Remove from state.
	d.SetId("")

	return nil
}

func setPrivateLinkEndpointData(d *schema.ResourceData, databaseID string, datacenterID string, endpointID string, astraEndpointID string) error {
	d.SetId(fmt.Sprintf("%s/datacenter/%s/endpoint/%s", databaseID, datacenterID, endpointID))

	if err := d.Set("database_id", databaseID); err != nil {
		return err
	}
	if err := d.Set("datacenter_id", datacenterID); err != nil {
		return err
	}
	if err := d.Set("endpoint_id", endpointID); err != nil {
		return err
	}
    if err := d.Set("astra_endpoint_id", astraEndpointID); err != nil {
		return err
	}
	return nil
}

func parsePrivateLinkEndpointID(id string) (string, string, string, error) {
	re := regexp.MustCompile(`(?P<databaseid>.*)/datacenter/(?P<datacenterid>.*)/endpoint/(?P<endpointid>.*)`)
	if !re.MatchString(id) {
		return "", "", "", errors.New("invalid private link id format: expected dataceneter/endpoint")
	}
	matches := re.FindStringSubmatch(id)
	dbIdIndex := re.SubexpIndex("databaseid")
	dcIdIndex := re.SubexpIndex("datacenterid")
	epIdIndex := re.SubexpIndex("endpointid")
	return matches[dbIdIndex], matches[dcIdIndex], matches[epIdIndex], nil
}
