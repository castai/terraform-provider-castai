package castai

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func TestWorkloadScalingPolicyAPISchemaIsValid(t *testing.T) {
	require.NoError(t, Provider("test").InternalValidate())
}

func TestStartupTwoPhaseRecommendationsRoundTrip(t *testing.T) {
	input := map[string]any{
		"period_seconds": 180,
		FieldTwoPhaseRecommendations: []any{map[string]any{
			FieldEnabled: true,
			FieldRequestsOnStartup: []any{map[string]any{
				FieldCPUCores:  0.5,
				FieldMemoryGiB: 1.5,
			}},
		}},
	}

	got := toStartup(input)
	require.NotNil(t, got)
	require.Equal(t, int32(180), *got.PeriodSeconds)
	require.True(t, got.TwoPhaseRecommendations.Enabled)
	require.Equal(t, 0.5, *got.TwoPhaseRecommendations.RequestsOnStartup.CpuCores)
	require.Equal(t, 1.5, *got.TwoPhaseRecommendations.RequestsOnStartup.MemoryGib)

	roundTripped := toStartup(toStartupMap(got)[0])
	require.Equal(t, got, roundTripped)
}

func TestAdditionalRecommendationPolicySettingsRoundTrip(t *testing.T) {
	anomaly := toAnomalyDetection(map[string]any{
		FieldAnomalyDetectionInfiniteMemoryScaling: []any{map[string]any{
			FieldEnabled: true,
		}},
	})
	require.Equal(t, &sdk.WorkloadoptimizationV1AnomalyDetectionSettings{
		InfiniteMemoryScaling: &sdk.WorkloadoptimizationV1InfiniteMemoryScalingSettings{
			Enabled: true,
		},
	}, anomaly)
	require.Equal(t, []map[string]any{{
		FieldAnomalyDetectionInfiniteMemoryScaling: []map[string]any{{
			FieldEnabled: true,
		}},
	}}, toAnomalyDetectionMap(anomaly))

	gpu := toGPU(map[string]any{"management_option": "MANAGED"})
	require.Equal(t, &sdk.WorkloadoptimizationV1GPUSettings{
		ManagementOption: sdk.WorkloadoptimizationV1ManagementOption("MANAGED"),
	}, gpu)
	require.Equal(t, []map[string]any{{"management_option": "MANAGED"}}, toGPUMap(gpu))
}

