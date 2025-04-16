package sdk

import (
	"context"
	"fmt"

	"github.com/castai/terraform-provider-castai/castai/sdk/client"
)

// Currently, sdk doesn't have generated constants for cluster status and agent status, declaring our own.
const (
	ClusterStatusReady    = "ready"
	ClusterStatusDeleting = "deleting"
	ClusterStatusDeleted  = "deleted"
	ClusterStatusArchived = "archived"
	ClusterStatusFailed   = "failed"

	ClusterAgentStatusDisconnected  = "disconnected"
	ClusterAgentStatusDisconnecting = "disconnecting"
)

func CreateClient(apiURL, apiToken, userAgent string) (*ClientWithResponses, error) {
	httpClient, editors := client.GetHttpClient(apiToken, userAgent)
	httpClientOption := func(client *Client) error {
		client.Client = httpClient

		for _, editor := range editors {
			client.RequestEditors = append(client.RequestEditors, editor)
		}

		return nil
	}

	apiClient, err := NewClientWithResponses(apiURL, httpClientOption)
	if err != nil {
		return nil, err
	}

	if checkErr := CheckGetResponse(apiClient.AuthTokenAPIListAuthTokensWithResponse(context.Background(), &AuthTokenAPIListAuthTokensParams{})); checkErr != nil {
		return nil, fmt.Errorf("validating api token (by listing auth tokens): %v", checkErr)
	}

	return apiClient, nil
}
