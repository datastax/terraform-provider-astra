package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"os"
	"testing"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = New("1")()
	testAccProviders = map[string]*schema.Provider{
		"astra": testAccProvider,
	}
	configure("dev", testAccProvider)
}

func testAccPreCheck(t *testing.T) {
	if err := os.Getenv("ASTRA_API_TOKEN"); err == "" {
		t.Fatal("ASTRA_API_TOKEN must be set for acceptance tests")
	}
}

func TestProvider(t *testing.T) {
	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}