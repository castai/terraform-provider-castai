package castai

import (
	"context"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_dataSourceGKEPoliciesRead(t *testing.T) {
	tests := []struct {
		name     string
		features []interface{}
		expected int
		hasError bool
	}{
		{
			name: "all features",
			features: []interface{}{
				loadBalancersTargetBackendPoolsFeature,
				loadBalancersUnmanagedInstanceGroupsFeature,
			},
			expected: 42, // -1 for the duplicate policy
			hasError: false,
		},
		{
			name: "loadBalancersTargetBackendPoolsFeature",
			features: []interface{}{
				loadBalancersTargetBackendPoolsFeature,
			},
			expected: 41, // -1 for the duplicate policy
			hasError: false,
		},
		{
			name:     "empty features",
			expected: 37,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			//mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

			ctx := context.Background()
			provider := &ProviderConfig{
				api: &sdk.ClientWithResponses{
					ClientInterface: nil,
				},
			}

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)

			resource := dataSourceGKEPolicies()
			data := resource.Data(state)
			r.NoError(data.Set(fieldGKEPoliciesFeatures, tt.features))

			result := resource.ReadContext(ctx, data, provider)
			if tt.hasError {
				r.True(result.HasError())
			} else {
				r.Nil(result)
				r.False(result.HasError())
				actualPolicies := data.Get(fieldGKEPoliciesPolicy).([]interface{})
				r.Len(actualPolicies, tt.expected)
			}
		})
	}
}

func TestAccDataSourceGKEPolicies_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceGKEPoliciesConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.castai_gke_user_policies.gke", "features.#", "2"),
					resource.TestCheckResourceAttr("data.castai_gke_user_policies.gke", "policy.#", "42"),
				),
			},
			{
				Config: testAccDataSourceGKEPoliciesConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.castai_gke_user_policies.gke", "features.#", "1"),
					resource.TestCheckResourceAttr("data.castai_gke_user_policies.gke", "policy.#", "41"),
				),
			},
		},
	})
}

const testAccDataSourceGKEPoliciesConfig = `
data "castai_gke_user_policies" "gke" {
  features = [
    "load_balancers_target_backend_pools",
    "load_balancers_unmanaged_instance_groups"
  ]
}
`
const testAccDataSourceGKEPoliciesConfigUpdated = `
data "castai_gke_user_policies" "gke" {
  features = [
    "load_balancers_target_backend_pools"
  ]
}
`
