package castai

import (
	"context"

	"github.com/castai/terraform-provider-castai/castai/policies"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAKSPolicies() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAKSPoliciesRead,
		Schema: map[string]*schema.Schema {
			"policy":  {
				Type: schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAKSPoliciesRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	policies, _ := policies.GetAKSPolicy()
	data.SetId("aks")
	data.Set("policy", policies)

	return nil
}
