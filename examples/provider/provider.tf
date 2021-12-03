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
