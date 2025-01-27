package castai

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	minResourceMultiplierValue      = 1.0
	minApplyThresholdValue          = 0.01
	maxApplyThresholdValue          = 2.5
	defaultApplyThresholdPercentage = 0.1
)

const (
	FieldLimitStrategy           = "limit"
	FieldLimitStrategyType       = "type"
	FieldLimitStrategyMultiplier = "multiplier"

	DeprecatedFieldApplyThreshold             = "apply_threshold"
	FieldApplyThresholdStrategy               = "apply_threshold_strategy"
	FieldApplyThresholdStrategyType           = "type"
	FieldApplyThresholdStrategyPercentage     = "percentage"
	FieldApplyThresholdStrategyPercentageType = "PERCENTAGE"
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
				Elem:     workloadScalingPolicyResourceSchema("cpu", "QUANTILE", 0, 0.01),
			},
			"memory": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem:     workloadScalingPolicyResourceSchema("memory", "MAX", 0.1, 10),
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
			"downscaling": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"apply_type": {
							Type:     schema.TypeString,
							Optional: true,
							Description: `Defines the apply type to be used when downscaling.
	- IMMEDIATE - pods are restarted immediately when new recommendation is generated.
	- DEFERRED - pods are not restarted and recommendation values are applied during natural restarts only (new deployment, etc.)`,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"IMMEDIATE", "DEFERRED"}, false)),
						},
					},
				},
			},
			"memory_event": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"apply_type": {
							Type:     schema.TypeString,
							Optional: true,
							Description: `Defines the apply type to be used when applying recommendation for memory related event.
	- IMMEDIATE - pods are restarted immediately when new recommendation is generated.
	- DEFERRED - pods are not restarted and recommendation values are applied during natural restarts only (new deployment, etc.)`,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"IMMEDIATE", "DEFERRED"}, false)),
						},
					},
				},
			},
			"anti_affinity": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"consider_anti_affinity": {
							Type:     schema.TypeBool,
							Optional: true,
							Description: `Defines if anti-affinity should be considered when scaling the workload.
	If enabled, requiring host ports, or having anti-affinity on hostname will force all recommendations to be deferred.`,
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

func workloadScalingPolicyResourceSchema(resource, function string, overhead, minRecommended float64) *schema.Resource {
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
			DeprecatedFieldApplyThreshold: {
				Type:     schema.TypeFloat,
				Optional: true,
				Description: "The threshold of when to apply the recommendation. Recommendation will be applied when " +
					"diff of current requests and new recommendation is greater than set value",
				ValidateDiagFunc: validation.ToDiagFunc(validation.FloatBetween(minApplyThresholdValue, maxApplyThresholdValue)),
				Deprecated:       "Use apply_threshold_strategy instead",
			},
			FieldApplyThresholdStrategy: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Description: "Resource apply threshold strategy settings. " +
					"The default strategy is `PERCENTAGE` with percentage value set to 0.1.",
				Elem: workloadScalingPolicyResourceApplyThresholdStrategySchema(),
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return suppressThresholdStrategyDefaultValueDiff(resource, old, new, d)
				},
				ConflictsWith: []string{fmt.Sprintf("%s.0.%s", resource, DeprecatedFieldApplyThreshold)},
			},
			"look_back_period_seconds": {
				Type:             schema.TypeInt,
				Optional:         true,
				Description:      "The look back period in seconds for the recommendation.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(24*60*60, 7*24*60*60)),
			},
			"min": {
				Type:             schema.TypeFloat,
				Default:          minRecommended,
				Optional:         true,
				Description:      "Min values for the recommendation, applies to every container. For memory - this is in MiB, for CPU - this is in cores.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.FloatAtLeast(minRecommended)),
			},
			"max": {
				Type:        schema.TypeFloat,
				Optional:    true,
				Description: "Max values for the recommendation, applies to every container. For memory - this is in MiB, for CPU - this is in cores.",
			},
			FieldLimitStrategy: {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Resource limit settings",
				Elem:        workloadScalingPolicyResourceLimitSchema(),
			},
			"management_option": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "Disables management for a single resource when set to `READ_ONLY`. " +
					"The resource will use its original workload template requests and limits. " +
					"Supported value: `READ_ONLY`. Minimum required workload-autoscaler version: `v0.23.1`.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"READ_ONLY"}, false)),
			},
		},
	}
}

func workloadScalingPolicyResourceLimitSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			FieldLimitStrategyType: {
				Type:     schema.TypeString,
				Required: true,
				Description: fmt.Sprintf(`Defines limit strategy type.
	- %s - removes the resource limit even if it was specified in the workload spec.
	- %s - used to calculate the resource limit. The final value is determined by multiplying the resource request by the specified factor.`, sdk.NOLIMIT, sdk.MULTIPLIER),
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{string(sdk.MULTIPLIER), string(sdk.NOLIMIT)}, false)),
			},
			FieldLimitStrategyMultiplier: {
				Type:             schema.TypeFloat,
				Optional:         true,
				Description:      "Multiplier used to calculate the resource limit. It must be defined for the MULTIPLIER strategy.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.FloatAtLeast(minResourceMultiplierValue)),
			},
		},
	}
}

func workloadScalingPolicyResourceApplyThresholdStrategySchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			FieldApplyThresholdStrategyType: {
				Type:     schema.TypeString,
				Required: true,
				Description: fmt.Sprintf(`Defines apply theshold strategy type.
	- %s - recommendation will be applied when diff of current requests and new recommendation is greater than set value`, FieldApplyThresholdStrategyPercentageType),
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{FieldApplyThresholdStrategyPercentageType}, false)),
			},
			FieldApplyThresholdStrategyPercentage: {
				Type:     schema.TypeFloat,
				Optional: true,
				Description: fmt.Sprintf("Percentage of a how much difference should there be between the current pod requests and the new recommendation. "+
					"It must be defined for the %s strategy.", FieldApplyThresholdStrategyPercentageType),
				ValidateDiagFunc: validation.ToDiagFunc(validation.FloatBetween(minApplyThresholdValue, maxApplyThresholdValue)),
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

	req.RecommendationPolicies.Downscaling = toDownscaling(toSection(d, "downscaling"))

	req.RecommendationPolicies.MemoryEvent = toMemoryEvent(toSection(d, "memory_event"))

	req.RecommendationPolicies.AntiAffinity = toAntiAffinity(toSection(d, "anti_affinity"))

	create, err := client.WorkloadOptimizationAPICreateWorkloadScalingPolicyWithResponse(ctx, clusterID, req)
	if err != nil {
		return diag.FromErr(err)
	}

	switch create.StatusCode() {
	case http.StatusOK:
		d.SetId(create.JSON200.Id)
		return resourceWorkloadScalingPolicyRead(ctx, d, meta)
	case http.StatusConflict:
		policy, err := getWorkloadScalingPolicyByName(ctx, client, clusterID, req.Name)
		if err != nil {
			return diag.FromErr(err)
		}
		if policy.IsDefault {
			d.SetId(policy.Id)
			return resourceWorkloadScalingPolicyUpdate(ctx, d, meta)
		}
		return diag.Errorf("scaling policy with name %q already exists", req.Name)
	default:
		return diag.Errorf("expected status code %d, received: status=%d body=%s", http.StatusOK, create.StatusCode(), string(create.GetBody()))
	}
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
	if err := d.Set("cpu", toWorkloadScalingPoliciesMap(getResourceFrom(d, "cpu"), sp.RecommendationPolicies.Cpu)); err != nil {
		return diag.FromErr(fmt.Errorf("setting cpu: %w", err))
	}
	if err := d.Set("memory", toWorkloadScalingPoliciesMap(getResourceFrom(d, "memory"), sp.RecommendationPolicies.Memory)); err != nil {
		return diag.FromErr(fmt.Errorf("setting memory: %w", err))
	}
	if err := d.Set("startup", toStartupMap(sp.RecommendationPolicies.Startup)); err != nil {
		return diag.FromErr(fmt.Errorf("setting startup: %w", err))
	}
	if err := d.Set("downscaling", toDownscalingMap(sp.RecommendationPolicies.Downscaling)); err != nil {
		return diag.FromErr(fmt.Errorf("setting downscaling: %w", err))
	}
	if err := d.Set("memory_event", toMemoryEventMap(sp.RecommendationPolicies.MemoryEvent)); err != nil {
		return diag.FromErr(fmt.Errorf("setting memory event: %w", err))
	}
	if err := d.Set("anti_affinity", toAntiAffinityMap(sp.RecommendationPolicies.AntiAffinity)); err != nil {
		return diag.FromErr(fmt.Errorf("setting anti-affinity: %w", err))
	}

	return nil
}

