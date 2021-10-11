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
resource "astra_database" "dev" {
  name           = "puppies"
  keyspace       = "puppies"
  cloud_provider = "gcp"
  region         = "us-east1"
}

resource "astra_keyspace" "keyspace-1" {
  name        = "ks1"
  database_id = astra_database.dev.id

}

resource "astra_keyspace" "keyspace-2" {
  name        = "ks2"
  database_id = astra_database.dev.id

}

resource "astra_keyspace" "keyspace-3" {
  name        = "ks3"
  database_id = astra_database.dev.id

}

resource "astra_keyspace" "keyspace-4" {
  name        = "ks4"
  database_id = astra_database.dev.id

}
`)
}