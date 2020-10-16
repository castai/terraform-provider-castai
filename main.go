package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/castai/terraform-provider-castai/castai"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: castai.Provider,
	})
}
