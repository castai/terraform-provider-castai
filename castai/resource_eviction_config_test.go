package castai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/workload_eviction"
	mock_workload_eviction "github.com/castai/terraform-provider-castai/castai/sdk/workload_eviction/mock"
)

func TestEvictionConfig_ReadContext(t *testing.T) {
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	ctx := context.Background()

	tests := map[string]struct {
		data     *workload_eviction.AdvancedConfig
		testFunc func(*testing.T, diag.Diagnostics, *schema.ResourceData)
	}{
		"should work with empty config": {
			data: &workload_eviction.AdvancedConfig{EvictionConfig: []workload_eviction.EvictionConfig{}},
			testFunc: func(t *testing.T, res diag.Diagnostics, data *schema.ResourceData) {
				r := require.New(t)
				r.Empty(res)
				r.False(res.HasError())
				eac := data.Get(FieldEvictorAdvancedConfig)
				d, ok := eac.([]interface{})
				r.True(ok)
				r.Len(d, 0)
			},
		},
		"should read config": {
			data: &workload_eviction.AdvancedConfig{EvictionConfig: []workload_eviction.EvictionConfig{
				{
					PodSelector: &workload_eviction.PodSelector{
						Kind: lo.ToPtr("Job"),
						LabelSelector: &workload_eviction.LabelSelector{
							MatchLabels: &map[string]string{"key1": "value1"},
						},
					},
					Settings: workload_eviction.EvictionSettings{
						Aggressive: &workload_eviction.EvictionSettingsSettingEnabled{Enabled: true},
					},
				},
			}},
			testFunc: func(t *testing.T, res diag.Diagnostics, data *schema.ResourceData) {
				r := require.New(t)
				r.Empty(res)
				r.False(res.HasError())
				eac := data.Get(FieldEvictorAdvancedConfig)
				r.NotNil(eac)
				podSelectorKind := data.Get(fmt.Sprintf("%s.0.%s.0.kind", FieldEvictorAdvancedConfig, FieldPodSelector))
				r.Equal("Job", podSelectorKind)
				podSelectorLabelValue := data.Get(fmt.Sprintf("%s.0.%s.0.%s.key1", FieldEvictorAdvancedConfig, FieldPodSelector, FieldMatchLabels))
				r.Equal("value1", podSelectorLabelValue)
			},
		},
		"should handle multiple evictionConfig objects": {
			data: &workload_eviction.AdvancedConfig{EvictionConfig: []workload_eviction.EvictionConfig{
				{
					PodSelector: &workload_eviction.PodSelector{
						Kind: lo.ToPtr("Job"),
						LabelSelector: &workload_eviction.LabelSelector{
							MatchLabels: &map[string]string{"key1": "value1"},
						},
					},
					Settings: workload_eviction.EvictionSettings{
						Aggressive: &workload_eviction.EvictionSettingsSettingEnabled{Enabled: true},
					},
				},
				{
					NodeSelector: &workload_eviction.NodeSelector{
						LabelSelector: workload_eviction.LabelSelector{
							MatchLabels: &map[string]string{"node-label": "value1"},
						},
					},
					Settings: workload_eviction.EvictionSettings{
						Disposable: &workload_eviction.EvictionSettingsSettingEnabled{Enabled: true},
					},
				},
			}},
			testFunc: func(t *testing.T, res diag.Diagnostics, data *schema.ResourceData) {
				r := require.New(t)
				r.Empty(res)
				r.False(res.HasError())
				eac := data.Get(FieldEvictorAdvancedConfig)
				r.NotNil(eac)
				podSelectorKind := data.Get(fmt.Sprintf("%s.0.%s.0.kind", FieldEvictorAdvancedConfig, FieldPodSelector))
				r.Equal("Job", podSelectorKind)
				podSelectorLabelValue := data.Get(fmt.Sprintf("%s.0.%s.0.%s.key1", FieldEvictorAdvancedConfig, FieldPodSelector, FieldMatchLabels))
				r.Equal("value1", podSelectorLabelValue)
				nodeSelectorLabelValue := data.Get(fmt.Sprintf("%s.1.%s.0.%s.node-label", FieldEvictorAdvancedConfig, FieldNodeSelector, FieldMatchLabels))
				r.Equal("value1", nodeSelectorLabelValue)
			},
		},
		"should read pod_selector replicas_min": {
			data: &workload_eviction.AdvancedConfig{EvictionConfig: []workload_eviction.EvictionConfig{
				{
					PodSelector: &workload_eviction.PodSelector{
						Kind:        lo.ToPtr("Deployment"),
						ReplicasMin: lo.ToPtr(int32(2)),
						LabelSelector: &workload_eviction.LabelSelector{
							MatchLabels: &map[string]string{"key1": "value1"},
						},
					},
					Settings: workload_eviction.EvictionSettings{
						Aggressive: &workload_eviction.EvictionSettingsSettingEnabled{Enabled: true},
					},
				},
			}},
			testFunc: func(t *testing.T, res diag.Diagnostics, data *schema.ResourceData) {
				r := require.New(t)
				r.Empty(res)
				r.False(res.HasError())
				replicasMin := data.Get(fmt.Sprintf("%s.0.%s.0.%s", FieldEvictorAdvancedConfig, FieldPodSelector, FieldPodSelectorReplicasMin))
				r.Equal(2, replicasMin)
			},
		},
		"should handle label expressions": {
			data: &workload_eviction.AdvancedConfig{EvictionConfig: []workload_eviction.EvictionConfig{
				{
					PodSelector: &workload_eviction.PodSelector{
						Kind: lo.ToPtr("Job"),
						LabelSelector: &workload_eviction.LabelSelector{
							MatchExpressions: &[]workload_eviction.LabelSelectorExpression{{
								Key:      "value1",
								Operator: workload_eviction.LabelSelectorExpressionOperatorIN,
								Values:   &[]string{"v1", "v2"},
							}},
						},
					},
					Settings: workload_eviction.EvictionSettings{
						Aggressive: &workload_eviction.EvictionSettingsSettingEnabled{Enabled: true},
					},
				},
			}},
			testFunc: func(t *testing.T, res diag.Diagnostics, data *schema.ResourceData) {
				r := require.New(t)
				r.Empty(res)
				r.False(res.HasError())
				eac := data.Get(FieldEvictorAdvancedConfig)
				r.NotNil(eac)
				podSelectorKeyValue := data.Get(fmt.Sprintf("%s.0.%s.0.%s.0.key", FieldEvictorAdvancedConfig, FieldPodSelector, FieldMatchExpressions))
				r.Equal("value1", podSelectorKeyValue)
				podSelectorValues := data.Get(fmt.Sprintf("%s.0.%s.0.%s.0.values", FieldEvictorAdvancedConfig, FieldPodSelector, FieldMatchExpressions))
				r.Equal([]interface{}{"v1", "v2"}, podSelectorValues)
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)
			provider := &ProviderConfig{workloadEvictionClient: mockClient}
			resource := resourceEvictionConfig()
			initialState := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				FieldClusterId: cty.StringVal(clusterId),
			}), 0)

			mockClient.EXPECT().
				EvictorAPIGetEvictorAdvancedConfigWithResponse(mock.Anything, clusterId).
				Return(&workload_eviction.EvictorAPIGetEvictorAdvancedConfigResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Header: map[string][]string{"Content-Type": {"json"}}},
					JSON200:      test.data,
				}, nil)
			data := resource.Data(initialState)

			result := resource.ReadContext(ctx, data, provider)
			test.testFunc(t, result, data)
		})
	}
}

