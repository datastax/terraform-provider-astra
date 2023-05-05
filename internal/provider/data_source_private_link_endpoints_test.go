package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestDataSourcePrivateLinkEndpoints(t *testing.T) {
	checkRequiredTestVars(t, "ASTRA_TEST_DATABASE_ID", "ASTRA_TEST_DATACENTER_ID", "ASTRA_TEST_ENDPOINT_ID")
	databaseID := os.Getenv("ASTRA_TEST_DATABASE_ID")
	datacenterID := os.Getenv("ASTRA_TEST_DATACENTER_ID")
	endpointID := os.Getenv("ASTRA_TEST_ENDPOINT_ID")

	id.UniqueId()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateLinkEndpointsDataSource(databaseID, datacenterID, endpointID),
			},
		},
	})
}

// https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccPrivateLinkEndpointsDataSource(databaseID, datacenterID, endpointID string) string {
	return fmt.Sprintf(`
data "astra_private_link_endpoints" "dev" {
  database_id = "%s"
  datacenter_id = "%s"
  endpoint_id = "%s"
}
`, databaseID, datacenterID, endpointID)
}
