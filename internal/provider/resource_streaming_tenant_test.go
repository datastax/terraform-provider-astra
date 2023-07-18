package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestStreamingTenant(t *testing.T) {
	t.Parallel()
	tenantName := "terraform-test-" + randomString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccStreamingTenantConfiguration(tenantName),
			},
		},
	})
}

// https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccStreamingTenantConfiguration(tenantName string) string {
	return fmt.Sprintf(`
resource "astra_streaming_tenant" "streaming_tenant_1" {
  tenant_name         = "%s"
  cloud_provider      = "gcp"
  region              = "useast-4"
  user_email          = "terraform-test-user@datastax.com"
  deletion_protection = false
}
`, tenantName)
}

func TestStreamingTenantImport(t *testing.T) {
	t.Parallel()
	tenantName := "terraform-test-" + randomString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccStreamingTenantImport(tenantName),
			},
			{
				ResourceName:     "astra_streaming_tenant.streaming_tenant_2",
				ImportState:      true,
				ImportStateCheck: checkStreamingTenantImportState("azure", "uswest2", tenantName),
				// ImportStateVerify: true,
			},
		},
	})
}

func testAccStreamingTenantImport(tenantName string) string {
	return fmt.Sprintf(`
resource "astra_streaming_tenant" "streaming_tenant_2" {
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