func TestEvictionConfig_CreateContext(t *testing.T) {
	r := require.New(t)
	mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)

	ctx := context.Background()
	provider := &ProviderConfig{
		workloadEvictionClient: mockClient,
	}
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	evictionConfigResponse := `{"evictionConfig":[{"podSelector":{"kind":"Job","labelSelector":{"matchLabels":{"key1":"value1"}}},"settings":{"aggressive":{"enabled":true}}}]}`

	resource := resourceEvictionConfig()

	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
		FieldEvictorAdvancedConfig: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"pod_selector": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"kind": cty.StringVal("Job"),
					"match_labels": cty.MapVal(map[string]cty.Value{
						"key1": cty.StringVal("value1"),
					}),
				}),
				}),
				"aggressive": cty.BoolVal(true),
			}),
		}),
	})

	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	mockClient.EXPECT().EvictorAPIUpdateEvictorAdvancedConfigWithBodyWithResponse(mock.Anything, clusterId, "application/json", mock.Anything).
		RunAndReturn(func(ctx context.Context, clusterId string, contentType string, body io.Reader, reqEditors ...workload_eviction.RequestEditorFn) (*workload_eviction.EvictorAPIUpdateEvictorAdvancedConfigResponse, error) {

			got, _ := io.ReadAll(body)
			expected := []byte(evictionConfigResponse)

			eq, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(eq, fmt.Sprintf("got:      %v\n"+
				"expected: %v\n", string(got), string(expected)))

			return &workload_eviction.EvictorAPIUpdateEvictorAdvancedConfigResponse{
				HTTPResponse: &http.Response{StatusCode: 200, Header: map[string][]string{"Content-Type": {"json"}}},
				JSON200: &workload_eviction.AdvancedConfig{EvictionConfig: []workload_eviction.EvictionConfig{
					{
						PodSelector: &workload_eviction.PodSelector{
							Kind: lo.ToPtr("Job"),
							LabelSelector: &workload_eviction.LabelSelector{
								MatchLabels: &map[string]string{"key1": "value1"},
							},
						},
						Settings: workload_eviction.EvictionSettings{
							Aggressive: &workload_eviction.EvictionSettingsSettingEnabled{Enabled: true},
						},
					},
				}},
			}, nil
		}).Once()

	result := resource.CreateContext(ctx, data, provider)

	r.Nil(result)
	r.False(result.HasError())
	eac, isOK := data.GetOk(FieldEvictorAdvancedConfig)
	r.True(isOK)
	r.NotNil(eac)
	podSelectorKind, isOK := data.GetOk(fmt.Sprintf("%s.0.%s.0.kind", FieldEvictorAdvancedConfig, FieldPodSelector))
	r.True(isOK)
	r.NotNil(podSelectorKind)
	r.Equal("Job", podSelectorKind)
}

