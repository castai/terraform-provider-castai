package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func dataSourceWorkloadScalingPolicies() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceWorkloadScalingPoliciesRead,
		Description: "Lists all workload scaling policies for a cluster. Useful for dynamically " +
			"resolving policy IDs by name without maintaining a hardcoded UUID map.",
		Schema: map[string]*schema.Schema{
			FieldClusterID: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "CAST AI cluster id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			"policies": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of all scaling policies in the cluster.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Policy UUID.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Policy name.",
						},
						"is_default": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether this is the default scaling policy for the cluster.",
						},
						"is_readonly": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether this policy is read-only (cannot be updated or deleted).",
						},
						"is_castware": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether this policy is managed by CAST AI and only applies to castware workloads.",
						},
					},
				},
			},
		},
	}
}

func dataSourceWorkloadScalingPoliciesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := d.Get(FieldClusterID).(string)

	list, err := client.WorkloadOptimizationAPIListWorkloadScalingPoliciesWithResponse(ctx, clusterID)
	if e := sdk.CheckOKResponse(list, err); e != nil {
		return diag.FromErr(fmt.Errorf("listing scaling policies: %w", e))
	}

	policies := make([]map[string]any, 0, len(list.JSON200.Items))
	for _, p := range list.JSON200.Items {
		policies = append(policies, map[string]any{
			"id":          p.Id,
			"name":        p.Name,
			"is_default":  p.IsDefault,
			"is_readonly": p.IsReadonly,
			"is_castware": p.IsCastware,
		})
	}

	d.SetId(clusterID)
	if err := d.Set("policies", policies); err != nil {
		return diag.FromErr(fmt.Errorf("setting policies: %w", err))
	}

	return nil
}
