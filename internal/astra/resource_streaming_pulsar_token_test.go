package astra

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccStreamingPulsarTokenResource(t *testing.T) {
	clusterName := getEnvVarOrDefault("ASTRA_TEST_STREAMING_CLUSTER_NAME", testDefaultStreamingClusterName)
	tenant := getEnvVarOrDefault("ASTRA_TEST_STREAMING_TENANT_NAME", "terraform-"+randomString(4))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig + "\n" + streamingPulsarTokenTestConfig(clusterName, tenant),
			},
		},
	})
}

func streamingPulsarTokenTestConfig(cluster, tenant string) string {
	return fmt.Sprintf(`
resource "astra_streaming_tenant" "streaming_tenant_1" {
  cluster_name        = "%s"
  tenant_name         = "%s"
  user_email          = "terraform-test-user@datastax.com"
  deletion_protection = false
}

resource "astra_streaming_pulsar_token" "pulsar_token_1" {
  depends_on = [
    astra_streaming_tenant.streaming_tenant_1
  ]
  cluster   = "%s"
  tenant    = "%s"
}`, cluster, tenant, cluster, tenant)
}
