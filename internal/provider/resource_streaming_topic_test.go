package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestStreamingTopic(t *testing.T) {
	// Disable this test by default until test works with non-prod clusters
	checkRequiredTestVars(t, "ASTRA_TEST_STREAMING_TOPIC_TEST_ENABLED")

	t.Parallel()
	tenantName := "terraform-test-" + randomString(5)

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
  topic              = "testtopic1"
  region             = "useast-4"
  cloud_provider     = "gcp"
  namespace          = "default"
}

resource "astra_streaming_topic" "streaming_topic-2" {
  tenant             = astra_streaming_tenant.streaming_tenant_1.tenant_name
  topic              = "testtopic2"
  cluster            = "pulsar-gcp-useast4"
  namespace          = "default"
  persistent         = false
  partitioned        = true
  num_partitions     = 4
}
`, tenantName)
}

func TestParseStreamingTopicID(t *testing.T) {
	topicID := "my-cluster:persistent://my-tenant/my-namespace/topic1"
	topic, err := parseStreamingTopicID(topicID)
	assert.Nil(t, err)
	assert.Equal(t, "my-cluster", topic.Cluster.ValueString())
	assert.Equal(t, "my-tenant", topic.Tenant.ValueString())
	assert.Equal(t, "my-namespace", topic.Namespace.ValueString())
	assert.Equal(t, "topic1", topic.Topic.ValueString())
	assert.Equal(t, "my-cluster", topic.Cluster.ValueString())
	assert.True(t, topic.Persistent.ValueBool())
	assert.False(t, topic.Partitioned.ValueBool())
}
