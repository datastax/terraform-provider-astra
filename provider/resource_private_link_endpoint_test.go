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
provider "aws" {
  region = "us-east-2"
  version = "~> 3.0"
}
resource "astra_private_link" "example" {
  allowed_principals = ["arn:aws:iam::337811753388:user/merkle-terraform"]
  database_id = "aba3cf20-d579-4091-a36d-9c9f75096031"
  datacenter_id = "aba3cf20-d579-4091-a36d-9c9f75096031-1"
}
resource "aws_vpc_endpoint" "example" {
  vpc_id             = "vpc-5fbb2e34"
  service_name       = astra_private_link.example.service_name
  vpc_endpoint_type  = "Interface"
  subnet_ids         = ["subnet-c0396b8c","subnet-8059e4eb","subnet-d37c97ae"]
  security_group_ids = ["sg-18ba256c"]
}
resource "astra_private_link_endpoint" "example" {
  database_id = "aba3cf20-d579-4091-a36d-9c9f75096031"
  datacenter_id = "aba3cf20-d579-4091-a36d-9c9f75096031-1"
  endpoint_id = aws_vpc_endpoint.example.id
}
`)
}

func TestParsePrivateLinkEndpointId(t *testing.T){
	id := "b504911d-4982-4e45-84c2-607524cb533b/datacenter/b504911d-4982-4e45-84c2-607524cb533b-1/endpoint//subscriptions/123456/resourceGroups/123456/providers/Microsoft.Network/privateEndpoints/123456"
	databaseID, datacenterID, endpointID, err := parsePrivateLinkEndpointID(id)
	if err != nil {
		t.Logf("Private link ID failed to parse: \"%s\", %s", id, err)
		t.Fail()
	}
	// assert databaseID, dataceneterID and endpointID
	if databaseID != "b504911d-4982-4e45-84c2-607524cb533b" {
		t.Logf("Database ID parsed from private link ID: \"%s\", expected \"%s\"", databaseID, "b504911d-4982-4e45-84c2-607524cb533b")
		t.Fail()
	}
	if datacenterID != "b504911d-4982-4e45-84c2-607524cb533b-1" {
		t.Logf("Datacenter ID parsed from private link ID: \"%s\", expected \"%s\"", datacenterID, "b504911d-4982-4e45-84c2-607524cb533b-1")
		t.Fail()
	}
	if (endpointID != "/subscriptions/123456/resourceGroups/123456/providers/Microsoft.Network/privateEndpoints/123456") {
		t.Logf("endpoint ID parsed from private link ID: \"%s\", expected \"%s\"", endpointID, "/subscriptions/123456/resourceGroups/123456/providers/Microsoft.Network/privateEndpoints/123456")
		t.Fail()
	}
}