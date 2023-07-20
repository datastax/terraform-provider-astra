# Generate a random pet name to avoid naming conflicts
resource "random_pet" "pet_name" {}

# Create a new database
resource "astra_database" "example_db" {
  # Required
  name                = substr("my-database-${random_pet.pet_name.id}", 0, 50)
  keyspace            = "example_keyspace"
  cloud_provider      = "gcp"
  regions             = ["us-central1"]
  deletion_protection = false
}

# Create a new table
resource "astra_table" "example_table" {
  # Required
  keyspace           = astra_database.example_db.keyspace
  database_id        = astra_database.example_db.id
  region             = astra_database.example_db.regions[0]
  table              = "a_table_of_data"
  clustering_columns = "a:b"
  partition_keys     = "c:d"
  column_definitions = [
    {
      Name : "a"
      Static : false
      TypeDefinition : "text"
    },
    {
      Name : "b"
      Static : false
      TypeDefinition : "text"
    },
    {
      Name : "c"
      Static : false
      TypeDefinition : "text"
    },
    {
      Name : "d"
      Static : false
      TypeDefinition : "text"
    },
    {
      Name : "e"
      Static : false
      TypeDefinition : "text"
    },
    {
      Name : "f"
      Static : false
      TypeDefinition : "text"
    }
  ]
}
