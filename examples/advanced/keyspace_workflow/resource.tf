# This needs to be a token associated with an Administrator role
provider "astra" {
  token = "AstraCS:abcd..."
}

resource "astra_database" "db1" {
  name           = "mydatabase"
  cloud_provider = "gcp"
  regions        = ["us-east1"]
  keyspace       = "myks"
}

resource "astra_role" "role1" {
    role_name = "role1"
    description = "role description"
    effect = "allow"
    resources = [
                  "drn:astra:org:<org id>:db:${astra_database.db1.id}",
                  # the following line grants permissions to ALL keyspaces within an org
                  "drn:astra:org:<org id>:db:${astra_database.db1.id}:keyspace:*",
                  # if you want to grant permissions only to specific keyspaces, do not use the above line,
                  # instead specify each keyspace. NOTE: The keyspaces below do not have to exist at this point.
                  "drn:astra:org:<org id>:db:${astra_database.db1.id}:keyspace:app1keyspace",
                  "drn:astra:org:<org id>:db:${astra_database.db1.id}:keyspace:app2keyspace",
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

# This will create a token associated with the role above.
resource "astra_token" "token1" {
  roles = [astra_role.role1.id]
}

# Tokens associated with the above "role1" role will have permissions for all Keyspace operations
# for the keyspaces listeed in the "resources" section of role definition.