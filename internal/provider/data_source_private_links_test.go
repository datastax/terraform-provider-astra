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
  database_id = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
  datacenter_id = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
}
`)
}