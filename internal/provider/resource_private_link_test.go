package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestPrivateLink(t *testing.T) {
	checkRequiredTestVars(t, "ASTRA_TEST_DATABASE_ID", "ASTRA_TEST_DATACENTER_ID")
	databaseID := os.Getenv("ASTRA_TEST_DATABASE_ID")
	datacenterID := os.Getenv("ASTRA_TEST_DATACENTER_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateLinkConfiguration(databaseID, datacenterID, `"community-ecosystem"`),
				Check:  resource.TestCheckResourceAttr("astra_private_link.example", "allowed_principals.#", "1"),
			},
			{
				// changing allowed_principals should update in place
				Config: testAccPrivateLinkConfiguration(databaseID, datacenterID, `"community-ecosystem-2", "community-ecosystem"`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("astra_private_link.example", "allowed_principals.#", "2"),
					resource.TestCheckTypeSetElemAttr("astra_private_link.example", "allowed_principals.*", "community-ecosystem-2"),
				),
			},
		},
	})
}

// https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccPrivateLinkConfiguration(databaseID, datacenterID, allowedPrincipals string) string {
	return fmt.Sprintf(`
resource "astra_private_link" "example" {
  allowed_principals = [%s]
  database_id = "%s"
  datacenter_id = "%s"
}
`, allowedPrincipals, databaseID, datacenterID)
}

func TestParsePrivateLinkId(t *testing.T) {
	id := "b504911d-4982-4e45-84c2-607524cb533b/datacenter/b504911d-4982-4e45-84c2-607524cb533b-1/serviceNames/projects/astra-serverless-prod-22/regions/us-east1/serviceAttachments/pl-prod"
	databaseID, datacenterID, serviceName, err := parsePrivateLinkID(id)
	if err != nil {
		t.Logf("Private link ID failed to parse: \"%s\", %s", id, err)
		t.Fail()
	}
	// assert databaseID, dataceneterID and serviceName
	if databaseID != "b504911d-4982-4e45-84c2-607524cb533b" {
		t.Logf("Database ID parsed from private link ID: \"%s\", expected \"%s\"", databaseID, "b504911d-4982-4e45-84c2-607524cb533b")
		t.Fail()
	}
	if datacenterID != "b504911d-4982-4e45-84c2-607524cb533b-1" {
		t.Logf("Datacenter ID parsed from private link ID: \"%s\", expected \"%s\"", datacenterID, "b504911d-4982-4e45-84c2-607524cb533b-1")
		t.Fail()
	}
	if serviceName != "projects/astra-serverless-prod-22/regions/us-east1/serviceAttachments/pl-prod" {
		t.Logf("serviceName parsed from private link ID: \"%s\", expected \"%s\"", serviceName, "projects/astra-serverless-prod-22/regions/us-east1/serviceAttachments/pl-prod")
		t.Fail()
	}
}
