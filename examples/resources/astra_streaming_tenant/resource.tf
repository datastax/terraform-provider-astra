# Generate a random pet name to avoid naming conflicts
resource "random_pet" "pet_name" {}

# Create a new tenant
resource "astra_streaming_tenant" "streaming_tenant" {
  # Required
  tenant_name = substr("my-tenant-${random_pet.pet_name.id}", 0, 32)
  user_email  = "someuser@example.com"

  # Optional
  cloud_provider      = "gcp"
  deletion_protection = false # let terraform destroy the tenant
  region              = "us-central1"
}

output "cluster_name" {
  description = "Cluster name"
  value       = astra_streaming_tenant.streaming_tenant.cluster_name
}

output "tenant_name" {
  description = "Tenant name"
  value       = astra_streaming_tenant.streaming_tenant.tenant_name
}

output "broker_service_url" {
  description = "Broker service"
  value       = astra_streaming_tenant.streaming_tenant.broker_service_url
}

output "web_service_url" {
  description = "Web service"
  value       = astra_streaming_tenant.streaming_tenant.web_service_url
}

output "web_socket_url" {
  description = "Socket service"
  value       = astra_streaming_tenant.streaming_tenant.web_socket_url
}

# --Formatted Outputs--
# astra_streaming_tenant.streaming_tenant.tenant_name
# astra_streaming_tenant.streaming_tenant.cluster_name
# astra_streaming_tenant.streaming_tenant.broker_service_url
# astra_streaming_tenant.streaming_tenant.id
# astra_streaming_tenant.streaming_tenant.tenant_id
# astra_streaming_tenant.streaming_tenant.user_metrics_url
# astra_streaming_tenant.streaming_tenant.web_service_url
# astra_streaming_tenant.streaming_tenant.web_socket_query_param_url
# astra_streaming_tenant.streaming_tenant.web_socket_url
