package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccessList(t *testing.T) {
	checkRequiredTestVars(t, "ASTRA_TEST_DATABASE_ID")
	databaseID := os.Getenv("ASTRA_TEST_DATABASE_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAccessListDestroy,
		Steps: []resource.TestStep{
			{
				// addresses are intentionally not in sorted order; the post-apply plan must still be empty
				Config: testAccAccessListConfiguration(databaseID, "first", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("astra_access_list.example", "addresses.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("astra_access_list.example", "addresses.*", map[string]string{
						"address":     "198.51.100.10/32",
						"description": "first",
					}),
				),
			},
			{
				// changing an address or the top-level flag should update in place
				Config: testAccAccessListConfiguration(databaseID, "second", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("astra_access_list.example", "enabled", "false"),
					resource.TestCheckTypeSetElemNestedAttrs("astra_access_list.example", "addresses.*", map[string]string{
						"address":     "198.51.100.10/32",
						"description": "second",
					}),
				),
			},
		},
	})
}

func testAccCheckAccessListDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(astraClients).astraClient.(*astra.ClientWithResponses)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "astra_access_list" {
			continue
		}
		accessList, err := listAccessList(context.Background(), client, rs.Primary.ID)
		if err != nil {
			return err
		}
		if accessList != nil {
			return fmt.Errorf("access list for database %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccAccessListConfiguration(databaseID, description string, enabled bool) string {
	return fmt.Sprintf(`
resource "astra_access_list" "example" {
  database_id = "%s"
  addresses {
      address     = "198.51.100.10/32"
      description = "%s"
      enabled     = true
  }
  addresses {
      address = "192.0.2.10/32"
      enabled = true
  }
  addresses {
      address = "192.0.2.20/32"
      enabled = true
  }
  enabled = %t
}
`, databaseID, description, enabled)
}

func TestTimeUnmarshal(t *testing.T) {
	msg := `{"lastUpdateDateTime":"2021-08-03 15:20:29.008 +0000 UTC"}`
	//msg := `{"lastUpdateDateTime":"2021-08-03T15:20:29Z"}`
	bodyBytes := []byte(msg)
	type TestStruct struct {
		LastUpdateDateTime *string
	}
	var dest TestStruct
	if err := json.Unmarshal(bodyBytes, &dest); err != nil {
		fmt.Printf("fail with error: %s", err)
		return
	}
	fmt.Printf("succeed")
}

func TestMsgUnmarshal(t *testing.T) {
	msg := `{"organizationId":"f9f4b1e0-4c05-451e-9bba-d631295a7f73","databaseId":"aba3cf20-d579-4091-a36d-9c9f75096031","addresses":[{"address":"0.0.0.0/0","description":"","enabled":true,"lastUpdateDateTime":"2021-08-03 15:20:29.008 +0000 UTC"}],"configurations":{"accessListEnabled":false}}`
	bodyBytes := []byte(msg)

	var dest astra.AccessListResponse
	if err := json.Unmarshal(bodyBytes, &dest); err != nil {
		fmt.Printf("fail with error: %s", err)
		return
	}

	fmt.Printf("succeed")

}

func TestMsgNewStructMarshal(t *testing.T) {
	type AddressResponse struct {

		// The address (ip address and subnet mask in CIDR notation) of the address to allow
		Address *string `json:"address,omitempty"`

		// Description of this addresses use
		Description *string `json:"description,omitempty"`

		// The indication if the access address is enabled or not
		Enabled *bool `json:"enabled,omitempty"`

		// The last update date/time for the access list
		LastUpdateDateTime *string `json:"lastUpdateDateTime,omitempty"`
	}
	// AccessListConfigurations defines model for AccessListConfigurations.
	type AccessListConfigurations struct {
		AccessListEnabled bool `json:"accessListEnabled"`
	}

	// The response for a requested access list
	type MyAccessListResponse struct {

		// A listing of the allowed addresses
		Addresses      *[]AddressResponse        `json:"addresses,omitempty"`
		Configurations *AccessListConfigurations `json:"configurations,omitempty"`

		// The unique identifier of the database
		DatabaseId *string `json:"databaseId,omitempty"`

		// The unique identifier of the organization
		OrganizationId *string `json:"organizationId,omitempty"`
	}

	msg := `{"organizationId":"f9f4b1e0-4c05-451e-9bba-d631295a7f73","databaseId":"aba3cf20-d579-4091-a36d-9c9f75096031","addresses":[{"address":"0.0.0.0/0","description":"","enabled":true,"lastUpdateDateTime":"2021-08-03 15:20:29.008 +0000 UTC"}],"configurations":{"accessListEnabled":false}}`
	bodyBytes := []byte(msg)

	var dest MyAccessListResponse
	if err := json.Unmarshal(bodyBytes, &dest); err != nil {
		fmt.Printf("fail with error: %s", err)
		return
	}

	fmt.Printf("succeed")

}
