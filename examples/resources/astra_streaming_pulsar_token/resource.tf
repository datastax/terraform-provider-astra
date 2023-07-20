# Generate a random pet name to avoid naming conflicts
resource "random_pet" "pet_name" {}

# Create a new tenant
resource "astra_streaming_tenant" "streaming_tenant" {
  # Required
  tenant_name         = substr("my-tenant-${random_pet.pet_name.id}", 0, 32)
  user_email          = "someuser@example.com"
  cloud_provider      = "gcp"
  deletion_protection = false
  region              = "us-central1"
}

# Create a new pulsar token for the tenant
resource "astra_streaming_pulsar_token" "streaming_token" {
  # Required
  cluster = astra_streaming_tenant.streaming_tenant.cluster_name
  tenant  = astra_streaming_tenant.streaming_tenant.tenant_name

  # Optional
  time_to_live = "30d" # Token will be valid for 30 days
}

output "streaming_token" {
  description = "The streaming token"
  value       = nonsensitive(astra_streaming_pulsar_token.streaming_token.token)
}

# --Formatted Outputs--
# astra_streaming_pulsar_token.streaming_token.id
# astra_streaming_pulsar_token.streaming_token.token
