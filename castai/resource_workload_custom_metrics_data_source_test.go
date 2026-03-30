package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestWorkloadCustomMetricsDataSource_CreateWithPresets(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	res := resourceWorkloadCustomMetricsDataSource()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gk1"
	dsID := "ds-123"

	// GIVEN
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterID: cty.StringVal(clusterID),
		"name":         cty.StringVal("my-prometheus"),
		"prometheus": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"url":     cty.StringVal("http://prometheus:9090"),
				"timeout": cty.StringVal("30s"),
				"presets": cty.ListVal([]cty.Value{cty.StringVal("jvm")}),
				"metric":  cty.SetValEmpty(cty.Object(map[string]cty.Type{"name": cty.String, "query": cty.String})),
			}),
		}),
		"status":             cty.StringVal(""),
		"kube_resource_name": cty.StringVal(""),
		"managed_by_cast":    cty.False,
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := res.Data(state)

	createResponse := sdk.WorkloadoptimizationV1CustomMetricsDataSource{
		Id:               dsID,
		ClusterId:        clusterID,
		Name:             "my-prometheus",
		Type:             sdk.PROMETHEUS,
		Status:           sdk.WorkloadoptimizationV1CustomMetricsDataSourceStatusCONNECTING,
		KubeResourceName: "my-prometheus",
		ManagedByCast:    true,
		Data: sdk.WorkloadoptimizationV1CustomMetricsDataSourceData{
			Prometheus: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus{
				DataSource: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusDataSource{
					Url:     "http://prometheus:9090",
					Timeout: toPtr("30s"),
				},
				Metrics: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetrics{
					Presets: &[]string{"jvm"},
				},
			},
		},
	}

	createBody, _ := json.Marshal(createResponse)
	mockClient.EXPECT().
		WorkloadOptimizationAPICreateCustomMetricsDataSource(ctx, clusterID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(createBody)),
		}, nil)

	listResponse := sdk.WorkloadoptimizationV1ListCustomMetricsDataSourcesResponse{
		Items: []sdk.WorkloadoptimizationV1CustomMetricsDataSource{createResponse},
	}
	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		WorkloadOptimizationAPIListCustomMetricsDataSources(ctx, clusterID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	// WHEN
	diags := resourceWorkloadCustomMetricsDataSourceCreate(ctx, data, provider)

	// THEN
	r.Empty(diags)
	r.Equal(dsID, data.Id())
	r.Equal("my-prometheus", data.Get("name"))
	r.Equal("CONNECTING", data.Get("status"))
	r.Equal("my-prometheus", data.Get("kube_resource_name"))
	r.Equal(true, data.Get("managed_by_cast"))
}

