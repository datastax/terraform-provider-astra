# Get all available "serverless" regions. With no filtering only "serverless" regions are returned.
data "astra_available_regions" "serverless_regions" {
}

# Get all available "serverless" regions, with explicit filtering for "serverless"
data "astra_available_regions" "serverless_regions" {
  region_type = "serverless"
}

# Get all available "vector" regions
data "astra_available_regions" "vector_regions" {
  region_type = "vector"
}

# Get all available regions, regardless of type
data "astra_available_regions" "all_regions" {
  region_type = "all"
}

# Filter regions by cloud provider (one of "aws", "azure", "gcp")
data "astra_available_regions" "aws_serverless_regions" {
  cloud_provider = "aws"
}

# Filter regions by enabled status
data "astra_available_regions" "enabled_regions" {
  only_enabled = true
}

# Filter only enabled regions in GCP for vector
data "astra_available_regions" "gcp_vector_regions" {
  region_type    = "vector"
  cloud_provider = "gcp"
  only_enabled   = true
}
