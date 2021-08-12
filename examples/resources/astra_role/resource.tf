resource "astra_role" "example" {
  role_name = "puppies"
  description = "test role"
  effect = "allow"
  resources = ["drn:astra:org:f9f4b1e0-4c05-451e-9bba-d631295a7f73"]
  policy = ["db-all-keyspace-create"]
}