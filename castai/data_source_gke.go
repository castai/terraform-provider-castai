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
	// policiesResourceName is the name of the resource
	policiesResourceName = "policy"
	// featuresResourceName is the name of the policies per feature
	featuresResourceName                        = "features"
	loadBalancersNetworkEndpointGroupFeature    = "load_balancers_network_endpoint_group"
	loadBalancersTargetBackendPoolsFeature      = "load_balancers_target_backend_pools"
	loadBalancersUnmanagedInstanceGroupsFeature = "load_balancers_unmanaged_instance_groups"
)

func dataSourceGKEPolicies() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGKEPoliciesRead,
		Description: "Data source for retrieving GKE policies",
		Schema: map[string]*schema.Schema{
			featuresResourceName: {
				Description: "Includes list of policies needed for the GCP features",
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						loadBalancersNetworkEndpointGroupFeature,
						loadBalancersTargetBackendPoolsFeature,
						loadBalancersUnmanagedInstanceGroupsFeature,
					}, false),
				},
			},
			policiesResourceName: {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceGKEPoliciesRead(_ context.Context, data *schema.ResourceData, _ interface{}) diag.Diagnostics {
	// add policies per specified features
	features := data.Get(featuresResourceName).([]interface{})
	policySet := make(map[string]struct{})

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

	if err := data.Set(policiesResourceName, allPolicies); err != nil {
		return diag.FromErr(fmt.Errorf("setting %s policy: %w", policiesResourceName, err))
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
