package castai

import (
	"context"
	"fmt"

	"github.com/castai/terraform-provider-castai/castai/policies/gke"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	// fieldGKEPoliciesPolicy is the name of the resource
	fieldGKEPoliciesPolicy = "policy"
	// fieldGKEPoliciesFeatures is the name of the policies per feature
	fieldGKEPoliciesFeatures                    = "features"
	loadBalancersNetworkEndpointGroupFeature    = "load_balancers_network_endpoint_group"
	loadBalancersTargetBackendPoolsFeature      = "load_balancers_target_backend_pools"
	loadBalancersUnmanagedInstanceGroupsFeature = "load_balancers_unmanaged_instance_groups"
)

func dataSourceGKEPolicies() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGKEPoliciesRead,
		Description: "Returns list of GCP policies needed for onboarding a cluster into CAST AI",
		Schema: map[string]*schema.Schema{
			fieldGKEPoliciesFeatures: {
				Description: "Provide a list of GCP feature names to include the necessary policies for them to work.",
				Type:        schema.TypeList,
				ForceNew:    true,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						loadBalancersNetworkEndpointGroupFeature,
						loadBalancersTargetBackendPoolsFeature,
						loadBalancersUnmanagedInstanceGroupsFeature,
					}, false),
				},
			},
			fieldGKEPoliciesPolicy: {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceGKEPoliciesRead(_ context.Context, data *schema.ResourceData, _ interface{}) diag.Diagnostics {
	// add policies per specified features
	features, ok := data.Get(fieldGKEPoliciesFeatures).([]interface{})
	if !ok {
		return diag.FromErr(fmt.Errorf("failed to retrieve features"))
	}

	// Initialize policy set
	policySet := make(map[string]struct{})

	// Process each feature
	for _, feature := range features {
		var err error
		var policies []string

		switch feature {
		case loadBalancersNetworkEndpointGroupFeature:
			policies, err = gke.GetLoadBalancersNetworkEndpointGroupPolicy()
		case loadBalancersTargetBackendPoolsFeature:
			policies, err = gke.GetLoadBalancersTargetBackendPoolsPolicy()
		case loadBalancersUnmanagedInstanceGroupsFeature:
			policies, err = gke.GetLoadBalancersUnmanagedInstanceGroupsPolicy()
		default:
			return diag.FromErr(fmt.Errorf("unknown feature: %s", feature))
		}

		if err != nil {
			return diag.FromErr(fmt.Errorf("getting %s policy: %w", feature, err))
		}

		policySet = appendArrayToMap(policies, policySet)
	}

	// Add base user policies
	userPolicy, err := gke.GetUserPolicy()
	if err != nil {
		return diag.FromErr(fmt.Errorf("getting user policy: %w", err))
	}
	policySet = appendArrayToMap(userPolicy, policySet)

	var allPolicies []string
	for policy := range policySet {
		allPolicies = append(allPolicies, policy)
	}

	if err := data.Set(fieldGKEPoliciesPolicy, allPolicies); err != nil {
		return diag.FromErr(fmt.Errorf("setting %s policy: %w", fieldGKEPoliciesPolicy, err))
	}
	data.SetId("gke")

	return nil
}

func appendArrayToMap(arr []string, m map[string]struct{}) map[string]struct{} {
	if m == nil {
		m = make(map[string]struct{})
	}
	for _, v := range arr {
		m[v] = struct{}{}
	}
	return m
}
