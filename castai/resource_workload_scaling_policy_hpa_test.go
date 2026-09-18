package castai

import (
	"context"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestWorkloadScalingPolicyHpaSettings(t *testing.T) {
	require.NoError(t, Provider("test").InternalValidate())

	input := map[string]any{
		"management_option": "READ_ONLY",
		fieldTakeOwnership:  false,
		fieldNativeHpaSpec: []any{map[string]any{
			fieldMinReplicas: 1,
			fieldMaxReplicas: 10,
			fieldMetrics: []any{
				map[string]any{
					FieldLimitStrategyType: "RESOURCE",
					fieldResource: []any{map[string]any{
						"name": "cpu",
						fieldTarget: []any{map[string]any{
							FieldLimitStrategyType: "UTILIZATION",
							"value":                "80",
						}},
					}},
				},
				map[string]any{
					FieldLimitStrategyType: "RESOURCE",
					fieldResource: []any{map[string]any{
						"name": "memory",
						fieldTarget: []any{map[string]any{
							FieldLimitStrategyType: "UTILIZATION",
							"value":                "70",
						}},
					}},
				},
			},
			fieldBehavior: []any{map[string]any{
				fieldScaleUp: []any{map[string]any{
					fieldStabilizationWindowSeconds: 0,
					fieldPolicies:                   []any{},
				}},
				fieldScaleDown: []any{map[string]any{
					fieldStabilizationWindowSeconds: 300,
					fieldSelectPolicy:               "MAX_CHANGE_POLICY_SELECT",
					fieldPolicies: []any{map[string]any{
						FieldLimitStrategyType: "PODS_SCALING_POLICY",
						"value":                1,
						"period_seconds":       60,
					}},
				}},
			}},
		}},
	}

	data := schema.TestResourceDataRaw(t, resourceWorkloadScalingPolicy().Schema, map[string]any{
		FieldHpaSettings: []any{input},
	})
	settings, err := toHpaSettings(toSection(data, FieldHpaSettings))
	require.NoError(t, err)
	require.False(t, settings.TakeOwnership)
	require.Equal(t, int32(0), *settings.NativeHpaSpec.Behavior.ScaleUp.StabilizationWindowSeconds)
	require.NotNil(t, settings.NativeHpaSpec.Behavior.ScaleUp.Policies)
	require.Empty(t, *settings.NativeHpaSpec.Behavior.ScaleUp.Policies)
	require.Len(t, settings.NativeHpaSpec.Metrics, 2)

	flattened, err := toHpaSettingsMap(settings)
	require.NoError(t, err)
	require.NoError(t, data.Set(FieldHpaSettings, flattened))
	roundTripped, err := toHpaSettings(toSection(data, FieldHpaSettings))
	require.NoError(t, err)
	require.Equal(t, settings, roundTripped)

	invalid := map[string]any{
		fieldNativeHpaSpec: []any{map[string]any{
			fieldMinReplicas: 5,
			fieldMaxReplicas: 4,
		}},
	}
	require.EqualError(t, validateHpaSettings([]any{invalid}), "hpa_settings: max_replicas must be greater than or equal to min_replicas")

	unsupportedType := sdk.WorkloadoptimizationV1MetricSourceType("PODS")
	_, err = toHpaSettingsMap(&sdk.WorkloadoptimizationV1ScalingPolicyHPASettings{
		NativeHpaSpec: sdk.WorkloadoptimizationV1ScalingPolicyNativeHPASpec{
			Metrics: []sdk.WorkloadoptimizationV1MetricSpec{{Type: &unsupportedType}},
		},
	})
	require.EqualError(t, err, "hpa_settings.metrics.0: only RESOURCE metrics are supported")

	clusterID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
	policyID := "98173807-6568-4e2b-9fe1-bcece3301649"
	require.NoError(t, data.Set(FieldClusterID, clusterID))
	require.NoError(t, data.Set("name", "native-hpa"))
	require.NoError(t, data.Set(FieldApplyType, "IMMEDIATE"))
	require.NoError(t, data.Set("management_option", "MANAGED"))

	client := mock_sdk.NewMockClientInterface(gomock.NewController(t))
	policy := &sdk.WorkloadoptimizationV1WorkloadScalingPolicy{
		Id:          policyID,
		ClusterId:   clusterID,
		Name:        "native-hpa",
		ApplyType:   sdk.WorkloadoptimizationV1ApplyType("IMMEDIATE"),
		HpaSettings: settings,
		RecommendationPolicies: sdk.WorkloadoptimizationV1RecommendationPolicies{
			ManagementOption: sdk.WorkloadoptimizationV1ManagementOption("MANAGED"),
		},
	}
	client.EXPECT().
		WorkloadOptimizationAPICreateWorkloadScalingPolicy(gomock.Any(), clusterID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, request sdk.WorkloadOptimizationAPICreateWorkloadScalingPolicyJSONRequestBody, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			require.Equal(t, settings, request.HpaSettings)
			return toResponse(require.New(t), policy, http.StatusOK)
		})
	client.EXPECT().
		WorkloadOptimizationAPIGetWorkloadScalingPolicy(gomock.Any(), clusterID, policyID).
		DoAndReturn(func(_ context.Context, _, _ string, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			return toResponse(require.New(t), policy, http.StatusOK)
		})

	diagnostics := resourceWorkloadScalingPolicy().CreateContext(t.Context(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: client},
	})
	require.False(t, diagnostics.HasError())
	require.Equal(t, policyID, data.Id())
}
