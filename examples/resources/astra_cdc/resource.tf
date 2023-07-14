# Generate a random pet name to avoid naming conflicts
resource "random_uuid" "identifier" {}

# Create a new tenant
resource "astra_streaming_tenant" "streaming_tenant" {
  tenant_name           = "webstore-clicks-${random_uuid.identifier.id}"
  user_email            = "someuser@example.com"
  cloud_provider        = "gcp"
  deletion_protection   = true
  region                = "us-central1"
}

# Create a new database
resource "astra_database" "db_database" {
  name           = "webstore-clicks"
  keyspace       = "click_data"
  cloud_provider = "gcp" # this must match the cloud_provider of the tenant
  regions        = ["us-central1"] # this must match the region of the tenant
}

# Create a new table in that database
resource "astra_table" "db_table" {
  keyspace            = astra_database.db_database.keyspace
  database_id         = astra_database.db_database.id
  region              = astra_database.db_database.regions[0]
  table               = "all_clicks-${random_uuid.identifier.id}"
  clustering_columns  = "click_timestamp:visitor_id"
  partition_keys      = "click_timestamp"
  column_definitions= [
    {
      Name: "click_timestamp"
      Static: false
      TypeDefinition: "bigint"
    },
    {
      Name: "visitor_id"
      Static: false
      TypeDefinition: "uuid"
    },
    {
      Name: "click_url"
      Static: false
      TypeDefinition: "text"
    }
  ]
}

# Create a new CDC connection between tenant topic and db table
resource "astra_cdc" "db_cdc" {
  database_id      = astra_database.db_database.id
  database_name    = astra_database.db_database.name
  table            = astra_table.db_table.table
  keyspace         = astra_database.db_database.keyspace
  tenant_name      = astra_streaming_tenant.streaming_tenant.tenant_name
  topic_partitions = 3
}

# --Formatted Outputs--
# astra_cdc.db_cdc.connector_status
# astra_cdc.db_cdc.data_topic
# astra_cdc.db_cdc.id