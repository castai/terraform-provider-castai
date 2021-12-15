package castai

import (
	"context"
	"github.com/castai/terraform-provider-castai/castai/policies/gke"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGkePolicies() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGkePoliciesRead,
		Schema: map[string]*schema.Schema {
			"policy":  {
				Type: schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceGkePoliciesRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	policies, _ := gke.GetUserPolicy()
	data.SetId("gke")
	data.Set("policy", policies)

	return nil
}
