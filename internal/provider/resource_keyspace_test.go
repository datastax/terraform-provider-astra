package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestKeyspace(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKeyspaceConfiguration(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccKeyspaceConfiguration() string {
	return fmt.Sprintf(`
resource "astra_keyspace" "keyspace-1" {
  name        = "ks1"
  database_id = "aba3cf20-d579-4091-a36d-9c9f75096031"

}

resource "astra_keyspace" "keyspace-2" {
  name        = "ks2"
  database_id = "aba3cf20-d579-4091-a36d-9c9f75096031"

}

resource "astra_keyspace" "keyspace-3" {
  name        = "ks3"
  database_id = "aba3cf20-d579-4091-a36d-9c9f75096031"

}

resource "astra_keyspace" "keyspace-4" {
  name        = "ks4"
  database_id = "aba3cf20-d579-4091-a36d-9c9f75096031"

}
`)
}