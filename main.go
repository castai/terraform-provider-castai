package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"

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

	ctx := context.Background()

	// Upgrade SDKv2 provider from protocol v5 to v6
	upgradedSDKProvider, err := tf5to6server.UpgradeServer(
		ctx,
		castai.Provider(version).GRPCProvider,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Create muxed provider combining SDKv2 and Framework providers
	muxServer, err := tf6muxserver.NewMuxServer(
		ctx,
		providerserver.NewProtocol6(castai.NewFrameworkProvider(version)),
		func() tfprotov6.ProviderServer { return upgradedSDKProvider },
	)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf6server.ServeOpt
	if debug {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}

	err = tf6server.Serve(
		"registry.terraform.io/castai/castai",
		func() tfprotov6.ProviderServer {
			return muxServer.ProviderServer()
		},
		serveOpts...,
	)

	if err != nil {
		log.Fatal(err)
	}
}
