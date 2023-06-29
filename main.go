package main

import (
	"context"
	"flag"
	"log"

	"github.com/datastax/terraform-provider-astra/v2/internal/astra"
	"github.com/datastax/terraform-provider-astra/v2/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
)

const providerName = "datastax/astra"

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary
	version string = "dev"

	// goreleaser can also pass the specific commit if you want
	// commit  string = ""
)

func main() {
	debugFlag := flag.Bool("debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	ctx := context.Background()

	upgradedLegacySdkProvider, err := tf5to6server.UpgradeServer(
		context.Background(),
		provider.New(version)().GRPCProvider,
	)
	if err != nil {
		log.Fatal(err)
	}

	providers := []func() tfprotov6.ProviderServer{
		// Legacy Astra provider using the Terraform Plugin SDK
		func() tfprotov6.ProviderServer {
			return upgradedLegacySdkProvider
		},

		// New Astra provider using the Terraform Plugin Framework
		providerserver.NewProtocol6(astra.New(version)()),
	}

	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)

	// oldProviders := []func() tfprotov5.ProviderServer{
	// 	// Legacy plugin sdk provider
	// 	provider.New(version)().GRPCProvider,

	// 	// New provider using plugin framework
	// 	providerserver.NewProtocol5(
	// 		astra.New(version),
	// 	),
	// }

	// muxServer, err := tf5muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf6server.ServeOpt

	if *debugFlag {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}

	err = tf6server.Serve(
		"registry.terraform.io/"+providerName,
		muxServer.ProviderServer,
		serveOpts...,
	)

	if err != nil {
		log.Fatal(err)
	}
}
