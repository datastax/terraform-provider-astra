resource "astra_database" "dev" {
  name           = "puppies"
  keyspace       = "puppies"
  cloud_provider = "gcp"
  regions        = ["us-east1"]
}
resource "astra_table" "table-1" {
  table       = "table"
  keyspace = "puppies"
  database_id = astra_database.dev.id
  region = "us-east1"
  clustering_columns = "a:b"
  partition_keys = "c:d"
  column_definitions= [
    {
      Name: "a"
      Static: false
      TypeDefinition: "text"
    },
    {
      Name: "b"
      Static: false
      TypeDefinition: "text"
    },
    {
      Name: "c"
      Static: false
      TypeDefinition: "text"
    },
    {
      Name: "d"
      Static: false
      TypeDefinition: "text"
    },
    {
      Name: "e"
      Static: false
      TypeDefinition: "text"
    },
    {
      Name: "f"
      Static: false
      TypeDefinition: "text"
    }
  ]
}