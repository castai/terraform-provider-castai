package castai

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldTwoPhaseRecommendations                 = "two_phase_recommendations"
	FieldRequestsOnStartup                       = "requests_on_startup"
	FieldCPUCores                                = "cpu_cores"
	FieldMemoryGiB                               = "memory_gib"
	FieldHPASettings                             = "hpa_settings"
	FieldHPAConverters                           = "hpa_converters"
	FieldGPU                                     = "gpu"
	FieldIsOpsPilot                              = "is_ops_pilot"
	FieldAnomalyDetectionInfiniteMemoryScaling   = "infinite_memory_scaling"
	fieldNativeHPASpec                           = "native_hpa_spec"
	fieldTakeOwnership                           = "take_ownership"
	fieldMinReplicas                             = "min_replicas"
	fieldMaxReplicas                             = "max_replicas"
	fieldMetrics                                 = "metrics"
	fieldBehavior                                = "behavior"
	fieldScaleUp                                 = "scale_up"
	fieldScaleDown                               = "scale_down"
	fieldStabilizationWindowSeconds              = "stabilization_window_seconds"
	fieldSelectPolicy                            = "select_policy"
	fieldPolicies                                = "policies"
	fieldTolerance                               = "tolerance"
	fieldResource                                = "resource"
	fieldPods                                    = "pods"
	fieldObject                                  = "object"
	fieldExternal                                = "external"
	fieldContainerResource                       = "container_resource"
	fieldTarget                                  = "target"
	fieldMetric                                  = "metric"
	fieldSelector                                = "selector"
	fieldDescribedObject                         = "described_object"
	fieldAPIVersion                              = "api_version"
	hpaConverterAverageValueFromOriginalRequests = "AVERAGE_VALUE_FROM_ORIGINAL_REQUESTS"
	hpaMetricSourceResource                      = "RESOURCE"
	hpaMetricSourcePods                          = "PODS"
	hpaMetricSourceObject                        = "OBJECT"
	hpaMetricSourceExternal                      = "EXTERNAL"
	hpaMetricSourceContainerResource             = "CONTAINER_RESOURCE"
	hpaMetricTargetValue                         = "VALUE"
	hpaMetricTargetAverageValue                  = "AVERAGE_VALUE"
	hpaMetricTargetUtilization                   = "UTILIZATION"
	hpaScalingPolicyPods                         = "PODS_SCALING_POLICY"
	hpaScalingPolicyPercent                      = "PERCENT_SCALING_POLICY"
	hpaScalingPolicySelectUnspecified            = "SCALING_POLICY_SELECT_UNSPECIFIED"
	hpaScalingPolicySelectMaxChange              = "MAX_CHANGE_POLICY_SELECT"
	hpaScalingPolicySelectMinChange              = "MIN_CHANGE_POLICY_SELECT"
	hpaScalingPolicySelectDisabled               = "DISABLED_POLICY_SELECT"
)

func twoPhaseRecommendationsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "Configures startup recommendations that use original requests during startup and optimized recommendations afterwards.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				FieldEnabled: {
					Type:        schema.TypeBool,
					Required:    true,
					Description: "Enables two-phase startup recommendations.",
				},
				FieldRequestsOnStartup: {
					Type:        schema.TypeList,
					Optional:    true,
					MaxItems:    1,
					Description: "Overrides the requests used during the startup phase.",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							FieldCPUCores: {
								Type:             schema.TypeFloat,
								Optional:         true,
								Description:      "CPU request in cores used during startup.",
								ValidateDiagFunc: validation.ToDiagFunc(validation.FloatAtLeast(0)),
							},
							FieldMemoryGiB: {
								Type:             schema.TypeFloat,
								Optional:         true,
								Description:      "Memory request in GiB used during startup.",
								ValidateDiagFunc: validation.ToDiagFunc(validation.FloatAtLeast(0)),
							},
						},
					},
				},
			},
		},
	}
}

func infiniteMemoryScalingSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "Configures infinite memory scaling anomaly detection.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				FieldEnabled: {
					Type:        schema.TypeBool,
					Required:    true,
					Description: "Enables infinite memory scaling detection for workloads using this policy.",
				},
			},
		},
	}
}

func hpaConvertersSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "Configures conversion of existing HPAs when vertical optimization is used without HPA management.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				FieldLimitStrategyType: {
					Type:             schema.TypeString,
					Required:         true,
					Description:      "HPA converter strategy.",
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{hpaConverterAverageValueFromOriginalRequests}, false)),
				},
			},
		},
	}
}

func gpuSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "Configures GPU optimization.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"management_option": managementOptionSchema(),
			},
		},
	}
}

func managementOptionSchema() *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		Description:      "Defines whether CAST AI observes or manages the feature.",
		ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"READ_ONLY", "MANAGED"}, false)),
	}
}

func hpaSettingsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MaxItems:    1,
		Description: "Configures horizontal pod autoscaling for workloads using this policy.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"management_option": managementOptionSchema(),
				fieldTakeOwnership: {
					Type:        schema.TypeBool,
					Required:    true,
					Description: "Allows CAST AI to take ownership of eligible existing HPAs.",
				},
				fieldNativeHPASpec: nativeHPASpecSchema(),
			},
		},
	}
}

func validateHPASettings(d *schema.ResourceDiff) error {
	settings, ok := d.Get(FieldHPASettings).([]any)
	if !ok || len(settings) == 0 {
		return nil
	}
	setting, ok := settings[0].(map[string]any)
	if !ok {
		return nil
	}
	native := getFirstElem(setting, fieldNativeHPASpec)
	if native == nil {
		return nil
	}
	if minReplicas, maxReplicas := intValue(native, fieldMinReplicas), intValue(native, fieldMaxReplicas); maxReplicas < minReplicas {
		return fmt.Errorf("%s: max_replicas must be greater than or equal to min_replicas", FieldHPASettings)
	}
	for index, rawMetric := range listValue(native, fieldMetrics) {
		metric, ok := rawMetric.(map[string]any)
		if !ok {
			continue
		}
		metricType := scalingPolicyStringValue(metric, FieldLimitStrategyType)
		sourceByType := map[string]string{
			hpaMetricSourceResource:          fieldResource,
			hpaMetricSourcePods:              fieldPods,
			hpaMetricSourceObject:            fieldObject,
			hpaMetricSourceExternal:          fieldExternal,
			hpaMetricSourceContainerResource: fieldContainerResource,
		}
		expectedSource := sourceByType[metricType]
		configuredSources := make([]string, 0, len(sourceByType))
		for _, source := range sourceByType {
			if getFirstElem(metric, source) != nil {
				configuredSources = append(configuredSources, source)
			}
		}
		if len(configuredSources) != 1 || configuredSources[0] != expectedSource {
			return fmt.Errorf(
				"%s: metric %d with type %q must configure exactly its %q source block",
				FieldHPASettings,
				index,
				metricType,
				expectedSource,
			)
		}
	}
	return nil
}

func isOpsPilotSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "Marks a policy as managed by OpsPilot.",
	}
}

func withAPIDefaultDiffSuppression(s *schema.Schema, path ...string) *schema.Schema {
	s.DiffSuppressFunc = func(_ string, oldValue, newValue string, d *schema.ResourceData) bool {
		if !rawConfigHasField(d.GetRawConfig(), path...) {
			return true
		}
		return oldValue == newValue
	}
	return s
}

func nativeHPASpecSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Required:    true,
		MaxItems:    1,
		Description: "Native Kubernetes HPA specification.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
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
					Description: "Metrics used by the HPA.",
					Elem:        hpaMetricSpecSchema(),
				},
				fieldBehavior: hpaBehaviorSchema(),
			},
		},
	}
}

func hpaMetricSpecSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			FieldLimitStrategyType: {
				Type:     schema.TypeString,
				Required: true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
					hpaMetricSourceResource,
					hpaMetricSourcePods,
					hpaMetricSourceObject,
					hpaMetricSourceExternal,
					hpaMetricSourceContainerResource,
				}, false)),
			},
			fieldResource:          hpaResourceMetricSourceSchema(),
			fieldPods:              hpaPodsMetricSourceSchema(),
			fieldObject:            hpaObjectMetricSourceSchema(),
			fieldExternal:          hpaExternalMetricSourceSchema(),
			fieldContainerResource: hpaContainerResourceMetricSourceSchema(),
		},
	}
}

func hpaResourceMetricSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			"name":      requiredStringSchema(),
			fieldTarget: hpaMetricTargetSchema(),
		}},
	}
}

func hpaPodsMetricSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			fieldMetric: hpaMetricIdentifierSchema(),
			fieldTarget: hpaMetricTargetSchema(),
		}},
	}
}

func hpaObjectMetricSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			fieldMetric:          hpaMetricIdentifierSchema(),
			fieldDescribedObject: hpaDescribedObjectSchema(),
			fieldTarget:          hpaMetricTargetSchema(),
		}},
	}
}

func hpaExternalMetricSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			fieldMetric: hpaMetricIdentifierSchema(),
			fieldTarget: hpaMetricTargetSchema(),
		}},
	}
}

func hpaContainerResourceMetricSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			"name":      requiredStringSchema(),
			"container": requiredStringSchema(),
			fieldTarget: hpaMetricTargetSchema(),
		}},
	}
}

func hpaMetricIdentifierSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Required: true,
		MaxItems: 1,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			"name": requiredStringSchema(),
			fieldSelector: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Metric label selector.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		}},
	}
}

func hpaDescribedObjectSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Required: true,
		MaxItems: 1,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			"kind":          optionalStringSchema(),
			"name":          optionalStringSchema(),
			fieldAPIVersion: optionalStringSchema(),
		}},
	}
}

func hpaMetricTargetSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Required: true,
		MaxItems: 1,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			FieldLimitStrategyType: {
				Type:     schema.TypeString,
				Required: true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
					hpaMetricTargetValue,
					hpaMetricTargetAverageValue,
					hpaMetricTargetUtilization,
				}, false)),
			},
			"value": requiredStringSchema(),
		}},
	}
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
					hpaScalingPolicySelectUnspecified,
					hpaScalingPolicySelectMaxChange,
					hpaScalingPolicySelectMinChange,
					hpaScalingPolicySelectDisabled,
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
							hpaScalingPolicyPods,
							hpaScalingPolicyPercent,
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
			fieldTolerance: optionalStringSchema(),
		}},
	}
}

func requiredStringSchema() *schema.Schema {
	return &schema.Schema{Type: schema.TypeString, Required: true}
}

func optionalStringSchema() *schema.Schema {
	return &schema.Schema{Type: schema.TypeString, Optional: true}
}

func toTwoPhaseRecommendations(m map[string]any) *sdk.WorkloadoptimizationV1TwoPhaseRecommendations {
	if len(m) == 0 {
		return nil
	}
	result := &sdk.WorkloadoptimizationV1TwoPhaseRecommendations{}
	if enabled, ok := m[FieldEnabled].(bool); ok {
		result.Enabled = enabled
	}
	if requests := getFirstElem(m, FieldRequestsOnStartup); requests != nil {
		resourceQuantity := &sdk.WorkloadoptimizationV1ResourceQuantity{}
		if cpuCores, ok := requests[FieldCPUCores].(float64); ok {
			resourceQuantity.CpuCores = lo.ToPtr(cpuCores)
		}
		if memoryGiB, ok := requests[FieldMemoryGiB].(float64); ok {
			resourceQuantity.MemoryGib = lo.ToPtr(memoryGiB)
		}
		if resourceQuantity.CpuCores != nil || resourceQuantity.MemoryGib != nil {
			result.RequestsOnStartup = resourceQuantity
		}
	}
	return result
}

