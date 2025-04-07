/*
NOTE:

The streaming tenant and database will be created at the same time because they have no dependent resources in the flow.
The table will then be created, and then the CDC connection after that. This all follows terraform dependency rules.
*/

# Generate a random pet name to avoid naming conflicts
resource "random_uuid" "identifier" {}

# Create a new tenant
resource "astra_streaming_tenant" "streaming_tenant" {
  tenant_name         = substr("webstore-clicks-${random_uuid.identifier.id}", 0, 32)
  user_email          = "someuser@example.com"
  cloud_provider      = "gcp"
  deletion_protection = false
  region              = "us-central1"
}

# Create a new database
resource "astra_database" "db_database" {
  name                = substr("webstore-clicks-${random_uuid.identifier.id}", 0, 50)
  keyspace            = "click_data"    # 48 characters max
  cloud_provider      = "gcp"           # this must match the cloud_provider of the tenant
  regions             = ["us-central1"] # this must match the region of the tenant
  deletion_protection = false
}

# Create a new table in that database
resource "astra_table" "db_table" {
  keyspace           = astra_database.db_database.keyspace
  database_id        = astra_database.db_database.id
  region             = astra_database.db_database.regions[0]
  table              = "all_product_clicks"
  clustering_columns = "visitor_id:click_timestamp"
  partition_keys     = "visitor_id:click_url"
  column_definitions = [
    {
      Name : "click_timestamp"
      Static : false
      TypeDefinition : "bigint"
    },
    {
      Name : "visitor_id"
      Static : false
      TypeDefinition : "uuid"
    },
    {
      Name : "click_url"
      Static : false
      TypeDefinition : "text"
    }
  ]
}

# Create a new CDC connection between tenant topic and db table
resource "astra_cdc" "db_cdc" {
  database_id      = astra_database.db_database.id
  database_name    = astra_database.db_database.name
  table            = astra_table.db_table.table
  keyspace         = astra_database.db_database.keyspace
  pulsar_cluster   = astra_streaming_tenant.cluster_name
  tenant_name      = astra_streaming_tenant.streaming_tenant.tenant_name
  topic_partitions = 3
}

# --Formatted Outputs--
# astra_cdc.db_cdc.connector_status
# astra_cdc.db_cdc.data_topic
# astra_cdc.db_cdc.id
