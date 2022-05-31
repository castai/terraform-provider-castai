package castai

import (
	"github.com/castai/terraform-provider-castai/castai/sdk"
)

type Config struct {
	ApiUrl   string
	ApiToken string
}

type ProviderConfig struct {
	api *sdk.ClientWithResponses
}

func (c *Config) configureProvider() (interface{}, error) {
	apiClient, err := sdk.CreateClient(c.ApiUrl, c.ApiToken, "castai-terraform-provider")

	if err != nil {
		return nil, err
	}

	return &ProviderConfig{
		api: apiClient,
	}, nil
}