func TestWorkloadCustomMetricsDataSource_CreateWithManualMetrics(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	res := resourceWorkloadCustomMetricsDataSource()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gk1"
	dsID := "ds-456"

	// GIVEN
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterID: cty.StringVal(clusterID),
		"name":         cty.StringVal("custom-prom"),
		"prometheus": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"url":     cty.StringVal("http://prometheus:9090"),
				"timeout": cty.StringVal(""),
				"presets": cty.ListValEmpty(cty.String),
				"metric": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name":  cty.StringVal("http_requests_total"),
						"query": cty.StringVal("sum(rate(http_requests_total[5m])) by (pod)"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"name":  cty.StringVal("queue_depth"),
						"query": cty.StringVal("avg(queue_depth) by (pod)"),
					}),
				}),
			}),
		}),
		"status":             cty.StringVal(""),
		"kube_resource_name": cty.StringVal(""),
		"managed_by_cast":    cty.False,
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := res.Data(state)

	createResponse := sdk.WorkloadoptimizationV1CustomMetricsDataSource{
		Id:               dsID,
		ClusterId:        clusterID,
		Name:             "custom-prom",
		Type:             sdk.PROMETHEUS,
		Status:           sdk.WorkloadoptimizationV1CustomMetricsDataSourceStatusCONNECTING,
		KubeResourceName: "custom-prom",
		ManagedByCast:    true,
		Data: sdk.WorkloadoptimizationV1CustomMetricsDataSourceData{
			Prometheus: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus{
				DataSource: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusDataSource{
					Url: "http://prometheus:9090",
				},
				Metrics: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetrics{
					Resolved: &[]sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetric{
						{Name: "http_requests_total", Queries: []sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQuery{
							{Value: "sum(rate(http_requests_total[5m])) by (pod)", Origin: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQueryOriginMANUAL},
						}},
						{Name: "queue_depth", Queries: []sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQuery{
							{Value: "avg(queue_depth) by (pod)", Origin: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQueryOriginMANUAL},
						}},
					},
				},
			},
		},
	}

	createBody, _ := json.Marshal(createResponse)
	mockClient.EXPECT().
		WorkloadOptimizationAPICreateCustomMetricsDataSource(ctx, clusterID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(createBody)),
		}, nil)

	listResponse := sdk.WorkloadoptimizationV1ListCustomMetricsDataSourcesResponse{
		Items: []sdk.WorkloadoptimizationV1CustomMetricsDataSource{createResponse},
	}
	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		WorkloadOptimizationAPIListCustomMetricsDataSources(ctx, clusterID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	// WHEN
	diags := resourceWorkloadCustomMetricsDataSourceCreate(ctx, data, provider)

	// THEN
	r.Empty(diags)
	r.Equal(dsID, data.Id())
	r.Equal("custom-prom", data.Get("name"))

	promList := data.Get("prometheus").([]interface{})
	r.Len(promList, 1)
	promMap := promList[0].(map[string]interface{})
	r.Equal("http://prometheus:9090", promMap["url"])

	metricSet := promMap["metric"].(*schema.Set)
	r.Equal(2, metricSet.Len())

	metricList := metricSet.List()
	names := map[string]bool{}
	for _, m := range metricList {
		names[m.(map[string]interface{})["name"].(string)] = true
	}
	r.True(names["http_requests_total"])
	r.True(names["queue_depth"])
}

func TestWorkloadCustomMetricsDataSource_ReadNotFound(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	res := resourceWorkloadCustomMetricsDataSource()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gk1"
	dsID := "ds-not-exist"

	// GIVEN
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterID: cty.StringVal(clusterID),
		"name":         cty.StringVal("test"),
		"prometheus": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"url":     cty.StringVal("http://prometheus:9090"),
				"timeout": cty.StringVal(""),
				"presets": cty.ListValEmpty(cty.String),
				"metric":  cty.SetValEmpty(cty.Object(map[string]cty.Type{"name": cty.String, "query": cty.String})),
			}),
		}),
		"status":             cty.StringVal(""),
		"kube_resource_name": cty.StringVal(""),
		"managed_by_cast":    cty.False,
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = dsID
	data := res.Data(state)

	listResponse := sdk.WorkloadoptimizationV1ListCustomMetricsDataSourcesResponse{
		Items: []sdk.WorkloadoptimizationV1CustomMetricsDataSource{},
	}
	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		WorkloadOptimizationAPIListCustomMetricsDataSources(ctx, clusterID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	// WHEN
	diags := resourceWorkloadCustomMetricsDataSourceRead(ctx, data, provider)

	// THEN
	r.Empty(diags)
	r.Empty(data.Id())
}

