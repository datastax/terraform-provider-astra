package provider

import (
	"fmt"
	"testing"

	"github.com/datastax/terraform-provider-astra/v2/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestStreamingTopic(t *testing.T) {
	// Disable this test by default until test works with non-prod clusters
	checkRequiredTestVars(t, "ASTRA_TEST_STREAMING_TOPIC_TEST_ENABLED")

	t.Parallel()
	tenantName := "terraform-test-" + common.RandomString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccStreamingTopicConfiguration(tenantName),
			},
		},
	})
}

// https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccStreamingTopicConfiguration(tenantName string) string {
	return fmt.Sprintf(`
resource "astra_streaming_tenant" "streaming_tenant_1" {
  tenant_name         = "%s"
  topic               = "default-topic-1"
  region              = "useast-4"
  cloud_provider      = "gcp"
  user_email          = "terraform-test-user@datastax.com"
  deletion_protection = false
}

resource "astra_streaming_topic" "streaming_topic-1" {
  tenant_name        = astra_streaming_tenant.streaming_tenant_1.tenant_name
  topic              = "testtopic"
  region             = "useast-4"
  cloud_provider     = "gcp"
  namespace          = "default"
}

`, tenantName)
}
