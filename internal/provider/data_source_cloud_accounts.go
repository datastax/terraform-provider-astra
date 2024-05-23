package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceCloudAccounts() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieve a list of Cloud Accounts within an Organization",

		ReadContext: dataSourceCloudAccountsRead,

		Schema: map[string]*schema.Schema{
			// Required inputs
			"cloud_provider": {
				Description:      "The cloud provider where the Customer Key exists (Currently supported: aws, gcp)",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateFunc:     validation.StringInSlice(availableBYOKCloudProviders, true),
				DiffSuppressFunc: ignoreCase,
			},
            "region": {
				Description:  "Cloud provider region",
				Type:         schema.TypeString,
				Required:     true,
			},
			// Computed outputs
			"results": {
				Type:        schema.TypeList,
				Description: "The list of Cloud Accounts for the given Organization.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"organization_id": {
							Description: "Organization ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"provider": {
							Description: "The cloud provider",
							Type:        schema.TypeString,
							Required:    true,
						},
						"provider_id": {
							Description: "The provider account ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceCloudAccountsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	provider := d.Get("cloud_provider").(string)
	region := d.Get("region").(string)

	cloudAccounts, err := listCloudAccounts(ctx, client, provider, region)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("results", cloudAccounts); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.UniqueId())
	return nil
}

func listCloudAccounts(ctx context.Context, client *astra.ClientWithResponses, cloudProvider, region string) ([]map[string]interface{}, error) {
	resp, err := client.GetCloudAccountsWithResponse(ctx, cloudProvider, region)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Error fetching Customer Keys. Status: %d, Message: %s", resp.StatusCode(), (resp.Body))
	}
	cloudAccounts := resp.JSON200
	result := make([]map[string]interface{}, 0, len(*cloudAccounts))
	for _, account := range *cloudAccounts {
		result = append(result, map[string]interface{}{
			"organization_id" : account.OrganizationId,
			"provider"        : account.Provider,
			"provider_id"     : account.ProviderId,
		})
	}
	return result, nil
}