package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestStreamingTenant(t *testing.T) {
	t.Parallel()
	tenantName1 := "terraform-test-" + randomString(6)
	tenantName2 := "terraform-test-" + randomString(6)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStreamingTenantConfiguration(tenantName1, tenantName2),
			},
		},
	})
}

// https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccStreamingTenantConfiguration(tenantName1, tenantName2 string) string {
	return fmt.Sprintf(`
resource "astra_streaming_tenant" "streaming_tenant_1" {
  tenant_name         = "%s"
  cloud_provider      = "gcp"
  region              = "useast-4"
  user_email          = "terraform-test-user@datastax.com"
  deletion_protection = false
}
resource "astra_streaming_tenant" "streaming_tenant_2" {
  tenant_name         = "%s"
  cluster_name        = "pulsar-gcp-useast4-staging"
  user_email          = "terraform-test-user@datastax.com"
  deletion_protection = false
}

`, tenantName1, tenantName2)
}

func TestStreamingTenantImport(t *testing.T) {
	t.Parallel()
	tenantName := "terraform-test-" + randomString(6)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStreamingTenantImport(tenantName),
			},
			{
				ResourceName:     "astra_streaming_tenant.streaming_tenant_imported_1",
				ImportState:      true,
				ImportStateId:    "pulsar-azure-westus2-staging/" + tenantName,
				ImportStateCheck: checkStreamingTenantImportState("azure", "uswest2", tenantName),
			},
		},
	})
}

func testAccStreamingTenantImport(tenantName string) string {
	return fmt.Sprintf(`
resource "astra_streaming_tenant" "streaming_tenant_imported_1" {
  tenant_name         = "%s"
  topic               = "default-topic-1"
  cloud_provider      = "azure"
  region              = "us-west-2"
  user_email          = "terraform-test-user@datastax.com"
  deletion_protection = false
}
`, tenantName)
}

func checkStreamingTenantImportState(cloudProvider, region, tenantName string) func(state []*terraform.InstanceState) error {
	return func(state []*terraform.InstanceState) error {
		if len(state) != 1 {
			return fmt.Errorf("expected 1 state, got %d", len(state))
		}
		attributes := state[0].Attributes
		if attributes["cloud_provider"] != cloudProvider {
			return fmt.Errorf("expected cloud_provider to be %s, got %s", cloudProvider, attributes["cloud_provider"])
		}
		if attributes["region"] != region {
			return fmt.Errorf("expected region to be %s, got %s", region, attributes["region"])
		}
		if attributes["tenant_name"] != tenantName {
			return fmt.Errorf("expected tenant_name to be %s, got %s", tenantName, attributes["tenant_name"])
		}
		return nil
	}
}
