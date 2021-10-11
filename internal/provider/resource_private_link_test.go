package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestPrivateLink(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateLinkConfiguration(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccPrivateLinkConfiguration() string {
	return fmt.Sprintf(`
resource "astra_private_link" "example" {
  allowed_principals = ["arn:aws:iam::337811753388:user/merkle-terraform"]
  database_id = "aba3cf20-d579-4091-a36d-9c9f75096031"
  datacenter_id = "aba3cf20-d579-4091-a36d-9c9f75096031-1"
}
`)
}
