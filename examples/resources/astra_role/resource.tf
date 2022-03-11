resource "astra_role" "example" {
  role_name   = "puppies"
  description = "test role"
  effect      = "allow"
  resources   = ["drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73"]
  policy      = ["db-all-keyspace-create"]
}

resource "astra_role" "example2" {
  role_name   = "puppies"
  description = "complex role"
  effect      = "allow"
  resources = [
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:5b70892f-e01a-4595-98e6-19ecc9985d50",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:5b70892f-e01a-4595-98e6-19ecc9985d50:keyspace:system_schema:table:*",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:5b70892f-e01a-4595-98e6-19ecc9985d50:keyspace:system:table:*",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:5b70892f-e01a-4595-98e6-19ecc9985d50:keyspace:system_virtual_schema:table:*",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:5b70892f-e01a-4595-98e6-19ecc9985d50:keyspace:*",
    "drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73:db:5b70892f-e01a-4595-98e6-19ecc9985d50:keyspace:*:table:*"
  ]
  policy = ["accesslist-read", "db-all-keyspace-describe", "db-keyspace-describe", "db-table-select", "db-table-describe", "db-graphql", "db-rest", "db-cql"]
}