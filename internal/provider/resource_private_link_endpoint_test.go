package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestPrivateLinkEndpoint(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); checkAwsEnv(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateLinkEndpointConfiguration(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccPrivateLinkEndpointConfiguration() string {
	return fmt.Sprintf(`
resource "astra_private_link" "example" {
  allowed_principals = ["arn:aws:iam::337811753388:user/terraform-user"]
  database_id = "762c633f-dcde-47fe-8cb1-f6c47f6e9049"
  datacenter_id = "762c633f-dcde-47fe-8cb1-f6c47f6e9049-1"
}
resource "aws_vpc_endpoint" "example" {
  vpc_id             = "vpc-5fbb2e34"
  service_name       = astra_private_link.example.service_name
  vpc_endpoint_type  = "Interface"
  subnet_ids         = ["subnet-c0396b8c","subnet-8059e4eb","subnet-d37c97ae"]
  security_group_ids = ["sg-18ba256c"]
}
resource "astra_private_link_endpoint" "example" {
  database_id = "762c633f-dcde-47fe-8cb1-f6c47f6e9049"
  datacenter_id = "762c633f-dcde-47fe-8cb1-f6c47f6e9049-1"
  endpoint_id = aws_vpc_endpoint.example.id
}
`)
}