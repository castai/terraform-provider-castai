package castai

import (
	"context"
	"net/http"

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
		Importer:      &schema.ResourceImporter{},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Scaling policy name",
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

func resourceAllocationGroupRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	resp, err := client.AllocationGroupAPIGetAllocationGroupWithResponse(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if !d.IsNewResource() && resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "Scaling policy not found, removing from state", map[string]any{"id": d.Id()})
		d.SetId("")
		return nil
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	ag := resp.JSON200

	// TODO (romank): set fields in d from the ag
	//d.Set("name", ) etc.
	return nil
}

func resourceAllocationGroupCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterIds := toClusterIds(toSection(d, "clusterIds"))

	namespaces := toStringList(d.Get("namespaces").([]interface{}))

	labels := toLabels(toSection(d, "labels"))

	labelsOperator := toLabelsOperator(toSection(d, "labels_operator"))

	// TODO (romank): check what happens if we have both labels and namespaces selectors, can we have both?
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
	// TODO (romank): finish this status codes check, look into resource_workload_scaling_policy.go on how.
	switch create.StatusCode() {
	case http.StatusOK:
		d.SetId(*create.JSON200.Id)
		return resourceAllocationGroupRead(ctx, d, meta)
	// TODO (romank): do we have a status conflict in our response from API?
	case http.StatusConflict:
		return resourceAllocationGroupRead(ctx, d, meta)
	default:
		return diag.Errorf("expected status code %d, received: status=%d body=%s", http.StatusOK, create.StatusCode(), string(create.GetBody()))
	}
}

func toLabelsOperator(section map[string]interface{}) *sdk.CostreportV1beta1FilterOperator {
	if v, ok := section["labels_operator"]; ok {
		if lv := v.(string); lv != "" {
			labelOperator := sdk.CostreportV1beta1FilterOperator(lv)
			return &labelOperator
		}
	}
	return nil
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

			if len(labelsStringMap) > 0 {
				labels := make([]sdk.CostreportV1beta1AllocationGroupFilterLabelValue, 0, len(labelsStringMap))
				for labelKey, labelValue := range labelsStringMap {
					label := sdk.CostreportV1beta1AllocationGroupFilterLabelValue{
						Label: &labelKey,
						Value: &labelValue,
						// TODO (romank): check if the operator can be nil from the backend side and can we just enable
						// the equals one, check services/cost-report/internal/server/allocationgroup/allocationgroup.go
						// for that.
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
		tflog.Info(ctx, "scaling policy up to date")
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
