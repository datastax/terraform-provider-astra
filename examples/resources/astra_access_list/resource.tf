resource "astra_access_list" "example" {
  database_id = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
  enabled     = true
  addresses {
    request {
      address = "0.0.0.0/0"
      enabled = true
    }
  }
}