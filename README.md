# Terraform Provider for Astra

[Astra](https://astra.datastax.com/) is DataStax's (serverless) service platform for Apache Cassandra and Apache Pulsar.

## Prerequisites

### Astra

Before using this provider, you will need an existing or new [Astra account](https://astra.datastax.com/register).
You will also need an Astra token which is used for authentication.  You can generate a new token
using the [Astra Dashboard](https://astra.datastax.com).

### Terraform

You will need [Terraform](https://www.terraform.io/) version 0.13.x or higher.

## Getting Started

### Create a new Astra database using terraform

1. Create a file called `main.tf` in a new directory:

    ```hcl
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

    resource "astra_database" "example" {
      name           = "mydb1"
      keyspace       = "ks1"
      cloud_provider = "gcp"
      regions        = ["us-east1"]
    }
    ```

2. Initialize terraform

       terraform init

3. Preview the changes

       terraform plan

4. Create resources

       terraform apply

   If the changes look ok, then approve the changes with `yes`.

5. Wait for the resources to be created.  The new database should be visible
   in the [Astra Dashboard](https://astra.datastax.com/) .

## Examples

The [examples diretory](./examples) contains example configuration for the various resources.

## Building from Source

The build requires [Go](https://golang.org/doc/install) >= 1.15

The provider code can be built using `make`.

    make

### Documentation Updates

When modifying plugin services, updates to documentation may be required. Once you have changed a service description,
or added or deleted a service, you need to regenerate the docs and commit them with your changes.

#### Generating docs

The tool used to generate documentation is [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs). See the [installation](https://github.com/hashicorp/terraform-plugin-docs#installation)
section for installing the tool. Once installed, generate the docs by running `tfplugindocs` from the root of the project:

```sh
tfplugindocs
```

The tool will build the plugin and generate the docs based on the implementation. Make sure to add the `docs` folder to your commit to include any changes in the docs.
