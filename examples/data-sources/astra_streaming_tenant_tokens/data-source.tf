data "astra_streaming_tenant_tokens" "tokens" {
  tenant_name  = "mytenant"
  cluster_name = "pulsar-gcp-useast4"
}


// Example referencing astra_streaming_tenant
resource "astra_streaming_tenant" "tenant" {
  tenant_name    = "mytenant"
  topic          = "topic1"
  region         = "us-east4"
  cloud_provider = "gcp"
  user_email     = "user@example.com"
}

data "astra_streaming_tenant_tokens" "tokens" {
  tenant_name  = astra_streaming_tenant.tenant.tenant_name
  cluster_name = astra_streaming_tenant.tenant.cluster_name
}