func toTwoPhaseRecommendationsMap(s *sdk.WorkloadoptimizationV1TwoPhaseRecommendations) []map[string]any {
	if s == nil {
		return nil
	}
	m := map[string]any{FieldEnabled: s.Enabled}
	if s.RequestsOnStartup != nil {
		requests := map[string]any{}
		if s.RequestsOnStartup.CpuCores != nil {
			requests[FieldCPUCores] = *s.RequestsOnStartup.CpuCores
		}
		if s.RequestsOnStartup.MemoryGib != nil {
			requests[FieldMemoryGiB] = *s.RequestsOnStartup.MemoryGib
		}
		if len(requests) > 0 {
			m[FieldRequestsOnStartup] = []map[string]any{requests}
		}
	}
	return []map[string]any{m}
}

func toHPAConverters(d *schema.ResourceData) *[]sdk.WorkloadoptimizationV1HPAConverters {
	return toHPAConvertersValue(d.Get(FieldHPAConverters))
}

func toHPAConvertersValue(value any) *[]sdk.WorkloadoptimizationV1HPAConverters {
	raw := listFromValue(value)
	if len(raw) == 0 {
		return nil
	}
	result := make([]sdk.WorkloadoptimizationV1HPAConverters, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if converterType, ok := m[FieldLimitStrategyType].(string); ok && converterType != "" {
			result = append(result, sdk.WorkloadoptimizationV1HPAConverters{
				Type: sdk.WorkloadoptimizationV1HPAConverterType(converterType),
			})
		}
	}
	if len(result) == 0 {
		return nil
	}
	return &result
}

func listFromValue(value any) []any {
	switch list := value.(type) {
	case []any:
		return list
	case []map[string]any:
		result := make([]any, 0, len(list))
		for _, item := range list {
			result = append(result, item)
		}
		return result
	default:
		return nil
	}
}

func toHPAConvertersMap(s *[]sdk.WorkloadoptimizationV1HPAConverters) []map[string]any {
	if s == nil || len(*s) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(*s))
	for _, converter := range *s {
		result = append(result, map[string]any{
			FieldLimitStrategyType: string(converter.Type),
		})
	}
	return result
}

func toGPU(m map[string]any) *sdk.WorkloadoptimizationV1GPUSettings {
	if len(m) == 0 {
		return nil
	}
	managementOption, ok := m["management_option"].(string)
	if !ok || managementOption == "" {
		return nil
	}
	return &sdk.WorkloadoptimizationV1GPUSettings{
		ManagementOption: sdk.WorkloadoptimizationV1ManagementOption(managementOption),
	}
}

func toGPUMap(s *sdk.WorkloadoptimizationV1GPUSettings) []map[string]any {
	if s == nil {
		return nil
	}
	return []map[string]any{{
		"management_option": string(s.ManagementOption),
	}}
}

func toHPASettings(m map[string]any) *sdk.WorkloadoptimizationV1ScalingPolicyHPASettings {
	if len(m) == 0 {
		return nil
	}
	native := getFirstElem(m, fieldNativeHPASpec)
	if native == nil {
		return nil
	}
	result := &sdk.WorkloadoptimizationV1ScalingPolicyHPASettings{
		ManagementOption: sdk.WorkloadoptimizationV1ManagementOption(scalingPolicyStringValue(m, "management_option")),
		TakeOwnership:    boolValue(m, fieldTakeOwnership),
		NativeHpaSpec: sdk.WorkloadoptimizationV1ScalingPolicyNativeHPASpec{
			MinReplicas: int32(intValue(native, fieldMinReplicas)),
			MaxReplicas: int32(intValue(native, fieldMaxReplicas)),
			Metrics:     toHPAMetrics(listValue(native, fieldMetrics)),
			Behavior:    toHPABehavior(getFirstElem(native, fieldBehavior)),
		},
	}
	return result
}

