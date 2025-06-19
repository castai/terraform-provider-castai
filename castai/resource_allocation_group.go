package castai

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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
		Importer: &schema.ResourceImporter{
			StateContext: allocationGroupImporter,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Allocation group name",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(k8sNameRegex, "name must adhere to the format guidelines of Kubernetes labels/annotations")),
			},
			"cluster_ids": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of cluster ids",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"namespaces": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of namespaces",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"labels": {
				Type:        schema.TypeMap,
				Description: "Labels",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"labels_operator": {
				Type:        schema.TypeString,
				Description: "Labels Operator",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
					string(sdk.AND), string(sdk.OR),
				}, false)),
			},
		},
	}
}

func allocationGroupImporter(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	client := meta.(*ProviderConfig).api
	if _, err := uuid.ParseUUID(d.Id()); err != nil {
		return nil, fmt.Errorf("error parsing uuid: %w", err)
	}
	resp, err := client.AllocationGroupAPIGetAllocationGroupWithResponse(ctx, d.Id())
	if err != nil {
		return nil, fmt.Errorf("error getting allocation group: %w", err)
	}

	err = sdk.CheckOKResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("error checking response: %w", err)
	}

	d.SetId(resp.JSON200.Id)
	return []*schema.ResourceData{d}, nil
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

	if err := d.Set("name", ag.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %w", err))
	}
	if err := d.Set("cluster_ids", ag.Filter.ClusterIds); err != nil {
		return diag.FromErr(fmt.Errorf("setting cluster_ids: %w", err))
	}
	if err := d.Set("namespaces", ag.Filter.Namespaces); err != nil {
		return diag.FromErr(fmt.Errorf("setting namespaces: %w", err))
	}
	if err := d.Set("labels", ag.Filter.Labels); err != nil {
		return diag.FromErr(fmt.Errorf("setting labels: %w", err))
	}
	if err := d.Set("labels_operator", ag.Filter.LabelsOperator); err != nil {
		return diag.FromErr(fmt.Errorf("setting labels_operator: %w", err))
	}
	return nil
}

func resourceAllocationGroupCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterIds := toClusterIds(toSection(d, "clusterIds"))

	listParams := &sdk.AllocationGroupAPIListAllocationGroupsParams{
		ClusterIds: &clusterIds,
	}

	response, err := client.AllocationGroupAPIListAllocationGroupsWithResponse(ctx, listParams)

	if err != nil {
		return diag.FromErr(err)
	}
	if err := sdk.CheckOKResponse(response, err); err != nil {
		return diag.FromErr(err)
	}

	allocationGroups := *response.JSON200.Items

	allocationGroupName := d.Get("name").(*string)
	if len(allocationGroups) > 0 {
		for _, ag := range allocationGroups {
			if *ag.Name == *allocationGroupName {
				return diag.Errorf("allocation group already exists")
			}
		}
	}
	namespaces := toStringList(d.Get("namespaces").([]interface{}))

	labels := toLabels(toSection(d, "labels"))

	labelsOperator := toLabelsOperator(toSection(d, "labels_operator"))

	body := sdk.AllocationGroupAPICreateAllocationGroupJSONRequestBody{
		Filter: &sdk.CostreportV1beta1AllocationGroupFilter{
			ClusterIds:     &clusterIds,
			Labels:         &labels,
			LabelsOperator: labelsOperator,
			Namespaces:     &namespaces,
		},
		Name: d.Get("name").(*string),
	}
	create, err := client.AllocationGroupAPICreateAllocationGroupWithResponse(ctx, body)
	if err != nil {
		return nil
	}
	switch create.StatusCode() {
	case http.StatusOK:
		d.SetId(*create.JSON200.Id)
		return resourceAllocationGroupRead(ctx, d, meta)
	default:
		return diag.Errorf("expected status code %d, received: status=%d body=%s", http.StatusOK, create.StatusCode(), string(create.GetBody()))
	}
}

func toLabelsOperator(section map[string]interface{}) *sdk.CostreportV1beta1FilterOperator {
	defaultLabelOperator := sdk.OR
	if v, ok := section["labels_operator"]; ok {
		if lv := v.(string); lv != "" {
			labelOperator := sdk.CostreportV1beta1FilterOperator(lv)
			return &labelOperator
		}
	}
	return &defaultLabelOperator
}

func toClusterIds(i map[string]interface{}) []string {
	if v, ok := i["clusterIds"]; ok {
		if lv := v.([]interface{}); len(lv) > 0 {
			return toStringList(lv)
		}
	}
	return nil
}

func toLabels(i map[string]interface{}) []sdk.CostreportV1beta1AllocationGroupFilterLabelValue {
	if v, ok := i["labels"]; ok {
		if lv := v.(map[string]interface{}); len(lv) > 0 {
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
					return labels
				}
			}
		}
	}
	return nil
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

	clusterIds := toClusterIds(toSection(d, "clusterIds"))
	labels := toLabels(toSection(d, "labels"))
	namespaces := toStringList(d.Get("namespaces").([]interface{}))

	req := sdk.AllocationGroupAPIUpdateAllocationGroupJSONRequestBody{
		Filter: &sdk.CostreportV1beta1AllocationGroupFilter{
			ClusterIds:     &clusterIds,
			Labels:         &labels,
			LabelsOperator: toLabelsOperator(toSection(d, "labels_operator")),
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

	resp, err := client.AllocationGroupAPIGetAllocationGroupWithResponse(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		tflog.Debug(ctx, "Allocation group not found, skipping delete", map[string]any{"id": d.Id()})
		return nil
	}
	if err := sdk.StatusOk(resp); err != nil {
		return diag.FromErr(err)
	}

	response, err := client.AllocationGroupAPIDeleteAllocationGroupWithResponse(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := sdk.StatusOk(response); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
