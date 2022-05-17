package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestStreamingSink(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccStreamingSinkConfiguration(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccStreamingSinkConfiguration() string {
	return fmt.Sprintf(`
resource "astra_streaming_tenant" "streaming_tenant-1" {
  tenant_name        = "terraform_test"
  topic              = "terraform_test"
  region             = "useast-4"
  cloud_provider     = "gcp"
  user_email         = "seb@datastax.com"
}
resource "astra_streaming_sink" "streaming_sink-1" {
  depends_on            = [ astra_streaming_tenant.streaming_tenant-1 ]
  tenant_name           = "terraform_test"
  topic                 = "terraform_test"
  region                = "useast-4"
  cloud_provider        = "gcp"
  sink_name             = "snowflake"
  retain_ordering       = true
  processing_guarantees = "ATLEAST_ONCE"
  parallelism           = 3
  namespace             = "default"
  sink_configs          = "fix this"
  destination           = "persistent://aws-new/default/snowflake"
  auto_ack              = true
  class_name            = "com.test.othertest"
}
`)
}