package castai

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	fieldNativeHpaSpec              = "native_hpa_spec"
	fieldTakeOwnership              = "take_ownership"
	fieldMinReplicas                = "min_replicas"
	fieldMaxReplicas                = "max_replicas"
	fieldMetrics                    = "metrics"
	fieldBehavior                   = "behavior"
	fieldScaleUp                    = "scale_up"
	fieldScaleDown                  = "scale_down"
	fieldStabilizationWindowSeconds = "stabilization_window_seconds"
	fieldSelectPolicy               = "select_policy"
	fieldPolicies                   = "policies"
	fieldResource                   = "resource"
	fieldTarget                     = "target"
)

func hpaSettingsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "Configures native Kubernetes horizontal pod autoscaling for workloads using this policy.",
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			"management_option": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Defines whether CAST AI observes or manages horizontal pod autoscaling.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"READ_ONLY", "MANAGED"}, false)),
			},
			fieldTakeOwnership: {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Allows CAST AI to take ownership of eligible existing HPAs.",
			},
			fieldNativeHpaSpec: nativeHpaSpecSchema(),
		}},
	}
}

func nativeHpaSpecSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Required:    true,
		MaxItems:    1,
		Description: "Native Kubernetes HPA specification.",
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			fieldMinReplicas: {
				Type:             schema.TypeInt,
				Required:         true,
				Description:      "Minimum number of replicas.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
			},
			fieldMaxReplicas: {
				Type:             schema.TypeInt,
				Required:         true,
				Description:      "Maximum number of replicas.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
			},
			fieldMetrics: {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "Resource utilization metrics used by the HPA.",
				Elem:        hpaMetricSchema(),
			},
			fieldBehavior: hpaBehaviorSchema(),
		}},
	}
}

func hpaMetricSchema() *schema.Resource {
	return &schema.Resource{Schema: map[string]*schema.Schema{
		FieldLimitStrategyType: {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"RESOURCE"}, false)),
		},
		fieldResource: {
			Type:     schema.TypeList,
			Required: true,
			MaxItems: 1,
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"name": {
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringIsNotEmpty,
				},
				fieldTarget: {
					Type:     schema.TypeList,
					Required: true,
					MaxItems: 1,
					Elem: &schema.Resource{Schema: map[string]*schema.Schema{
						FieldLimitStrategyType: {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"UTILIZATION"}, false)),
						},
						"value": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},
					}},
				},
			}},
		},
	}}
}

func hpaBehaviorSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			fieldScaleUp:   hpaScalingRulesSchema(),
			fieldScaleDown: hpaScalingRulesSchema(),
		}},
	}
}

func hpaScalingRulesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			fieldStabilizationWindowSeconds: {
				Type:             schema.TypeInt,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 3600)),
			},
			fieldSelectPolicy: {
				Type:     schema.TypeString,
				Optional: true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
					"MAX_CHANGE_POLICY_SELECT",
					"MIN_CHANGE_POLICY_SELECT",
					"DISABLED_POLICY_SELECT",
				}, false)),
			},
			fieldPolicies: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					FieldLimitStrategyType: {
						Type:     schema.TypeString,
						Required: true,
						ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
							"PODS_SCALING_POLICY",
							"PERCENT_SCALING_POLICY",
						}, false)),
					},
					"value": {
						Type:             schema.TypeInt,
						Required:         true,
						ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
					},
					"period_seconds": {
						Type:             schema.TypeInt,
						Required:         true,
						ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 1800)),
					},
				}},
			},
		}},
	}
}

func validateHpaSettings(value any) error {
	settings, _ := value.([]any)
	if len(settings) == 0 {
		return nil
	}
	setting, ok := settings[0].(map[string]any)
	if !ok {
		return fmt.Errorf("%s must be an object", FieldHpaSettings)
	}
	native := getFirstElem(setting, fieldNativeHpaSpec)
	if native == nil {
		return fmt.Errorf("%s.%s is required", FieldHpaSettings, fieldNativeHpaSpec)
	}
	minReplicas, minOK := native[fieldMinReplicas].(int)
	maxReplicas, maxOK := native[fieldMaxReplicas].(int)
	if !minOK || !maxOK {
		return fmt.Errorf("%s: min_replicas and max_replicas are required", FieldHpaSettings)
	}
	if maxReplicas < minReplicas {
		return fmt.Errorf("%s: max_replicas must be greater than or equal to min_replicas", FieldHpaSettings)
	}
	return nil
}

