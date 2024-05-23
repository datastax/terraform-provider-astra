# AWS example
data "astra_cloud_accounts" "awsaccounts" {
  cloud_provider = "aws"
  region         = "us-east-1"
}

# GCP example
data "astra_cloud_accounts" "gcpaccounts" {
  cloud_provider = "gcp"
  region         = "us-east1"
}
