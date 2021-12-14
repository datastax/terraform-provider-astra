resource "astra_database" "example" {
  name           = "name"
  keyspace       = "keyspace"
  cloud_provider = "gcp"
  regions        = ["us-east1"]
}
