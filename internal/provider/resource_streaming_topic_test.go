package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/assert"
)

func TestStreamingTopic(t *testing.T) {

	t.Parallel()
	tenantName := "terraform-test-" + randomString(6)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
  tenant_name         = astra_streaming_tenant.streaming_tenant_1.tenant_name
  topic               = "testtopic1"
  cloud_provider      = "gcp"
  region              = "useast-4"
  namespace           = "default"
  deletion_protection = false
}

resource "astra_streaming_topic" "streaming_topic-2" {
  tenant              = astra_streaming_tenant.streaming_tenant_1.tenant_name
  topic               = "testtopic2"
  cluster             = "pulsar-gcp-useast4-staging"
  namespace           = "default"
  persistent          = true
  partitioned         = true
  num_partitions      = 4
  deletion_protection = false
}

resource "astra_streaming_topic" "streaming_topic-3" {
  tenant_name         = astra_streaming_tenant.streaming_tenant_1.tenant_name
  topic               = "testtopic3"
  cloud_provider      = "gcp"
  region              = "us-east4"
  namespace           = "default"
  persistent          = true
  deletion_protection = false
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

	topicID = "my-cluster:non-persistent://my-tenant/my-namespace/topic1-partition"
	topic, err = parseStreamingTopicID(topicID)
	assert.Nil(t, err)
	assert.Equal(t, "my-cluster", topic.Cluster.ValueString())
	assert.Equal(t, "my-tenant", topic.Tenant.ValueString())
	assert.Equal(t, "my-namespace", topic.Namespace.ValueString())
	assert.Equal(t, "topic1", topic.Topic.ValueString())
	assert.Equal(t, "my-cluster", topic.Cluster.ValueString())
	assert.False(t, topic.Persistent.ValueBool())
	assert.True(t, topic.Partitioned.ValueBool())
}
