package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestCDC(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCDCConfiguration(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccCDCConfiguration() string {
	return fmt.Sprintf(`
resource "astra_cdc" "cdc-1" {
  database_id = "5b70892f-e01a-4595-98e6-19ecc9985d50"
  database_name = "sai_test"
  table = "test"
  keyspace = "sai_test"
  topic_partitions = 3
  tenant_name = "sebtest123"
}

`)
}