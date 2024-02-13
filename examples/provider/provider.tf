terraform {
  required_providers {
    astra = {
      source  = "datastax/astra"
      version = "2.2.8"
    }
  }
}

# Create a new Astra database
resource "astra_database" "example" {
  name           = "name"
  keyspace       = "keyspace"
  cloud_provider = "gcp"
  regions        = ["us-east1"]
}

# Create a new Astra streaming tenant
resource "astra_streaming_tenant" "example_tenant" {
  tenant_name    = "mystreamingtenant1"
  cloud_provider = "gcp"
  region         = "useast-1"
  user_email     = "someuser@example.com"
}
