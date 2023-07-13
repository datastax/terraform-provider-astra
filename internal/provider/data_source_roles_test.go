package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestRolessDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccRolesDataSource(),
			},
		},
	})
}

// https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccRolesDataSource() string {
	return fmt.Sprintf(`
data "astra_roles" "dev" {
}
`)
}
