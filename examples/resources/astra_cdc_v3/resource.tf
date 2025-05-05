/*
NOTE:

The streaming tenant and database will be created at the same time because they have no dependent resources in the flow.
The table will then be created, and then the CDC connection after that. This all follows terraform dependency rules.
*/

# Generate a random pet name to avoid naming conflicts
resource "random_uuid" "identifier" {}

# Create a new tenant
resource "astra_streaming_tenant" "streaming_tenant1" {
  cluster_name = "pulsar-gcp-uscentral1"
  tenant_name  = "streaming-uscentral1"
  user_email   = "someuser@example.com"
}

resource "astra_streaming_tenant" "streaming_tenant2" {
  cluster_name = "pulsar-gcp-useast1"
  tenant_name  = "streaming-useast1"
  user_email   = "someuser@example.com"
}

# Create a new database
resource "astra_database" "db_database" {
  name                = "mydb"
  keyspace            = "click_data"                # 48 characters max
  cloud_provider      = "gcp"                       # this must match the cloud_provider of the streaming tenants
  regions             = ["us-central1", "us-east1"] # this must match the regions of the streaming tenants
  deletion_protection = false
}

# Create a new table in that database
resource "astra_table" "db_table1" {
  keyspace           = astra_database.db_database.keyspace
  database_id        = astra_database.db_database.id
  region             = astra_database.db_database.regions[0]
  table              = "all_product_clicks"
  clustering_columns = "click_timestamp"
  partition_keys     = "visitor_id:click_url"
  column_definitions = [
    {
      Name : "visitor_id"
      Static : false
      TypeDefinition : "uuid"
    },
    {
      Name : "click_url"
      Static : false
      TypeDefinition : "text"
    },
    {
      Name : "click_timestamp"
      Static : false
      TypeDefinition : "bigint"
    },
  ]
}

# Create a new CDC connection between tenant topic and db table
resource "astra_cdc_v3" "db_cdc" {
  database_id   = astra_database.db_database.id
  database_name = astra_database.db_database.name
  tables = [
    {
      keyspace = astra_table.db_table1.keyspace
      table    = astra_table.db_table1.table
    },
  ]
  regions = [
    {
      region            = "us-central-1"
      datacenter_id     = "${astra_database.db_database.id}-1"
      streaming_cluster = astra_streaming_tenant.streaming_tenant1.cluster_name
      streaming_tenant  = astra_streaming_tenant.streaming_tenant1.tenant_name
    },
    {
      region            = "us-east-1"
      datacenter_id     = "${astra_database.db_database.id}-2"
      streaming_cluster = astra_streaming_tenant.streaming_tenant2.cluster_name
      streaming_tenant  = astra_streaming_tenant.streaming_tenant2.tenant_name
    },
  ]
}
