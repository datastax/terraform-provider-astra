package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccStreamingNamespaceResource(t *testing.T) {
	clusterName := envVarOrDefault("ASTRA_TEST_STREAMING_CLUSTER_NAME", testDefaultStreamingClusterName)
	tenant := envVarOrDefault("ASTRA_TEST_STREAMING_TENANT_NAME", "terraform-"+randomString(4))
	namespace := "terraform-test-" + randomString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
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

resource "astra_streaming_namespace" "terraform_test" {
  depends_on = [
    astra_streaming_tenant.streaming_tenant_1
  ]
  cluster   = "%s"
  tenant    = "%s"
  namespace = "%s"
}`, cluster, tenant, cluster, tenant, namespace)
}
