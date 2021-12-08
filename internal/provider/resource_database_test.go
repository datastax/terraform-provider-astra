package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestDatabase(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabaseConfiguration(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccDatabaseConfiguration() string {
	return fmt.Sprintf(`
resource "astra_database" "dev" {
  name           = "puppies"
  keyspace       = "puppies"
  cloud_provider = "gcp"
  region         = ["us-east1"]
}

data "astra_secure_connect_bundle_url" "dev" {
  database_id = astra_database.dev.id
}
`)
}