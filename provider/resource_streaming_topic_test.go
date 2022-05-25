package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestStreamingTopic(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccStreamingTopicConfiguration(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccStreamingTopicConfiguration() string {
	return fmt.Sprintf(`
resource "astra_streaming_topic" "streaming_topic-1" {
  tenant_name        = "terraformtest2"
  topic              = "testtopic"
  region             = "useast-4"
  cloud_provider     = "gcp"
  namespace          = "default"
}

`)
}