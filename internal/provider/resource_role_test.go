package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestRole(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfiguration(),
			},
		},
	})
}

func testAccRoleConfiguration() string {
	return fmt.Sprintf(`
resource "astra_role" "example" {
  role_name = "puppies"
  description = "test role"
  effect = "allow"
  resources = ["drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73"]
  policy = ["db-all-keyspace-create"]
}
`)
}