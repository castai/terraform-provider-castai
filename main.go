package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"

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

	// Create muxed provider combining SDKv2 and Framework providers
	muxServer, err := tf5muxserver.NewMuxServer(
		ctx,
		providerserver.NewProtocol5(castai.NewFrameworkProvider(version)),
		castai.Provider(version).GRPCProvider,
	)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf5server.ServeOpt
	if debug {
		serveOpts = append(serveOpts, tf5server.WithManagedDebug())
	}

	err = tf5server.Serve(
		"registry.terraform.io/castai/castai",
		func() tfprotov5.ProviderServer {
			return muxServer.ProviderServer()
		},
		serveOpts...,
	)

	if err != nil {
		log.Fatal(err)
	}
}
