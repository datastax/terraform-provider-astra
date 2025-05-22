# Terraform Provider for Astra

[Astra](https://astra.datastax.com/) is the DataStax (serverless) service platform for Apache Cassandra and Apache Pulsar.
[Complete API documentation](https://registry.terraform.io/providers/datastax/astra/latest/docs) for this terraform provider is
available in the Terrfarm provider registry.

## Prerequisites

### Astra

Before using this provider, you will need an [Astra](https://astra.datastax.com/) account, and an Astra token for authentication.
From the [Astra Dashboard](https://astra.datastax.com), you can generate a new token using the
[`Token Management` section](https://docs.datastax.com/en/astra-serverless/docs/getting-started/gs-grant-user-access.html#_generate_an_application_token).

### Terraform

You will need [Terraform](https://www.terraform.io/) version 1.5 or higher.

## Getting Started

### Create a new Astra database using terraform

1. Create a file called `main.tf` in a new directory:

    ```hcl
    terraform {
      required_providers {
        astra = {
          source = "datastax/astra"
          version = "2.2.8"
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

The build requires [Go](https://golang.org/doc/install) >= 1.23

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
         direct {
         }
       }

2. Build the provider binary

       cd $HOME/go/src/github.com/datastax/terraform-provider-astra
       make

3. Create a new Terraform config file or run an existing one and the locally built
   provider will be used.  You may see a warning about using an unverified binary.

       │ Warning: Provider development overrides are in effect

   Note: `terraform init` should be skipped when developing locally.


By default, Terraform will run against the public servers.  To run against a test server,
set the following environment variables.

```sh
export ASTRA_API_URL="<Astra test server URL>"
export ASTRA_STREAMING_API_URL="<Astra streaming test server URL>"
export ASTRA_API_TOKEN="<Astra test server Token>"
```

### Running the tests

The tests require several environment variables to be set in order to successfully
run.  By default any tests which are missing the required environment variables
will be skipped.

```sh
export ASTRA_TEST_DATABASE_ID="<Astra database UUID>"
export ASTRA_TEST_DATACENTER_ID="<Astra datacenter id>"
export ASTRA_TEST_ENDPOINT_ID="<Astra endpoint ID>"
```

An example of these variables can be found in the file `test/example-test.env`.  If a
file called `test/test.env` is created it will be automatically loaded by the test script.

The tests can be run via Make.

```sh
make test
```

A single test can be run using golang test args.

```sh
export TESTARGS="-run TestStreamingTenant"
make test
```

## Adding a new resource

This project uses both the [terraform-plugin-sdk](https://github.com/hashicorp/terraform-plugin-sdk) which is now deprecated, and the
newer [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework).  In addition,
[terraform-plugin-mux](https://github.com/hashicorp/terraform-plugin-mux/) is used to allow the sdk and framework to work together.

New resources should use the `terraform-plugin-framework` and should be added under the `internal/astra` directory.
For an example of how to use the `terraform-plugin-framework`, see the [hashicups provider](https://github.com/hashicorp/terraform-provider-hashicups-pf).

## Documentation Updates

When modifying plugin services, updates to documentation may be required. Once you have changed a service description,
or added or deleted a service, you need to regenerate the docs and commit them with your changes.

#### Update Generated docs

The tool used to generate documentation is [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs). The `Makefile` is configured
with a target to generate the docs.

```sh
make docs
```

The tool will build the plugin and generate the docs based on the implementation. Make sure to add the `docs` folder to your commit to include any changes in the docs.
