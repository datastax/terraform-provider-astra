package provider

import (
	"encoding/json"
	"fmt"
	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccessList(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAccessListConfiguration(),
			},
		},
	})
}

func testAccAccessListConfiguration() string {
	return fmt.Sprintf(`
resource "astra_access_list" "example" {
  database_id = "f6e6b500-61a0-48d5-a29f-3406d28974ee"
  addresses {
    request {
      address= "0.0.0.1/0"
      enabled= true
    }
    request {
      address= "0.0.0.2/0"
      enabled= true
    }
    request {
      address= "0.0.0.3/0"
      enabled= true
    }
  }
  enabled = true
}
`)
}

func TestTimeUnmarshal(t *testing.T) {
	msg := `{"lastUpdateDateTime":"2021-08-03 15:20:29.008 +0000 UTC"}`
	//msg := `{"lastUpdateDateTime":"2021-08-03T15:20:29Z"}`
	bodyBytes := []byte(msg)
	type TestStruct struct{
		LastUpdateDateTime *string
	}
	var dest TestStruct
	if err := json.Unmarshal(bodyBytes, &dest); err != nil {
		fmt.Printf("fail with error: %s",err)
		return
	}
	fmt.Printf("succeed")
}

func TestMsgUnmarshal(t *testing.T) {
	msg := `{"organizationId":"f9f4b1e0-4c05-451e-9bba-d631295a7f73","databaseId":"aba3cf20-d579-4091-a36d-9c9f75096031","addresses":[{"address":"0.0.0.0/0","description":"","enabled":true,"lastUpdateDateTime":"2021-08-03 15:20:29.008 +0000 UTC"}],"configurations":{"accessListEnabled":false}}`
	bodyBytes := []byte(msg)

	var dest astra.AccessListResponse
	if err := json.Unmarshal(bodyBytes, &dest); err != nil {
		fmt.Printf("fail with error: %s",err)
		return
	}

	fmt.Printf("succeed")

}

func TestMsgNewStructMarshal(t *testing.T){
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
		fmt.Printf("fail with error: %s",err)
		return
	}

	fmt.Printf("succeed")


}
