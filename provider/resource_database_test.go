package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestDatabase(t *testing.T){
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabaseConfiguration(),
			},
		},
	})
}

//https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html
func testAccDatabaseConfiguration() string {
	return fmt.Sprintf(`
resource "astra_database" "dev" {
  name           = "puppies"
  keyspace       = "puppies"
  cloud_provider = "gcp"
  regions        = ["us-east1"]
}

data "astra_secure_connect_bundle_url" "dev" {
  database_id = astra_database.dev.id
}
`)
}

func TestGetRegionUpdatesOnlyDeletes(t *testing.T) {
	oldData := []interface{} {"region1", "region2", "region3", "region4", "region5"}
	newData := []interface{} {"region1", "region2", "region3"}

	regionsToAdd, regionsToDelete := getRegionUpdates(oldData, newData)

	testFailed := false
	// verify no adds and 2 deletes
	if len(regionsToAdd) != 0 {
		testFailed = true
		t.Logf("getRegionUpdates returned regions to add, but expected none. Regions to add: %v", regionsToAdd)
	}
	if len(regionsToDelete) != 2 {
		testFailed = true
		t.Logf("getRegionUpdates returned an unexpected number of regions to delete. Expected [region4 region5] but got: %v", regionsToDelete)
	} else {
		// make sure it's the correct regions
		expectedMap := map[string]bool{}
		expectedMap["region4"] = true
		expectedMap["region5"] = true
		for _, v := range regionsToDelete {
			if !expectedMap[v] {
				testFailed = true
				t.Logf("Unexpected region to delete: %s", v)
			}
		}
	}

	if testFailed {
		t.Fail()
	}
}

func TestGetRegionUpdatesOnlyAdds(t *testing.T) {
	oldData := []interface{} {"region1", "region2", "region3"}
	newData := []interface{} {"region1", "region2", "region3", "region4", "region5"}

	regionsToAdd, regionsToDelete := getRegionUpdates(oldData, newData)

	testFailed := false
	// verify no deletes and 2 adds
	if len(regionsToAdd) != 2 {
		testFailed = true
		t.Logf("getRegionUpdates returned an unexpected number of regions to add. Expected [region4 region5] but got]: %v", regionsToAdd)
	} else {
		// make sure it's the correct regions
		expectedMap := map[string]bool{}
		expectedMap["region4"] = true
		expectedMap["region5"] = true
		for _, v := range regionsToAdd {
			if !expectedMap[v] {
				testFailed = true
				t.Logf("Unexpected region to add: %s", v)
			}
		}
	}
	if len(regionsToDelete) != 0 {
		testFailed = true
		t.Logf("getRegionUpdates returned regions to delete, but expected none. Regions to delete: %v", regionsToDelete)
	}

	if testFailed {
		t.Fail()
	}
}

func TestGetRegionUpdatesAddsAndDeletes(t *testing.T) {
	oldData := []interface{} {"region1", "region3", "region5"}
	newData := []interface{} {"region1", "region2", "region4"}

	regionsToAdd, regionsToDelete := getRegionUpdates(oldData, newData)

	testFailed := false
	// verify 2 adds and 2 deletes
	if len(regionsToAdd) != 2 {
		testFailed = true
		t.Logf("getRegionUpdates returned an unexpected number of regions to add. Expected [region2 region4] but got]: %v", regionsToAdd)
	} else {
		// make sure it's the correct regions
		expectedMap := map[string]bool{}
		expectedMap["region2"] = true
		expectedMap["region4"] = true
		for _, v := range regionsToAdd {
			if !expectedMap[v] {
				testFailed = true
				t.Logf("Unexpected region to add: %s", v)
			}
		}
	}
	if len(regionsToDelete) != 2 {
		testFailed = true
		t.Logf("getRegionUpdates returned an unexpected number of regions to delete. Expected [region3 region5] but got]: %v", regionsToDelete)
	} else {
		// make sure it's the correct regions
		expectedMap := map[string]bool{}
		expectedMap["region3"] = true
		expectedMap["region5"] = true
		for _, v := range regionsToDelete {
			if !expectedMap[v] {
				testFailed = true
				t.Logf("Unexpected region to delete: %s", v)
			}
		}
	}
	if testFailed {
		t.Fail()
	}
}