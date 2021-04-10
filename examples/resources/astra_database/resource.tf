resource "astra_database" "example" {
  name           = "name"
  keyspace       = "keyspace"
  cloud_provider = "gcp"
  region         = "us-east1"
}