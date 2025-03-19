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
	// GKEPoliciesResourceName is the name of the resource
	GKEPoliciesResourceName = "policy"
	// GKEFeaturesResourceName is the name of the policies per feature
	GKEFeaturesResourceName                        = "features"
	GKELoadBalancersNetworkEndpointGroupFeature    = "load_balancers_network_endpoint_group"
	GKELoadBalancersTargetBackendPoolsFeature      = "load_balancers_target_backend_pools"
	GKELoadBalancersUnmanagedInstanceGroupsFeature = "load_balancers_unmanaged_instance_groups"
)

func dataSourceGKEPolicies() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGKEPoliciesRead,
		Schema: map[string]*schema.Schema{
			GKEFeaturesResourceName: {
				Type:     schema.TypeList,
				Computed: true,
				Default:  []string{},
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						GKELoadBalancersNetworkEndpointGroupFeature,
						GKELoadBalancersTargetBackendPoolsFeature,
						GKELoadBalancersUnmanagedInstanceGroupsFeature,
					}, false),
				},
			},
			GKEPoliciesResourceName: {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceGKEPoliciesRead(_ context.Context, data *schema.ResourceData, _ interface{}) diag.Diagnostics {
	// add policies per specified features
	features := data.Get(GKEFeaturesResourceName).([]interface{})
	policySet := make(map[string]struct{})

	for _, feature := range features {
		var err error
		var policies []string

		switch feature {
		case GKELoadBalancersNetworkEndpointGroupFeature:
			policies, err = gke.GetLoadBalancersNetworkEndpointGroupPolicy()
		case GKELoadBalancersTargetBackendPoolsFeature:
			policies, err = gke.GetLoadBalancersTargetBackendPoolsPolicy()
		case GKELoadBalancersUnmanagedInstanceGroupsFeature:
			policies, err = gke.GetLoadBalancersUnmanagedInstanceGroupsPolicy()
		}

		if err != nil {
			return diag.FromErr(fmt.Errorf("getting %s policy: %w", feature, err))
		}

		policySet = appendArrayToMap(policies, policySet)
	}

	// add base user policies
	userPolicy, err := gke.GetUserPolicy()
	if err != nil {
		return diag.FromErr(fmt.Errorf("getting user policy: %w", err))
	}
	policySet = appendArrayToMap(userPolicy, policySet)

	var allPolicies []string
	for policy := range policySet {
		allPolicies = append(allPolicies, policy)
	}

	if err := data.Set(GKEPoliciesResourceName, allPolicies); err != nil {
		return diag.FromErr(fmt.Errorf("setting %s policy: %w", GKEPoliciesResourceName, err))
	}

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