func TestEvictionConfig_CreateContext_ReplicasMin(t *testing.T) {
	r := require.New(t)
	mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)

	ctx := context.Background()
	provider := &ProviderConfig{
		workloadEvictionClient: mockClient,
	}
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	evictionConfigResponse := `{
  "evictionConfig": [
    {
      "podSelector": {
        "kind": "Deployment",
        "labelSelector": {
          "matchLabels": {
            "key1": "value1"
          }
        },
		"replicasMin": 2
      },
      "settings": {
        "aggressive": {
          "enabled": true
        }
      }
    }
  ]
}`

	resource := resourceEvictionConfig()

	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
		FieldEvictorAdvancedConfig: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"pod_selector": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"kind":         cty.StringVal("Deployment"),
					"replicas_min": cty.NumberIntVal(2),
					"match_labels": cty.MapVal(map[string]cty.Value{
						"key1": cty.StringVal("value1"),
					}),
				}),
				}),
				"aggressive": cty.BoolVal(true),
			}),
		}),
	})

	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	mockClient.EXPECT().EvictorAPIUpdateEvictorAdvancedConfigWithBodyWithResponse(mock.Anything, clusterId, "application/json", mock.Anything).
		RunAndReturn(func(ctx context.Context, clusterId string, contentType string, body io.Reader, reqEditors ...workload_eviction.RequestEditorFn) (*workload_eviction.EvictorAPIUpdateEvictorAdvancedConfigResponse, error) {

			got, _ := io.ReadAll(body)
			expected := []byte(evictionConfigResponse)

			eq, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(eq, fmt.Sprintf("got:      %v\n"+
				"expected: %v\n", string(got), string(expected)))

			return &workload_eviction.EvictorAPIUpdateEvictorAdvancedConfigResponse{
				HTTPResponse: &http.Response{StatusCode: 200, Header: map[string][]string{"Content-Type": {"json"}}},
				JSON200: &workload_eviction.AdvancedConfig{EvictionConfig: []workload_eviction.EvictionConfig{
					{
						PodSelector: &workload_eviction.PodSelector{
							Kind:        lo.ToPtr("Deployment"),
							ReplicasMin: lo.ToPtr(int32(2)),
							LabelSelector: &workload_eviction.LabelSelector{
								MatchLabels: &map[string]string{"key1": "value1"},
							},
						},
						Settings: workload_eviction.EvictionSettings{
							Aggressive: &workload_eviction.EvictionSettingsSettingEnabled{Enabled: true},
						},
					},
				}},
			}, nil
		}).Once()

	result := resource.CreateContext(ctx, data, provider)

	r.Nil(result)
	r.False(result.HasError())
	replicasMin, isOK := data.GetOk(fmt.Sprintf("%s.0.%s.0.%s", FieldEvictorAdvancedConfig, FieldPodSelector, FieldPodSelectorReplicasMin))
	r.True(isOK)
	r.Equal(2, replicasMin)
}

