package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestTable(t *testing.T) {
	checkRequiredTestVars(t, "ASTRA_TEST_DATABASE_ID")
	databaseID := os.Getenv("ASTRA_TEST_DATABASE_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccTableConfiguration(databaseID),
			},
		},
	})
}

// https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccTableConfiguration(databaseID string) string {
	return fmt.Sprintf(`
resource "astra_table" "table-1" {
  table       = "mytable"
  keyspace = "puppies"
  database_id = "%s"
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
`, databaseID)
}