func toHPASettingsMap(s *sdk.WorkloadoptimizationV1ScalingPolicyHPASettings) []map[string]any {
	if s == nil {
		return nil
	}
	native := map[string]any{
		fieldMinReplicas: int(s.NativeHpaSpec.MinReplicas),
		fieldMaxReplicas: int(s.NativeHpaSpec.MaxReplicas),
		fieldMetrics:     toHPAMetricsMap(s.NativeHpaSpec.Metrics),
	}
	if behavior := toHPABehaviorMap(s.NativeHpaSpec.Behavior); behavior != nil {
		native[fieldBehavior] = behavior
	}
	return []map[string]any{{
		"management_option": string(s.ManagementOption),
		fieldTakeOwnership:  s.TakeOwnership,
		fieldNativeHPASpec:  []map[string]any{native},
	}}
}

func toHPAMetrics(raw []any) []sdk.WorkloadoptimizationV1MetricSpec {
	result := make([]sdk.WorkloadoptimizationV1MetricSpec, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		metricType := sdk.WorkloadoptimizationV1MetricSourceType(scalingPolicyStringValue(m, FieldLimitStrategyType))
		metric := sdk.WorkloadoptimizationV1MetricSpec{Type: &metricType}
		if source := getFirstElem(m, fieldResource); source != nil {
			metric.Resource = &sdk.WorkloadoptimizationV1ResourceMetricSource{
				Name:   scalingPolicyStringValue(source, "name"),
				Target: toHPAMetricTarget(getFirstElem(source, fieldTarget)),
			}
		}
		if source := getFirstElem(m, fieldPods); source != nil {
			metric.Pods = &sdk.WorkloadoptimizationV1PodsMetricSource{
				Metric: toHPAMetricIdentifier(getFirstElem(source, fieldMetric)),
				Target: toHPAMetricTarget(getFirstElem(source, fieldTarget)),
			}
		}
		if source := getFirstElem(m, fieldObject); source != nil {
			metric.Object = &sdk.WorkloadoptimizationV1ObjectMetricSource{
				Metric:          toHPAMetricIdentifier(getFirstElem(source, fieldMetric)),
				DescribedObject: toHPADescribedObject(getFirstElem(source, fieldDescribedObject)),
				Target:          toHPAMetricTarget(getFirstElem(source, fieldTarget)),
			}
		}
		if source := getFirstElem(m, fieldExternal); source != nil {
			metric.External = &sdk.WorkloadoptimizationV1ExternalMetricSource{
				Metric: toHPAMetricIdentifier(getFirstElem(source, fieldMetric)),
				Target: toHPAMetricTarget(getFirstElem(source, fieldTarget)),
			}
		}
		if source := getFirstElem(m, fieldContainerResource); source != nil {
			metric.ContainerResource = &sdk.WorkloadoptimizationV1ContainerResourceMetricSource{
				Name:      scalingPolicyStringValue(source, "name"),
				Container: scalingPolicyStringValue(source, "container"),
				Target:    toHPAMetricTarget(getFirstElem(source, fieldTarget)),
			}
		}
		result = append(result, metric)
	}
	return result
}

