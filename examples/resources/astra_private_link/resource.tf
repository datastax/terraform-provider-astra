resource "astra_private_link" "example" {
  allowed_principals = ["arn:aws:iam::111708290731:user/sebastian.estevez"]
  database_id        = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
  datacenter_id      = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
}