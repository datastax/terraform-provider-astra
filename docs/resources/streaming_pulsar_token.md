---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "astra_streaming_pulsar_token Resource - terraform-provider-astra"
subcategory: ""
description: |-
  A Pulsar Token.
---

# astra_streaming_pulsar_token (Resource)

A Pulsar Token.

## Example Usage

```terraform
# Generate a random pet name to avoid naming conflicts
resource "random_pet" "pet_name" {}

# Create a new tenant
resource "astra_streaming_tenant" "streaming_tenant" {
  # Required
  tenant_name         = substr("my-tenant-${random_pet.pet_name.id}", 0, 32)
  user_email          = "someuser@example.com"
  cloud_provider      = "gcp"
  deletion_protection = false
  region              = "us-central1"
}

# Create a new pulsar token for the tenant
resource "astra_streaming_pulsar_token" "streaming_token" {
  # Required
  cluster = astra_streaming_tenant.streaming_tenant.cluster_name
  tenant  = astra_streaming_tenant.streaming_tenant.tenant_name

  # Optional
  time_to_live = "30d" # Token will be valid for 30 days
}

output "streaming_token" {
  description = "The streaming token"
  value       = nonsensitive(astra_streaming_pulsar_token.streaming_token.token)
}

# --Formatted Outputs--
# astra_streaming_pulsar_token.streaming_token.id
# astra_streaming_pulsar_token.streaming_token.token
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cluster` (String) Cluster where the Pulsar tenant is located.
- `tenant` (String) Name of the tenant.

### Optional

- `time_to_live` (String) The relative time until the token expires.  For example 1h, 1d, 1y, etc.

### Read-Only

- `id` (String) Full path to the namespace
- `token` (String, Sensitive) String values of the token

## Import

Import is supported using the following syntax:

```shell
# the ID must be in the format <cluster_name>/<tenant_name>/<namespace_name>
terraform import astra_streaming_pulsar_token.mynamespace1 pulsar-gcp-useast4-staging/acme1/incoming-orders
```