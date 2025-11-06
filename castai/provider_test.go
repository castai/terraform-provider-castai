package castai

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	ProviderName   = "castai"
	ResourcePrefix = "tf-acc-test"
)

var (
	testAccProviderConfigure sync.Once

	providerFactories               map[string]func() (*schema.Provider, error)
	testAccProvider                 *schema.Provider
	testAccProtoV6ProviderFactories map[string]func() (tfprotov6.ProviderServer, error)
)

func init() {
	testAccProvider = Provider("v1.0.0")
	providerFactories = map[string]func() (*schema.Provider, error){
		ProviderName: func() (*schema.Provider, error) {
			return testAccProvider, nil
		},
	}

	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		ProviderName: func() (tfprotov6.ProviderServer, error) {
			ctx := context.Background()
			upgradedSdkProvider, err := tf5to6server.UpgradeServer(ctx, testAccProvider.GRPCProvider)
			if err != nil {
				return nil, err
			}

			providers := []func() tfprotov6.ProviderServer{
				func() tfprotov6.ProviderServer {
					return upgradedSdkProvider
				},
				providerserver.NewProtocol6(NewFrameworkProvider("v1.0.0")),
			}

			muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
			if err != nil {
				return nil, err
			}

			return muxServer.ProviderServer(), nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := Provider("v1.0.0").InternalValidate(); err != nil {
		t.Fatalf("internal consistency validation failed: %v", err)
	}
}

func testAccPreCheck(t *testing.T) {
	testAccProviderConfigure.Do(func() {
		if os.Getenv("CASTAI_API_URL") == "" {
			// Run acceptance on dev by default if not set.
			os.Setenv("CASTAI_API_URL", "https://api.dev-master.cast.ai")
		}

		if v := os.Getenv("CASTAI_API_TOKEN"); v == "" {
			t.Fatal("CASTAI_API_TOKEN must be set for acceptance tests")
		}

		if v := os.Getenv("ACCEPTANCE_TEST_ORGANIZATION_ID"); v == "" {
			t.Fatal("ACCEPTANCE_TEST_ORGANIZATION_ID must be set for acceptance tests")
		}

		if err := testAccProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil)); err != nil {
			t.Fatal(err)
		}
	})
}

// ConfigCompose can be called to concatenate multiple strings to build test configurations
func ConfigCompose(config ...string) string {
	var str strings.Builder
	for _, conf := range config {
		str.WriteString(conf)
	}
	return str.String()
}

// testAccGetOrganizationID returns the organization ID from environment variable
func testAccGetOrganizationID() string {
	return os.Getenv("ACCEPTANCE_TEST_ORGANIZATION_ID")
}