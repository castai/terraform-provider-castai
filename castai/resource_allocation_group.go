package castai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func resourceAllocationGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAllocationGroupCreate,
		ReadContext:   resourceAllocationGroupRead,
		UpdateContext: resourceAllocationGroupUpdate,
		DeleteContext: resourceAllocationGroupDelete,
		CustomizeDiff: resourceAllocationGroupDiff,
		Description:   "Manage allocation group. Allocation group [reference](https://docs.cast.ai/docs/allocation-groups)",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Allocation group name",
			},
			"cluster_ids": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "List of CAST AI cluster ids",
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				},
			},
			"namespaces": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of cluster namespaces to track",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"labels": {
				Type:        schema.TypeMap,
				Description: "Labels used to select workloads to track",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"labels_operator": {
				Type: schema.TypeString,
				Description: `Operator with which to connect the labels
	OR (default) - workload needs to have at least one label to be included
	AND - workload needs to have all the labels to be included`,
				Optional: true,
				Default:  sdk.OR,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
					string(sdk.AND), string(sdk.OR),
				}, false)),
			},
		},
	}
}

func resourceAllocationGroupDiff(_ context.Context, d *schema.ResourceDiff, _ any) error {
	var clusterIds []string
	if cids, ok := d.GetOk("cluster_ids"); ok {
		clusterIds = toClusterIds(cids.(*schema.Set).List())
	}
	namespaces := toStringList(d.Get("namespaces").([]interface{}))

	var labels []sdk.CostreportV1beta1AllocationGroupFilterLabelValue
	if ls, ok := d.GetOk("labels"); ok {
		labels = toLabels(ls.(map[string]interface{}))
	}

	if len(clusterIds) == 0 && len(namespaces) == 0 && len(labels) == 0 {
		return errors.New("allocation group must specify at least one of: cluster_ids, namespaces, or labels")
	}
	return nil
}

func resourceAllocationGroupRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	resp, err := client.AllocationGroupAPIGetAllocationGroupWithResponse(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if !d.IsNewResource() && resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "Allocation group not found, removing from state", map[string]any{"id": d.Id()})
		d.SetId("")
		return nil
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	ag := resp.JSON200

	d.SetId(*ag.Id)

	if err := d.Set("name", ag.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %w", err))
	}
	if err := d.Set("cluster_ids", *ag.Filter.ClusterIds); err != nil {
		return diag.FromErr(fmt.Errorf("setting cluster_ids: %w", err))
	}
	if err := d.Set("namespaces", ag.Filter.Namespaces); err != nil {
		return diag.FromErr(fmt.Errorf("setting namespaces: %w", err))
	}
	if err := d.Set("labels", fromLabels(*ag.Filter.Labels)); err != nil {
		return diag.FromErr(fmt.Errorf("setting labels: %w", err))
	}
	if err := d.Set("labels_operator", ag.Filter.LabelsOperator); err != nil {
		return diag.FromErr(fmt.Errorf("setting labels_operator: %w", err))
	}
	return nil
}

func resourceAllocationGroupCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	var clusterIds []string
	if cid, ok := d.GetOk("cluster_ids"); ok {
		clusterIds = toClusterIds(cid.(*schema.Set).List())
	}

	allocationGroupName := d.Get("name").(string)

	namespaces := toStringList(d.Get("namespaces").([]interface{}))

	var labels []sdk.CostreportV1beta1AllocationGroupFilterLabelValue
	if ls, ok := d.GetOk("labels"); ok {
		labels = toLabels(ls.(map[string]interface{}))
	}

	labelsOperator := toLabelsOperator(d)

	if len(clusterIds) == 0 && len(namespaces) == 0 && len(labels) == 0 {
		return diag.FromErr(errors.New("allocation group must specify at least one of: cluster_ids, namespaces, or labels"))
	}

	body := sdk.AllocationGroupAPICreateAllocationGroupJSONRequestBody{
		Filter: &sdk.CostreportV1beta1AllocationGroupFilter{
			ClusterIds:     &clusterIds,
			Labels:         &labels,
			LabelsOperator: labelsOperator,
			Namespaces:     &namespaces,
		},
		Name: &allocationGroupName,
	}
	create, err := client.AllocationGroupAPICreateAllocationGroupWithResponse(ctx, body)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error calling create allocation group: %w", err))
	}
	switch create.StatusCode() {
	case http.StatusOK:
		d.SetId(*create.JSON200.Id)
		return resourceAllocationGroupRead(ctx, d, meta)
	default:
		return diag.Errorf("expected status code %d, received: status=%d body=%s", http.StatusOK, create.StatusCode(), string(create.GetBody()))
	}
}

func resourceAllocationGroupUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	if !d.HasChanges(
		"name",
		"cluster_ids",
		"namespaces",
		"labels",
		"labels_operator",
	) {
		tflog.Info(ctx, "allocation group up to date")
		return nil
	}

	client := meta.(*ProviderConfig).api

	allocationGroupName := d.Get("name").(string)

	var clusterIds []string
	if cids, ok := d.GetOk("cluster_ids"); ok {
		clusterIds = toClusterIds(cids.(*schema.Set).List())
	}

	var labels []sdk.CostreportV1beta1AllocationGroupFilterLabelValue
	if ls, ok := d.GetOk("labels"); ok {
		labels = toLabels(ls.(map[string]interface{}))
	}

	namespaces := toStringList(d.Get("namespaces").([]interface{}))

	if len(clusterIds) == 0 && len(namespaces) == 0 && len(labels) == 0 {
		return diag.FromErr(errors.New("allocation group must specify at least one of: cluster_ids, namespaces, or labels"))
	}

	req := sdk.AllocationGroupAPIUpdateAllocationGroupJSONRequestBody{
		Name: &allocationGroupName,
		Filter: &sdk.CostreportV1beta1AllocationGroupFilter{
			ClusterIds:     &clusterIds,
			Labels:         &labels,
			LabelsOperator: toLabelsOperator(d),
			Namespaces:     &namespaces,
		},
	}

	resp, err := client.AllocationGroupAPIUpdateAllocationGroupWithResponse(ctx, d.Id(), req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}
	return resourceAllocationGroupRead(ctx, d, meta)
}

func resourceAllocationGroupDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	response, err := client.AllocationGroupAPIDeleteAllocationGroupWithResponse(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := sdk.StatusOk(response); err != nil {
		return diag.FromErr(err)
	}

	// Poll until the resource no longer exists
	err = retry.RetryContext(ctx, 2*time.Second, func() *retry.RetryError {
		readresp, err := client.AllocationGroupAPIGetAllocationGroupWithResponse(ctx, d.Id())
		if err != nil {
			// Retryable error
			return retry.RetryableError(err)
		}
		if readresp.StatusCode() == http.StatusNotFound {
			// Resource is gone
			return nil
		}

		// Still exists, retry
		return retry.RetryableError(fmt.Errorf("resource %s still exists", d.Id()))
	})

	if err != nil {
		return diag.Errorf("Error waiting for resource %s to be deleted: %s", d.Id(), err)
	}

	return nil
}

func toLabelsOperator(d *schema.ResourceData) *sdk.CostreportV1beta1FilterOperator {
	defaultLabelOperator := sdk.OR
	if v, ok := d.GetOk("labels_operator"); ok {
		if lv := v.(string); lv != "" {
			labelOperator := sdk.CostreportV1beta1FilterOperator(lv)
			return &labelOperator
		}
	}
	return &defaultLabelOperator
}

func toClusterIds(lv []interface{}) []string {
	if len(lv) > 0 {
		return toStringList(lv)
	}
	return nil
}

func fromLabels(labels []sdk.CostreportV1beta1AllocationGroupFilterLabelValue) map[string]string {
	result := make(map[string]string)
	for _, label := range labels {
		result[*label.Label] = *label.Value
	}
	return result
}

func toLabels(lv map[string]interface{}) []sdk.CostreportV1beta1AllocationGroupFilterLabelValue {
	if len(lv) > 0 {
		labelsStringMap := toStringMap(lv)

		operator := sdk.CostreportV1beta1AllocationGroupFilterLabelValueOperatorEqual

		if len(labelsStringMap) > 0 {
			labels := make([]sdk.CostreportV1beta1AllocationGroupFilterLabelValue, 0, len(labelsStringMap))
			for labelKey, labelValue := range labelsStringMap {
				label := sdk.CostreportV1beta1AllocationGroupFilterLabelValue{
					Label:    &labelKey,
					Value:    &labelValue,
					Operator: &operator,
				}
				labels = append(labels, label)
			}
			return labels
		}
	}
	return nil
}
