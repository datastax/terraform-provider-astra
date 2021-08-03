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
  database_id = "8d356587-73b3-430a-9c0e-d780332e2afb"
  datacenter_id = "8d356587-73b3-430a-9c0e-d780332e2afb"
  endpoint_id = "com.amazonaws.vpce.us-east-1.vpce-svc-03ac5a4b18ee480df"
}
`)
}