func getResourceFrom(d *schema.ResourceData, resource string) map[string]any {
	if v, ok := d.GetOk(resource); ok {
		return v.([]any)[0].(map[string]any)
	}
	return map[string]any{}
}

func resourceWorkloadScalingPolicyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !d.HasChanges(
		"name",
		"apply_type",
		"management_option",
		"cpu",
		"memory",
		"startup",
		"downscaling",
		"memory_event",
		"anti_affinity",
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
			Downscaling:      toDownscaling(toSection(d, "downscaling")),
			MemoryEvent:      toMemoryEvent(toSection(d, "memory_event")),
			AntiAffinity:     toAntiAffinity(toSection(d, "anti_affinity")),
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

	if err := validateResourcePolicy(cpu, "cpu"); err != nil {
		return err
	}
	return validateResourcePolicy(memory, "memory")
}

func validateResourcePolicy(r sdk.WorkloadoptimizationV1ResourcePolicies, res string) error {
	if r.Function == "QUANTILE" && len(r.Args) == 0 {
		return fmt.Errorf("field %q: QUANTILE function requires args to be provided", res)
	}
	if r.Function == "MAX" && len(r.Args) > 0 {
		return fmt.Errorf("field %q: MAX function doesn't accept any args", res)
	}

	err := validateResourceLimit(r)
	if err != nil {
		return fmt.Errorf("field %q: %w", res, err)
	}

	err = validateResourceApplyThresholdStrategy(r.ApplyThresholdStrategy)
	if err != nil {
		return fmt.Errorf("field %q: %w", res, err)
	}
	return nil
}

func validateResourceApplyThresholdStrategy(r *sdk.WorkloadoptimizationV1ApplyThresholdStrategy) error {
	if r == nil {
		return nil
	}

	if r.PercentageThreshold != nil {
		if err := validatePercentageThresholdStrategy(r.PercentageThreshold); err != nil {
			return fmt.Errorf(`field %q: %w`, FieldApplyThresholdStrategy, err)
		}
		return nil
	}

	return fmt.Errorf(`field %q: field %q: unknown apply threshold strategy type`, FieldApplyThresholdStrategy, FieldApplyThresholdStrategyType)
}

func validatePercentageThresholdStrategy(r *sdk.WorkloadoptimizationV1ApplyThresholdStrategyPercentageThreshold) error {
	if r.Percentage == 0 {
		return fmt.Errorf(
			`field %q: value must be set for strategy type %s`,
			FieldApplyThresholdStrategyPercentage,
			FieldApplyThresholdStrategyPercentageType,
		)
	}
	return nil
}

