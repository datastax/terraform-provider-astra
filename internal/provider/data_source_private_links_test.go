package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestPrivateDataSourceLink(t *testing.T){
	resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateLinksDataSource(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccPrivateLinksDataSource() string {
	return fmt.Sprintf(`
data "astra_private_links" "dev" {
  database_id = "8d356587-73b3-430a-9c0e-d780332e2afb"
  datacenter_id = "8d356587-73b3-430a-9c0e-d780332e2afb"
}
`)
}