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
  database_id = "aba3cf20-d579-4091-a36d-9c9f75096031"
  datacenter_id = "aba3cf20-d579-4091-a36d-9c9f75096031-1"
  endpoint_id = "vpc-5fbb2e34"
}
`)
}