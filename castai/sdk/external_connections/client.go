package external_connections

import (
	"github.com/castai/terraform-provider-castai/castai/sdk/client"
)

// CreateClient builds an authenticated ClientWithResponses for the ExternalConnectionsAPI.
// It mirrors the hand-written helpers in the other SDK packages (e.g. cluster_autoscaler),
// wiring the shared authed HTTP client and request editors into the generated
// NewClientWithResponses constructor.
func CreateClient(apiURL, apiToken, userAgent string) (*ClientWithResponses, error) {
	httpClient, editors := client.GetHttpClient(apiToken, userAgent)
	httpClientOption := func(c *Client) error {
		c.Client = httpClient

		for _, editor := range editors {
			c.RequestEditors = append(c.RequestEditors, editor)
		}

		return nil
	}

	apiClient, err := NewClientWithResponses(apiURL, httpClientOption)
	if err != nil {
		return nil, err
	}

	return apiClient, nil
}
