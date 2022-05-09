# AWS example
resource "astra_private_link" "example" {
  allowed_principals = ["arn:aws:iam::445559476293:user/Sebastian"]
  database_id        = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
  datacenter_id      = "a6bc9c26-e7ce-424f-84c7-0a00afb12588-1"
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
  datacenter_id = "a6bc9c26-e7ce-424f-84c7-0a00afb12588-1"
  endpoint_id   = aws_vpc_endpoint.example.id
}

# GCP example
# To use GCP terraform provider, you will need to explicitly include it, configure it
# and authenticate to GCP service. Please see the following for more details:
# https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/getting_started#configuring-the-provider
# https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/getting_started#adding-credentials
provider "google" {
  project = "my-project"
  region = "us-east1"
  zone = "us-east1-b"
}

resource "astra_private_link" "example" {
  allowed_principals = ["my-project"]
  database_id        = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
  datacenter_id      = "a6bc9c26-e7ce-424f-84c7-0a00afb12588-1"
}

resource "google_compute_network" "example" {
 name                    = "example-network"
 auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "example" {
 name          = "example-subnetwork"
 ip_cidr_range = "10.142.0.0/20"
 region        = "us-east1"
 network       = google_compute_network.example.id
}

resource "google_compute_address" "example" {
 name         = "endpoint-address"
 subnetwork   = google_compute_subnetwork.example.id
 address_type = "INTERNAL"
 region       = "us-east1"
}

resource "google_compute_forwarding_rule" "example" {
 name                  = "psc-endpoint"
 target                = "https://www.googleapis.com/compute/v1/${astra_private_link.example.service_name}"
 project               = google_compute_network.example.project
 ip_address            = google_compute_address.example.id
 network               = google_compute_network.example.id
 region                = "us-east1"
 load_balancing_scheme = ""
}

# The endpoint ID (PSC Connection ID) is not currently accessible from the google_compute_forwarding_rule terraform object.
# It must be retrieved via the GCP UI (https://console.cloud.google.com/net-services/psc/list) or via the gcloud CLI:
#    gcloud compute forwarding-rules describe psc-endpoint --region=us-east1
resource "astra_private_link_endpoint" "endpoint" {
  database_id   = "a6bc9c26-e7ce-424f-84c7-0a00afb12588"
  datacenter_id = "a6bc9c26-e7ce-424f-84c7-0a00afb12588-1"
  endpoint_id   = "13585698993864708"
}
