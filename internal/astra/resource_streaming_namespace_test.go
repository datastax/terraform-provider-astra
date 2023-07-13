package astra

import (
	"fmt"
	"testing"

	"github.com/datastax/terraform-provider-astra/v2/internal/util"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccStreamingNamespaceResource(t *testing.T) {
	clusterName := util.EnvVarOrDefault("ASTRA_TEST_STREAMING_CLUSTER_NAME", testDefaultStreamingClusterName)
	tenant := util.EnvVarOrDefault("ASTRA_TEST_STREAMING_TENANT_NAME", "terraform-"+util.RandomString(4))
	namespace := "tf-test-" + util.RandomString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testProviderConfig + "\n" + streamingNamespaceTestConfig(clusterName, tenant, namespace),
			},
		},
	})
}

func streamingNamespaceTestConfig(cluster, tenant, namespace string) string {
	return fmt.Sprintf(`
resource "astra_streaming_tenant" "streaming_tenant_1" {
	cluster_name        = "%s"
	tenant_name         = "%s"
	user_email          = "terraform-test-user@datastax.com"
	deletion_protection = false
}

resource "astra_streaming_namespace" "test_namespace" {
  depends_on = [
    astra_streaming_tenant.streaming_tenant_1
  ]
  cluster   = astra_streaming_tenant.streaming_tenant_1.cluster_name
  tenant    = astra_streaming_tenant.streaming_tenant_1.tenant_name
  namespace = "%s"
}

resource "astra_streaming_namespace" "test_namespace_with_policies" {
  depends_on = [
    astra_streaming_tenant.streaming_tenant_1
  ]
  cluster   = astra_streaming_tenant.streaming_tenant_1.cluster_name
  tenant    = astra_streaming_tenant.streaming_tenant_1.tenant_name
  namespace = "%s-with-policies"
  policies = {
    schema_validation_enforced = true
    auto_topic_creation_override = {
      allow_auto_topic_creation = true
	  topic_type = "non-partitioned"
    }
    retention_policies = {
      retention_size_in_mb = -1
      retention_time_in_minutes = 4
    }
  }
}
`, cluster, tenant, namespace, namespace)
}
