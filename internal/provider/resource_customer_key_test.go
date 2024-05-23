package provider

import (
	"testing"
)

func TestCustomerKeyIdParser(t *testing.T) {
	testId := "28b7d281-a2ae-4e5b-bc3f-9f80df5f5223/cloudProvider/aws/region/us-east-1/keyId/arn:aws:kms:us-east-1:388533891461:key/85e37e2b-d897-49f0-9d18-3c0daf4a7ff5"
	orgId, cloudProvider, region, keyId, err := parseCustomerKeyId(testId)
	if err != nil {
		t.Logf("Customer Key ID failed to parse: \"%s\", %s", testId, err)
		t.Fail()
	}
	if orgId != "28b7d281-a2ae-4e5b-bc3f-9f80df5f5223" {
		t.Logf("Organization ID parsed from Customer Key ID: \"%s\", expected \"%s\"", orgId, "28b7d281-a2ae-4e5b-bc3f-9f80df5f5223")
		t.Fail()
	}
	if cloudProvider != "aws" {
		t.Logf("Cloud Provider parsed from Customer Key ID: \"%s\", expected \"%s\"", cloudProvider, "aws")
		t.Fail()
	}
	if region != "us-east-1" {
		t.Logf("Region parsed from Customer Key ID: \"%s\", expected \"%s\"", region, "us-east-1")
		t.Fail()
	}
	if keyId != "arn:aws:kms:us-east-1:388533891461:key/85e37e2b-d897-49f0-9d18-3c0daf4a7ff5" {
		t.Logf("Key ID parsed from Customer Key ID: \"%s\", expected \"%s\"", keyId, "arn:aws:kms:us-east-1:388533891461:key/85e37e2b-d897-49f0-9d18-3c0daf4a7ff5")
		t.Fail()
	}
}