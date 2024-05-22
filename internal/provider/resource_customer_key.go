package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var availableBYOKCloudProviders = []string{
	"aws",
	"gcp",
}

func resourceCustomerKey() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_customer_key` provides a Customer Key resource for Astra's Bring Your Own Key (BYOK).",
		CreateContext: resourceCustomerKeyCreate,
		ReadContext:   resourceCustomerKeyRead,
		DeleteContext: resourceCustomerKeyDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Required
			"cloud_provider": {
				Description:      "The cloud provider where the Customer Key exists (Currently supported: aws, gcp)",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateFunc:     validation.StringInSlice(availableBYOKCloudProviders, true),
				DiffSuppressFunc: ignoreCase,
			},
			"key_id": {
				Description:      "Customer Key ID.",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
			},
			"region": {
				Description:     "Region in which the Customer Key exists.",
				Type:            schema.TypeString,
				Required:        true,
				ForceNew:        true,
			},
			// Computed
			"organization_id": {
				Description:    "The Astra organization ID (this is derived from the token used to create the Customer Key).",
				Type:           schema.TypeString,
				Computed:       true,
			},
		},
	}
}

func resourceCustomerKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	cloudProvider := d.Get("cloud_provider").(string)
	keyId := d.Get("key_id").(string)
	region := d.Get("region").(string)

	// build the create Key request
	createKeyReq := &astra.ExternalKMS{}
	if strings.EqualFold("aws", cloudProvider) {
		createKeyReq.Aws = buildAWSKms(region, keyId)
	} else if strings.EqualFold("gcp", cloudProvider) {
		createKeyReq.Gcp = buildGCPKms(region, keyId)
	}
	// create the Customer Key
	resp, err := client.CreateKeyWithResponse(ctx, *createKeyReq)
	if err != nil {
		return diag.FromErr(err)
	}
	if resp.StatusCode() != http.StatusOK {
		return diag.Errorf("Unexpected error creating Customer Key. Status: %d, Message: %s", resp.StatusCode(), string(resp.Body))
	}
	orgId, err := getOrgId(ctx, client)
	if err != nil {
		return diag.FromErr(err)
	}
	// set the data
	if err := setCustomerKeyData(d, orgId, cloudProvider, region, keyId); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceCustomerKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	id := d.Id()

	orgId, cloudProvider, region, keyId, err := parseCustomerKeyId(id)
	if err != nil {
		return diag.FromErr(err)
	}

	setCustomerKeyData(d, orgId, cloudProvider, region, keyId)
	return nil
}

func resourceCustomerKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Delete not yet supported via DevOps API
	return nil
}

func buildAWSKms(region, keyId string) *astra.AWSKMS {
	return &astra.AWSKMS{
		KeyID: &keyId,
		Region: &region,
	}
}

func buildGCPKms(region, keyId string) *astra.GCPKMS {
	return &astra.GCPKMS{
		KeyID: &keyId,
		Region: &region,
	}
}

func setCustomerKeyData(d *schema.ResourceData, orgId, cloudProvider, region, keyId string) error {
	if err := d.Set("organization_id", orgId); err != nil {
		return err
	}
	if err:= d.Set("cloud_provider", cloudProvider); err != nil {
		return err
	}
	if err := d.Set("region", region); err != nil {
		return err
	}
	if err := d.Set("key_id", keyId); err != nil {
		return err
	}

	// generate the resource ID
	// format: <organization_id>/<cloud_provider>/<region>/<key_id>
	d.SetId(fmt.Sprintf("%s/%s/%s/%s", orgId, cloudProvider, region, keyId))
	return nil
}

func getOrgId(ctx context.Context, client *astra.ClientWithResponses) (string, error) {
	// get the current Org ID
	resp, err := client.GetCurrentOrganizationWithResponse(ctx)
	if err != nil {
		return "", err
	}
	return resp.JSON200.Id, nil
}

func parseCustomerKeyId(id string) (string, string, string, string, error) {
	re := regexp.MustCompile(`(?P<orgid>.*)/(?P<cloudprovider>.*)/(?P<region>.*)/(?P<keyid>.*)`)
	if !re.MatchString(id) {
		return "", "", "", "", errors.New("invalid customer key id format: expected <organization_id>/<cloud_provider>/<region>/<key_id>")
	}
	matches := re.FindStringSubmatch(id)
	orgIdIndex := re.SubexpIndex("orgid")
	cloudProviderIndex := re.SubexpIndex("cloudprovider")
	regionIndex := re.SubexpIndex("region")
	keyIdIndex := re.SubexpIndex("keyid")
	return matches[orgIdIndex], matches[cloudProviderIndex], matches[regionIndex], matches[keyIdIndex], nil
}