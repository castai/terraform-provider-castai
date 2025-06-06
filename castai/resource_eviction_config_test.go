package castai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestEvictionConfig_ReadContext(t *testing.T) {

	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	resource := resourceEvictionConfig()
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
	})
	initialState := terraform.NewInstanceStateShimmedFromValue(val, 0)

	tests := map[string]struct {
		data     string
		testFunc func(*testing.T, diag.Diagnostics, *schema.ResourceData)
	}{
		"should work with empty config": {
			data: `{"evictionConfig":[]}`,
			testFunc: func(t *testing.T, res diag.Diagnostics, data *schema.ResourceData) {
				r := require.New(t)
				r.Nil(res)
				r.False(res.HasError())
				eac, isOK := data.GetOk(FieldEvictorAdvancedConfig)
				r.False(isOK)
				fmt.Printf("is not %T, %+v", eac, eac)
				d, ok := eac.([]interface{})
				r.True(ok)
				r.Len(d, 0)

			},
		},
		"should read config": {
			data: `{"evictionConfig":[{"podSelector":{"kind":"Job","labelSelector":{"matchLabels":{"key1":"value1"}}},"settings":{"aggressive":{"enabled":true}}}]}`,
			testFunc: func(t *testing.T, res diag.Diagnostics, data *schema.ResourceData) {
				r := require.New(t)
				r.Nil(res)
				r.False(res.HasError())
				eac, isOK := data.GetOk(FieldEvictorAdvancedConfig)
				r.True(isOK)
				r.NotNil(eac)
				podSelectorKind, isOK := data.GetOk(fmt.Sprintf("%s.0.%s.0.kind", FieldEvictorAdvancedConfig, FieldPodSelector))
				r.True(isOK)
				r.NotNil(podSelectorKind)
				r.Equal("Job", podSelectorKind)
				podSelectorLabelValue, isOK := data.GetOk(fmt.Sprintf("%s.0.%s.0.%s.key1", FieldEvictorAdvancedConfig, FieldPodSelector, FieldMatchLabels))
				r.True(isOK)
				r.NotNil(podSelectorLabelValue)
				r.Equal("value1", podSelectorLabelValue)
			},
		},
		"should handle multiple evictionConfig objects": {
			data: `{"evictionConfig":[
					{"podSelector":{"kind":"Job","labelSelector":{"matchLabels":{"key1":"value1"}}},"settings":{"aggressive":{"enabled":true}}}, 
					{"nodeSelector":{"labelSelector":{"matchLabels":{"node-label":"value1"}}},"settings":{"disposable":{"enabled":true}}}]}`,
			testFunc: func(t *testing.T, res diag.Diagnostics, data *schema.ResourceData) {
				r := require.New(t)
				r.Nil(res)
				r.False(res.HasError())
				eac, isOK := data.GetOk(FieldEvictorAdvancedConfig)
				r.True(isOK)
				r.NotNil(eac)
				podSelectorKind, isOK := data.GetOk(fmt.Sprintf("%s.0.%s.0.kind", FieldEvictorAdvancedConfig, FieldPodSelector))
				r.True(isOK)
				r.NotNil(podSelectorKind)
				r.Equal("Job", podSelectorKind)
				podSelectorLabelValue, isOK := data.GetOk(fmt.Sprintf("%s.0.%s.0.%s.key1", FieldEvictorAdvancedConfig, FieldPodSelector, FieldMatchLabels))
				r.True(isOK)
				r.NotNil(podSelectorLabelValue)
				r.Equal("value1", podSelectorLabelValue)
				nodeSelectorLabelValue, isOK := data.GetOk(fmt.Sprintf("%s.1.%s.0.%s.node-label", FieldEvictorAdvancedConfig, FieldNodeSelector, FieldMatchLabels))
				r.True(isOK)
				r.NotNil(nodeSelectorLabelValue)
				r.Equal("value1", nodeSelectorLabelValue)
			},
		},
		"should handle label expressions": {
			data: `{"evictionConfig":[ {"podSelector":{"kind":"Job","labelSelector":{"matchExpressions":[{"key":"value1", "operator":"In", "values":["v1", "v2"]}]}},"settings":{"aggressive":{"enabled":true}}} ]}`,
			testFunc: func(t *testing.T, res diag.Diagnostics, data *schema.ResourceData) {
				r := require.New(t)
				r.Nil(res)
				r.False(res.HasError())
				eac, isOK := data.GetOk(FieldEvictorAdvancedConfig)
				r.True(isOK)
				r.NotNil(eac)
				podSelectorKeyValue, isOK := data.GetOk(fmt.Sprintf("%s.0.%s.0.%s.0.key", FieldEvictorAdvancedConfig, FieldPodSelector, FieldMatchExpressions))
				r.True(isOK)
				r.NotNil(podSelectorKeyValue)
				r.Equal("value1", podSelectorKeyValue)
				podSelectorValues, isOK := data.GetOk(fmt.Sprintf("%s.0.%s.0.%s.0.values", FieldEvictorAdvancedConfig, FieldPodSelector, FieldMatchExpressions))
				r.True(isOK)
				r.NotNil(podSelectorValues)
				r.Equal([]interface{}{"v1", "v2"}, podSelectorValues)
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			body := io.NopCloser(bytes.NewReader([]byte(test.data)))

			mockClient.EXPECT().
				EvictorAPIGetAdvancedConfig(gomock.Any(), clusterId).
				Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
			data := resource.Data(initialState)

			result := resource.ReadContext(ctx, data, provider)
			test.testFunc(t, result, data)

		})
	}

}

