package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestStreamingTenant(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccStreamingTenantConfiguration(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccStreamingTenantConfiguration() string {
	return fmt.Sprintf(`
resource "astra_streaming_tenant" "streaming_tenant-1" {
  tenant_name        = "terraform_test"
  topic              = "terraform_test"
  region             = "useast4"
  cloud_provider     = "gcp"
  user_email         = "seb@datastax.com"
}

`)
}