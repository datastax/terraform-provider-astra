data "astra_databases" "databaselist" {
  status = "ACTIVE"
}

output "existing_dbs" {
  value = [for db in data.astra_databases.databaselist.results : db.id]
}