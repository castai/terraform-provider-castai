package castai

import (
	"context"
	"fmt"

	"github.com/castai/terraform-provider-castai/v7/castai/policies/gke"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGKEPolicies() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGKEPoliciesRead,
		Schema: map[string]*schema.Schema{
			"policy": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceGKEPoliciesRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	policies, _ := gke.GetUserPolicy()
	data.SetId("gke")
	if err := data.Set("policy", policies); err != nil {
		return diag.FromErr(fmt.Errorf("setting gke policy: %w", err))
	}

	return nil
}
