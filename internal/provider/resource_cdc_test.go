package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func TestAccAstraCDC(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAstraCDCConfig(),
			},
		},
	})
}

func testAstraCDCConfig() string {
	return `

resource "astra_cdc" "cdc-1" {
  database_id        = "de76e588-761f-4e74-afed-1d2092aaaa84"
  database_name      = "terraform-cdc-test"
  keyspace           = "ks1"
  table              = "tbl1"
  topic_partitions   = 3
  pulsar_cluster     = "pulsar-gcp-useast1-staging"
  tenant_name        = "terraform-tests1"
}`
}

func TestAstraCDCFull(t *testing.T) {
	checkRequiredTestVars(t, "ASTRA_TEST_CDC_FULL_TEST_ENABLED")
	streamingTenant := "terraform-cdc-test-" + randomString(6)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAstraCDCConfigFull("GCP", "us-east1", streamingTenant),
			},
		},
	})
}

func testAstraCDCConfigFull(cloud_provider, region, streamingTenant string) string {
	return fmt.Sprintf(`

resource "astra_database" "database_1" {
  cloud_provider      = "%s"
  regions             = ["%s"]
  name                = "terraform-cdc-test"
  keyspace            = "ks1"
  deletion_protection = "false"
}

resource "astra_table" "table_1" {
  database_id        = astra_database.database_1.id
  keyspace           = astra_database.database_1.keyspace
  region             = "%s"
  table              = "cdctable1"
  clustering_columns = "a"
  partition_keys     = "b"
  column_definitions = [
    {
      Name: "a"
      Static: false
      TypeDefinition: "text"
    },
    {
      Name: "b"
      Static: false
      TypeDefinition: "text"
    }
  ]
}

resource "astra_streaming_tenant" "streaming_tenant_1" {
  tenant_name         = "%s"
  cloud_provider      = lower(astra_database.database_1.cloud_provider)
  region              = astra_table.table_1.region
  user_email          = "test@datastax.com"
  deletion_protection = "false"
}

 resource "astra_cdc" "cdc-1" {
  depends_on         = [ astra_database.database_1, astra_streaming_tenant.streaming_tenant_1 ]
  database_id        = astra_database.database_1.id
  database_name      = astra_database.database_1.name
  keyspace           = astra_database.database_1.keyspace
  table              = astra_table.table_1.table
  topic_partitions   = 3
  tenant_name        = astra_streaming_tenant.streaming_tenant_1.tenant_name
  pulsar_cluster     = astra_streaming_tenant.streaming_tenant_1.cluster_name
}`, cloud_provider, region, region, streamingTenant)

}
