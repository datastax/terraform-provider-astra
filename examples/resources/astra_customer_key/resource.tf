# AWS example
resource "astra_customer_key" "customerkey" {
  cloud_provider = "aws"
  region         = "us-east-1"
  key_id         = "arn:aws:kms:us-east-1:123456789012:key/1a2b3c4d-5e6f-1a2b-3c4d-5e6f1a2b3c4d"
}

# GCP example
resource "astra_customer_key" "customerKey" {
  cloud_provider = "gcp"
  region         = "us-east1"
  key_id         = "projects/my-project/locations/us-east1/keyRings/my-key-ring/cryptoKeys/my-key"
}