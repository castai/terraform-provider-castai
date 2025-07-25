package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldScalingPolicyIDs = "policy_ids"
)

func dataSourceWorkloadScalingPolicies() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceWorkloadScalingPoliciesRead,
		Description: "Returns a list of all workload scaling policies attached to a Cast AI cluster",
		Schema: map[string]*schema.Schema{
			FieldClusterID: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "CAST AI cluster id",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldScalingPolicyIDs: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of scaling policy IDs in the order they should be applied.",
				Elem: &schema.Schema{
					ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
					Type:             schema.TypeString,
				},
			},
		},
	}
}

func dataSourceWorkloadScalingPoliciesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := d.Get(FieldClusterID).(string)
	list, err := client.WorkloadOptimizationAPIListWorkloadScalingPoliciesWithResponse(ctx, clusterID)
	if err := sdk.CheckOKResponse(list, err); err != nil {
		return diag.FromErr(err)
	}

	policyIDs := make([]string, 0, len(list.JSON200.Items))
	for _, item := range list.JSON200.Items {
		policyIDs = append(policyIDs, item.Id)
	}

	d.SetId(fmt.Sprintf("policies-%s", clusterID))
	if err := d.Set(FieldScalingPolicyIDs, policyIDs); err != nil {
		return diag.FromErr(fmt.Errorf("setting policy IDs: %w", err))
	}

	return nil
}
