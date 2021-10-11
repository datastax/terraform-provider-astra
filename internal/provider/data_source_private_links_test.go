package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestPrivateLinksDataSource(t *testing.T){
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
  database_id = "762c633f-dcde-47fe-8cb1-f6c47f6e9049"
  datacenter_id = "762c633f-dcde-47fe-8cb1-f6c47f6e9049-1"
}
`)
}