package castai

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

var (
	k8sNameRegex = regexp.MustCompile("^[a-z0-9A-Z][a-z0-9A-Z._-]{0,61}[a-z0-9A-Z]$")
)

func resourceWorkloadScalingPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceWorkloadScalingPolicyCreate,
		ReadContext:   resourceWorkloadScalingPolicyRead,
		UpdateContext: resourceWorkloadScalingPolicyUpdate,
		DeleteContext: resourceWorkloadScalingPolicyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: workloadScalingPolicyImporter,
		},
		Description: "Manage workload scaling policy. Scaling policy [reference](https://docs.cast.ai/docs/woop-scaling-policies)",
		Schema: map[string]*schema.Schema{
			FieldClusterID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "CAST AI cluster id",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Scaling policy name",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(k8sNameRegex, "name must adhere to the format guidelines of Kubernetes labels/annotations")),
			},
			"apply_type": {
				Type:     schema.TypeString,
				Required: true,
				Description: `Recommendation apply type. 
	- IMMEDIATE - pods are restarted immediately when new recommendation is generated.
	- DEFERRED - pods are not restarted and recommendation values are applied during natural restarts only (new deployment, etc.)`,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"IMMEDIATE", "DEFERRED"}, false)),
			},
			"management_option": {
				Type:     schema.TypeString,
				Required: true,
				Description: `Defines possible options for workload management.
	- READ_ONLY - workload watched (metrics collected), but no actions performed by CAST AI.
	- MANAGED - workload watched (metrics collected), CAST AI may perform actions on the workload.`,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"READ_ONLY", "MANAGED"}, false)),
			},
			"cpu": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem:     resourceSchema("QUANTILE", 0, []string{"0.8"}),
			},
			"memory": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem:     resourceSchema("MAX", 0.1, []string{}),
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(15 * time.Second),
			Read:   schema.DefaultTimeout(15 * time.Second),
			Update: schema.DefaultTimeout(15 * time.Second),
			Delete: schema.DefaultTimeout(15 * time.Second),
		},
	}
}

func resourceSchema(function string, overhead float64, args []string) *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"function": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The function used to calculate the resource recommendation. Supported values: `QUANTILE`, `MAX`",
				Default:          function,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"QUANTILE", "MAX"}, false)),
			},
			"args": {
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 1,
				MaxItems: 1,
				Description: "The arguments for the function - i.e. for `QUANTILE` this should be a [0, 1] float. " +
					"`MAX` doesn't accept any args",
				Elem: &schema.Schema{
					Type:    schema.TypeString,
					Default: args,
				},
			},
			"overhead": {
				Type:             schema.TypeFloat,
				Optional:         true,
				Description:      "Overhead for the recommendation, e.g. `0.1` will result in 10% higher recommendation",
				Default:          overhead,
				ValidateDiagFunc: validation.ToDiagFunc(validation.FloatBetween(0, 1)),
			},
			"apply_threshold": {
				Type:     schema.TypeFloat,
				Optional: true,
				Description: "The threshold of when to apply the recommendation. Recommendation will be applied when " +
					"diff of current requests and new recommendation is greater than set value",
				Default:          0.1,
				ValidateDiagFunc: validation.ToDiagFunc(validation.FloatBetween(0.01, 1)),
			},
		},
	}
}

func resourceWorkloadScalingPolicyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := d.Get(FieldClusterID).(string)
	req := sdk.WorkloadOptimizationAPICreateWorkloadScalingPolicyJSONRequestBody{
		Name:      d.Get("name").(string),
		ApplyType: sdk.WorkloadoptimizationV1ApplyType(d.Get("apply_type").(string)),
		RecommendationPolicies: sdk.WorkloadoptimizationV1RecommendationPolicies{
			ManagementOption: sdk.WorkloadoptimizationV1ManagementOption(d.Get("management_option").(string)),
		},
	}

	if v, ok := d.GetOk("cpu"); ok {
		req.RecommendationPolicies.Cpu = toResourcePolicies(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("memory"); ok {
		req.RecommendationPolicies.Memory = toResourcePolicies(v.([]interface{})[0].(map[string]interface{}))
	}

	resp, err := client.WorkloadOptimizationAPICreateWorkloadScalingPolicyWithResponse(ctx, clusterID, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	d.SetId(resp.JSON200.Id)

	return resourceWorkloadScalingPolicyRead(ctx, d, meta)
}

func resourceWorkloadScalingPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := d.Get(FieldClusterID).(string)
	resp, err := client.WorkloadOptimizationAPIGetWorkloadScalingPolicyWithResponse(ctx, clusterID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if !d.IsNewResource() && resp.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Scaling policy (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	sp := resp.JSON200

	if err := d.Set("name", sp.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %w", err))
	}
	if err := d.Set("apply_type", sp.ApplyType); err != nil {
		return diag.FromErr(fmt.Errorf("setting apply type: %w", err))
	}
	if err := d.Set("management_option", sp.RecommendationPolicies.ManagementOption); err != nil {
		return diag.FromErr(fmt.Errorf("setting management option: %w", err))
	}
	if err := d.Set("cpu", toResourceMap(sp.RecommendationPolicies.Cpu)); err != nil {
		return diag.FromErr(fmt.Errorf("setting cpu: %w", err))
	}
	if err := d.Set("memory", toResourceMap(sp.RecommendationPolicies.Memory)); err != nil {
		return diag.FromErr(fmt.Errorf("setting memory: %w", err))
	}

	return nil
}

func resourceWorkloadScalingPolicyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !d.HasChanges(
		"name",
		"apply_type",
		"management_option",
		"cpu",
		"memory",
	) {
		log.Printf("[INFO] scaling policy up to date")
		return nil
	}

	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)
	req := sdk.WorkloadOptimizationAPIUpdateWorkloadScalingPolicyJSONBody{
		Name:      d.Get("name").(string),
		ApplyType: sdk.WorkloadoptimizationV1ApplyType(d.Get("apply_type").(string)),
		RecommendationPolicies: sdk.WorkloadoptimizationV1RecommendationPolicies{
			ManagementOption: sdk.WorkloadoptimizationV1ManagementOption(d.Get("management_option").(string)),
			Cpu:              toResourcePolicies(d.Get("cpu").([]interface{})[0].(map[string]interface{})),
			Memory:           toResourcePolicies(d.Get("memory").([]interface{})[0].(map[string]interface{})),
		},
	}

	resp, err := client.WorkloadOptimizationAPIUpdateWorkloadScalingPolicyWithResponse(ctx, clusterID, d.Id(), req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	return resourceWorkloadScalingPolicyRead(ctx, d, meta)
}

func resourceWorkloadScalingPolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)

	resp, err := client.WorkloadOptimizationAPIGetWorkloadScalingPolicyWithResponse(ctx, clusterID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[DEBUG] Scaling policy (%s) not found, skipping delete", d.Id())
		return nil
	}
	if err := sdk.StatusOk(resp); err != nil {
		return diag.FromErr(err)
	}

	delResp, err := client.WorkloadOptimizationAPIDeleteWorkloadScalingPolicyWithResponse(ctx, clusterID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if delResp.StatusCode() == http.StatusBadRequest {
		log.Printf("[WARN] Scaling policy has active workloads (%s) and can't be deleted, removing from state", d.Id())
		return nil
	}
	if err := sdk.StatusOk(delResp); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func workloadScalingPolicyImporter(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	ids := strings.Split(d.Id(), "/")
	if len(ids) != 2 || ids[0] == "" || ids[1] == "" {
		return nil, fmt.Errorf("expected import id with format: <cluster_id>/<scaling_policy name or id>, got: %q", d.Id())
	}

	clusterID, id := ids[0], ids[1]
	if err := d.Set(FieldClusterID, clusterID); err != nil {
		return nil, fmt.Errorf("setting cluster id: %w", err)
	}
	d.SetId(id)

	// Return if scaling policy ID provided.
	if _, err := uuid.Parse(id); err == nil {
		return []*schema.ResourceData{d}, nil
	}

	// Find scaling policy ID by name.
	client := meta.(*ProviderConfig).api
	resp, err := client.WorkloadOptimizationAPIListWorkloadScalingPoliciesWithResponse(ctx, clusterID)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return nil, err
	}

	for _, sp := range resp.JSON200.Items {
		if sp.Name == id {
			d.SetId(sp.Id)
			return []*schema.ResourceData{d}, nil
		}
	}

	return nil, fmt.Errorf("failed to find workload scaling policy with the following name: %v", id)
}

func toResourcePolicies(obj map[string]interface{}) sdk.WorkloadoptimizationV1ResourcePolicies {
	out := sdk.WorkloadoptimizationV1ResourcePolicies{}

	if v, ok := obj["function"].(string); ok {
		out.Function = sdk.WorkloadoptimizationV1ResourcePoliciesFunction(v)
	}
	if v, ok := obj["args"].([]interface{}); ok && len(v) > 0 {
		out.Args = toStringList(v)
	}
	if v, ok := obj["overhead"].(float64); ok {
		out.Overhead = v
	}
	if v, ok := obj["apply_threshold"].(float64); ok {
		out.ApplyThreshold = v
	}

	return out
}

func toResourceMap(p sdk.WorkloadoptimizationV1ResourcePolicies) []map[string]interface{} {
	m := map[string]interface{}{
		"function":        p.Function,
		"args":            p.Args,
		"overhead":        p.Overhead,
		"apply_threshold": p.ApplyThreshold,
	}

	return []map[string]interface{}{m}
}
