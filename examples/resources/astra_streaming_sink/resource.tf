# Generate a random pet name to avoid naming conflicts
resource "random_pet" "server" {}

# Create a new tenant
resource "astra_streaming_tenant" "streaming_tenant" {
  tenant_name         = "my-tenant-${random_pet.server.id}"
  user_email          = "someuser@example.com"
  cloud_provider      = "gcp"
  deletion_protection = false
  region              = "us-central1"
}

# Create a new namespace
resource "astra_streaming_namespace" "streaming_namespace" {
  cluster   = astra_streaming_tenant.streaming_tenant.cluster_name
  tenant    = astra_streaming_tenant.streaming_tenant.tenant_name
  namespace = "my-namespace"
}

# Create a new topic
resource "astra_streaming_topic" "streaming_topic" {
  cluster             = astra_streaming_tenant.streaming_tenant.cluster_name
  tenant              = astra_streaming_tenant.streaming_tenant.tenant_name
  namespace           = astra_streaming_namespace.streaming_namespace.namespace
  topic               = "my-topic"
  deletion_protection = false
}

# Create a new sink
# Refer to Astra Streaming documentation for more information on sinks
#   https://docs.datastax.com/en/streaming/streaming-learning/pulsar-io/connectors/index.html
resource "astra_streaming_sink" "streaming_sink" {
  # Required
  cluster               = astra_streaming_tenant.streaming_tenant.cluster_name
  tenant_name           = astra_streaming_tenant.streaming_tenant.tenant_name
  namespace             = astra_streaming_namespace.streaming_namespace.namespace
  sink_name             = "sink1"
  archive               = "builtin://jdbc-clickhouse"
  topic                 = astra_streaming_topic.streaming_topic.topic_fqn
  auto_ack              = true
  parallelism           = 1
  retain_ordering       = false
  processing_guarantees = "ATLEAST_ONCE"
  sink_configs = jsonencode({
    "userName" : "clickhouse",
    "password" : "password",
    "jdbcUrl" : "jdbc:clickhouse://fake.clickhouse.url:8123/pulsar_clickhouse_jdbc_sink",
    "tableName" : "pulsar_clickhouse_jdbc_sink"
  })

  # Optional
  deletion_protection = false
}

# --Formatted Outputs--
# astra_streaming_topic.streaming_sink.id
