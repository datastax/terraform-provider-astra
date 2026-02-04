# Create a basic PCU group
# NOTE: This creates a committed reserved-capacity group! Make sure this is what you want!
resource "astra_pcu_group" "example" {
  title             = "production-pcu-group"
  cloud_provider    = "AWS"
  region            = "us-east-1"
  min_capacity      = 2
  max_capacity      = 10
  reserved_capacity = 2
  description       = "PCU group for production databases"
  # You can uncomment the line below to park the group
  # The group must be created first as a resource before parking can be performed successfully
  # park = true
}

# Create a PCU group with custom settings
resource "astra_pcu_group" "custom" {
  title          = "dev-pcu-group"
  cloud_provider = "GCP"
  region         = "us-central1"
  cache_type     = "STANDARD"
  provision_type = "SHARED"
  min_capacity   = 1
  max_capacity   = 5
  description    = "PCU group for development environment"

  # Disable deletion protection for dev environment
  deletion_protection = false

  # Enable reserved capacity protection
  reserved_protection = true
}

output "pcu_group_id" {
  value = astra_pcu_group.example.id
}

output "pcu_group_status" {
  value = astra_pcu_group.example.status
}
