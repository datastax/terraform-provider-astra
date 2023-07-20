# Generate a random pet name to avoid naming conflicts
resource "random_pet" "pet_name" {}

# Create a new database
resource "astra_database" "example_db" {
  # Required
  name                  = substr( "my-database-${random_pet.pet_name.id}", 0, 50)
  keyspace              = "example_keyspace"
  cloud_provider        = "gcp"
  regions               = ["us-central1"]
  deletion_protection   = false
}

resource "astra_keyspace" "example_keyspace" {
  # Required
  database_id = astra_database.example_db.id
  name        = "example_keyspace_2"
}