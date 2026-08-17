package castai

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler_v2"
	mock_cluster_autoscaler_v2 "github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler_v2/mock"
)

func newAutoscalerPoliciesProvider(mockClient *mock_cluster_autoscaler_v2.MockClientWithResponsesInterface) *ProviderConfig {
	return &ProviderConfig{
		clusterAutoscalerV2Client: mockClient,
	}
}

func TestResourceAutoscalerPolicies_ReadContext(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	version := "v1"
	enabled := true
	scopedMode := false
	maxCores := int32(20)
	minCores := int32(1)
	clusterLimitsEnabled := true
	emptyNodesDelay := "5m"
	emptyNodesEnabled := true
	unschedulablePodsEnabled := true
	podPinnerEnabled := true

	policies := &cluster_autoscaler_v2.PoliciesV2{
		Enabled:    &enabled,
		ScopedMode: &scopedMode,
		Version:    &version,
		ClusterLimits: &cluster_autoscaler_v2.ClusterLimitsPolicy{
			Enabled: &clusterLimitsEnabled,
			Cpu: &cluster_autoscaler_v2.ClusterLimitsCpu{
				MaxCores: maxCores,
				MinCores: &minCores,
			},
		},
		NodeDownscaler: &cluster_autoscaler_v2.NodeDownscalerPolicy{
			EmptyNodesDelay:   &emptyNodesDelay,
			EmptyNodesEnabled: &emptyNodesEnabled,
		},
		UnschedulablePods: &cluster_autoscaler_v2.UnschedulablePodsPolicy{
			Enabled: &unschedulablePodsEnabled,
			PodPinner: &cluster_autoscaler_v2.PodPinner{
				Enabled: &podPinnerEnabled,
			},
		},
	}

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	provider := newAutoscalerPoliciesProvider(mockClient)

	mockClient.EXPECT().
		PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterId).
		Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}},
			JSON200:      policies,
		}, nil)

	resource := resourceAutoscalerPolicies()
	stateValue := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	data := resource.Data(state)

	diags := resource.ReadContext(context.Background(), data, provider)

	r := require.New(t)
	r.False(diags.HasError())
	r.Equal(clusterId, data.Id())
	r.Equal(clusterId, data.Get(FieldClusterId))
	r.Equal(enabled, data.Get(FieldAutoscalerPoliciesEnabled))
	r.Equal(scopedMode, data.Get(FieldAutoscalerPoliciesScopedMode))
	r.Equal(version, data.Get(FieldAutoscalerPoliciesVersion))
	r.Equal(clusterLimitsEnabled, data.Get(FieldAutoscalerPoliciesClusterLimits+".0."+FieldClusterLimitsEnabled))
	r.Equal(int(maxCores), data.Get(FieldAutoscalerPoliciesClusterLimits+".0."+FieldClusterLimitsCPU+".0."+FieldClusterLimitsCPUMaxCores))
	r.Equal(int(minCores), data.Get(FieldAutoscalerPoliciesClusterLimits+".0."+FieldClusterLimitsCPU+".0."+FieldClusterLimitsCPUMinCores))
	r.Equal(emptyNodesDelay, data.Get(FieldAutoscalerPoliciesNodeDownscaler+".0."+FieldNodeDownscalerEmptyNodesDelay))
	r.Equal(emptyNodesEnabled, data.Get(FieldAutoscalerPoliciesNodeDownscaler+".0."+FieldNodeDownscalerEmptyNodesEnabled))
	r.Equal(unschedulablePodsEnabled, data.Get(FieldAutoscalerPoliciesUnschedulablePods+".0."+FieldUnschedulablePodsEnabled))
	r.Equal(podPinnerEnabled, data.Get(FieldAutoscalerPoliciesUnschedulablePods+".0."+FieldUnschedulablePodsPodPinner+".0."+FieldPodPinnerEnabled))
}

