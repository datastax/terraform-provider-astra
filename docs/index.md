---
layout: ""
page_title: "Provider: DataStax Astra - Serverless Cassandra DBaaS"
description: |-
  The Astra provider provides Terraform resources to interact with DataStax Astra databases.
---

# DataStax Astra Provider

  This provider allows DataStax Astra users to manage their full database lifecycle for Astra Serverless databases (built on Apache Cassandra(TM))
  using Terraform.

  To get started, log into [Astra](https://astra.datastax.com/register) and create an authorization token (in your organization settings). The provider will prompt you for the token
  on apply if it does not detect it in your environment variable `ASTRA_API_TOKEN`.

  Currently Astra Streaming (based on Apache Pulsar) is not supported.

## Example Usage

  ```terraform
terraform {
  required_providers {
    astra = {
      source = "datastax/astra"
    }
  }
}

variable "token" {}

provider "astra" {
  // This can also be set via ASTRA_API_TOKEN environment variable.
  token = var.token
}
```

## Additional Info

To report bugs or feature requests [file an issue on github](https://github.com/datastax/terraform-provider-astra/issues).

For general help contact [support](https://houston.datastax.com/).