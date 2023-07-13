package provider

import (
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

func TestProvider(t *testing.T) {
	if err := NewSDKProvider("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
