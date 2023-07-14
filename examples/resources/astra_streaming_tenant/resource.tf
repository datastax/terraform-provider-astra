# Generate a random pet name to avoid naming conflicts
resource "random_pet" "pet_name" {}

# Create a new tenant
resource "astra_streaming_tenant" "streaming_tenant" {
  # Required
  tenant_name           = "my-tenant-${random_pet.pet_name.id}"
  user_email            = "someuser@example.com"

  # Optional
  cloud_provider        = "gcp"
  deletion_protection   = false # let terraform destroy the tenant
  region                = "us-central1"
}

# --Formatted Outputs--
# astra_streaming_tenant.example_tenant.broker_service_url
# astra_streaming_tenant.example_tenant.id
# astra_streaming_tenant.example_tenant.tenant_id
# astra_streaming_tenant.example_tenant.user_metrics_url
# astra_streaming_tenant.example_tenant.web_service_url
# astra_streaming_tenant.example_tenant.web_socket_query_param_url
# astra_streaming_tenant.example_tenant.web_socket_url