func TestEvictionConfig_UpdateContext(t *testing.T) {
	r := require.New(t)
	mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)

	ctx := context.Background()
	provider := &ProviderConfig{
		workloadEvictionClient: mockClient,
	}
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	evictionConfigJson := `{"evictionConfig":[{"podSelector":{"kind":"Job","labelSelector":{"matchLabels":{"key1":"value1"}}},"settings":{"aggressive":{"enabled":true}}},{"nodeSelector":{"labelSelector":{"matchExpressions":[{"key":"key1","operator":"IN","values":["val1","val2"]}]}},"settings":{"disposable":{"enabled":true}}}]}`

	initialConfig := workload_eviction.EvictionConfig{
		Settings: workload_eviction.EvictionSettings{Aggressive: &workload_eviction.EvictionSettingsSettingEnabled{Enabled: true}},
		PodSelector: &workload_eviction.PodSelector{
			Kind: lo.ToPtr("Job"),
			LabelSelector: &workload_eviction.LabelSelector{
				MatchLabels: &map[string]string{
					"key1": "value1",
				}}}}

	newConfig := workload_eviction.EvictionConfig{
		Settings: workload_eviction.EvictionSettings{Disposable: &workload_eviction.EvictionSettingsSettingEnabled{Enabled: true}},
		NodeSelector: &workload_eviction.NodeSelector{
			LabelSelector: workload_eviction.LabelSelector{MatchExpressions: &[]workload_eviction.LabelSelectorExpression{{
				Key:      "key1",
				Operator: workload_eviction.LabelSelectorExpressionOperatorIN,
				Values:   &[]string{"val1", "val2"},
			}}}}}
	finalConfiuration := []workload_eviction.EvictionConfig{initialConfig, newConfig}
	resource := resourceEvictionConfig()

	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
	})

	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	mockClient.EXPECT().
		EvictorAPIGetEvictorAdvancedConfigWithResponse(mock.Anything, clusterId).
		Return(&workload_eviction.EvictorAPIGetEvictorAdvancedConfigResponse{
			HTTPResponse: &http.Response{StatusCode: 200, Header: map[string][]string{"Content-Type": {"json"}}},
			JSON200: &workload_eviction.AdvancedConfig{EvictionConfig: []workload_eviction.EvictionConfig{
				initialConfig,
			}},
		}, nil)

	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())

	mockClient.EXPECT().EvictorAPIUpdateEvictorAdvancedConfigWithBodyWithResponse(mock.Anything, clusterId, "application/json", mock.Anything).
		RunAndReturn(func(ctx context.Context, clusterId string, contentType string, body io.Reader, reqEditors ...workload_eviction.RequestEditorFn) (*workload_eviction.EvictorAPIUpdateEvictorAdvancedConfigResponse, error) {
			got, _ := io.ReadAll(body)
			expected := []byte(evictionConfigJson)

			eq, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(eq, fmt.Sprintf("got:      %v\n"+
				"expected: %v\n", string(got), string(expected)))

			return &workload_eviction.EvictorAPIUpdateEvictorAdvancedConfigResponse{
				HTTPResponse: &http.Response{StatusCode: 200, Header: map[string][]string{"Content-Type": {"json"}}},
				JSON200:      &workload_eviction.AdvancedConfig{EvictionConfig: finalConfiuration},
			}, nil
		}).Once()
	err := data.Set(FieldEvictorAdvancedConfig, flattenEvictionConfig(finalConfiuration))
	r.NoError(err)
	updateResult := resource.UpdateContext(ctx, data, provider)

	r.Nil(updateResult)
	r.False(updateResult.HasError())
	eac := data.Get(FieldEvictorAdvancedConfig)
	r.NotNil(eac)
}