func TestResourceAutoscalerPolicies_CreateContext(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	version := "v2"
	enabled := true
	maxCores := int32(16)
	emptyNodesEnabled := true
	unschedulablePodsEnabled := true

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	provider := newAutoscalerPoliciesProvider(mockClient)

	var capturedBody cluster_autoscaler_v2.PoliciesV2
	mockClient.EXPECT().
		PoliciesV2APIUpdateClusterPoliciesWithResponse(mock.Anything, clusterId, mock.Anything).
		RunAndReturn(func(_ context.Context, _ string, body cluster_autoscaler_v2.PoliciesV2, _ ...cluster_autoscaler_v2.RequestEditorFn) (*cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse, error) {
			capturedBody = body
			return &cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}},
				JSON200: &cluster_autoscaler_v2.PoliciesV2{
					Version: &version,
					Enabled: &enabled,
				},
			}, nil
		})

	mockClient.EXPECT().
		PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterId).
		Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}},
			JSON200: &cluster_autoscaler_v2.PoliciesV2{
				Version: &version,
				Enabled: &enabled,
			},
		}, nil)

	resource := resourceAutoscalerPolicies()
	stateValue := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:                   cty.StringVal(clusterId),
		FieldAutoscalerPoliciesEnabled:   cty.BoolVal(enabled),
		FieldAutoscalerPoliciesClusterLimits: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				FieldClusterLimitsEnabled: cty.BoolVal(true),
				FieldClusterLimitsCPU: cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						FieldClusterLimitsCPUMaxCores: cty.NumberIntVal(int64(maxCores)),
					}),
				}),
			}),
		}),
		FieldAutoscalerPoliciesNodeDownscaler: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				FieldNodeDownscalerEmptyNodesEnabled: cty.BoolVal(emptyNodesEnabled),
			}),
		}),
		FieldAutoscalerPoliciesUnschedulablePods: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				FieldUnschedulablePodsEnabled: cty.BoolVal(unschedulablePodsEnabled),
			}),
		}),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	data := resource.Data(state)

	diags := resource.CreateContext(context.Background(), data, provider)

	r := require.New(t)
	r.False(diags.HasError())
	r.Equal(clusterId, data.Id())
	r.NotNil(capturedBody.Enabled)
	r.Equal(enabled, *capturedBody.Enabled)
	r.NotNil(capturedBody.ClusterLimits)
	r.NotNil(capturedBody.ClusterLimits.Cpu)
	r.Equal(maxCores, capturedBody.ClusterLimits.Cpu.MaxCores)
	r.NotNil(capturedBody.ClusterLimits.Enabled)
	r.Equal(true, *capturedBody.ClusterLimits.Enabled)
	r.NotNil(capturedBody.NodeDownscaler)
	r.NotNil(capturedBody.NodeDownscaler.EmptyNodesEnabled)
	r.Equal(emptyNodesEnabled, *capturedBody.NodeDownscaler.EmptyNodesEnabled)
	r.NotNil(capturedBody.UnschedulablePods)
	r.NotNil(capturedBody.UnschedulablePods.Enabled)
	r.Equal(unschedulablePodsEnabled, *capturedBody.UnschedulablePods.Enabled)
}

func TestResourceAutoscalerPolicies_UpdateContext(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	version := "v3"
	newVersion := "v4"
	emptyNodesDelay := "10m"

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	provider := newAutoscalerPoliciesProvider(mockClient)

	var capturedBody cluster_autoscaler_v2.PoliciesV2
	mockClient.EXPECT().
		PoliciesV2APIUpdateClusterPoliciesWithResponse(mock.Anything, clusterId, mock.Anything).
		RunAndReturn(func(_ context.Context, _ string, body cluster_autoscaler_v2.PoliciesV2, _ ...cluster_autoscaler_v2.RequestEditorFn) (*cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse, error) {
			capturedBody = body
			return &cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}},
				JSON200: &cluster_autoscaler_v2.PoliciesV2{
					Version: &newVersion,
				},
			}, nil
		})

	mockClient.EXPECT().
		PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterId).
		Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}},
			JSON200: &cluster_autoscaler_v2.PoliciesV2{
				Version: &newVersion,
			},
		}, nil)

	resource := resourceAutoscalerPolicies()
	stateValue := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:                   cty.StringVal(clusterId),
		FieldAutoscalerPoliciesVersion:   cty.StringVal(version),
		FieldAutoscalerPoliciesNodeDownscaler: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				FieldNodeDownscalerEmptyNodesDelay: cty.StringVal(emptyNodesDelay),
			}),
		}),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	data := resource.Data(state)

	diags := resource.UpdateContext(context.Background(), data, provider)

	r := require.New(t)
	r.False(diags.HasError())
	r.NotNil(capturedBody.Version)
	r.Equal(version, *capturedBody.Version)
	r.NotNil(capturedBody.NodeDownscaler)
	r.NotNil(capturedBody.NodeDownscaler.EmptyNodesDelay)
	r.Equal(emptyNodesDelay, *capturedBody.NodeDownscaler.EmptyNodesDelay)
}

func TestResourceAutoscalerPolicies_DeleteContext(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	provider := newAutoscalerPoliciesProvider(mockClient)

	resource := resourceAutoscalerPolicies()
	stateValue := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	data := resource.Data(state)

	diags := resource.DeleteContext(context.Background(), data, provider)

	r := require.New(t)
	r.False(diags.HasError())
	r.Empty(data.Id())
}

func TestResourceAutoscalerPolicies_Import(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	provider := newAutoscalerPoliciesProvider(mockClient)

	resource := resourceAutoscalerPolicies()
	stateValue := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	data := resource.Data(state)
	data.SetId(clusterId)

	imported, err := resource.Importer.StateContext(context.Background(), data, provider)

	r := require.New(t)
	r.NoError(err)
	r.Len(imported, 1)
	r.Equal(clusterId, imported[0].Id())
	r.Equal(clusterId, imported[0].Get(FieldClusterId))
}

