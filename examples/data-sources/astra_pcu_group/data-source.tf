data "astra_pcu_group" "example" {
  pcu_group_id = "6c57916e-7bd8-4bb6-b264-bae906c8859f"
}

output "pcu_group_info" {
  value = {
    title          = data.astra_pcu_group.example.title
    cloud_provider = data.astra_pcu_group.example.cloud_provider
    region         = data.astra_pcu_group.example.region
    min_capacity   = data.astra_pcu_group.example.min_capacity
    max_capacity   = data.astra_pcu_group.example.max_capacity
    status         = data.astra_pcu_group.example.status
  }
}
