---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "astra_customer_key Resource - terraform-provider-astra"
subcategory: ""
description: |-
  astra_customer_key provides a Customer Key resource for Astra's Bring Your Own Key (BYOK). Note that DELETE is not supported through Terraform currently. A support ticket must be created to delete Customer Keys in Astra.
---

# astra_customer_key (Resource)

`astra_customer_key` provides a Customer Key resource for Astra's Bring Your Own Key (BYOK). Note that DELETE is not supported through Terraform currently. A support ticket must be created to delete Customer Keys in Astra.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cloud_provider` (String) The cloud provider where the Customer Key exists (Currently supported: aws, gcp)
- `key_id` (String) Customer Key ID.
- `region` (String) Region in which the Customer Key exists.

### Read-Only

- `id` (String) The ID of this resource.
- `organization_id` (String) The Astra organization ID (this is derived from the token used to create the Customer Key).