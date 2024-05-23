# Read in a customer key for a given cloud provider and region
data "astra_customer_key" "key" {
  cloud_provider = "aws"
  region         = "us-east-1"
}
