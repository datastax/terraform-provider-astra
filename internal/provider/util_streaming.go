package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
)

const (
	authHeader          = "Authorization"
	pulsarClusterHeader = "X-Datastax-Pulsar-Cluster"
	organizationHeader  = "X-Datastax-Current-Org"
)

// setPulsarClusterHeaders returns a function that can be used to set the request headers for a Pulsar admin API requests.
// This overrides the provider Authorization header because the Pulsar admin API requires a Pulsar token instead of the AstraCS
// token required by the Astra API.
func setPulsarClusterHeaders(organizationID, cluster, pulsarToken string) func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		if pulsarToken == "" {
			return fmt.Errorf("missing required pulsar token")
		}
		req.Header.Set(authHeader, fmt.Sprintf("Bearer %s", pulsarToken))
		if cluster == "" {
			return fmt.Errorf("missing required pulsar cluster name")
		}
		req.Header.Set(pulsarClusterHeader, cluster)
		if organizationID != "" {
			req.Header.Set(organizationHeader, organizationID)
		}
		return nil
	}
}

type StreamingToken struct {
	Iat     int    `json:"iat"`
	Iss     string `json:"iss"`
	Sub     string `json:"sub"`
	Tokenid string `json:"tokenid"`
}

func getPulsarTokenByID(ctx context.Context, streamingClient *astrastreaming.ClientWithResponses, orgID string, pulsarCluster string, tenantName string, tokenID string) (string, error) {

	if pulsarCluster == "" {
		return "", fmt.Errorf("missing pulsar cluster")
	}
	if tenantName == "" {
		return "", fmt.Errorf("missing tenant name")
	}
	if tokenID == "" {
		return "", fmt.Errorf("missing token ID")
	}
	tokenParams := astrastreaming.GetPulsarTokenByIDParams{
		XDataStaxCurrentOrg:    orgID,
		XDataStaxPulsarCluster: pulsarCluster,
	}

	pulsarTokenResponse, err := streamingClient.GetPulsarTokenByIDWithResponse(ctx, tenantName, tokenID, &tokenParams)
	if err != nil {
		return "", fmt.Errorf("failed to get pulsar tokens: %w", err)
	}

	pulsarToken := string(pulsarTokenResponse.Body)
	return pulsarToken, nil
}

func getLatestPulsarToken(ctx context.Context, streamingClient *astrastreaming.ClientWithResponses, astraToken string, orgID string, pulsarCluster string, tenantName string) (string, error) {

	if pulsarCluster == "" {
		return "", fmt.Errorf("missing pulsar cluster")
	}
	if tenantName == "" {
		return "", fmt.Errorf("missing tenant name")
	}
	tenantTokenParams := astrastreaming.GetPulsarTokensByTenantParams{
		Authorization:          fmt.Sprintf("Bearer %s", astraToken),
		XDataStaxCurrentOrg:    orgID,
		XDataStaxPulsarCluster: pulsarCluster,
	}

	pulsarTokenResponse, err := streamingClient.GetPulsarTokensByTenant(ctx, tenantName, &tenantTokenParams)
	if err != nil {
		return "", fmt.Errorf("failed to get pulsar tokens: %w", err)
	}
	respBody, err := io.ReadAll(pulsarTokenResponse.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read pulsar token response body: %w", err)
	}
	if pulsarTokenResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get pulsar tokens, invalid status code: %d, message: %s", pulsarTokenResponse.StatusCode, string(respBody))
	}

	var streamingTokens []StreamingToken
	err = json.Unmarshal(respBody, &streamingTokens)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal pulsar token list: %w\n Response Body: %s", err, string(respBody))
	}

	if len(streamingTokens) == 0 {
		return "", fmt.Errorf("no valid pulsar tokens found for tenant '%s'", tenantName)
	}
	tokenId := streamingTokens[0].Tokenid
	getTokenByIdParams := astrastreaming.GetPulsarTokenByIDParams{
		Authorization:          fmt.Sprintf("Bearer %s", astraToken),
		XDataStaxCurrentOrg:    orgID,
		XDataStaxPulsarCluster: pulsarCluster,
	}

	getTokenResponse, err := streamingClient.GetPulsarTokenByIDWithResponse(ctx, tenantName, tokenId, &getTokenByIdParams)
	if err != nil {
		return "", fmt.Errorf("failed to get pulsar token: %w", err)
	}

	pulsarToken := string(getTokenResponse.Body)
	return pulsarToken, nil
}

type OrgId struct {
	ID string `json:"id"`
}

// getCurrentOrgID returns the organization ID from the Astra server
func getCurrentOrgID(ctx context.Context, astraClient *astra.ClientWithResponses) (string, error) {
	orgResponse, err := astraClient.GetCurrentOrganization(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get organization ID: %w", err)
	} else if orgResponse.StatusCode > 300 {
		body, err := io.ReadAll(orgResponse.Body)
		message := string(body)
		if err != nil {
			message = err.Error()
		}
		return "", fmt.Errorf("failed to get organization ID: %s", message)
	}
	var orgID OrgId
	err = json.NewDecoder(orgResponse.Body).Decode(&orgID)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal organization ID: %w", err)
	} else if orgID.ID == "" {
		return "", errors.New("failed to get organization ID, found empty string")
	}
	return orgID.ID, nil
}

func getProviderRegionFromClusterName(clusterName string) (string, string, error) {
	provider := ""
	if strings.Contains(clusterName, "-aws") {
		provider = "aws"
	} else if strings.Contains(clusterName, "-azure") {
		provider = "azure"
	} else if strings.Contains(clusterName, "-gcp") {
		provider = "gcp"
	} else {
		return "", "", fmt.Errorf("failed to parse streaming cluster name '%s', unknown cloud provider", clusterName)
	}
	parts := strings.Split(clusterName, "-"+provider)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("failed to parse streaming cluster name '%s'", clusterName)
	}
	region := parts[1]
	region = strings.TrimSuffix(region, "-dev")
	region = strings.TrimSuffix(region, "-staging")
	region = strings.ReplaceAll(region, "-", "")

	if region == "ue1" {
		region = "useast1"
	}

	return provider, region, nil
}

// getPulsarCluster TODO: this is unreliable because not all clusters might follow this format
func getPulsarCluster(clusterName, cloudProvider, region, suffix string) string {
	if strings.TrimSpace(clusterName) != "" {
		return clusterName
	}
	// In most astra APIs there are dashes in region names depending on the cloud provider, this seems not to be the case for streaming
	normalizedRegion := strings.ReplaceAll(region, "-", "")
	return strings.ToLower(fmt.Sprintf("pulsar-%s-%s%s", cloudProvider, normalizedRegion, suffix))
}

func int32ToInt64Pointer(v *int32) *int64 {
	if v == nil {
		return nil
	}
	x := int64(*v)
	return &x
}
