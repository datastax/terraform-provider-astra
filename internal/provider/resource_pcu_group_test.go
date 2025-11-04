package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccPcuGroupResource_lifecycle(t *testing.T) {
	prefix := "tf-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPcuGroupConfiguration(prefix, false),
				Check:  resource.ComposeAggregateTestCheckFunc(
				//resource.TestCheckResourceAttr("astra_pcu_group.test-1", "title", prefix+"-group"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-1", "park", "false"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-1", "deletion_protection", "false"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-1", "reserved_protection", "true"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-1", "status", "CREATED"),
				//
				//resource.TestCheckResourceAttr("astra_pcu_group.test-2", "title", prefix+"-group"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-2", "park", "false"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-2", "deletion_protection", "false"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-2", "reserved_protection", "true"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-2", "status", "CREATED"),
				),
			},
			{
				Config: testAccPcuGroupConfiguration(prefix, false) + testAccPcuGroupDbAndAssociationConfiguration(prefix, false),
				Check:  resource.ComposeAggregateTestCheckFunc(
				//resource.TestCheckResourceAttr("astra_pcu_group.test-1", "status", "ACTIVE"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-2", "status", "ACTIVE"),
				),
			},
			{
				Config: testAccPcuGroupConfiguration(prefix, true) + testAccPcuGroupDbAndAssociationConfiguration(prefix, false),
				Check:  resource.ComposeAggregateTestCheckFunc(
				//resource.TestCheckResourceAttr("astra_pcu_group.test-1", "park", "true"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-1", "deletion_protection", "false"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-1", "reserved_protection", "true"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-1", "status", "PARKED"),
				//
				//resource.TestCheckResourceAttr("astra_pcu_group.test-2", "park", "true"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-2", "deletion_protection", "false"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-2", "reserved_protection", "true"),
				//resource.TestCheckResourceAttr("astra_pcu_group.test-2", "status", "PARKED"),
				),
			},
		},
	})
}

func testAccPcuGroupConfiguration(prefix string, park bool) string {
	impl := func(suffix int) string {
		return fmt.Sprintf(`
			resource "astra_pcu_group" "test-%d" {
        	    title               = "%s-group-1"
                cloud_provider      = "AWS"
                region              = "us-west-2"
                min_capacity        = 1
                max_capacity        = 1
                deletion_protection = false
                park			    = %t
        	}
		`, suffix, prefix, park)
	}

	return impl(1) + impl(2)
}

func testAccPcuGroupDbAndAssociationConfiguration(prefix string, swapAssociation bool) string {
	impl := func(suffix int, groupSuffix int) string {
		return fmt.Sprintf(`
		    resource "astra_database" "test-%d" {
            	name                = "%s-db"
                cloud_provider      = "AWS"
                regions             = ["us-west-2"]
                db_type             = "vector"
                keyspace            = "default_keyspace"
                deletion_protection = false
            }

            resource "astra_pcu_group_association" "test-%d" {
              pcu_group_id  = astra_pcu_group.test-%d.id
              datacenter_id = astra_database.test-%d.datacenters["AWS.us-west-2"]
            }
		`, suffix, prefix, suffix, groupSuffix, suffix)
	}

	if swapAssociation {
		return impl(1, 2) + impl(2, 1)
	} else {
		return impl(1, 1) + impl(2, 2)
	}
}
