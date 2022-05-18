resource "astra_streaming_tenant" "streaming_tenant-1" {
  tenant_name    = "terraformtest"
  topic          = "terraformtest"
  region         = "useast-4"
  cloud_provider = "gcp"
  user_email     = "seb@datastax.com"
}
resource "astra_cdc" "cdc-1" {
  depends_on       = [astra_streaming_tenant.streaming_tenant-1]
  database_id      = "5b70892f-e01a-4595-98e6-19ecc9985d50"
  database_name    = "sai_test"
  table            = "test"
  keyspace         = "sai_test"
  topic_partitions = 3
  tenant_name      = astra_streaming_tenant.streaming_tenant-1.tenant_name
}