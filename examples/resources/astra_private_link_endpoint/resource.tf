resource "astra_private_link" "example" {
  allowed_principals = ["arn:aws:iam::445559476293:user/Sebastian"]
  database_id        = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
  datacenter_id      = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
}
resource "aws_vpc_endpoint" "example" {
  vpc_id             = "vpc-f939e884"
  service_name       = astra_private_link.example.service_name
  vpc_endpoint_type  = "Interface"
  subnet_ids         = ["subnet-4d376300", "subnet-4d85066c", "subnet-030e8b65"]
  security_group_ids = ["sg-74ae4d41"]
}
resource "astra_private_link_endpoint" "example" {
  database_id   = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
  datacenter_id = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
  endpoint_id   = aws_vpc_endpoint.example.id
}
