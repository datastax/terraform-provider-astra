resource "astra_database" "dev" {
  name           = "puppies"
  keyspace       = "puppies"
  cloud_provider = "gcp"
  region         = "us-east1"
}

data "astra_secure_connect_bundle_url" "dev" {
  database_id = astra_database.dev.id
}

resource "astra_keyspace" "keyspace-1" {
  name        = "1"
  database_id = astra_database.dev.id

}

resource "astra_keyspace" "keyspace-2" {
  name        = "2"
  database_id = astra_database.dev.id

}

resource "astra_keyspace" "keyspace-3" {
  name        = "3"
  database_id = astra_database.dev.id

}

resource "astra_keyspace" "keyspace-4" {
  name        = "4"
  database_id = astra_database.dev.id

}
