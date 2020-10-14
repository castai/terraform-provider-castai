package castai

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/terraform-providers/terraform-provider-castai/castai/sdk"
)

type Config struct {
	ApiUrl   string
	ApiToken string
}

type ProviderConfig struct {
	api *sdk.ClientWithResponses
}

func (c *Config) configureProvider() (interface{}, error) {
	baseURL, err := url.Parse(c.ApiUrl)
	if err != nil {
		return nil, err
	}
	if baseURL.String() == "" {
		baseURL.Path = "https://api.cast.ai/"
	}

	apiClient, err := sdk.NewClientWithResponses(c.ApiUrl, sdk.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-API-Key", c.ApiToken)
		return nil
	}))
	if err != nil {
		return nil, err
	}

	if checkErr := sdk.CheckGetResponse(apiClient.ListAuthTokensWithResponse(context.Background())); checkErr != nil {
		return nil, fmt.Errorf("validating api token (by listing auth tokens): %v", checkErr)
	}

	return &ProviderConfig{
		api: apiClient,
	}, nil
}
