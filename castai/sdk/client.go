package sdk

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
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
	httpClientOption := func(client *Client) error {
		client.Client = &http.Client{
			Transport: logging.NewSubsystemLoggingHTTPTransport("CAST.AI", http.DefaultTransport),
			Timeout:   1 * time.Minute,
		}
		client.RequestEditors = append(client.RequestEditors, func(_ context.Context, req *http.Request) error {
			req.Header.Set("user-agent", userAgent)
			return nil
		})
		return nil
	}

	apiTokenOption := WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-API-Key", apiToken)
		return nil
	})

	apiClient, err := NewClientWithResponses(apiURL, httpClientOption, apiTokenOption)
	if err != nil {
		return nil, err
	}

	if checkErr := CheckGetResponse(apiClient.ListAuthTokensWithResponse(context.Background(), &ListAuthTokensParams{})); checkErr != nil {
		return nil, fmt.Errorf("validating api token (by listing auth tokens): %v", checkErr)
	}

	return apiClient, nil
}
