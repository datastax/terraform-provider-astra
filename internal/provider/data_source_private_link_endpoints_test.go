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
  database_id = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
  datacenter_id = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
  endpoint_id = "vpce-04cf066a99d812a8d"
}
`)
}