func TestEvictionConfig_CreateContext(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	evictionConfigResponse := `{
  "evictionConfig": [
    {
      "podSelector": {
        "kind": "Job",
        "labelSelector": {
          "matchLabels": {
            "key1": "value1"
          }
        },
		"replicasMin": null
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

	mockClient.EXPECT().EvictorAPIUpsertAdvancedConfigWithBody(gomock.Any(), clusterId, "application/json", gomock.Any()).
		DoAndReturn(func(ctx context.Context, clusterId string, contentType string, body io.Reader) (*http.Response, error) {

			got, _ := io.ReadAll(body)
			expected := []byte(evictionConfigResponse)

			eq, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(eq, fmt.Sprintf("got:      %v\n"+
				"expected: %v\n", string(got), string(expected)))

			return &http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(evictionConfigResponse))),
			}, nil
		}).Times(1)

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

func TestEvictionConfig_UpdateContext(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	initialConfigJson := `
		{
  "evictionConfig": [
    {
      "podSelector": {
		"kind": "Job",
        "labelSelector": {
          "matchLabels": {
            "key1":     "value1"
          }
        }
      },
		"settings": {
            "aggressive": {
              "enabled": true
            }
          }
    }
  ]
}`
	evictionConfigJson := `
		{
  "evictionConfig": [
    {
      "podSelector": {
        "kind": "Job",
        "labelSelector": {
          "matchLabels": {
            "key1": "value1"
          }
        },
        "replicasMin": null
      },
      "settings": {
        "aggressive": {
          "enabled": true
        }
      }
    },
    {
      "nodeSelector": {
  "labelSelector": {
        "matchExpressions": [
          {
            "key": "key1",
            "operator": "In",
            "values": [
              "val1",
              "val2"
            ]
          }
        ]}
      },
      "settings": {
        "disposable": {
          "enabled": true
        }
      }
    }
  ]
}`

	initialConfig := sdk.CastaiEvictorV1EvictionConfig{
		Settings: sdk.CastaiEvictorV1EvictionSettings{Aggressive: &sdk.CastaiEvictorV1EvictionSettingsSettingEnabled{Enabled: true}},
		PodSelector: &sdk.CastaiEvictorV1PodSelector{
			Kind: lo.ToPtr("Job"),
			LabelSelector: &sdk.CastaiEvictorV1LabelSelector{
				MatchLabels: &map[string]string{
					"key1": "value1",
				}}}}

	newConfig := sdk.CastaiEvictorV1EvictionConfig{
		Settings: sdk.CastaiEvictorV1EvictionSettings{Disposable: &sdk.CastaiEvictorV1EvictionSettingsSettingEnabled{Enabled: true}},
		NodeSelector: &sdk.CastaiEvictorV1NodeSelector{
			LabelSelector: sdk.CastaiEvictorV1LabelSelector{MatchExpressions: &[]sdk.CastaiEvictorV1LabelSelectorExpression{{
				Key:      "key1",
				Operator: "In",
				Values:   &[]string{"val1", "val2"},
			}}}}}
	finalConfiuration := []sdk.CastaiEvictorV1EvictionConfig{initialConfig, newConfig}
	resource := resourceEvictionConfig()

	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
	})

	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	body := io.NopCloser(bytes.NewReader([]byte(initialConfigJson)))
	mockClient.EXPECT().
		EvictorAPIGetAdvancedConfig(gomock.Any(), clusterId).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())

	mockClient.EXPECT().EvictorAPIUpsertAdvancedConfigWithBody(gomock.Any(), clusterId, "application/json", gomock.Any()).
		DoAndReturn(func(ctx context.Context, clusterId string, contentType string, body io.Reader) (*http.Response, error) {
			got, _ := io.ReadAll(body)
			expected := []byte(evictionConfigJson)

			eq, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(eq, fmt.Sprintf("got:      %v\n"+
				"expected: %v\n", string(got), string(expected)))

			return &http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(evictionConfigJson))),
			}, nil
		}).Times(1)
	err := data.Set(FieldEvictorAdvancedConfig, flattenEvictionConfig(finalConfiuration))
	r.NoError(err)
	updateResult := resource.UpdateContext(ctx, data, provider)

	r.Nil(updateResult)
	r.False(result.HasError())
	eac, isOK := data.GetOk(FieldEvictorAdvancedConfig)
	r.True(isOK)
	r.NotNil(eac)
}

func TestEvictionConfig_DeleteContext(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	evictionConfigJson := `{"evictionConfig": []}`

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

	mockClient.EXPECT().EvictorAPIUpsertAdvancedConfigWithBody(gomock.Any(), clusterId, "application/json", gomock.Any()).
		DoAndReturn(func(ctx context.Context, clusterId string, contentType string, body io.Reader) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(evictionConfigJson))),
			}, nil
		}).Times(1)

	result := resource.DeleteContext(ctx, data, provider)

	r.Nil(result)
	r.False(result.HasError())
	eac, isOK := data.GetOk(FieldEvictorAdvancedConfig)
	r.False(isOK)
	r.Equal([]interface{}{}, eac)
}