func TestWorkloadCustomMetricsDataSource_Update(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	res := resourceWorkloadCustomMetricsDataSource()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gk1"
	dsID := "ds-123"

	// GIVEN
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterID: cty.StringVal(clusterID),
		"name":         cty.StringVal("updated-name"),
		"prometheus": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"url":     cty.StringVal("http://new-prometheus:9090"),
				"timeout": cty.StringVal("60s"),
				"presets": cty.ListVal([]cty.Value{cty.StringVal("jvm")}),
				"metric":  cty.SetValEmpty(cty.Object(map[string]cty.Type{"name": cty.String, "query": cty.String})),
			}),
		}),
		"status":             cty.StringVal(""),
		"kube_resource_name": cty.StringVal(""),
		"managed_by_cast":    cty.False,
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = dsID
	data := res.Data(state)

	updatedDS := sdk.WorkloadoptimizationV1CustomMetricsDataSource{
		Id:               dsID,
		ClusterId:        clusterID,
		Name:             "updated-name",
		Type:             sdk.PROMETHEUS,
		Status:           sdk.WorkloadoptimizationV1CustomMetricsDataSourceStatusCONNECTED,
		KubeResourceName: "my-prometheus",
		ManagedByCast:    true,
		Data: sdk.WorkloadoptimizationV1CustomMetricsDataSourceData{
			Prometheus: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus{
				DataSource: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusDataSource{
					Url:     "http://new-prometheus:9090",
					Timeout: toPtr("60s"),
				},
				Metrics: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetrics{
					Presets: &[]string{"jvm"},
				},
			},
		},
	}

	updateBody, _ := json.Marshal(updatedDS)
	mockClient.EXPECT().
		WorkloadOptimizationAPIUpdateCustomMetricsDataSource(ctx, clusterID, dsID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(updateBody)),
		}, nil)

	listResponse := sdk.WorkloadoptimizationV1ListCustomMetricsDataSourcesResponse{
		Items: []sdk.WorkloadoptimizationV1CustomMetricsDataSource{updatedDS},
	}
	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		WorkloadOptimizationAPIListCustomMetricsDataSources(ctx, clusterID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	// WHEN
	diags := resourceWorkloadCustomMetricsDataSourceUpdate(ctx, data, provider)

	// THEN
	r.Empty(diags)
	r.Equal(dsID, data.Id())
	r.Equal("updated-name", data.Get("name"))
	r.Equal("CONNECTED", data.Get("status"))
}

func TestWorkloadCustomMetricsDataSource_Delete(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	res := resourceWorkloadCustomMetricsDataSource()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gk1"
	dsID := "ds-123"

	// GIVEN
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterID: cty.StringVal(clusterID),
		"name":         cty.StringVal("test"),
		"prometheus": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"url":     cty.StringVal("http://prometheus:9090"),
				"timeout": cty.StringVal(""),
				"presets": cty.ListValEmpty(cty.String),
				"metric":  cty.SetValEmpty(cty.Object(map[string]cty.Type{"name": cty.String, "query": cty.String})),
			}),
		}),
		"status":             cty.StringVal(""),
		"kube_resource_name": cty.StringVal(""),
		"managed_by_cast":    cty.False,
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = dsID
	data := res.Data(state)

	deleteBody, _ := json.Marshal(map[string]interface{}{})
	mockClient.EXPECT().
		WorkloadOptimizationAPIDeleteCustomMetricsDataSource(ctx, clusterID, dsID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(deleteBody)),
		}, nil)

	// WHEN
	diags := resourceWorkloadCustomMetricsDataSourceDelete(ctx, data, provider)

	// THEN
	r.Empty(diags)
}

