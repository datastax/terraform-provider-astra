resource "astra_database" "db" {
  # ...
}

resource "astra_pcu_group" "pcu_group" {
  # ...
}

resource "astra_pcu_group_association" "assoc" {
  pcu_group_id  = astra_pcu_group.pcu_group.id
  datacenter_id = provider::astra::resolve_datacenter(astra_database.db, "us-west-2")
}
