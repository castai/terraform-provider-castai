package castai

import (
	"context"

	"github.com/castai/terraform-provider-castai/castai/policies/gcp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGcpPolicies() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGcpPoliciesRead,
		Schema: map[string]*schema.Schema {
			"policy":  {
				Type: schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceGcpPoliciesRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	policies, _ := gcp.GetIAMPolicy()
	data.Set("policy", policies)

	return nil
}