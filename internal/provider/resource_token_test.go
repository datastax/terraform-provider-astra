package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestToken(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenConfiguration(),
			},
		},
	})
}

func testAccTokenConfiguration() string {
	return fmt.Sprintf(`
resource "astra_role" "example" {
  role_name = "example-role"
  description = "test role"
  effect = "allow"
  resources = []
  policy = ["org-db-view"]
}
resource "astra_token" "example" {
  roles = [astra_role.example.role_id]
}
`)
}
