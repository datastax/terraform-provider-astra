# Get all PCU groups in the organization
data "astra_pcu_groups" "all" {
}

output "all_pcu_groups" {
  value = data.astra_pcu_groups.all.results[*].title
}

# Get specific PCU groups by ID
data "astra_pcu_groups" "specific" {
  pcu_group_ids = [
    "6c57916e-7bd8-4bb6-b264-bae906c8859f",
    "7d68027f-8ce9-5cc7-c375-cbf017d9960g"
  ]
}

output "specific_pcu_groups" {
  value = {
    for group in data.astra_pcu_groups.specific.results : group.id => {
      title          = group.title
      cloud_provider = group.cloud_provider
      region         = group.region
      status         = group.status
    }
  }
}
