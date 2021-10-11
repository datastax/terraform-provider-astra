package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestDataSourcePrivateLinkEndpoints(t *testing.T){
	resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateLinkEndpointsDataSource(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccPrivateLinkEndpointsDataSource() string {
	return fmt.Sprintf(`
data "astra_private_link_endpoints" "dev" {
  database_id = "762c633f-dcde-47fe-8cb1-f6c47f6e9049"
  datacenter_id = "762c633f-dcde-47fe-8cb1-f6c47f6e9049-1"
  endpoint_id = "vpc-5fbb2e34"
}
`)
}