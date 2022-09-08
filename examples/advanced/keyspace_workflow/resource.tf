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

resource "astra_keyspace" "appks1" {
  name = "appks1"
  database_id = astra_database.db1.id
}

resource "astra_keyspace" "appks2" {
  name = "appks2"
  database_id = astra_database.db1.id
}

resource "astra_keyspace" "appks3" {
  name = "appks3"
  database_id = astra_database.db1.id
}

resource "astra_role" "alldbsrole" {
    role_name = "alldbsrole"
    description = "role description"
    effect = "allow"
    resources = [
                  "drn:astra:org:<org id>:db:*",
                  "drn:astra:org:<org id>:db:*:keyspace:*",
                  "drn:astra:org:<org id>:db:*:keyspace:*:table:*",
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

resource "astra_role" "selectkeyspacesrole" {
    role_name = "selectkeyspacesrole"
    description = "role description"
    effect = "allow"
    resources = [
                  "drn:astra:org:<org id>:db:${astra_database.db1.id}",
                  # NOTE: The keyspaces below do not have to exist when this role is created
                  "drn:astra:org:<org id>:db:${astra_database.db1.id}:keyspace:appks1",
                  "drn:astra:org:<org id>:db:${astra_database.db1.id}:keyspace:appks1:table:*",
                  "drn:astra:org:<org id>:db:${astra_database.db1.id}:keyspace:appks3",
                  "drn:astra:org:<org id>:db:${astra_database.db1.id}:keyspace:appks3:table:*",
                  "drn:astra:org:<org id>:db:${astra_database.db1.id}:keyspace:notcreatedappks",
                  "drn:astra:org:<org id>:db:${astra_database.db1.id}:keyspace:notcreatedappks:table:*",
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