package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-aws/aws"
	"os"
	"testing"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = New("1")()
	testAccProviders = map[string]*schema.Provider{
		"astra": testAccProvider,
		"aws": aws.Provider(),
	}
	configure("dev", testAccProvider)
}

func testAccPreCheck(t *testing.T) {
	if err := os.Getenv("ASTRA_API_TOKEN"); err == "" {
		t.Fatal("ASTRA_API_TOKEN must be set for acceptance tests")
	}
}

func checkAwsEnv(t *testing.T) {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" ||
		os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		t.Fatal("`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` must be set for acceptance testing")
	}
}

func TestProvider(t *testing.T) {
	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}