func TestHPASettingsRoundTrip(t *testing.T) {
	input := map[string]any{
		"management_option": "MANAGED",
		fieldTakeOwnership:  true,
		fieldNativeHPASpec: []any{map[string]any{
			fieldMinReplicas: 2,
			fieldMaxReplicas: 20,
			fieldMetrics: []any{
				map[string]any{
					FieldLimitStrategyType: hpaMetricSourceResource,
					fieldResource: []any{map[string]any{
						"name": "cpu",
						fieldTarget: []any{map[string]any{
							FieldLimitStrategyType: hpaMetricTargetUtilization,
							"value":                "80",
						}},
					}},
				},
				map[string]any{
					FieldLimitStrategyType: hpaMetricSourcePods,
					fieldPods: []any{map[string]any{
						fieldMetric: []any{map[string]any{
							"name":        "requests_per_second",
							fieldSelector: map[string]any{"service": "api"},
						}},
						fieldTarget: []any{map[string]any{
							FieldLimitStrategyType: hpaMetricTargetAverageValue,
							"value":                "100",
						}},
					}},
				},
				map[string]any{
					FieldLimitStrategyType: hpaMetricSourceObject,
					fieldObject: []any{map[string]any{
						fieldMetric: []any{map[string]any{
							"name":        "queue_depth",
							fieldSelector: map[string]any{},
						}},
						fieldDescribedObject: []any{map[string]any{
							"kind":          "Service",
							"name":          "worker",
							fieldAPIVersion: "v1",
						}},
						fieldTarget: []any{map[string]any{
							FieldLimitStrategyType: hpaMetricTargetValue,
							"value":                "10",
						}},
					}},
				},
				map[string]any{
					FieldLimitStrategyType: hpaMetricSourceExternal,
					fieldExternal: []any{map[string]any{
						fieldMetric: []any{map[string]any{
							"name":        "external_queue_depth",
							fieldSelector: map[string]any{},
						}},
						fieldTarget: []any{map[string]any{
							FieldLimitStrategyType: hpaMetricTargetAverageValue,
							"value":                "5",
						}},
					}},
				},
				map[string]any{
					FieldLimitStrategyType: hpaMetricSourceContainerResource,
					fieldContainerResource: []any{map[string]any{
						"name":      "memory",
						"container": "application",
						fieldTarget: []any{map[string]any{
							FieldLimitStrategyType: hpaMetricTargetUtilization,
							"value":                "75",
						}},
					}},
				},
			},
			fieldBehavior: []any{map[string]any{
				fieldScaleUp: []any{map[string]any{
					fieldStabilizationWindowSeconds: 0,
					fieldSelectPolicy:               hpaScalingPolicySelectMaxChange,
					fieldTolerance:                  "0.05",
					fieldPolicies: []any{map[string]any{
						FieldLimitStrategyType: hpaScalingPolicyPercent,
						"value":                100,
						"period_seconds":       15,
					}},
				}},
				fieldScaleDown: []any{map[string]any{
					fieldStabilizationWindowSeconds: 300,
					fieldSelectPolicy:               hpaScalingPolicySelectMinChange,
					fieldPolicies: []any{map[string]any{
						FieldLimitStrategyType: hpaScalingPolicyPods,
						"value":                1,
						"period_seconds":       60,
					}},
				}},
			}},
		}},
	}

	got := toHPASettings(input)
	require.NotNil(t, got)
	require.Equal(t, sdk.WorkloadoptimizationV1ManagementOption("MANAGED"), got.ManagementOption)
	require.True(t, got.TakeOwnership)
	require.Equal(t, int32(2), got.NativeHpaSpec.MinReplicas)
	require.Equal(t, int32(20), got.NativeHpaSpec.MaxReplicas)
	require.Len(t, got.NativeHpaSpec.Metrics, 5)
	require.Equal(t, "api", got.NativeHpaSpec.Metrics[1].Pods.Metric.Selector["service"])
	require.Equal(t, "application", got.NativeHpaSpec.Metrics[4].ContainerResource.Container)
	require.Equal(t, int32(300), *got.NativeHpaSpec.Behavior.ScaleDown.StabilizationWindowSeconds)
	require.Equal(t, "0.05", *got.NativeHpaSpec.Behavior.ScaleUp.Tolerance)

	roundTripped := toHPASettings(toHPASettingsMap(got)[0])
	require.Equal(t, got, roundTripped)
}

func TestHPAConvertersMapping(t *testing.T) {
	got := toHPAConvertersValue([]any{
		map[string]any{
			FieldLimitStrategyType: hpaConverterAverageValueFromOriginalRequests,
		},
	})

	require.Equal(t, &[]sdk.WorkloadoptimizationV1HPAConverters{{
		Type: sdk.WorkloadoptimizationV1HPAConverterType(hpaConverterAverageValueFromOriginalRequests),
	}}, got)
	require.Equal(t, []map[string]any{{
		FieldLimitStrategyType: hpaConverterAverageValueFromOriginalRequests,
	}}, toHPAConvertersMap(got))
}

func TestScalingPolicySchemaAcceptsCurrentAPIRanges(t *testing.T) {
	resource := resourceWorkloadScalingPolicy()

	overheadDiagnostics := resource.Schema["cpu"].Elem.(*schema.Resource).
		Schema["overhead"].ValidateDiagFunc(2.5, cty.Path{})
	require.False(t, overheadDiagnostics.HasError())

	limitDiagnostics := resource.Schema["cpu"].Elem.(*schema.Resource).
		Schema[FieldLimitStrategy].Elem.(*schema.Resource).
		Schema[FieldLimitStrategyType].ValidateDiagFunc("MAINTAIN_RATIO", cty.Path{})
	require.False(t, limitDiagnostics.HasError())
}
