data "astra_database" "example_db" {
  # ...
}

locals {
  # If the database has multiple regions, you can specify the desired one explicitly
  dc_id1 = provider::astra::resolve_datacenter(data.astra_database.example_db, "us-central1")

  # Or, if the database has only one region, you can omit the second argument
  dc_id2 = provider::astra::resolve_datacenter(data.astra_database.example_db)

  # This is all shorthand for the following much longer expression:
  dc_id3 = data.astra_database.example_db.datacenters["${data.astra_database.example_db.cloud_provider}.${data.astra_database.example_db.regions[0]}"]
}