func toHPAMetricsMap(metrics []sdk.WorkloadoptimizationV1MetricSpec) []map[string]any {
	result := make([]map[string]any, 0, len(metrics))
	for _, metric := range metrics {
		m := map[string]any{}
		if metric.Type != nil {
			m[FieldLimitStrategyType] = string(*metric.Type)
		}
		if metric.Resource != nil {
			m[fieldResource] = []map[string]any{{
				"name":      metric.Resource.Name,
				fieldTarget: toHPAMetricTargetMap(metric.Resource.Target),
			}}
		}
		if metric.Pods != nil {
			m[fieldPods] = []map[string]any{{
				fieldMetric: toHPAMetricIdentifierMap(metric.Pods.Metric),
				fieldTarget: toHPAMetricTargetMap(metric.Pods.Target),
			}}
		}
		if metric.Object != nil {
			m[fieldObject] = []map[string]any{{
				fieldMetric:          toHPAMetricIdentifierMap(metric.Object.Metric),
				fieldDescribedObject: toHPADescribedObjectMap(metric.Object.DescribedObject),
				fieldTarget:          toHPAMetricTargetMap(metric.Object.Target),
			}}
		}
		if metric.External != nil {
			m[fieldExternal] = []map[string]any{{
				fieldMetric: toHPAMetricIdentifierMap(metric.External.Metric),
				fieldTarget: toHPAMetricTargetMap(metric.External.Target),
			}}
		}
		if metric.ContainerResource != nil {
			m[fieldContainerResource] = []map[string]any{{
				"name":      metric.ContainerResource.Name,
				"container": metric.ContainerResource.Container,
				fieldTarget: toHPAMetricTargetMap(metric.ContainerResource.Target),
			}}
		}
		result = append(result, m)
	}
	return result
}

func toHPAMetricTarget(m map[string]any) sdk.WorkloadoptimizationV1MetricTarget {
	return sdk.WorkloadoptimizationV1MetricTarget{
		Type:  sdk.WorkloadoptimizationV1MetricTargetType(scalingPolicyStringValue(m, FieldLimitStrategyType)),
		Value: scalingPolicyStringValue(m, "value"),
	}
}

func toHPAMetricTargetMap(target sdk.WorkloadoptimizationV1MetricTarget) []map[string]any {
	return []map[string]any{{
		FieldLimitStrategyType: string(target.Type),
		"value":                target.Value,
	}}
}

func toHPAMetricIdentifier(m map[string]any) sdk.WorkloadoptimizationV1MetricIdentifier {
	return sdk.WorkloadoptimizationV1MetricIdentifier{
		Name:     scalingPolicyStringValue(m, "name"),
		Selector: stringMapValue(m, fieldSelector),
	}
}

func toHPAMetricIdentifierMap(identifier sdk.WorkloadoptimizationV1MetricIdentifier) []map[string]any {
	return []map[string]any{{
		"name":        identifier.Name,
		fieldSelector: identifier.Selector,
	}}
}

func toHPADescribedObject(m map[string]any) sdk.WorkloadoptimizationV1CrossVersionObjectReference {
	result := sdk.WorkloadoptimizationV1CrossVersionObjectReference{}
	if v := scalingPolicyStringValue(m, "kind"); v != "" {
		result.Kind = &v
	}
	if v := scalingPolicyStringValue(m, "name"); v != "" {
		result.Name = &v
	}
	if v := scalingPolicyStringValue(m, fieldAPIVersion); v != "" {
		result.ApiVersion = &v
	}
	return result
}

func toHPADescribedObjectMap(object sdk.WorkloadoptimizationV1CrossVersionObjectReference) []map[string]any {
	m := map[string]any{}
	if object.Kind != nil {
		m["kind"] = *object.Kind
	}
	if object.Name != nil {
		m["name"] = *object.Name
	}
	if object.ApiVersion != nil {
		m[fieldAPIVersion] = *object.ApiVersion
	}
	return []map[string]any{m}
}

func toHPABehavior(m map[string]any) *sdk.WorkloadoptimizationV1HorizontalPodAutoscalerBehavior {
	if len(m) == 0 {
		return nil
	}
	result := &sdk.WorkloadoptimizationV1HorizontalPodAutoscalerBehavior{
		ScaleUp:   toHPAScalingRules(getFirstElem(m, fieldScaleUp)),
		ScaleDown: toHPAScalingRules(getFirstElem(m, fieldScaleDown)),
	}
	if result.ScaleUp == nil && result.ScaleDown == nil {
		return nil
	}
	return result
}

