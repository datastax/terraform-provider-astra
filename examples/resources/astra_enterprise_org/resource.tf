# Create a new Enterprise organization
resource "astra_enterprise_org" "entorg" {
  name          = "My Enterprise Organization"
  email         = "admin@example.com"
  admin_user_id = "a1b2c3d4-5e6f-7a8b-9c0d-1e2f3a4b5c6d"
  enterprise_id = "a1b2c3d4-5e6f-7a8b-9c0d-1e2f3a4b5c6d"
}