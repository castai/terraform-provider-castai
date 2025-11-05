package omni_provisioner

import (
	"context"
	"fmt"
	"net/http"

	"github.com/castai/terraform-provider-castai/castai/sdk/client"
)

func CreateClient(apiURL, apiToken, agent string) (ClientWithResponsesInterface, error) {
	httpClient := client.New(apiURL, apiToken, agent)

	c, err := NewClientWithResponses(apiURL, WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("creating omni provisioner client: %w", err)
	}
	return c, nil
}

// WithHTTPClient is a functional option to set the HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) error {
		c.Client = client
		return nil
	}
}

// RequestEditorFn is a function that edits an HTTP request before sending.
type RequestEditorFn func(ctx context.Context, req *http.Request) error