func validateResourceLimit(r sdk.WorkloadoptimizationV1ResourcePolicies) error {
	if r.Limit == nil {
		return nil
	}

	switch r.Limit.Type {
	case sdk.NOLIMIT:
		if r.Limit.Multiplier != nil {
			return fmt.Errorf(`field %q: %q limit type doesn't accept multiplier value`, FieldLimitStrategy, sdk.NOLIMIT)
		}
	case sdk.MULTIPLIER:
		if r.Limit.Multiplier == nil {
			return fmt.Errorf(`field %q: %q limit type requires multiplier value to be provided`, FieldLimitStrategy, sdk.MULTIPLIER)
		}
	default:
		return fmt.Errorf(`field %q: unknown limit type %q`, FieldLimitStrategy, r.Limit.Type)
	}
	return nil
}

func workloadScalingPolicyImporter(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	clusterID, nameOrID, found := strings.Cut(d.Id(), "/")
	if !found {
		return nil, fmt.Errorf("expected import id with format: <cluster_id>/<scaling_policy name or id>, got: %q", d.Id())
	}

	if err := d.Set(FieldClusterID, clusterID); err != nil {
		return nil, fmt.Errorf("setting cluster ID: %w", err)
	}

	// Return if scaling policy ID provided.
	if _, err := uuid.Parse(nameOrID); err == nil {
		d.SetId(nameOrID)
		return []*schema.ResourceData{d}, nil
	}

	// Find scaling policy ID by name.
	client := meta.(*ProviderConfig).api
	policy, err := getWorkloadScalingPolicyByName(ctx, client, clusterID, nameOrID)
	if err != nil {
		return nil, err
	}

	d.SetId(policy.Id)
	return []*schema.ResourceData{d}, nil
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
	if v, ok := obj["look_back_period_seconds"].(int); ok && v > 0 {
		out.LookBackPeriodSeconds = lo.ToPtr(int32(v))
	}
	if v, ok := obj["min"].(float64); ok {
		out.Min = lo.ToPtr(v)
	}
	if v, ok := obj["max"].(float64); ok && v > 0 {
		out.Max = lo.ToPtr(v)
	}
	if v, ok := obj[FieldLimitStrategy].([]any); ok && len(v) > 0 {
		out.Limit = toWorkloadResourceLimit(v[0].(map[string]any))
	}
	if v, ok := obj["management_option"].(string); ok && v != "" {
		out.ManagementOption = lo.ToPtr(sdk.WorkloadoptimizationV1ManagementOption(v))
	}
	out.ApplyThresholdStrategy = resolveApplyThresholdStrategy(obj)

	return out
}

func resolveApplyThresholdStrategy(obj map[string]interface{}) *sdk.WorkloadoptimizationV1ApplyThresholdStrategy {
	if v, ok := obj[DeprecatedFieldApplyThreshold].(float64); ok && v > 0 {
		return &sdk.WorkloadoptimizationV1ApplyThresholdStrategy{
			PercentageThreshold: &sdk.WorkloadoptimizationV1ApplyThresholdStrategyPercentageThreshold{
				Percentage: v,
			},
		}
	}
	if v, ok := obj[FieldApplyThresholdStrategy].([]any); ok && len(v) > 0 {
		return toWorkloadResourceApplyThresholdStrategy(v[0].(map[string]any))
	}

	return toWorkloadResourcePercentageThresholdStrategy(defaultApplyThresholdPercentage)
}

func toWorkloadResourceApplyThresholdStrategy(obj map[string]any) *sdk.WorkloadoptimizationV1ApplyThresholdStrategy {
	if len(obj) == 0 {
		return nil
	}

	out := &sdk.WorkloadoptimizationV1ApplyThresholdStrategy{}
	strategy, _ := obj[FieldApplyThresholdStrategyType].(string)
	switch strategy {
	case FieldApplyThresholdStrategyPercentageType:
		if percentage, ok := obj[FieldApplyThresholdStrategyPercentage].(float64); ok {
			out = toWorkloadResourcePercentageThresholdStrategy(percentage)
		}
	}
	return out
}

