---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "astra_streaming_namespace Resource - terraform-provider-astra"
subcategory: ""
description: |-
  A Pulsar Namespace.
---

# astra_streaming_namespace (Resource)

A Pulsar Namespace.

## Example Usage

```terraform
# Generate a random pet name to avoid naming conflicts
resource "random_pet" "server" {}

# Create a new tenant
resource "astra_streaming_tenant" "streaming_tenant" {
  tenant_name         = "my-tenant-${random_pet.server.id}"
  user_email          = "someuser@example.com"
  cloud_provider      = "gcp"
  deletion_protection = false # let terraform destroy the tenant
  region              = "us-central1"
}

# Create a new namespace
resource "astra_streaming_namespace" "streaming_namespace" {
  # Required
  cluster   = astra_streaming_tenant.streaming_tenant.cluster_name
  tenant    = astra_streaming_tenant.streaming_tenant.tenant_name
  namespace = "my-namespace"

  # Optional
  policies = {
    auto_topic_creation_override = {
      allow_auto_topic_creation = true
      default_num_partitions    = 1
      topic_type                = "partitioned" # "partitioned" or "non_partitioned"
    }
    backlog_quota_map = {
      "destination_storage" = {
        "limit" : 500170751,
        "limit_size" : 500170751,
        "limit_time" : 0,
        "policy" : "producer_exception" # "producer_exception" or "producer_request_hold" or "consumer_backlog_eviction"
      }
    }
    is_allow_auto_update_schema = true
    message_ttl_in_seconds      = 0
    retention_policies = {
      retention_size_in_mb      = 0
      retention_time_in_minutes = 0
    }
    schema_auto_update_compatibility_strategy = "Full"
    schema_compatibility_strategy             = "UNDEFINED",
    schema_validation_enforce                 = true
  }
}

# --Formatted Outputs--
# astra_streaming_namespace.streaming_namespace.id
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cluster` (String) Cluster where the tenant is located.
- `namespace` (String) Name of the Pulsar namespace.
- `tenant` (String) Name of the tenant.

### Optional

- `policies` (Attributes) Policies to be applied to the Pulsar namespace. For more details related to valid policy configuration, refer to the Pulsar namespace policies documentation (https://pulsar.apache.org/docs/3.0.x/admin-api-namespaces/). (see [below for nested schema](#nestedatt--policies))

### Read-Only

- `id` (String) Full path to the namespace

<a id="nestedatt--policies"></a>
### Nested Schema for `policies`

Optional:

- `auto_topic_creation_override` (Attributes) (see [below for nested schema](#nestedatt--policies--auto_topic_creation_override))
- `backlog_quota_map` (Attributes Map) (see [below for nested schema](#nestedatt--policies--backlog_quota_map))
- `is_allow_auto_update_schema` (Boolean)
- `message_ttl_in_seconds` (Number)
- `retention_policies` (Attributes) (see [below for nested schema](#nestedatt--policies--retention_policies))
- `schema_auto_update_compatibility_strategy` (String)
- `schema_compatibility_strategy` (String)
- `schema_validation_enforced` (Boolean)

<a id="nestedatt--policies--auto_topic_creation_override"></a>
### Nested Schema for `policies.auto_topic_creation_override`

Optional:

- `allow_auto_topic_creation` (Boolean)
- `default_num_partitions` (Number)
- `topic_type` (String)


<a id="nestedatt--policies--backlog_quota_map"></a>
### Nested Schema for `policies.backlog_quota_map`

Optional:

- `limit` (Number)
- `limit_size` (Number)
- `limit_time` (Number)
- `policy` (String)


<a id="nestedatt--policies--retention_policies"></a>
### Nested Schema for `policies.retention_policies`

Optional:

- `retention_size_in_mb` (Number)
- `retention_time_in_minutes` (Number)

## Import

Import is supported using the following syntax:

```shell
# the ID must be in the format <cluster_name>/<tenant_name>/<namespace_name>
terraform import astra_streaming_namespace.mynamespace1 pulsar-gcp-useast4-staging/acme1/incoming-orders
```