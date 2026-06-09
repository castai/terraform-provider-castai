package castai

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldScalingPolicyOrderIDs = "policy_ids"
)

func resourceWorkloadScalingPolicyOrder() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceWorkloadScalingPolicyOrderSet,
		ReadContext:   resourceWorkloadScalingPolicyOrderRead,
		UpdateContext: resourceWorkloadScalingPolicyOrderSet,
		DeleteContext: resourceWorkloadScalingPolicyOrderDelete,
		Importer: &schema.ResourceImporter{
			StateContext: workloadScalingPolicyOrderImporter,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldClusterID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "CAST AI cluster id",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldScalingPolicyOrderIDs: {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of scaling policy IDs in the order they should be applied.",
				Elem: &schema.Schema{
					ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
					Type:             schema.TypeString,
				},
			},
		},
	}
}

func resourceWorkloadScalingPolicyOrderRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := d.Get(FieldClusterID).(string)
	list, err := client.WorkloadOptimizationAPIListWorkloadScalingPoliciesWithResponse(ctx, clusterID)
	if err := sdk.CheckOKResponse(list, err); err != nil {
		return diag.FromErr(err)
	}

	if !d.IsNewResource() && len(list.JSON200.Items) == 0 {
		tflog.Warn(ctx, "Scaling policies not found, removing order from state", map[string]any{FieldClusterID: clusterID})
		d.SetId("")
		return nil
	}

	var policyIds []string
	for _, sp := range list.JSON200.Items {
		policyIds = append(policyIds, sp.Id)
	}

	if d.Id() != clusterID {
		d.SetId(clusterID)
	}
	if err := d.Set(FieldScalingPolicyOrderIDs, policyIds); err != nil {
		return diag.FromErr(fmt.Errorf("setting policy: %w", err))
	}

	return nil
}

func resourceWorkloadScalingPolicyOrderSet(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterId := d.Get(FieldClusterID).(string)
	policyIds := toStringList(d.Get(FieldScalingPolicyOrderIDs).([]any))

	req := sdk.WorkloadOptimizationAPISetScalingPoliciesOrderRequest{
		PolicyIds: &policyIds,
	}

	resp, err := client.WorkloadOptimizationAPISetScalingPoliciesOrderWithResponse(ctx, clusterId, req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(clusterId)
	return resourceWorkloadScalingPolicyOrderRead(ctx, d, meta)
}

func resourceWorkloadScalingPolicyOrderDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	// We don't need to do anything special for deletion.
	// We could try to restore the initial order before the TF was used, but we don't have that information.
	return nil
}

func workloadScalingPolicyOrderImporter(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	clusterId := d.Id()
	if err := d.Set(FieldClusterID, clusterId); err != nil {
		return nil, err
	}

	tflog.Info(ctx, "imported workload scaling policy order", map[string]any{
		FieldClusterID: clusterId,
	})

	return []*schema.ResourceData{d}, nil
}