func TestResourceAutoscalerPolicies_toPoliciesV2(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	version := "v5"
	maxCores := int32(8)
	minCores := int32(2)

	resource := resourceAutoscalerPolicies()
	stateValue := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:                    cty.StringVal(clusterId),
		FieldAutoscalerPoliciesVersion:    cty.StringVal(version),
		FieldAutoscalerPoliciesEnabled:    cty.BoolVal(true),
		FieldAutoscalerPoliciesScopedMode: cty.BoolVal(true),
		FieldAutoscalerPoliciesClusterLimits: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				FieldClusterLimitsEnabled: cty.BoolVal(true),
				FieldClusterLimitsCPU: cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						FieldClusterLimitsCPUMaxCores: cty.NumberIntVal(int64(maxCores)),
						FieldClusterLimitsCPUMinCores: cty.NumberIntVal(int64(minCores)),
					}),
				}),
			}),
		}),
		FieldAutoscalerPoliciesNodeDownscaler: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				FieldNodeDownscalerEmptyNodesDelay:  cty.StringVal("3m"),
				FieldNodeDownscalerEmptyNodesEnabled: cty.BoolVal(false),
			}),
		}),
		FieldAutoscalerPoliciesUnschedulablePods: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				FieldUnschedulablePodsEnabled: cty.BoolVal(true),
				FieldUnschedulablePodsPodPinner: cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						FieldPodPinnerEnabled: cty.BoolVal(true),
					}),
				}),
			}),
		}),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	data := resource.Data(state)

	policies, err := toPoliciesV2(data)

	r := require.New(t)
	r.NoError(err)
	r.NotNil(policies)
	r.NotNil(policies.Version)
	r.Equal(version, *policies.Version)
	r.NotNil(policies.Enabled)
	r.Equal(true, *policies.Enabled)
	r.NotNil(policies.ScopedMode)
	r.Equal(true, *policies.ScopedMode)
	r.NotNil(policies.ClusterLimits)
	r.NotNil(policies.ClusterLimits.Enabled)
	r.Equal(true, *policies.ClusterLimits.Enabled)
	r.NotNil(policies.ClusterLimits.Cpu)
	r.Equal(maxCores, policies.ClusterLimits.Cpu.MaxCores)
	r.NotNil(policies.ClusterLimits.Cpu.MinCores)
	r.Equal(minCores, *policies.ClusterLimits.Cpu.MinCores)
	r.NotNil(policies.NodeDownscaler)
	r.NotNil(policies.NodeDownscaler.EmptyNodesDelay)
	r.Equal("3m", *policies.NodeDownscaler.EmptyNodesDelay)
	r.NotNil(policies.NodeDownscaler.EmptyNodesEnabled)
	r.Equal(false, *policies.NodeDownscaler.EmptyNodesEnabled)
	r.NotNil(policies.UnschedulablePods)
	r.NotNil(policies.UnschedulablePods.Enabled)
	r.Equal(true, *policies.UnschedulablePods.Enabled)
	r.NotNil(policies.UnschedulablePods.PodPinner)
	r.NotNil(policies.UnschedulablePods.PodPinner.Enabled)
	r.Equal(true, *policies.UnschedulablePods.PodPinner.Enabled)
}

func TestResourceAutoscalerPolicies_toJSON(t *testing.T) {
	t.Parallel()

	enabled := true
	maxCores := int32(10)
	policies := &cluster_autoscaler_v2.PoliciesV2{
		Enabled: &enabled,
		ClusterLimits: &cluster_autoscaler_v2.ClusterLimitsPolicy{
			Cpu: &cluster_autoscaler_v2.ClusterLimitsCpu{
				MaxCores: maxCores,
			},
		},
	}

	data, err := policiesV2ToJSON(policies)

	r := require.New(t)
	r.NoError(err)

	var decoded map[string]interface{}
	r.NoError(json.Unmarshal(data, &decoded))
	r.Equal(true, decoded["enabled"])
	r.NotNil(decoded["clusterLimits"])
}

func TestResourceAutoscalerPolicies_ReadContext_Error(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	tests := map[string]struct {
		response *cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse
	}{
		"non-200 status": {
			response: &cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError, Header: map[string][]string{"Content-Type": {"application/json"}}},
				JSON200:      nil,
			},
		},
		"nil JSON200": {
			response: &cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}},
				JSON200:      nil,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
			provider := newAutoscalerPoliciesProvider(mockClient)

			mockClient.EXPECT().
				PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterId).
				Return(tc.response, nil)

			resource := resourceAutoscalerPolicies()
			stateValue := cty.ObjectVal(map[string]cty.Value{
				FieldClusterId: cty.StringVal(clusterId),
			})
			state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
			data := resource.Data(state)

			diags := resource.ReadContext(context.Background(), data, provider)

			r := require.New(t)
			r.True(diags.HasError())
		})
	}
}

// policiesV2ToJSON is a test helper to serialize PoliciesV2 to JSON bytes.
func policiesV2ToJSON(policies *cluster_autoscaler_v2.PoliciesV2) ([]byte, error) {
	return json.Marshal(policies)
}

func TestAccResourceAutoscalerPolicies(t *testing.T) {
	t.Skip("requires CO-4292 V2 API to be live")
}
