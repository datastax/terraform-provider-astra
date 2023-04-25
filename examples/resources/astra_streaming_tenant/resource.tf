resource "astra_streaming_tenant" "example_tenant" {
  tenant_name    = "terraformtest1"
  cloud_provider = "gcp"
  region         = "useast-4"
  user_email     = "someuser@example.com"
}
