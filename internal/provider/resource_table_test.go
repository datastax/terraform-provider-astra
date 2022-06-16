package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestTable(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccTableConfiguration(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccTableConfiguration() string {
	return fmt.Sprintf(`
resource "astra_database" "dev" {
  name           = "puppies"
  keyspace       = "puppies"
  cloud_provider = "gcp"
  regions        = ["us-east1"]
}
resource "astra_table" "table-1" {
  table       = "mytable"
  keyspace = "puppies"
  database_id = astra_database.dev.id
  region = "us-east1"
  clustering_columns = "a:b"
  partition_keys = "c:d"
  column_definitions= [
    {
      Name: "a"
      Static: false
      TypeDefinition: "text"
    },
    {
      Name: "b"
      Static: false
      TypeDefinition: "text"
    },
    {
      Name: "c"
      Static: false
      TypeDefinition: "text"
    },
    {
      Name: "d"
      Static: false
      TypeDefinition: "text"
    },
    {
      Name: "e"
      Static: false
      TypeDefinition: "text"
    },
    {
      Name: "f"
      Static: false
      TypeDefinition: "text"
    }
  ]
}
`)
}