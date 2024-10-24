package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/castai/terraform-provider-castai/v7/castai"
)

var (
	commit  = "undefined" // nolint:unused
	version = "local"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return castai.Provider(version)
		},
	})
}