func toHpaSettings(setting map[string]any) (*sdk.WorkloadoptimizationV1ScalingPolicyHPASettings, error) {
	if len(setting) == 0 {
		return nil, nil
	}
	native := getFirstElem(setting, fieldNativeHpaSpec)
	if native == nil {
		return nil, fmt.Errorf("%s.%s is required", FieldHpaSettings, fieldNativeHpaSpec)
	}
	metrics, err := toHpaMetrics(native[fieldMetrics])
	if err != nil {
		return nil, err
	}
	return &sdk.WorkloadoptimizationV1ScalingPolicyHPASettings{
		ManagementOption: sdk.WorkloadoptimizationV1ManagementOption(setting["management_option"].(string)),
		TakeOwnership:    setting[fieldTakeOwnership].(bool),
		NativeHpaSpec: sdk.WorkloadoptimizationV1ScalingPolicyNativeHPASpec{
			MinReplicas: int32(native[fieldMinReplicas].(int)),
			MaxReplicas: int32(native[fieldMaxReplicas].(int)),
			Metrics:     metrics,
			Behavior:    toHpaBehavior(getFirstElem(native, fieldBehavior)),
		},
	}, nil
}

func toHpaMetrics(value any) ([]sdk.WorkloadoptimizationV1MetricSpec, error) {
	raw, _ := value.([]any)
	metrics := make([]sdk.WorkloadoptimizationV1MetricSpec, 0, len(raw))
	for i, item := range raw {
		metric, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s.%s.%d must be an object", FieldHpaSettings, fieldMetrics, i)
		}
		resource := getFirstElem(metric, fieldResource)
		if resource == nil {
			return nil, fmt.Errorf("%s.%s.%d.%s is required", FieldHpaSettings, fieldMetrics, i, fieldResource)
		}
		target := getFirstElem(resource, fieldTarget)
		if target == nil {
			return nil, fmt.Errorf("%s.%s.%d.%s.%s is required", FieldHpaSettings, fieldMetrics, i, fieldResource, fieldTarget)
		}
		metricType := sdk.WorkloadoptimizationV1MetricSourceType(metric[FieldLimitStrategyType].(string))
		metrics = append(metrics, sdk.WorkloadoptimizationV1MetricSpec{
			Type: &metricType,
			Resource: &sdk.WorkloadoptimizationV1ResourceMetricSource{
				Name: resource["name"].(string),
				Target: sdk.WorkloadoptimizationV1MetricTarget{
					Type:  sdk.WorkloadoptimizationV1MetricTargetType(target[FieldLimitStrategyType].(string)),
					Value: target["value"].(string),
				},
			},
		})
	}
	return metrics, nil
}

func toHpaBehavior(value map[string]any) *sdk.WorkloadoptimizationV1HorizontalPodAutoscalerBehavior {
	if len(value) == 0 {
		return nil
	}
	return &sdk.WorkloadoptimizationV1HorizontalPodAutoscalerBehavior{
		ScaleUp:   toHpaScalingRules(getFirstElem(value, fieldScaleUp)),
		ScaleDown: toHpaScalingRules(getFirstElem(value, fieldScaleDown)),
	}
}

func toHpaScalingRules(value map[string]any) *sdk.WorkloadoptimizationV1HPAScalingRules {
	if len(value) == 0 {
		return nil
	}
	rules := &sdk.WorkloadoptimizationV1HPAScalingRules{}
	if stabilizationWindowSeconds, ok := value[fieldStabilizationWindowSeconds].(int); ok {
		rules.StabilizationWindowSeconds = lo.ToPtr(int32(stabilizationWindowSeconds))
	}
	if selectPolicy, ok := value[fieldSelectPolicy].(string); ok && selectPolicy != "" {
		rules.SelectPolicy = lo.ToPtr(sdk.WorkloadoptimizationV1ScalingPolicySelect(selectPolicy))
	}
	if rawPolicies, ok := value[fieldPolicies].([]any); ok {
		policies := make([]sdk.WorkloadoptimizationV1HPAScalingPolicy, 0, len(rawPolicies))
		for _, item := range rawPolicies {
			policy := item.(map[string]any)
			policies = append(policies, sdk.WorkloadoptimizationV1HPAScalingPolicy{
				Type:          sdk.WorkloadoptimizationV1HPAScalingPolicyType(policy[FieldLimitStrategyType].(string)),
				Value:         int32(policy["value"].(int)),
				PeriodSeconds: int32(policy["period_seconds"].(int)),
			})
		}
		rules.Policies = &policies
	}
	return rules
}

