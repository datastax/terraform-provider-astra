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
  tenant_name        = "terraformtest2"
  topic              = "terraformtest"
  region             = "useast-4"
  cloud_provider     = "gcp"
  user_email         = "seb@datastax.com"
}
resource "astra_cdc" "cdc-1" {
  depends_on            = [ astra_streaming_tenant.streaming_tenant-1 ]
  database_id           = "5b70892f-e01a-4595-98e6-19ecc9985d50"
  database_name         = "sai_test"
  table                 = "test"
  keyspace              = "sai_test"
  topic_partitions      = 3
  tenant_name           = astra_streaming_tenant.streaming_tenant-1.tenant_name
}
resource "astra_streaming_sink" "streaming_sink-1" {
  depends_on            = [ astra_streaming_tenant.streaming_tenant-1, astra_cdc.cdc-1 ]
  tenant_name           = astra_streaming_tenant.streaming_tenant-1.tenant_name
  topic                 = astra_cdc.cdc-1.data_topic
  region                = "useast-4"
  cloud_provider        = "gcp"
  sink_name             = "jdbc-clickhouse"
  retain_ordering       = true
  processing_guarantees = "ATLEAST_ONCE"
  parallelism           = 3
  namespace             = "default"
  sink_configs          = jsonencode({
      "userName": "clickhouse",
      "password": "password",
      "jdbcUrl": "jdbc:clickhouse://fake.clickhouse.url:8123/pulsar_clickhouse_jdbc_sink",
      "tableName": "pulsar_clickhouse_jdbc_sink"
  })
  auto_ack              = true
}
`)
}