// To prevent terraform detecting changes, when there none, we want to set only the field that was set in previous
// configuration. If previous configuration is empty(during import), we use FieldApplyThresholdStrategy.
func mapApplyStrategyBasedOnPreviousConfig(
	p sdk.WorkloadoptimizationV1ResourcePolicies, previousCfg map[string]any) map[string]any {
	m := map[string]any{}
	if v, ok := previousCfg[DeprecatedFieldApplyThreshold].(float64); ok && v > 0 {
		m[DeprecatedFieldApplyThreshold] = p.ApplyThreshold
		return m
	}

	strategy := applyThresholdStrategyToMap(p.ApplyThresholdStrategy)
	if strategy != nil {
		m[FieldApplyThresholdStrategy] = strategy
	}
	return m
}

// When both DeprecatedFieldApplyThreshold and FieldApplyThresholdStrategy are unset in client configuration
// FieldApplyThresholdStrategy will be used with default value. It is not possible to set default in Schema, only when
// DeprecatedFieldApplyThreshold is missing. If configuration saved from API correspond with default value,
// we will supress diff.
func suppressThresholdStrategyDefaultValueDiff(resource, oldValue, newValue string, d *schema.ResourceData) bool {
	resourcePath := fmt.Sprintf("%s.0", resource)
	isApplyThresholdStrategyUnset := newValue == "0" || newValue == ""
	isApplyThresholdUnset := d.Get(fmt.Sprintf("%s.%s", resourcePath, DeprecatedFieldApplyThreshold)) == 0.
	if isApplyThresholdStrategyUnset && isApplyThresholdUnset {
		applyThresholdFromStrategy := d.Get(fmt.Sprintf("%s.%s.0.%s", resourcePath, FieldApplyThresholdStrategy, FieldApplyThresholdStrategyPercentage))
		// Suppress diff if configuration saved from API equals to default
		return applyThresholdFromStrategy == defaultApplyThresholdPercentage
	}

	return oldValue == newValue
}

func toWorkloadResourceLimit(obj map[string]any) *sdk.WorkloadoptimizationV1ResourceLimitStrategy {
	if len(obj) == 0 {
		return nil
	}

	out := &sdk.WorkloadoptimizationV1ResourceLimitStrategy{}
	if v, ok := obj[FieldLimitStrategyType].(string); ok {
		out.Type = sdk.WorkloadoptimizationV1ResourceLimitStrategyType(v)
	}
	if v, ok := obj[FieldLimitStrategyMultiplier].(float64); ok && v > 0 {
		out.Multiplier = lo.ToPtr(v)
	}
	return out
}

func toWorkloadResourcePercentageThresholdStrategy(percentage float64) *sdk.WorkloadoptimizationV1ApplyThresholdStrategy {
	return &sdk.WorkloadoptimizationV1ApplyThresholdStrategy{
		PercentageThreshold: &sdk.WorkloadoptimizationV1ApplyThresholdStrategyPercentageThreshold{
			Percentage: percentage,
		},
	}
}

func toWorkloadScalingPoliciesMap(previousCfg map[string]interface{}, p sdk.WorkloadoptimizationV1ResourcePolicies) []map[string]interface{} {
	m := map[string]interface{}{
		"function": p.Function,
		"args":     p.Args,
		"overhead": p.Overhead,
		"min":      p.Min,
		"max":      p.Max,
	}

	if p.LookBackPeriodSeconds != nil {
		m["look_back_period_seconds"] = int(*p.LookBackPeriodSeconds)
	}

	if p.Limit != nil {
		limit := map[string]any{}

		limit[FieldLimitStrategyType] = p.Limit.Type
		if p.Limit.Multiplier != nil {
			limit[FieldLimitStrategyMultiplier] = *p.Limit.Multiplier
		}
		m[FieldLimitStrategy] = []map[string]any{limit}
	}

	m = lo.Assign(m, mapApplyStrategyBasedOnPreviousConfig(p, previousCfg))

	if p.ManagementOption != nil {
		m["management_option"] = string(*p.ManagementOption)
	}

	return []map[string]interface{}{m}
}

