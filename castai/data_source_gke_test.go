package castai

import (
	"context"
	"github.com/castai/terraform-provider-castai/castai/policies/gke"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_dataSourceGKEPoliciesRead(t *testing.T) {
	up, _ := gke.GetUserPolicy()
	lbNeg, _ := gke.GetLoadBalancersNetworkEndpointGroupPolicy()
	lbTbp, _ := gke.GetLoadBalancersTargetBackendPoolsPolicy()
	lbUig, _ := gke.GetLoadBalancersUnmanagedInstanceGroupsPolicy()
	tests := []struct {
		name     string
		features []interface{}
		expected int
		hasError bool
	}{
		{
			name: "all features",
			features: []interface{}{
				loadBalancersNetworkEndpointGroupFeature,
				loadBalancersTargetBackendPoolsFeature,
				loadBalancersUnmanagedInstanceGroupsFeature,
			},
			expected: len(up) + len(lbNeg) + len(lbTbp) + len(lbUig) - 1, // -1 for the duplicate policy
			hasError: false,
		},
		{
			name:     "empty features",
			expected: len(up),
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
			r.NoError(data.Set(featuresResourceName, tt.features))

			result := resource.ReadContext(ctx, data, provider)
			if tt.hasError {
				r.True(result.HasError())
			} else {
				r.Nil(result)
				r.False(result.HasError())
				actualPolicies := data.Get(policiesResourceName).([]interface{})
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
					resource.TestCheckResourceAttr("data.castai_gke_user_policies.gke", "features.#", "3"),
					resource.TestCheckResourceAttr("data.castai_gke_user_policies.gke", "policy.#", "46"),
				),
			},
		},
	})
}

const testAccDataSourceGKEPoliciesConfig = `
data "castai_gke_user_policies" "gke" {
  features = [
    "load_balancers_network_endpoint_group",
    "load_balancers_target_backend_pools",
    "load_balancers_unmanaged_instance_groups"
  ]
}
`
