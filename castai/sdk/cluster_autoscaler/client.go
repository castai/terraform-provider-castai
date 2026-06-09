package cluster_autoscaler

import (
	"github.com/castai/terraform-provider-castai/castai/sdk/client"
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

	return apiClient, nil
}
