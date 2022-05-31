package provider

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestStreamingSink(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
//			{
//				Config: testAccStreamingSinkConfiguration(),
//			},
			{
				Config: testAccStreamingSnowflakeSinkConfiguration(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccStreamingSinkConfiguration() string {
	tenantName := fmt.Sprintf("terraformtest-%s", uuid.New().String())[0:20]
	return fmt.Sprintf(`
resource "astra_streaming_tenant" "streaming_tenant-1" {
  tenant_name        = "%s"
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
`, tenantName)
}
func testAccStreamingSnowflakeSinkConfiguration() string {
	tenantName := fmt.Sprintf("snowflake-%s", uuid.New().String())[0:20]
	return fmt.Sprintf(`
resource "astra_streaming_tenant" "streaming_tenant-1" {
  tenant_name        = "%s"
  topic              = "snowflake"
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
resource "astra_streaming_topic" "offset_topic" {
  depends_on         = [ astra_streaming_tenant.streaming_tenant-1 ]
  topic              = "offset"
  tenant_name        = astra_streaming_tenant.streaming_tenant-1.tenant_name
  region             = "useast-4"
  cloud_provider     = "gcp"
  namespace          = "default"
}
resource "astra_streaming_sink" "streaming_sink-1" { 
  depends_on            = [ astra_cdc.cdc-1 , astra_streaming_tenant.streaming_tenant-1] 
  tenant_name           = astra_streaming_tenant.streaming_tenant-1.tenant_name
  topic                 = astra_cdc.cdc-1.data_topic 
  region                = "useast-4" 
  cloud_provider        = "gcp" 
  sink_name             = "snowflake" 
  retain_ordering       = true 
  processing_guarantees = "ATLEAST_ONCE" 
  parallelism           = 3 
  namespace             = "astracdc" 
  auto_ack              = true
  sink_configs          = jsonencode({ 
    "lingerTimeMs": "10",
    "batchSize": "10",
    "topic": replace(astra_cdc.cdc-1.data_topic, "persistent://",""), 
    "offsetStorageTopic": format("%s/%s/%s",astra_streaming_tenant.streaming_tenant-1.tenant_name, astra_streaming_topic.offset_topic.namespace, astra_streaming_topic.offset_topic.topic), 
    "kafkaConnectorConfigProperties": {
      "connector.class": "com.snowflake.kafka.connector.SnowflakeSinkConnector",
      "key.converter": "org.apache.kafka.connect.storage.StringConverter",
      "name": "snowflake",
      "snowflake.database.name": "TF_DEMO",
      "snowflake.private.key": "MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDEqq7DaJik3M05QfEB97briX3d1C9cFORKgOSEuGbGb57Gt+BD9/pdZMtMGWRQxeEEeWq+XYsZEZufvPzYqZngK14h0S0ETpyrreG5YShqytg8/yv1rbBhaoTdHM+sCbw/JQMlGZ5uFa70uAKtB1+LTIcI1q0QNq+DWPKOJrBiIRc6TpQ94bp7Ndn/xw3tLKPSscIS+xY9fmN96YRw8scur4Hu9alF3r72mXHxakzuKB77ppeWRJQ8HTJn4XwJ+Yh5CyBL+mG7Mcnou1A/1gJp62PqY6tMGVU9mxfu69Id1mAqjmSptCwJWwDT1qDEsp77VFT8IuG/34lL2NKPuHfBAgMBAAECggEAVZffheaBAMekffYIGY4hS3PElwYhMpdZIF/mlSjYeibcWKpwbcSAb6NNo6otccYdf+AEKCP4RQnXzHbpKLbv5JObXWZ3jDdWkpWT9yWk/I2Z/TolfMCCUnOUrdM7QyndhzHpo3z56fl/8rwfVgUufSbqYltkaPkT/Dt7HYkTHTF8KtQ6afE5o/bglvBT7a9QgZ9SVzMuiAOsGmr9y3i/1+pHwSxeyrSK/Xs+Mn5U7KN9/sEgMSnJi3q6sHAFEWZc0mhnzH3j5zdgCvtaJ68aSeEaefW+xVVnhqw7UNRH3wILL6wu1ICPc5bVQMcX3x4k16Mj2cFXsHEdsF4/lC/ZEQKBgQDrgpwKFKfa8wN7MR+agi82daei16wZmMB1O/DVWMssx611f8cp+Rprj273M1wbFSDvqMwpPtKvcK9n1ZWJpZSZh9MP+wBLqzN6+OgFkx2wSiH+R0RFPatVFtXo9nbj/5G8n8gGSbdD4aDFkhcptu2VOEoafPhAi9in0Kc9sgNSbQKBgQDVxu8MY80yMSjTIK3nvJBAQSxMyQEoAEKlZ7POZ5t8/l1pBL5TSzF3lPSTxIWWlfhzapdzEaUU/TiMf1Wr41qQpEpYKfAA45TfyyYq5vCdE4p+22LivUPoeWFaL4X3wUkiOIXQfRdNqdtFSR9XZ8FJbxqamGTkjh8vOqmW7SsGJQKBgQC1r6IviLXiDL4vyLrn2O0RZ3z2/MlxSc2X47Es9f5wQm9ICVadw+Yk+peRr5ar8gXhvegbHbMt05IOWdCuSwYz13v1hR86QQ5LHUDqJA+wU9CbuWEqxaOq1h4aRiF8TUqiKOYIK9BtVuBP6x9heBUbnDxW6Pgf24M+G5MZ3n3/lQKBgQDIETOrbsOdU7CXVqAqnqiJ2fSxr/QrEYNN9W2rn/8+zXdyT4Qnw9l5xqfWmzdCpPwuV/WBNWQ/7nQ72Pe+tDoP4BHLzQPWcSblAuSnhhZtito0uvEirmqdaOuZUZyZMAVXx3pEkq16e5rAjxyL2ohkR1yojjLuS6wXsVkB7Ng1GQKBgDEdt34KchsMB/YrSc6HPhuCYMRil5aHlrQ4MOz28rZQhdQMPkqeJHPR1f39L/vMuYL9e5YCHDq/k60FjFPxrdRT1makNa2Zl5w95nT52uh2Cf6VsGh6FfNJh+U/iFb5jlzIOu04sXSnUG3WQ43rvJFpMmfx8MvXuaKnWjt0tebL",
      "snowflake.schema.name": "TF_DEMO",
      "snowflake.url.name": "https://MHB84648.us-east-1.snowflakecomputing.com",
      "snowflake.user.name": "tf-snow",
      "value.converter": "com.snowflake.kafka.connector.records.SnowflakeJsonConverter",
      "topic": replace(astra_cdc.cdc-1.data_topic, "persistent://",""), 
      "snowflake.topic2table.map": "terraformtestsnowflake/cdc/data-5b70892f-e01a-4595-98e6-19ecc9985d50-sai_test.test:test",
      "buffer.flush.time": 10,
      "buffer.count.records": 10,
      "buffer.size.bytes": 10
    }
 })
}
`,tenantName, "%s", "%s", "%s")
}