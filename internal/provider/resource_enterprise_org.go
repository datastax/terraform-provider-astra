package provider

import (
	"context"
	"errors"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceEnterpriseOrg() *schema.Resource {
	return &schema.Resource{
		Description:   "`enterprise_org` resource represents an Organization that is created under an Enterprise in Astra.",
		CreateContext: resourceEnterpriseOrgCreate,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Required
			"name": {
				Description: "Organization name",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"email": {
				Description: "Organization email address",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"admin_user_id": {
				Description: "UUID of the Astra user that will be the admin of the organization",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"enterprise_id": {
				Description: "UUID of the Enterprise under which the organization should be created",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			// Computed
			"organization_id": {
				Description:    "The Astra organization ID (UUID) for the created Enterprise organization.",
				Type:           schema.TypeString,
				Computed:       true,
			},
			"organization_type": {
				Description:    "The type of the organization.",
				Type:           schema.TypeString,
				Computed:       true,
			},
			"organization_group_id": {
				Description:    "The group ID (UUID) of the organization.",
				Type:           schema.TypeString,
				Computed:       true,
			},
			"created_at": {
				Description:    "The timestamp when the organization was created.",
				Type:           schema.TypeString,
				Computed:       true,
			},
			"last_modified": {
				Description:    "The timestamp when the organization was last modified.",
				Type:           schema.TypeString,
				Computed:       true,
			},
			// TODO: Add MarketPlaceData
		},
	}
}

func resourceEnterpriseOrgCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	orgName := d.Get("name").(string)
	orgEmail := d.Get("email").(string)
	adminUid := d.Get("admin_user_id").(string)
	enterpriseId := d.Get("enterprise_id").(string)

	orgReq := astra.CreateOrganizationInEnterpriseJSONRequestBody{
		Name:         orgName,
		Email:        orgEmail,
		AdminUserID:  adminUid,
		EnterpriseID: enterpriseId,
	}

	resp, err := client.CreateOrganizationInEnterpriseWithResponse(ctx, orgReq)
	if err != nil {
		return diag.FromErr(err)
	} else if resp.StatusCode() != 201 {
		return diag.Errorf("error adding Organization to Enterprise: Status: %s, %s", resp.Status(), resp.Body)
	}

	enterpriseOrg := resp.JSON201

	d.SetId(*enterpriseOrg.OrganizationID)
	if err := setEnterpriseOrgData(d, enterpriseOrg); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func setEnterpriseOrgData(d *schema.ResourceData, org *astra.CreateOrgInEnterpriseResponse) error {
	if org == nil {
		return errors.New("organization is nil")
	}

	d.Set("organization_id", *org.OrganizationID)
	d.Set("organization_type", *org.OrgType)
	d.Set("organization_group_id", *org.OrganizationGroupId)
	d.Set("created_at", *org.CreatedAt)
	d.Set("last_modified", *org.LastModified)

	return nil
}
