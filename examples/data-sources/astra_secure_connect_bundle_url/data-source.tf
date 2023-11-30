// For single DC databases, or to just get the primary datacenter Secure Connect Bundle
data "astra_secure_connect_bundle_url" "scb" {
  database_id = "f9f4b1e0-4c05-451e-9bba-d631295a7f73"
}
// Output for SCB reference
output "scb" {
  value = data.astra_secure_connect_bundle_url.scb.secure_bundles[0].url
}

// For mult-DC databases, specify the datacenter ID
// Example 1: Hard-coded IDs
// Primary datacenter SCB
data "astra_secure_connect_bundle_url" "scb1" {
  database_id   = "f9f4b1e0-4c05-451e-9bba-d631295a7f73"
  datacenter_id = "f9f4b1e0-4c05-451e-9bba-d631295a7f73-1" // for the primary dataceneter, the datacenter ID is not required
}
// Second datacenter SCB
data "astra_secure_connect_bundle_url" "scb2" {
  database_id   = "f9f4b1e0-4c05-451e-9bba-d631295a7f73"
  datacenter_id = "f9f4b1e0-4c05-451e-9bba-d631295a7f73-2"
}
// Third datacenter SCB
data "astra_secure_connect_bundle_url" "scb3" {
  database_id   = "f9f4b1e0-4c05-451e-9bba-d631295a7f73"
  datacenter_id = "f9f4b1e0-4c05-451e-9bba-d631295a7f73-3"
}
// Output for SCB reference
output "scb1" {
  value = data.astra_secure_connect_bundle_url.scb1.secure_bundles[0].url
}
output "scb2" {
  value = data.astra_secure_connect_bundle_url.scb2.secure_bundles[0].url
}
output "scb3" {
  value = data.astra_secure_connect_bundle_url.scb3.secure_bundles[0].url
}

// Example 2: Referenced IDs
// Database example for referencing
resource "astra_database" "mydb" {
  name           = "dbname"
  keyspace       = "testks"
  cloud_provider = "gcp"
  regions        = ["us-west4", "us-east4", "us-central1"]
  timeouts {
    create = "45m"
    delete = "45m"
    update = "45m"
  }
}
// Primary datacenter SCB (GCP.us-west4)
data "astra_secure_connect_bundle_url" "scb1" {
  database_id   = astra_database.mydb.id
  datacenter_id = astra_database.mydb.datacenters["${astra_database.mydb.cloud_provider}.${astra_database.mydb.regions[0]}"] // for the primary dataceneter, the datacenter ID is not required
}
// Second datacenter SCB (GCP.us-east4)
data "astra_secure_connect_bundle_url" "scb2" {
  database_id   = astra_database.mydb.id
  datacenter_id = astra_database.mydb.datacenters["${astra_database.mydb.cloud_provider}.${astra_database.mydb.regions[1]}"]
}
// Third datacenter SCB (GCP.us-central1)
data "astra_secure_connect_bundle_url" "scb3" {
  database_id   = astra_database.mydb.id
  datacenter_id = astra_database.mydb.datacenters["${astra_database.mydb.cloud_provider}.${astra_database.mydb.regions[2]}"]
}
// Output for SCB reference
output "scb1" {
  value = data.astra_secure_connect_bundle_url.scb1.secure_bundles[0].url
}
output "scb2" {
  value = data.astra_secure_connect_bundle_url.scb2.secure_bundles[0].url
}
output "scb3" {
  value = data.astra_secure_connect_bundle_url.scb3.secure_bundles[0].url
}
