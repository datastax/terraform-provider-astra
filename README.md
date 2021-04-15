# Terraform Provider for Astra

[Astra](https://astra.datastax.com/register) is DataStas's Serverless Apache Cassandra as a service platform.

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
