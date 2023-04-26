package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestKeyspace(t *testing.T) {
	checkRequiredTestVars(t, "ASTRA_TEST_DATABASE_ID")
	databaseID := os.Getenv("ASTRA_TEST_DATABASE_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKeyspaceConfiguration(databaseID),
			},
		},
	})
}

// https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccKeyspaceConfiguration(databaseID string) string {
	return fmt.Sprintf(`
resource "astra_keyspace" "keyspace-1" {
  name        = "ks1"
  database_id = "%s"

}

resource "astra_keyspace" "keyspace-2" {
  name        = "ks2"
  database_id = "%s"

}

resource "astra_keyspace" "keyspace-3" {
  name        = "ks3"
  database_id = "%s"

}

`, databaseID, databaseID, databaseID)
}
