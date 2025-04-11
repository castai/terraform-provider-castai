package castai

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	ProviderName   = "castai"
	ResourcePrefix = "tf-acc-test"
)

var (
	testAccProviderConfigure sync.Once

	providerFactories map[string]func() (*schema.Provider, error)
	testAccProvider   *schema.Provider
)

func init() {
	testAccProvider = Provider("v1.0.0")
	providerFactories = map[string]func() (*schema.Provider, error){
		ProviderName: func() (*schema.Provider, error) {
			return testAccProvider, nil
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
