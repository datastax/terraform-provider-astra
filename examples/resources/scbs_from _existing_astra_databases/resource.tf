data "astra_databases" "databaselist" {
  status = "ACTIVE"
}

locals {
  dbs = [for db in data.astra_databases.databaselist.results : db.id]
}

data "astra_secure_connect_bundle_url" "dev" {
  for_each    = toset(local.dbs)
  database_id = each.value
}