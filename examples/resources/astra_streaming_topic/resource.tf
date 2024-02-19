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
  # Required
  #depend_on ensures that the namespace is created before the creation of the topics
  depends_on = [astra_streaming_namespace.streaming_namespace]
  cluster    = astra_streaming_tenant.streaming_tenant.cluster_name
  tenant     = astra_streaming_tenant.streaming_tenant.tenant_name
  namespace  = astra_streaming_namespace.streaming_namespace.namespace
  topic      = "my-topic"

  # Optional
  deletion_protection = false
  num_partitions      = 2
  partitioned         = true
  persistent          = true
}

# --Formatted Outputs--
# astra_streaming_topic.streaming_topic.id
