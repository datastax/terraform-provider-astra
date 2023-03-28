resource "astra_database" "example" {
  name           = "name"
  keyspace       = "keyspace"
  cloud_provider = "gcp"
  region        = "us-east1"
  additional_regions = ["us-east4", "us-central1"]
}
