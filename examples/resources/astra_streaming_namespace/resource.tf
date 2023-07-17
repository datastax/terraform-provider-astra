# Generate a random pet name to avoid naming conflicts
resource "random_pet" "server" {}

# Create a new tenant
resource "astra_streaming_tenant" "streaming_tenant" {
  tenant_name           = "my-tenant-${random_pet.server.id}"
  user_email            = "someuser@example.com"
  cloud_provider        = "gcp"
  deletion_protection   = false # let terraform destroy the tenant
  region                = "us-central1"
}

# Create a new namespace
resource "astra_streaming_namespace" "streaming_namespace" {
  # Required
  cluster               = astra_streaming_tenant.streaming_tenant.cluster_name
  tenant                = astra_streaming_tenant.streaming_tenant.tenant_name
  namespace             = "my-namespace"

  # Optional
  policies              = {
    auto_topic_creation_override              = {
      allow_auto_topic_creation = false
      default_num_partitions = 0
      topic_type = "partitioned"
    }
    backlog_quota_map                       = {
      "destination_storage" = {
        "limit": 500170751,
        "limitSize": 500170751,
        "limitTime": 0,
        "policy": "producer_exception" # "producer_exception" or "producer_request_hold" or "consumer_backlog_eviction"
      }
    }
    is_allow_auto_update_schema               = true
    message_ttl_in_seconds                    = 0
    retention_policies                        = {
      retention_size_in_mb = 0
      retention_time_in_minutes = 0
    }
    schema_auto_update_compatibility_strategy = "Full"
    schema_compatibility_strategy             = "UNDEFINED",
    schema_validation_enforce                 = false
  }
}

# --Formatted Outputs--
# astra_streaming_namespace.streaming_namespace.id