package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

		},
	}
}

func resourcePrivateLinkEndpointCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	databaseID := d.Get("database_id").(string)
	datacenterID := d.Get("datacenter_id").(string)
	endpointID := d.Get("endpoint_id").(string)

	resp, err := client.AcceptEndpointToService(ctx,
		databaseID,
		datacenterID,
		astra.AcceptEndpointToServiceJSONRequestBody{
			EndpointID: &endpointID,
		},
	)

	if err != nil {
		return diag.FromErr(err)
	} else if resp.StatusCode >= 400 {
		return diag.Errorf("error adding private link to database: %s", resp.Body)
	}

	if err := setPrivateLinkEndpointData(d, databaseID, datacenterID, endpointID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourcePrivateLinkEndpointDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	id := d.Id()

	databaseID, datacenterID, endpointID, err := parsePrivateLinkEndpointID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := client.RejectEndpoint(ctx, databaseID, datacenterID, endpointID)

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

	databaseID, datacenterID, endpointID, err := parsePrivateLinkEndpointID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	privateLinks, err := listPrivateLinkEndpoints(ctx, client, databaseID, datacenterID, endpointID)
	if err != nil {
		return diag.FromErr(err)
	}

	if string(*privateLinks.EndpointID) == endpointID {
		if err := setPrivateLinkEndpointData(d, databaseID, datacenterID, endpointID); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}

	// Private Link not found. Remove from state.
	d.SetId("")

	return nil
}

func setPrivateLinkEndpointData(d *schema.ResourceData, databaseID string, datacenterID string, endpointID string) error {
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

	return nil
}

func parsePrivateLinkEndpointID(id string) (string, string, string, error) {
	idParts := strings.Split(strings.ToLower(id), "/")
	if len(idParts) != 5 {
		return "", "", "", errors.New("invalid private link id format: expected datacenter/servicenames")
	}
	return idParts[0], idParts[2], idParts[4], nil
}
