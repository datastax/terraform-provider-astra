resource "astra_streaming_tenant" "streaming_tenant-1" {
  tenant_name    = "terraformtest"
  topic          = "terraformtest"
  region         = "useast-4"
  cloud_provider = "gcp"
  user_email     = "seb@datastax.com"
}