# Generate a random pet name to avoid naming conflicts
resource "random_pet" "pet_name" {}

# Create a new database
resource "astra_database" "example_db" {
  # Required
  name           = substr("my-database-${random_pet.pet_name.id}", 0, 50)
  keyspace       = "example_keyspace" # optional, 48 characters max
  cloud_provider = "gcp"
  regions        = ["us-central1"]

  # Optional
  deletion_protection = false
  timeouts {
    create = "30m"
    update = "30m"
    delete = "30m"
  }
}

# --Formatted Outputs--
# astra_database.example_db.additional_keyspaces
# astra_database.example_db.cqlsh_url
# astra_database.example_db.data_endpoint_url
# astra_database.example_db.datacenters
# astra_database.example_db.grafana_url
# astra_database.example_db.graphql_url
# astra_database.example_db.node_count
# astra_database.example_db.organization_id
# astra_database.example_db.owner_id
# astra_database.example_db.replication_factor
# astra_database.example_db.status
# astra_database.example_db.total_storage

output "status" {
  description = "Database status"
  value       = astra_database.example_db.status
}

output "cqlsh_url" {
  description = "CQL shell URL"
  value       = astra_database.example_db.cqlsh_url
}
