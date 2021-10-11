package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestDataSourceAccessListEndpoints(t *testing.T){
	resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateAccessListDataSource(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccPrivateAccessListDataSource() string {
	return fmt.Sprintf(`
data "astra_access_list" "dev" {
  database_id = "762c633f-dcde-47fe-8cb1-f6c47f6e9049"
}
`)
}