func toHpaSettingsMap(settings *sdk.WorkloadoptimizationV1ScalingPolicyHPASettings) ([]map[string]any, error) {
	if settings == nil {
		return nil, nil
	}
	metrics, err := toHpaMetricsMap(settings.NativeHpaSpec.Metrics)
	if err != nil {
		return nil, err
	}
	native := map[string]any{
		fieldMinReplicas: int(settings.NativeHpaSpec.MinReplicas),
		fieldMaxReplicas: int(settings.NativeHpaSpec.MaxReplicas),
		fieldMetrics:     metrics,
	}
	if behavior := toHpaBehaviorMap(settings.NativeHpaSpec.Behavior); behavior != nil {
		native[fieldBehavior] = behavior
	}
	return []map[string]any{{
		"management_option": string(settings.ManagementOption),
		fieldTakeOwnership:  settings.TakeOwnership,
		fieldNativeHpaSpec:  []map[string]any{native},
	}}, nil
}

func toHpaMetricsMap(metrics []sdk.WorkloadoptimizationV1MetricSpec) ([]map[string]any, error) {
	result := make([]map[string]any, 0, len(metrics))
	for i, metric := range metrics {
		if metric.Type == nil || *metric.Type != sdk.WorkloadoptimizationV1MetricSourceType("RESOURCE") || metric.Resource == nil {
			return nil, fmt.Errorf("%s.%s.%d: only RESOURCE metrics are supported", FieldHpaSettings, fieldMetrics, i)
		}
		if metric.Resource.Target.Type != sdk.WorkloadoptimizationV1MetricTargetType("UTILIZATION") {
			return nil, fmt.Errorf("%s.%s.%d: only UTILIZATION targets are supported", FieldHpaSettings, fieldMetrics, i)
		}
		result = append(result, map[string]any{
			FieldLimitStrategyType: string(*metric.Type),
			fieldResource: []map[string]any{{
				"name": metric.Resource.Name,
				fieldTarget: []map[string]any{{
					FieldLimitStrategyType: string(metric.Resource.Target.Type),
					"value":                metric.Resource.Target.Value,
				}},
			}},
		})
	}
	return result, nil
}

func toHpaBehaviorMap(behavior *sdk.WorkloadoptimizationV1HorizontalPodAutoscalerBehavior) []map[string]any {
	if behavior == nil {
		return nil
	}
	result := map[string]any{}
	if scaleUp := toHpaScalingRulesMap(behavior.ScaleUp); scaleUp != nil {
		result[fieldScaleUp] = scaleUp
	}
	if scaleDown := toHpaScalingRulesMap(behavior.ScaleDown); scaleDown != nil {
		result[fieldScaleDown] = scaleDown
	}
	if len(result) == 0 {
		return nil
	}
	return []map[string]any{result}
}

func toHpaScalingRulesMap(rules *sdk.WorkloadoptimizationV1HPAScalingRules) []map[string]any {
	if rules == nil {
		return nil
	}
	result := map[string]any{}
	if rules.StabilizationWindowSeconds != nil {
		result[fieldStabilizationWindowSeconds] = int(*rules.StabilizationWindowSeconds)
	}
	if rules.SelectPolicy != nil {
		result[fieldSelectPolicy] = string(*rules.SelectPolicy)
	}
	if rules.Policies != nil {
		policies := make([]map[string]any, 0, len(*rules.Policies))
		for _, policy := range *rules.Policies {
			policies = append(policies, map[string]any{
				FieldLimitStrategyType: string(policy.Type),
				"value":                int(policy.Value),
				"period_seconds":       int(policy.PeriodSeconds),
			})
		}
		result[fieldPolicies] = policies
	}
	return []map[string]any{result}
}