func toHPABehaviorMap(behavior *sdk.WorkloadoptimizationV1HorizontalPodAutoscalerBehavior) []map[string]any {
	if behavior == nil {
		return nil
	}
	m := map[string]any{}
	if scaleUp := toHPAScalingRulesMap(behavior.ScaleUp); scaleUp != nil {
		m[fieldScaleUp] = scaleUp
	}
	if scaleDown := toHPAScalingRulesMap(behavior.ScaleDown); scaleDown != nil {
		m[fieldScaleDown] = scaleDown
	}
	if len(m) == 0 {
		return nil
	}
	return []map[string]any{m}
}

func toHPAScalingRules(m map[string]any) *sdk.WorkloadoptimizationV1HPAScalingRules {
	if len(m) == 0 {
		return nil
	}
	result := &sdk.WorkloadoptimizationV1HPAScalingRules{}
	if v, ok := m[fieldStabilizationWindowSeconds].(int); ok {
		result.StabilizationWindowSeconds = lo.ToPtr(int32(v))
	}
	if v, ok := m[fieldSelectPolicy].(string); ok && v != "" {
		result.SelectPolicy = lo.ToPtr(sdk.WorkloadoptimizationV1ScalingPolicySelect(v))
	}
	if v, ok := m[fieldTolerance].(string); ok && v != "" {
		result.Tolerance = &v
	}
	if raw := listValue(m, fieldPolicies); len(raw) > 0 {
		policies := make([]sdk.WorkloadoptimizationV1HPAScalingPolicy, 0, len(raw))
		for _, item := range raw {
			policyMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			policies = append(policies, sdk.WorkloadoptimizationV1HPAScalingPolicy{
				Type:          sdk.WorkloadoptimizationV1HPAScalingPolicyType(scalingPolicyStringValue(policyMap, FieldLimitStrategyType)),
				Value:         int32(intValue(policyMap, "value")),
				PeriodSeconds: int32(intValue(policyMap, "period_seconds")),
			})
		}
		result.Policies = &policies
	}
	return result
}

func toHPAScalingRulesMap(rules *sdk.WorkloadoptimizationV1HPAScalingRules) []map[string]any {
	if rules == nil {
		return nil
	}
	m := map[string]any{}
	if rules.StabilizationWindowSeconds != nil {
		m[fieldStabilizationWindowSeconds] = int(*rules.StabilizationWindowSeconds)
	}
	if rules.SelectPolicy != nil {
		m[fieldSelectPolicy] = string(*rules.SelectPolicy)
	}
	if rules.Tolerance != nil {
		m[fieldTolerance] = *rules.Tolerance
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
		m[fieldPolicies] = policies
	}
	if len(m) == 0 {
		return nil
	}
	return []map[string]any{m}
}

func scalingPolicyStringValue(m map[string]any, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}

func boolValue(m map[string]any, key string) bool {
	if value, ok := m[key].(bool); ok {
		return value
	}
	return false
}

func intValue(m map[string]any, key string) int {
	if value, ok := m[key].(int); ok {
		return value
	}
	return 0
}

func listValue(m map[string]any, key string) []any {
	switch value := m[key].(type) {
	case []any:
		return value
	case []map[string]any:
		result := make([]any, 0, len(value))
		for _, item := range value {
			result = append(result, item)
		}
		return result
	}
	return nil
}

func stringMapValue(m map[string]any, key string) map[string]string {
	result := map[string]string{}
	switch values := m[key].(type) {
	case map[string]any:
		for k, value := range values {
			if stringValue, ok := value.(string); ok {
				result[k] = stringValue
			}
		}
	case map[string]string:
		for k, value := range values {
			result[k] = value
		}
	}
	return result
}
