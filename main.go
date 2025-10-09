package main

import (
	"flag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/castai/terraform-provider-castai/castai"
)

var (
	commit  = "undefined" // nolint:unused
	version = "local"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{
		Debug:        debug,
		ProviderAddr: "registry.terraform.io/castai/castai",
		ProviderFunc: func() *schema.Provider {
			return castai.Provider(version)
		},
	}

	plugin.Serve(opts)
}
