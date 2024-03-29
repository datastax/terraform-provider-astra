package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestCDC(t *testing.T) {
	// Disable this test by default until test works with non-prod clusters
	checkRequiredTestVars(t, "ASTRA_TEST_CDC_TEST_ENABLED")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCDCConfiguration(),
			},
		},
	})
}

// https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccCDCConfiguration() string {
	return fmt.Sprintf(`
resource "astra_streaming_tenant" "streaming_tenant-1" {
  tenant_name        = "terraformtest"
  topic              = "terraformtest"
  region             = "useast-4"
  cloud_provider     = "gcp"
  user_email         = "seb@datastax.com"
}
resource "astra_cdc" "cdc-1" {
  depends_on            = [ astra_streaming_tenant.streaming_tenant-1 ]
  database_id           = "5b70892f-e01a-4595-98e6-19ecc9985d50"
  database_name         = "sai_test"
  table                 = "test"
  keyspace              = "sai_test"
  topic_partitions      = 3
  tenant_name           = astra_streaming_tenant.streaming_tenant-1.tenant_name
}

`)
}
