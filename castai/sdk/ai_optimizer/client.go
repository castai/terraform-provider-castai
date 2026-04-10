package ai_optimizer

import (
	"github.com/castai/terraform-provider-castai/castai/sdk/client"
)

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
