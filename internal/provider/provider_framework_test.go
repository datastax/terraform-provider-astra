package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
)

const (
	version                         = "testing"
	testDefaultStreamingClusterName = "pulsar-gcp-useast1-staging"

	// providerConfig is a shared configuration to combine with the actual
	// test configuration so the HashiCups client is properly configured.
	// It is also possible to use the HASHICUPS_ environment variables instead,
	// such as updating the Makefile and running the testing through that tool.
	testProviderConfig = `
provider "astra" {
}
`
)

var (
	// upgradedLegacySdkProvider, _ = tf5to6server.UpgradeServer(
	// 	context.Background(),
	// 	NewSDKProvider(version)().GRPCProvider,
	// )

	testAccProvidersFramework = []func() tfprotov6.ProviderServer{
		// Legacy provider using plugin sdk
		NewSDKProviderV6(version),

		// New provider using plugin framework
		providerserver.NewProtocol6(New(version)()),
	}
	testAccMuxProvider = func() (tfprotov6.ProviderServer, error) {
		ctx := context.Background()
		return tf6muxserver.NewMuxServer(ctx, testAccProvidersFramework...)
	}
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"astra": testAccMuxProvider,
	}
)

func testAccPreCheck(t *testing.T) {
	if err := os.Getenv("ASTRA_API_TOKEN"); err == "" {
		t.Fatal("ASTRA_API_TOKEN must be set for acceptance tests")
	}
}
