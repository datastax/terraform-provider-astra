package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-aws/aws"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = NewSDKProvider("1")()
	testAccProviders = map[string]*schema.Provider{
		"astra": testAccProvider,
		"aws":   aws.Provider(),
	}
	configure("dev", testAccProvider)
}

func testAccPreCheck(t *testing.T) {
	if err := os.Getenv("ASTRA_API_TOKEN"); err == "" {
		t.Fatal("ASTRA_API_TOKEN must be set for acceptance tests")
	}
}

func TestProvider(t *testing.T) {
	if err := NewSDKProvider("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
