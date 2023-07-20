# Example role that grants policy permissions to ALL Astra DBs in an organization
resource "astra_role" "alldbsrole" {
  role_name   = "alldbsrole"
  description = "Role that applies to all DBs in an org"
  effect      = "allow"
  resources = [
    # The following 3 resources are needed and wildcarded to associate the role to all dbs
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:*",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:*:keyspace:*",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:*:keyspace:*:table:*"
  ]
  policy = [
    # "org-db-view" is required to list databases
    "org-db-view",
    # the following are for CQl and table operations
    "db-cql", "db-table-alter", "db-table-create", "db-table-describe", "db-table-modify", "db-table-select",
    # the following are for Keysapce operations
    "db-keyspace-alter", "db-keyspace-describe", "db-keyspace-modify", "db-keyspace-authorize", "db-keyspace-drop", "db-keyspace-create", "db-keyspace-grant",
  ]
}

# Example resources for a more restricted role
# A Terraform managed Astra DB resource
resource "astra_database" "exampledb" {
  name           = "exampledb"
  keyspace       = "primaryks"
  cloud_provider = "gcp"
  regions        = ["us-east1"]
}

# Example application keyspaces
resource "astra_keyspace" "appks1" {
  name        = "appks1"
  database_id = astra_database.exampledb.id
}

resource "astra_keyspace" "appks2" {
  name        = "appks2"
  database_id = astra_database.exampledb.id
}

resource "astra_keyspace" "appks3" {
  name        = "appks3"
  database_id = astra_database.exampledb.id
}

# Example role that grants policy permissions to specific keyspaces within a single Astra DB
resource "astra_role" "singledbrole" {
  role_name   = "singledbrole"
  description = "Role that applies to specific keyspaces for a single Astra DB"
  effect      = "allow"
  resources = [
    # apply role to the primary keyspace in the database
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:${astra_database.exampledb.id}:keyspace:${astra_database.exampledb.keyspace}",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:${astra_database.exampledb.id}:keyspace:${astra_database.exampledb.keyspace}:table:*",
    # apply role to additional keyspaces defined above
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:${astra_database.exampledb.id}:keyspace:${astra_keyspace.appks1.name}",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:${astra_database.exampledb.id}:keyspace:${astra_keyspace.appks1.name}:table:*",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:${astra_database.exampledb.id}:keyspace:${astra_keyspace.appks2.name}",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:${astra_database.exampledb.id}:keyspace:${astra_keyspace.appks2.name}:table:*",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:${astra_database.exampledb.id}:keyspace:${astra_keyspace.appks3.name}",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:${astra_database.exampledb.id}:keyspace:${astra_keyspace.appks3.name}:table:*",
    # apply role to keyspaces that have not yet been created (the role will be associated if and when the keyspace is created)
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:${astra_database.exampledb.id}:keyspace:futureks",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:${astra_database.exampledb.id}:keyspace:futureks:table:*",
  ]
  policy = [
    # "org-db-view" is required to list databases
    "org-db-view",
    # the following are for CQl and table operations
    "db-cql", "db-table-alter", "db-table-create", "db-table-describe", "db-table-modify", "db-table-select",
    # the following are for Keysapce operations
    "db-keyspace-alter", "db-keyspace-describe", "db-keyspace-modify", "db-keyspace-authorize", "db-keyspace-drop", "db-keyspace-create", "db-keyspace-grant",
  ]
}
