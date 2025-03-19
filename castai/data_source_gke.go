package castai

import (
	"context"
	"fmt"

	"github.com/castai/terraform-provider-castai/castai/policies/gke"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	// GKEPoliciesResourceName is the name of the resource
	GKEPoliciesResourceName = "policy"
	// GKELoadBalancersNetworkEndpointGroupPoliciesResourceName is the name of the resource
	GKELoadBalancersNetworkEndpointGroupPoliciesResourceName = "castai_gke_load_balancers_network_endpoint_group_policies"
	// GKELoadBalancersTargetBackendPoolsPoliciesResourceName is the name of the resource
	GKELoadBalancersTargetBackendPoolsPoliciesResourceName = "castai_gke_load_balancers_target_backend_pools_policies"
	// GKELoadBalancersUnmanagedInstanceGroupsPoliciesResourceName is the name of the resource
	GKELoadBalancersUnmanagedInstanceGroupsPoliciesResourceName = "castai_gke_load_balancers_unmanaged_instance_groups_policies"
)

func dataSourceGKEPolicies() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGKEPoliciesRead,
		Schema: map[string]*schema.Schema{
			GKEPoliciesResourceName: {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			GKELoadBalancersNetworkEndpointGroupPoliciesResourceName: {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			GKELoadBalancersTargetBackendPoolsPoliciesResourceName: {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			GKELoadBalancersUnmanagedInstanceGroupsPoliciesResourceName: {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceGKEPoliciesRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	diags = append(diags, dataSourceGKEUserPoliciesRead(ctx, data, meta)...)
	diags = append(diags, dataSourceGKELoadBalancersNetworkEndpointGroupPoliciesRead(ctx, data, meta)...)
	diags = append(diags, dataSourceGKELoadBalancersTargetBackendPoolsPoliciesRead(ctx, data, meta)...)
	diags = append(diags, dataSourceGKELoadBalancersUnmanagedInstanceGroupsPoliciesRead(ctx, data, meta)...)

	return diags
}

func dataSourceGKEUserPoliciesRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	policies, _ := gke.GetUserPolicy()
	data.SetId("gke")
	if err := data.Set(GKEPoliciesResourceName, policies); err != nil {
		return diag.FromErr(fmt.Errorf("setting gke policy: %w", err))
	}

	return nil
}

func dataSourceGKELoadBalancersNetworkEndpointGroupPoliciesRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	policies, _ := gke.GetLoadBalancersNetworkEndpointGroupPolicy()
	data.SetId("gke")
	if err := data.Set(GKELoadBalancersNetworkEndpointGroupPoliciesResourceName, policies); err != nil {
		return diag.FromErr(fmt.Errorf("setting gke policy: %w", err))
	}

	return nil
}

func dataSourceGKELoadBalancersTargetBackendPoolsPoliciesRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	policies, _ := gke.GetLoadBalancersTargetBackendPoolsPolicy()
	data.SetId("gke")
	if err := data.Set(GKELoadBalancersTargetBackendPoolsPoliciesResourceName, policies); err != nil {
		return diag.FromErr(fmt.Errorf("setting gke policy: %w", err))
	}

	return nil
}

func dataSourceGKELoadBalancersUnmanagedInstanceGroupsPoliciesRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	policies, _ := gke.GetLoadBalancersUnmanagedInstanceGroupsPolicy()
	data.SetId("gke")
	if err := data.Set(GKELoadBalancersUnmanagedInstanceGroupsPoliciesResourceName, policies); err != nil {
		return diag.FromErr(fmt.Errorf("setting gke policy: %w", err))
	}

	return nil
}
