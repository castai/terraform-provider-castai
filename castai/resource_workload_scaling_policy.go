package castai

import (
	"context"
	"fmt"
	"github.com/samber/lo"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
		CustomizeDiff: resourceWorkloadScalingPolicyDiff,
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
				Elem:     workloadScalingPolicyResourceSchema("QUANTILE", 0),
			},
			"memory": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem:     workloadScalingPolicyResourceSchema("MAX", 0.1),
			},
			"startup": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"period_seconds": {
							Type:             schema.TypeInt,
							Optional:         true,
							Description:      "Defines the duration (in seconds) during which elevated resource usage is expected at startup.\nWhen set, recommendations will be adjusted to disregard resource spikes within this period.\nIf not specified, the workload will receive standard recommendations without startup considerations.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(120, 3600)),
						},
					},
				},
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

func workloadScalingPolicyResourceSchema(function string, overhead float64) *schema.Resource {
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
				MaxItems: 1,
				Description: "The arguments for the function - i.e. for `QUANTILE` this should be a [0, 1] float. " +
					"`MAX` doesn't accept any args",
				Elem: &schema.Schema{
					Type: schema.TypeString,
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
			"look_back_period_seconds": {
				Type:             schema.TypeInt,
				Optional:         true,
				Description:      "The look back period in seconds for the recommendation.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(24*60*60, 7*24*60*60)),
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
		req.RecommendationPolicies.Cpu = toWorkloadScalingPolicies(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("memory"); ok {
		req.RecommendationPolicies.Memory = toWorkloadScalingPolicies(v.([]interface{})[0].(map[string]interface{}))
	}

	req.RecommendationPolicies.Startup = toStartup(toSection(d, "startup"))

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
		tflog.Warn(ctx, "Scaling policy not found, removing from state", map[string]interface{}{"id": d.Id()})
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
	if err := d.Set("cpu", toWorkloadScalingPoliciesMap(sp.RecommendationPolicies.Cpu)); err != nil {
		return diag.FromErr(fmt.Errorf("setting cpu: %w", err))
	}
	if err := d.Set("memory", toWorkloadScalingPoliciesMap(sp.RecommendationPolicies.Memory)); err != nil {
		return diag.FromErr(fmt.Errorf("setting memory: %w", err))
	}

	if err := d.Set("startup", toStartupMap(sp.RecommendationPolicies.Startup)); err != nil {
		return diag.FromErr(fmt.Errorf("setting startup: %w", err))
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
		"startup",
	) {
		tflog.Info(ctx, "scaling policy up to date")
		return nil
	}

	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)
	req := sdk.WorkloadOptimizationAPIUpdateWorkloadScalingPolicyJSONBody{
		Name:      d.Get("name").(string),
		ApplyType: sdk.WorkloadoptimizationV1ApplyType(d.Get("apply_type").(string)),
		RecommendationPolicies: sdk.WorkloadoptimizationV1RecommendationPolicies{
			ManagementOption: sdk.WorkloadoptimizationV1ManagementOption(d.Get("management_option").(string)),
			Cpu:              toWorkloadScalingPolicies(d.Get("cpu").([]interface{})[0].(map[string]interface{})),
			Memory:           toWorkloadScalingPolicies(d.Get("memory").([]interface{})[0].(map[string]interface{})),
			Startup:          toStartup(toSection(d, "startup")),
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
		tflog.Debug(ctx, "Scaling policy not found, skipping delete", map[string]interface{}{"id": d.Id()})
		return nil
	}
	if err := sdk.StatusOk(resp); err != nil {
		return diag.FromErr(err)
	}

	if resp.JSON200.IsReadonly || resp.JSON200.IsDefault {
		tflog.Warn(ctx, "Default/readonly scaling policy can't be deleted, removing from state", map[string]interface{}{
			"id": d.Id(),
		})
		return nil
	}

	delResp, err := client.WorkloadOptimizationAPIDeleteWorkloadScalingPolicyWithResponse(ctx, clusterID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if err := sdk.StatusOk(delResp); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceWorkloadScalingPolicyDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	// Since tf doesn't support cross field validation, doing it here.
	cpu := toWorkloadScalingPolicies(d.Get("cpu").([]interface{})[0].(map[string]interface{}))
	memory := toWorkloadScalingPolicies(d.Get("memory").([]interface{})[0].(map[string]interface{}))

	if err := validateArgs(cpu, "cpu"); err != nil {
		return err
	}
	return validateArgs(memory, "memory")
}

func validateArgs(r sdk.WorkloadoptimizationV1ResourcePolicies, res string) error {
	if r.Function == "QUANTILE" && len(r.Args) == 0 {
		return fmt.Errorf("field %q: QUANTILE function requires args to be provided", res)
	}
	if r.Function == "MAX" && len(r.Args) > 0 {
		return fmt.Errorf("field %q: MAX function doesn't accept any args", res)
	}
	return nil
}

func workloadScalingPolicyImporter(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	ids := strings.Split(d.Id(), "/")
	if len(ids) != 2 || ids[0] == "" || ids[1] == "" {
		return nil, fmt.Errorf("expected import id with format: <cluster_id>/<scaling_policy name or id>, got: %q", d.Id())
	}

	clusterID, nameOrID := ids[0], ids[1]
	if err := d.Set(FieldClusterID, clusterID); err != nil {
		return nil, fmt.Errorf("setting cluster nameOrID: %w", err)
	}
	d.SetId(nameOrID)

	// Return if scaling policy ID provided.
	if _, err := uuid.Parse(nameOrID); err == nil {
		return []*schema.ResourceData{d}, nil
	}

	// Find scaling policy ID by name.
	client := meta.(*ProviderConfig).api
	resp, err := client.WorkloadOptimizationAPIListWorkloadScalingPoliciesWithResponse(ctx, clusterID)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return nil, err
	}

	for _, sp := range resp.JSON200.Items {
		if sp.Name == nameOrID {
			d.SetId(sp.Id)
			return []*schema.ResourceData{d}, nil
		}
	}

	return nil, fmt.Errorf("failed to find workload scaling policy with the following name: %v", nameOrID)
}

func toWorkloadScalingPolicies(obj map[string]interface{}) sdk.WorkloadoptimizationV1ResourcePolicies {
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
	if v, ok := obj["look_back_period_seconds"].(int); ok && v > 0 {
		out.LookBackPeriodSeconds = lo.ToPtr(int32(v))
	}

	return out
}

func toWorkloadScalingPoliciesMap(p sdk.WorkloadoptimizationV1ResourcePolicies) []map[string]interface{} {
	m := map[string]interface{}{
		"function":        p.Function,
		"args":            p.Args,
		"overhead":        p.Overhead,
		"apply_threshold": p.ApplyThreshold,
	}

	if p.LookBackPeriodSeconds != nil {
		m["look_back_period_seconds"] = int(*p.LookBackPeriodSeconds)
	}

	return []map[string]interface{}{m}
}

func toStartup(startup map[string]interface{}) *sdk.WorkloadoptimizationV1StartupSettings {
	if len(startup) == 0 {
		return nil
	}
	result := &sdk.WorkloadoptimizationV1StartupSettings{}

	if v, ok := startup["period_seconds"].(int); ok && v > 0 {
		result.PeriodSeconds = lo.ToPtr(int32(v))
	}

	return result
}

func toStartupMap(s *sdk.WorkloadoptimizationV1StartupSettings) []map[string]interface{} {
	if s == nil {
		return nil
	}

	m := map[string]interface{}{}

	if s.PeriodSeconds != nil {
		m["period_seconds"] = int(*s.PeriodSeconds)
	}

	if len(m) == 0 {
		return nil
	}

	return []map[string]interface{}{m}
}
