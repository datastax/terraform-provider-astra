data "astra_pcu_group" "pcu_group" {
  # ...
}

data "astra_pcu_group_associations" "assocs" {
  pcu_group_id = data.astra_pcu_group.pcu_group.id
}

output "datacenters" {
  value = data.astra_pcu_group_associations.assocs.results[*].datacenter_id
}