func applyThresholdStrategyToMap(s *sdk.WorkloadoptimizationV1ApplyThresholdStrategy) []map[string]any {
	if s == nil {
		return nil
	}
	m := map[string]any{}

	if s.PercentageThreshold != nil {
		m[FieldApplyThresholdStrategyType] = FieldApplyThresholdStrategyPercentageType
		m[FieldApplyThresholdStrategyPercentage] = s.PercentageThreshold.Percentage
	}

	if len(m) == 0 {
		return nil
	}

	return []map[string]any{m}
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

func toDownscaling(downscaling map[string]any) *sdk.WorkloadoptimizationV1DownscalingSettings {
	if len(downscaling) == 0 {
		return nil
	}

	result := &sdk.WorkloadoptimizationV1DownscalingSettings{}

	if v, ok := downscaling["apply_type"].(string); ok && v != "" {
		result.ApplyType = lo.ToPtr(sdk.WorkloadoptimizationV1ApplyType(v))
	}

	return result
}

func toDownscalingMap(s *sdk.WorkloadoptimizationV1DownscalingSettings) []map[string]any {
	if s == nil {
		return nil
	}

	m := map[string]any{}

	if s.ApplyType != nil {
		m["apply_type"] = string(*s.ApplyType)
	}

	if len(m) == 0 {
		return nil
	}

	return []map[string]any{m}
}

func toMemoryEvent(memoryEvent map[string]any) *sdk.WorkloadoptimizationV1MemoryEventSettings {
	if len(memoryEvent) == 0 {
		return nil
	}

	result := &sdk.WorkloadoptimizationV1MemoryEventSettings{}

	if v, ok := memoryEvent["apply_type"].(string); ok && v != "" {
		result.ApplyType = lo.ToPtr(sdk.WorkloadoptimizationV1ApplyType(v))
	}

	return result
}

func toMemoryEventMap(s *sdk.WorkloadoptimizationV1MemoryEventSettings) []map[string]any {
	if s == nil {
		return nil
	}

	m := map[string]any{}

	if s.ApplyType != nil {
		m["apply_type"] = string(*s.ApplyType)
	}

	if len(m) == 0 {
		return nil
	}

	return []map[string]any{m}
}

func toAntiAffinity(antiAffinity map[string]any) *sdk.WorkloadoptimizationV1AntiAffinitySettings {
	if len(antiAffinity) == 0 {
		return nil
	}

	result := &sdk.WorkloadoptimizationV1AntiAffinitySettings{}

	if v, ok := antiAffinity["consider_anti_affinity"].(bool); ok {
		result.ConsiderAntiAffinity = lo.ToPtr(v)
	}

	return result
}

func toAntiAffinityMap(s *sdk.WorkloadoptimizationV1AntiAffinitySettings) []map[string]any {
	if s == nil {
		return nil
	}

	m := map[string]any{}

	if s.ConsiderAntiAffinity != nil {
		m["consider_anti_affinity"] = *s.ConsiderAntiAffinity
	}

	if len(m) == 0 {
		return nil
	}

	return []map[string]any{m}
}

func getWorkloadScalingPolicyByName(ctx context.Context, client sdk.ClientWithResponsesInterface, clusterID, name string) (*sdk.WorkloadoptimizationV1WorkloadScalingPolicy, error) {
	list, err := client.WorkloadOptimizationAPIListWorkloadScalingPoliciesWithResponse(ctx, clusterID)
	if checkErr := sdk.CheckOKResponse(list, err); checkErr != nil {
		return nil, checkErr
	}

	for _, sp := range list.JSON200.Items {
		if sp.Name == name {
			return &sp, nil
		}
	}
	return nil, fmt.Errorf("policy with name %q not found", name)
}
