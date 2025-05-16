package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAstraCDCv3(t *testing.T) {
	checkRequiredTestVars(t, "ASTRA_TEST_CDC_V3_TEST_ENABLED")
	streamingTenant := "terraform-cdcv3-test-" + randomString(4)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAstraCDCv3Config("GCP", "us-east1", streamingTenant),
			},
		},
	})
}

func testAstraCDCv3Config(cloud_provider, region, streamingTenant string) string {
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
  region             = astra_database.database_1.region[0]
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

resource "astra_table" "table_2" {
  database_id        = astra_database.database_1.id
  keyspace           = astra_database.database_1.keyspace
  region             = astra_database.database_1.region[0]
  table              = "cdctable2"
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

resource "astra_streaming_tenant" "tenant_1" {
  cluster_name        = "pulsar-gcp-useast1"
  tenant_name         = "%s"
  user_email          = "test@datastax.com"
  deletion_protection = "false"
}

 resource "astra_cdc_v3" "cdc_1" {
  depends_on         = [ astra_database.database_1, astra_streaming_tenant.streaming_tenant_1 ]
  database_id        = astra_database.database_1.id
  database_name      = astra_database.database_1.name
  tables = [
    {
      keyspace = "ks1"
      table    = "table1"
    },
    {
      keyspace = "ks1"
      table    = "table2"
    },
  ]
  regions = [
    {
      region   = "us-east1"
      datacenter_id     = "${astra_database.example.id}-1"
      streaming_cluster = astra_streaming_tenant.tenant_1.cluster_name
      streaming_tenant  = astra_streaming_tenant.tenant_1.tenant_name
    },
  ]

}`, cloud_provider, region, streamingTenant)

}
