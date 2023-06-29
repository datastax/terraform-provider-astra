package astra

import (
	"fmt"
	"testing"

	"github.com/datastax/terraform-provider-astra/v2/internal/util"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccStreamingPulsarTokenResource(t *testing.T) {
	clusterName := util.EnvVarOrDefault("ASTRA_TEST_STREAMING_CLUSTER_NAME", testDefaultStreamingClusterName)
	tenant := util.EnvVarOrDefault("ASTRA_TEST_STREAMING_TENANT_NAME", "terraform-"+util.RandomString(4))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
  cluster   = astra_streaming_tenant.streaming_tenant_1.cluster_name
  tenant    = astra_streaming_tenant.streaming_tenant_1.tenant_name
}`, cluster, tenant)
}
