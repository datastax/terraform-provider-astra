package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetProviderRegionFromClusterName(t *testing.T) {
	clusterName := "pulsar-gcp-useast4"
	provider, region, err := getProviderRegionFromClusterName(clusterName)
	assert.Nil(t, err)
	assert.Equal(t, "gcp", provider)
	assert.Equal(t, "useast4", region)

	clusterName = "pulsar-azure-westus2-staging"
	provider, region, err = getProviderRegionFromClusterName(clusterName)
	assert.Nil(t, err)
	assert.Equal(t, "azure", provider)
	assert.Equal(t, "westus2", region)

	clusterName = "pulsar-foo-prod-aws-apsoutheast1"
	provider, region, err = getProviderRegionFromClusterName(clusterName)
	assert.Nil(t, err)
	assert.Equal(t, "aws", provider)
	assert.Equal(t, "apsoutheast1", region)

	clusterName = "pulsar-bar-dev-awsue1"
	provider, region, err = getProviderRegionFromClusterName(clusterName)
	assert.Nil(t, err)
	assert.Equal(t, "aws", provider)
	assert.Equal(t, "useast1", region)

	clusterName = "pulsar-baz-uat-azure-centralus"
	provider, region, err = getProviderRegionFromClusterName(clusterName)
	assert.Nil(t, err)
	assert.Equal(t, "azure", provider)
	assert.Equal(t, "centralus", region)
}
