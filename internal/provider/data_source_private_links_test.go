package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestPrivateLinksDataSource(t *testing.T) {
	checkRequiredTestVars(t, "ASTRA_TEST_DATABASE_ID", "ASTRA_TEST_DATACENTER_ID")
	databaseID := os.Getenv("ASTRA_TEST_DATABASE_ID")
	datacenterID := os.Getenv("ASTRA_TEST_DATACENTER_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateLinksDataSource(databaseID, datacenterID),
			},
		},
	})
}

// https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccPrivateLinksDataSource(databaseID, datacenterID string) string {
	return fmt.Sprintf(`
data "astra_private_links" "dev" {
  database_id = "%s"
  datacenter_id = "%s"
}
`, databaseID, datacenterID)
}