func TestEvictionConfig_DeleteContext(t *testing.T) {
	r := require.New(t)
	mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)

	ctx := context.Background()
	provider := &ProviderConfig{
		workloadEvictionClient: mockClient,
	}
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	resource := resourceEvictionConfig()

	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
		FieldEvictorAdvancedConfig: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"pod_selector": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"match_labels": cty.MapVal(map[string]cty.Value{
						"key1": cty.StringVal("val1"),
					}),
				}),
				}),
				"aggressive": cty.BoolVal(true),
			}),
		}),
	})

	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	mockClient.EXPECT().EvictorAPIUpdateEvictorAdvancedConfigWithBodyWithResponse(mock.Anything, clusterId, "application/json", mock.Anything).
		RunAndReturn(func(ctx context.Context, clusterId string, contentType string, body io.Reader, reqEditors ...workload_eviction.RequestEditorFn) (*workload_eviction.EvictorAPIUpdateEvictorAdvancedConfigResponse, error) {
			return &workload_eviction.EvictorAPIUpdateEvictorAdvancedConfigResponse{
				HTTPResponse: &http.Response{StatusCode: 200, Header: map[string][]string{"Content-Type": {"json"}}},
				JSON200:      &workload_eviction.AdvancedConfig{EvictionConfig: []workload_eviction.EvictionConfig{}},
			}, nil
		}).Once()

	result := resource.DeleteContext(ctx, data, provider)

	r.Nil(result)
	r.False(result.HasError())
	r.Empty(data.Id())
	eac, isOK := data.GetOk(FieldEvictorAdvancedConfig)
	r.False(isOK)
	r.Equal([]interface{}{}, eac)
}

func TestEvictionConfig_ReadContext_Error(t *testing.T) {
	tests := map[string]struct {
		response *workload_eviction.EvictorAPIGetEvictorAdvancedConfigResponse
	}{
		"non-200 status": {
			response: &workload_eviction.EvictorAPIGetEvictorAdvancedConfigResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError, Header: map[string][]string{"Content-Type": {"application/json"}}},
				JSON200:      nil,
			},
		},
		"nil JSON200": {
			response: &workload_eviction.EvictorAPIGetEvictorAdvancedConfigResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}},
				JSON200:      nil,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
			mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)
			provider := &ProviderConfig{workloadEvictionClient: mockClient}
			resource := resourceEvictionConfig()
			initialState := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				FieldClusterId: cty.StringVal(clusterId),
			}), 0)

			mockClient.EXPECT().
				EvictorAPIGetEvictorAdvancedConfigWithResponse(mock.Anything, clusterId).
				Return(tc.response, nil)
			data := resource.Data(initialState)

			result := resource.ReadContext(context.Background(), data, provider)

			r := require.New(t)
			r.NotNil(result)
			r.True(result.HasError())
		})
	}
}

func TestEvictionConfig_ReadContext_NotFound(t *testing.T) {
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)
	provider := &ProviderConfig{workloadEvictionClient: mockClient}
	resource := resourceEvictionConfig()

	initialState := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
	}), 0)
	data := resource.Data(initialState)
	data.SetId(clusterId)

	mockClient.EXPECT().
		EvictorAPIGetEvictorAdvancedConfigWithResponse(mock.Anything, clusterId).
		Return(&workload_eviction.EvictorAPIGetEvictorAdvancedConfigResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusNotFound, Header: map[string][]string{"Content-Type": {"application/json"}}},
			JSON200:      nil,
		}, nil)

	result := resource.ReadContext(context.Background(), data, provider)

	r := require.New(t)
	r.Nil(result)
	r.False(result.HasError())
	r.Empty(data.Id())
}

func TestToPodSelector_MalformedInput(t *testing.T) {
	t.Parallel()

	badInput := []interface{}{"not-a-map"}

	_, err := toPodSelector(badInput)

	r := require.New(t)
	r.Error(err)
	r.Contains(err.Error(), "expecting map[string]interface")
}

func TestToNodeSelector_MalformedInput(t *testing.T) {
	t.Parallel()

	badInput := []interface{}{"not-a-map"}

	_, err := toNodeSelector(badInput)

	r := require.New(t)
	r.Error(err)
	r.Contains(err.Error(), "expecting map[string]interface")
}
