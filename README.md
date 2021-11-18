# Terraform Provider for Astra

[Astra](https://astra.datastax.com/register) is DataStax's Serverless Apache Cassandra as a service platform.

## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
-	[Go](https://golang.org/doc/install) >= 1.15

## Using the provider

Obtain a client token from the [Astra Dashboard](https://astra.datastax.com).

Configure provider (provider.tf):

```hcl
provider "astra" {
    token = "your client token"
}
```


## Examples

See examples of `resources.tf` [here](https://github.com/datastax/terraform-provider-astra/tree/main/examples)

To run:

    terraform plan

    terraform apply

    terraform show

## Documentation Updates

When modifying plugin services, updates to documentation may be required. Once you have changed a service description,
or added or deleted a service, you need to regenerate the docs and commit them with your changes.

### Generating docs

The tool used to generate documentation is [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs). See the [installation](https://github.com/hashicorp/terraform-plugin-docs#installation)
section for installing the tool. Once installed, generate the docs by running `tfplugindocs` from the root of the project:

```sh
tfplugindocs
```

The tool will build the plugin and generate the docs based on the implementation. Make sure to add the `docs` folder to your commit to include any changes in the docs.
