terraform {
  required_providers {
    astra = {
      source = "datastax/astra"
      version = "2.1.0-rc1"
    }
  }
}

variable "token" {}

provider "astra" {
  // This can also be set via ASTRA_API_TOKEN environment variable.
  token = var.token
}
