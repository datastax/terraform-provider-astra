# Terraform Provider for Astra

[Astra](https://astra.datastax.com/) is DataStax's (serverless) service platform for Apache Cassandra and Apache Pulsar.

## Prerequisites

### Astra

Before using this provider, you will need an existing or new [Astra](https://astra.datastax.com/) account.
You will also need an Astra token which is used for authentication.  You can generate a new token using
the [`Token Management` section](https://docs.datastax.com/en/astra-serverless/docs/getting-started/gs-grant-user-access.html#_generate_an_application_token) on the [Astra Dashboard](https://astra.datastax.com).

### Terraform

You will need [Terraform](https://www.terraform.io/) version 1.0 or higher.

## Getting Started

Reference documentation can be found in the [terraform registry](https://registry.terraform.io/providers/datastax/astra/latest/docs)

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

## Local Development

### Build the provider from source

The build requires [Go](https://golang.org/doc/install) >= 1.15

In order to develop and test this provider, you'll need to configure your local environment
with a custom Terraform [config file](https://developer.hashicorp.com/terraform/cli/config/config-file).
This allows provider plugins to be retrieved from the local file system instead of from the
public servers.

1. Edit or create a .terraformrc file in your `$HOME` directory which includes custom
   `provider_installation` settings.  Note that you will need to manually
   expand `$HOME` to your actual home directory.

       provider_installation {
         # This disables the version and checksum verifications for locally installed astra providers.
         # See: https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers
         dev_overrides {
           "datastax/astra" = "$HOME/go/src/github.com/datastax/terraform-provider-astra/bin"
         }
       }

2. Build the provider binary

       cd $HOME/go/src/github.com/datastax/terraform-provider-astra
       make build

3. Create a new Terraform config file or run an existing one and the locally built
   provider will be used.  You may see a warning about using an unverified binary.

       │ Warning: Provider development overrides are in effect

   Note: `terraform init` should be skipped when developing locally.


### Running the test suite

The tests can be run via Make.

    make testacc

## Documentation Updates

When modifying plugin services, updates to documentation may be required. Once you have changed a service description,
or added or deleted a service, you need to regenerate the docs and commit them with your changes.

#### Generating docs

The tool used to generate documentation is [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs). See the [installation](https://github.com/hashicorp/terraform-plugin-docs#installation)
section for installing the tool. Once installed, generate the docs by running `tfplugindocs` from the root of the project:

```sh
tfplugindocs
```

The tool will build the plugin and generate the docs based on the implementation. Make sure to add the `docs` folder to your commit to include any changes in the docs.
