# Get all available PCU types
data "astra_pcu_types" "all" {}

output "all_pcu_types" {
  value = {
    for t in data.astra_pcu_types.all.results : "${t.cloud_provider}-${t.region}-${t.type}" => t
  }
}

# Get PCU types filtered by cloud provider and region
data "astra_pcu_types" "filtered" {
  cloud_provider = "aws"
  region         = "us-east-1"
}

output "filtered_pcu_types" {
  value = data.astra_pcu_types.filtered.results
}