func TestWorkloadCustomMetricsDataSource_DeleteNotFound(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	res := resourceWorkloadCustomMetricsDataSource()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gk1"
	dsID := "ds-gone"

	// GIVEN
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterID: cty.StringVal(clusterID),
		"name":         cty.StringVal("test"),
		"prometheus": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"url":     cty.StringVal("http://prometheus:9090"),
				"timeout": cty.StringVal(""),
				"presets": cty.ListValEmpty(cty.String),
				"metric":  cty.SetValEmpty(cty.Object(map[string]cty.Type{"name": cty.String, "query": cty.String})),
			}),
		}),
		"status":             cty.StringVal(""),
		"kube_resource_name": cty.StringVal(""),
		"managed_by_cast":    cty.False,
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = dsID
	data := res.Data(state)

	mockClient.EXPECT().
		WorkloadOptimizationAPIDeleteCustomMetricsDataSource(ctx, clusterID, dsID).
		Return(&http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"message":"not found"}`))),
		}, nil)

	// WHEN
	diags := resourceWorkloadCustomMetricsDataSourceDelete(ctx, data, provider)

	// THEN
	r.Empty(diags)
}

func TestFlattenPrometheusConfig(t *testing.T) {
	tests := map[string]struct {
		prom            *sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus
		expectedURL     string
		expectedTimeout string
		expectedPresets []string
		expectedMetric  []interface{} // nil means metric key should be nil
	}{
		"minimal config with nil metrics": {
			prom: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus{
				DataSource: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusDataSource{
					Url: "http://prometheus:9090",
				},
			},
			expectedURL:     "http://prometheus:9090",
			expectedTimeout: "",
			expectedPresets: []string{},
			expectedMetric:  nil,
		},
		"with timeout": {
			prom: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus{
				DataSource: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusDataSource{
					Url:     "http://prometheus:9090",
					Timeout: toPtr("30s"),
				},
			},
			expectedURL:     "http://prometheus:9090",
			expectedTimeout: "30s",
			expectedPresets: []string{},
			expectedMetric:  nil,
		},
		"with presets": {
			prom: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus{
				DataSource: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusDataSource{
					Url: "http://prometheus:9090",
				},
				Metrics: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetrics{
					Presets: &[]string{"jvm", "kafka"},
				},
			},
			expectedURL:     "http://prometheus:9090",
			expectedTimeout: "",
			expectedPresets: []string{"jvm", "kafka"},
			expectedMetric:  nil,
		},
		"resolved with only preset metrics": {
			prom: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus{
				DataSource: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusDataSource{
					Url: "http://prometheus:9090",
				},
				Metrics: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetrics{
					Presets: &[]string{"jvm"},
					Resolved: &[]sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetric{
						{Name: "jvm_threads", Queries: []sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQuery{
							{Value: "jvm_threads_current", Origin: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQueryOriginPRESET},
						}},
					},
				},
			},
			expectedURL:     "http://prometheus:9090",
			expectedTimeout: "",
			expectedPresets: []string{"jvm"},
			expectedMetric:  nil,
		},
		"resolved with empty list": {
			prom: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus{
				DataSource: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusDataSource{
					Url: "http://prometheus:9090",
				},
				Metrics: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetrics{
					Resolved: &[]sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetric{},
				},
			},
			expectedURL:     "http://prometheus:9090",
			expectedTimeout: "",
			expectedPresets: []string{},
			expectedMetric:  nil,
		},
		"resolved with manual metrics": {
			prom: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus{
				DataSource: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusDataSource{
					Url:     "http://prometheus:9090",
					Timeout: toPtr("15s"),
				},
				Metrics: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetrics{
					Resolved: &[]sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetric{
						{Name: "http_requests_total", Queries: []sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQuery{
							{Value: "sum(rate(http_requests_total[5m])) by (pod)", Origin: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQueryOriginMANUAL},
						}},
						{Name: "queue_depth", Queries: []sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQuery{
							{Value: "avg(queue_depth) by (pod)", Origin: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQueryOriginMANUAL},
						}},
					},
				},
			},
			expectedURL:     "http://prometheus:9090",
			expectedTimeout: "15s",
			expectedPresets: []string{},
			expectedMetric: []interface{}{
				map[string]interface{}{"name": "http_requests_total", "query": "sum(rate(http_requests_total[5m])) by (pod)"},
				map[string]interface{}{"name": "queue_depth", "query": "avg(queue_depth) by (pod)"},
			},
		},
		"resolved with mixed preset and manual metrics": {
			prom: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus{
				DataSource: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusDataSource{
					Url: "http://prometheus:9090",
				},
				Metrics: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetrics{
					Presets: &[]string{"jvm"},
					Resolved: &[]sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetric{
						{Name: "jvm_threads", Queries: []sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQuery{
							{Value: "jvm_threads_current", Origin: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQueryOriginPRESET},
						}},
						{Name: "custom_metric", Queries: []sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQuery{
							{Value: "my_custom_query", Origin: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQueryOriginMANUAL},
						}},
					},
				},
			},
			expectedURL:     "http://prometheus:9090",
			expectedTimeout: "",
			expectedPresets: []string{"jvm"},
			expectedMetric: []interface{}{
				map[string]interface{}{"name": "custom_metric", "query": "my_custom_query"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)

			result := flattenPrometheusConfig(tc.prom)
			r.Len(result, 1)

			promMap := result[0].(map[string]interface{})
			r.Equal(tc.expectedURL, promMap["url"])
			r.Equal(tc.expectedTimeout, promMap["timeout"])
			r.Equal(tc.expectedPresets, promMap["presets"])

			if tc.expectedMetric == nil {
				r.Nil(promMap["metric"])
			} else {
				r.Equal(tc.expectedMetric, promMap["metric"])
			}
		})
	}
}

func TestWorkloadCustomMetricsDataSource_Importer(t *testing.T) {
	type testCase struct {
		importID          string
		expectError       bool
		expectedClusterID string
		expectedID        string
	}

	tests := map[string]testCase{
		"valid import ID": {
			importID:          "b6bfc074-a267-400f-b8f1-db0850c36gk1/ds-123",
			expectedClusterID: "b6bfc074-a267-400f-b8f1-db0850c36gk1",
			expectedID:        "ds-123",
		},
		"missing separator": {
			importID:    "b6bfc074-a267-400f-b8f1-db0850c36gk1",
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			res := resourceWorkloadCustomMetricsDataSource()

			// GIVEN
			state := &terraform.InstanceState{ID: tc.importID}
			data := res.Data(state)

			// WHEN
			result, err := workloadCustomMetricsDataSourceImporter(context.Background(), data, nil)

			// THEN
			if tc.expectError {
				r.Error(err)
				return
			}
			r.NoError(err)
			r.Len(result, 1)
			r.Equal(tc.expectedID, result[0].Id())
			r.Equal(tc.expectedClusterID, result[0].Get(FieldClusterID))
		})
	}
}

func TestWorkloadCustomMetricsDataSource_ImportReadWithManualMetrics(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	res := resourceWorkloadCustomMetricsDataSource()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gk1"
	dsID := "ds-456"

	// GIVEN — simulate post-import state: cluster_id is set but no prometheus config in state.
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterID: cty.StringVal(clusterID),
		"name":         cty.StringVal(""),
		"prometheus": cty.ListValEmpty(cty.Object(map[string]cty.Type{
			"url":     cty.String,
			"timeout": cty.String,
			"presets": cty.List(cty.String),
			"metric": cty.Set(cty.Object(map[string]cty.Type{
				"name":  cty.String,
				"query": cty.String,
			})),
		})),
		"status":             cty.StringVal(""),
		"kube_resource_name": cty.StringVal(""),
		"managed_by_cast":    cty.False,
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = dsID
	data := res.Data(state)

	listResponse := sdk.WorkloadoptimizationV1ListCustomMetricsDataSourcesResponse{
		Items: []sdk.WorkloadoptimizationV1CustomMetricsDataSource{
			{
				Id:               dsID,
				ClusterId:        clusterID,
				Name:             "custom-prom",
				Type:             sdk.PROMETHEUS,
				Status:           sdk.WorkloadoptimizationV1CustomMetricsDataSourceStatusCONNECTED,
				KubeResourceName: "custom-prom",
				ManagedByCast:    true,
				Data: sdk.WorkloadoptimizationV1CustomMetricsDataSourceData{
					Prometheus: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus{
						DataSource: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusDataSource{
							Url: "http://prometheus:9090",
						},
						Metrics: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetrics{
							Resolved: &[]sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetric{
								{Name: "http_requests_total", Queries: []sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQuery{
									{Value: "sum(rate(http_requests_total[5m])) by (pod)", Origin: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQueryOriginMANUAL},
								}},
								{Name: "queue_depth", Queries: []sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQuery{
									{Value: "avg(queue_depth) by (pod)", Origin: sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQueryOriginMANUAL},
								}},
							},
						},
					},
				},
			},
		},
	}
	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		WorkloadOptimizationAPIListCustomMetricsDataSources(ctx, clusterID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	// WHEN
	diags := resourceWorkloadCustomMetricsDataSourceRead(ctx, data, provider)

	// THEN
	r.Empty(diags)
	r.Equal("custom-prom", data.Get("name"))

	promList := data.Get("prometheus").([]interface{})
	r.Len(promList, 1)
	promMap := promList[0].(map[string]interface{})
	r.Equal("http://prometheus:9090", promMap["url"])

	metricSet := promMap["metric"].(*schema.Set)
	r.Equal(2, metricSet.Len())

	metricsByName := map[string]string{}
	for _, m := range metricSet.List() {
		mm := m.(map[string]interface{})
		metricsByName[mm["name"].(string)] = mm["query"].(string)
	}
	r.Equal("sum(rate(http_requests_total[5m])) by (pod)", metricsByName["http_requests_total"])
	r.Equal("avg(queue_depth) by (pod)", metricsByName["queue_depth"])
}

