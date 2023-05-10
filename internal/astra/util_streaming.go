package astra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
func setPulsarClusterHeaders(pulsarToken, clusterName, organizationID string) func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		if pulsarToken == "" {
			return fmt.Errorf("missing required pulsar token")
		}
		req.Header.Set(authHeader, fmt.Sprintf("Bearer %s", pulsarToken))
		if clusterName == "" {
			return fmt.Errorf("missing required pulsar cluster name")
		}
		req.Header.Set(pulsarClusterHeader, clusterName)
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

func getPulsarToken(ctx context.Context, streamingClient *astrastreaming.ClientWithResponses, astraToken string, orgID string, pulsarCluster string, tenantName string) (string, error) {

	if pulsarCluster == "" {
		return "", fmt.Errorf("missing pulsar cluster")
	}
	if tenantName == "" {
		return "", fmt.Errorf("missing tenant name")
	}
	tenantTokenParams := astrastreaming.IdListTenantTokensParams{
		Authorization:          fmt.Sprintf("Bearer %s", astraToken),
		XDataStaxCurrentOrg:    orgID,
		XDataStaxPulsarCluster: pulsarCluster,
	}

	pulsarTokenResponse, err := streamingClient.IdListTenantTokens(ctx, tenantName, &tenantTokenParams)
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
	getTokenByIdParams := astrastreaming.GetTokenByIDParams{
		Authorization:          fmt.Sprintf("Bearer %s", astraToken),
		XDataStaxCurrentOrg:    orgID,
		XDataStaxPulsarCluster: pulsarCluster,
	}

	getTokenResponse, err := streamingClient.GetTokenByIDWithResponse(ctx, tenantName, tokenId, &getTokenByIdParams)
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
		return "", fmt.Errorf("failed to get current organization data: %w", err)
	}
	var orgID OrgId
	err = json.NewDecoder(orgResponse.Body).Decode(&orgID)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal current organization ID: %w", err)
	}
	return orgID.ID, nil
}
