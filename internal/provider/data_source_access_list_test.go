package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestDataSourceAccessListEndpoints(t *testing.T) {
	checkRequiredTestVars(t, "ASTRA_TEST_DATABASE_ID")
	databaseID := os.Getenv("ASTRA_TEST_DATABASE_ID")

	resource.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateAccessListDataSource(databaseID),
			},
		},
	})
}

// https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccPrivateAccessListDataSource(databaseID string) string {
	return fmt.Sprintf(`
data "astra_access_list" "dev" {
  database_id = "%s"
}
`, databaseID)
}
