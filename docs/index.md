---
layout: ""
page_title: "Provider: DataStax Astra - Serverless Cassandra DBaaS"
description: |-
  The Astra provider provides Terraform resources to interact with DataStax AstraDB and Astra Streaming, DataStax's cloud offerings based on Apache Cassandra, Apache Pulsar, and Kubernetes.
---

# DataStax Astra Provider

  This provider allows [DataStax Astra](https://astra.datastax.com/) users to manage their full database lifecycle for Astra Serverless databases (built on [Apache Cassandra](https://cassandra.apache.org/))
  using Terraform.

  To get started, log into [Astra](https://astra.datastax.com/register) and create an authorization token (in your organization settings). The provider will prompt you for the token
  on apply if it does not detect it in your environment variable `ASTRA_API_TOKEN`.

  Astra Streaming (based on [Apache Pulsar](https://pulsar.apache.org/)) is now supported.

## Example Usage

  ```terraform
terraform {
  required_providers {
    astra = {
      source = "datastax/astra"
      version = "2.1.15"
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

To report bugs or feature requests for the provider [file an issue on github](https://github.com/datastax/terraform-provider-astra/issues).

For general help contact [support](https://houston.datastax.com/).
