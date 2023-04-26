package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
resource "astra_streaming_tenant" "streaming_tenant-1" {
  tenant_name         = "%s"
  topic               = "topic-1"
  region              = "useast-4"
  cloud_provider      = "gcp"
  user_email          = "terraform-test-user@datastax.com"
  deletion_protection = false
}
`, tenantName)
}
