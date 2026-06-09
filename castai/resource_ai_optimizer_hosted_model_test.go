package castai

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/ai_optimizer"
	mock_ai_optimizer "github.com/castai/terraform-provider-castai/castai/sdk/ai_optimizer/mock"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func newHostedModelProvider(ctrl *gomock.Controller) (*mock_sdk.MockClientInterface, *mock_ai_optimizer.MockClientWithResponsesInterface, *ProviderConfig) {
	mockSDK := mock_sdk.NewMockClientInterface(ctrl)
	mockAIClient := mock_ai_optimizer.NewMockClientWithResponsesInterface(ctrl)
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockSDK,
		},
		aiOptimizerClient: mockAIClient,
		organizationID:    "org-1",
	}
	return mockSDK, mockAIClient, provider
}

func TestExpandFlattenVllmConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input *ai_optimizer.VLLMConfig
	}{
		"secret name only": {
			input: &ai_optimizer.VLLMConfig{SecretName: toPtr("my-secret")},
		},
		"hf token only": {
			input: &ai_optimizer.VLLMConfig{HuggingFaceToken: toPtr("hf-tok")},
		},
		"both fields": {
			input: &ai_optimizer.VLLMConfig{SecretName: toPtr("s"), HuggingFaceToken: toPtr("t")},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)
			flat := toAnySlice(flattenVllmConfig(tc.input))
			result := expandVllmConfig(flat)
			r.Equal(tc.input, result)
		})
	}
}

func TestExpandFlattenHorizontalAutoscaling(t *testing.T) {
	t.Parallel()

	enabled := true
	tests := map[string]struct {
		input *ai_optimizer.HorizontalAutoscaling
	}{
		"enabled with all fields": {
			input: &ai_optimizer.HorizontalAutoscaling{
				Enabled:      &enabled,
				MinReplicas:  1,
				MaxReplicas:  5,
				TargetMetric: ai_optimizer.HorizontalAutoscalingTargetMetricGPUCACHEUSAGEPERCENTAGE,
				TargetValue:  0.8,
			},
		},
		"no enabled flag": {
			input: &ai_optimizer.HorizontalAutoscaling{
				MinReplicas:  2,
				MaxReplicas:  10,
				TargetMetric: ai_optimizer.HorizontalAutoscalingTargetMetricNUMBEROFREQUESTSWAITING,
				TargetValue:  0.5,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)
			flat := toAnySlice(flattenHorizontalAutoscaling(tc.input))
			result := expandHorizontalAutoscaling(flat)
			r.Equal(tc.input, result)
		})
	}
}

func TestExpandFlattenHibernation(t *testing.T) {
	t.Parallel()

	enabled := true
	rc := uint32(10)
	tests := map[string]struct {
		input *ai_optimizer.Hibernation
	}{
		"enabled with request count": {
			input: &ai_optimizer.Hibernation{
				Enabled: &enabled,
				ResumeCondition: ai_optimizer.HibernationCondition{
					Duration:     "5m",
					RequestCount: &rc,
				},
				HibernateCondition: ai_optimizer.HibernationCondition{
					Duration:     "10m",
					RequestCount: &rc,
				},
			},
		},
		"no request count": {
			input: &ai_optimizer.Hibernation{
				ResumeCondition:    ai_optimizer.HibernationCondition{Duration: "1h"},
				HibernateCondition: ai_optimizer.HibernationCondition{Duration: "2h"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)
			flat := toAnySlice(flattenHibernation(tc.input))
			result := expandHibernation(flat)
			r.Equal(tc.input, result)
		})
	}
}

func TestExpandFlattenFallback(t *testing.T) {
	t.Parallel()

	enabled := true
	tests := map[string]struct {
		input *ai_optimizer.Fallback
	}{
		"all fields": {
			input: &ai_optimizer.Fallback{
				Enabled:    &enabled,
				ProviderId: toPtr("provider-1"),
				Model:      toPtr("gpt-4"),
			},
		},
		"enabled only": {
			input: &ai_optimizer.Fallback{Enabled: &enabled},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)
			flat := toAnySlice(flattenFallback(tc.input))
			result := expandFallback(flat)
			r.Equal(tc.input, result)
		})
	}
}

func TestExpandFlattenEdgeLocations(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    *ai_optimizer.EdgeLocations
		expected []string
	}{
		"with ids": {
			input:    &ai_optimizer.EdgeLocations{Ids: &[]string{"loc-1", "loc-2"}},
			expected: []string{"loc-1", "loc-2"},
		},
		"nil": {
			input:    nil,
			expected: nil,
		},
		"empty ids": {
			input:    &ai_optimizer.EdgeLocations{Ids: &[]string{}},
			expected: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)
			flat := flattenEdgeLocations(tc.input)
			r.Equal(tc.expected, flat)
			if flat != nil {
				raw := make([]interface{}, len(flat))
				for i, v := range flat {
					raw[i] = v
				}
				result := expandEdgeLocations(raw)
				if len(flat) == 0 {
					r.Nil(result)
				} else {
					r.Equal(tc.input, result)
				}
			}
		})
	}
}

func toAnySlice(in []map[string]interface{}) []interface{} {
	out := make([]interface{}, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

func TestAIHostedModelCreate(t *testing.T) {
	t.Parallel()

	modelID := "model-new-1"
	clusterID := "cluster-xyz"

	tests := map[string]struct {
		service        string
		port           int
		expectedStatus string
	}{
		"basic create": {
			service:        "llama-svc",
			port:           8080,
			expectedStatus: "DEPLOYING",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := require.New(t)
			ctrl := gomock.NewController(t)
			_, mockAIClient, provider := newHostedModelProvider(ctrl)

			status := ai_optimizer.HostedModelStatus(tc.expectedStatus)
			created := ai_optimizer.HostedModel{
				Id:           &modelID,
				ClusterId:    clusterID,
				ModelSpecsId: "specs-1",
				Service:      tc.service,
				Port:         int32(tc.port),
				Status:       &status,
			}

			mockAIClient.EXPECT().
				HostedModelsAPICreateHostedModelWithResponse(gomock.Any(), "org-1", clusterID, gomock.Any()).
				Return(&ai_optimizer.HostedModelsAPICreateHostedModelResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200:      &created,
				}, nil)

			mockAIClient.EXPECT().
				HostedModelsAPIListHostedModelsWithResponse(gomock.Any(), "org-1", clusterID, gomock.Any()).
				Return(&ai_optimizer.HostedModelsAPIListHostedModelsResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200: &ai_optimizer.ListHostedModelsResponse{
						Items:      []ai_optimizer.HostedModel{created},
						TotalCount: 1,
					},
				}, nil)

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			res := resourceAIHostedModel()
			data := res.Data(state)
			_ = data.Set(fieldAIHostedModelClusterID, clusterID)
			_ = data.Set(fieldAIHostedModelModelSpecsID, "specs-1")
			_ = data.Set(fieldAIHostedModelService, tc.service)
			_ = data.Set(fieldAIHostedModelPort, tc.port)

			result := res.CreateContext(context.Background(), data, provider)
			r.Nil(result)
			r.Equal(modelID, data.Id())
			r.Equal(tc.expectedStatus, data.Get(fieldAIHostedModelStatus).(string))
		})
	}
}

func TestAIHostedModelRead(t *testing.T) {
	t.Parallel()

	modelID := "model-abc"
	clusterID := "cluster-xyz"

	status := ai_optimizer.HostedModelStatus("RUNNING")
	replicas := int32(2)
	cloudProvider := "AWS"
	namespace := "ai-models"
	statusReason := "all replicas healthy"

	tests := map[string]struct {
		statusCode     int
		items          []ai_optimizer.HostedModel
		expectRemoved  bool
		expectedStatus string
	}{
		"found in list": {
			statusCode: 200,
			items: []ai_optimizer.HostedModel{
				{
					Id:              &modelID,
					ClusterId:       clusterID,
					ModelSpecsId:    "specs-1",
					Service:         "llama",
					Port:            8080,
					Status:          &status,
					StatusReason:    &statusReason,
					CurrentReplicas: &replicas,
					CloudProvider:   &cloudProvider,
					Namespace:       &namespace,
				},
			},
			expectedStatus: "RUNNING",
		},
		"not found removes from state": {
			statusCode:    200,
			items:         []ai_optimizer.HostedModel{},
			expectRemoved: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := require.New(t)
			ctrl := gomock.NewController(t)
			_, mockAIClient, provider := newHostedModelProvider(ctrl)

			ctx := context.Background()

			listResp := &ai_optimizer.HostedModelsAPIListHostedModelsResponse{
				Body:         []byte(`{}`),
				HTTPResponse: &http.Response{StatusCode: tc.statusCode},
				JSON200: &ai_optimizer.ListHostedModelsResponse{
					Items:      tc.items,
					TotalCount: int32(len(tc.items)),
				},
			}

			mockAIClient.EXPECT().
				HostedModelsAPIListHostedModelsWithResponse(gomock.Any(), "org-1", clusterID, gomock.Any()).
				Return(listResp, nil)

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			state.ID = modelID
			state.Attributes = map[string]string{
				fieldAIHostedModelClusterID: clusterID,
			}

			res := resourceAIHostedModel()
			data := res.Data(state)

			result := res.ReadContext(ctx, data, provider)
			r.Nil(result)

			if tc.expectRemoved {
				r.Equal("", data.Id())
			} else {
				r.Equal(modelID, data.Id())
				r.Equal(tc.expectedStatus, data.Get(fieldAIHostedModelStatus).(string))
				r.Equal(2, data.Get(fieldAIHostedModelCurrentReplicas).(int))
				r.Equal("AWS", data.Get(fieldAIHostedModelCloudProvider).(string))
				r.Equal("all replicas healthy", data.Get(fieldAIHostedModelStatusReason).(string))
			}
		})
	}
}

func TestAIHostedModelReadCluster404(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	_, mockAIClient, provider := newHostedModelProvider(ctrl)

	mockAIClient.EXPECT().
		HostedModelsAPIListHostedModelsWithResponse(gomock.Any(), "org-1", "gone-cluster", gomock.Any()).
		Return(&ai_optimizer.HostedModelsAPIListHostedModelsResponse{
			Body:         []byte(`{"message":"not found"}`),
			HTTPResponse: &http.Response{StatusCode: 404},
		}, nil)

	state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
	state.ID = "model-abc"
	state.Attributes = map[string]string{
		fieldAIHostedModelClusterID: "gone-cluster",
	}

	res := resourceAIHostedModel()
	data := res.Data(state)

	result := res.ReadContext(context.Background(), data, provider)
	r.Nil(result)
	r.Equal("", data.Id())
}

func TestAIHostedModelUpdate(t *testing.T) {
	t.Parallel()

	modelID := "model-abc"
	clusterID := "cluster-xyz"

	tests := map[string]struct {
		updateStatusCode int
		expectError      bool
	}{
		"successful update": {
			updateStatusCode: 200,
		},
		"api error": {
			updateStatusCode: 500,
			expectError:      true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := require.New(t)
			ctrl := gomock.NewController(t)
			_, mockAIClient, provider := newHostedModelProvider(ctrl)

			status := ai_optimizer.HostedModelStatus("RUNNING")
			updated := ai_optimizer.HostedModel{
				Id:           &modelID,
				ClusterId:    clusterID,
				ModelSpecsId: "specs-1",
				Service:      "llama",
				Port:         8080,
				Status:       &status,
			}

			updateResp := &ai_optimizer.HostedModelsAPIUpdateHostedModelResponse{
				Body:         []byte(`{}`),
				HTTPResponse: &http.Response{StatusCode: tc.updateStatusCode},
			}
			if tc.updateStatusCode == 200 {
				updateResp.JSON200 = &updated
			}

			gomock.InOrder(
				mockAIClient.EXPECT().
					HostedModelsAPIUpdateHostedModelWithResponse(gomock.Any(), "org-1", clusterID, modelID, gomock.Any()).
					Return(updateResp, nil),
			)

			if !tc.expectError {
				mockAIClient.EXPECT().
					HostedModelsAPIListHostedModelsWithResponse(gomock.Any(), "org-1", clusterID, gomock.Any()).
					Return(&ai_optimizer.HostedModelsAPIListHostedModelsResponse{
						Body:         []byte(`{}`),
						HTTPResponse: &http.Response{StatusCode: 200},
						JSON200: &ai_optimizer.ListHostedModelsResponse{
							Items:      []ai_optimizer.HostedModel{updated},
							TotalCount: 1,
						},
					}, nil)
			}

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			state.ID = modelID
			state.Attributes = map[string]string{
				fieldAIHostedModelClusterID:    clusterID,
				fieldAIHostedModelModelSpecsID: "specs-1",
				fieldAIHostedModelService:      "llama",
			}

			res := resourceAIHostedModel()
			data := res.Data(state)

			result := res.UpdateContext(context.Background(), data, provider)
			if tc.expectError {
				r.NotNil(result)
				r.True(result.HasError())
			} else {
				r.Nil(result)
				r.Equal(modelID, data.Id())
			}
		})
	}
}

func TestAIHostedModelDelete(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		statusCode  int
		expectError bool
	}{
		"successful delete": {
			statusCode: 200,
		},
		"api error": {
			statusCode:  500,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := require.New(t)
			ctrl := gomock.NewController(t)
			_, mockAIClient, provider := newHostedModelProvider(ctrl)

			mockAIClient.EXPECT().
				HostedModelsAPIDeleteHostedModelWithResponse(gomock.Any(), "org-1", "cluster-xyz", "model-abc").
				Return(&ai_optimizer.HostedModelsAPIDeleteHostedModelResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: tc.statusCode},
				}, nil)

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			state.ID = "model-abc"
			state.Attributes = map[string]string{
				fieldAIHostedModelClusterID: "cluster-xyz",
			}

			res := resourceAIHostedModel()
			data := res.Data(state)

			result := res.DeleteContext(context.Background(), data, provider)
			if tc.expectError {
				r.NotNil(result)
				r.True(result.HasError())
			} else {
				r.Nil(result)
			}
		})
	}
}

func TestAIHostedModelReadWithOrgFetch(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockSDK := mock_sdk.NewMockClientInterface(ctrl)
	mockAIClient := mock_ai_optimizer.NewMockClientWithResponsesInterface(ctrl)

	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockSDK,
		},
		aiOptimizerClient: mockAIClient,
	}

	orgBody := io.NopCloser(bytes.NewReader([]byte(`{"organizations":[{"id":"org-1"}]}`)))
	mockSDK.EXPECT().
		UsersAPIListOrganizations(gomock.Any(), gomock.Any()).
		Return(&http.Response{StatusCode: 200, Body: orgBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	modelID := "model-abc"
	clusterID := "cluster-xyz"
	status := ai_optimizer.HostedModelStatus("RUNNING")

	mockAIClient.EXPECT().
		HostedModelsAPIListHostedModelsWithResponse(gomock.Any(), "org-1", clusterID, gomock.Any()).
		Return(&ai_optimizer.HostedModelsAPIListHostedModelsResponse{
			Body:         []byte(`{}`),
			HTTPResponse: &http.Response{StatusCode: 200},
			JSON200: &ai_optimizer.ListHostedModelsResponse{
				Items: []ai_optimizer.HostedModel{
					{
						Id:           &modelID,
						ClusterId:    clusterID,
						ModelSpecsId: "specs-1",
						Service:      "llama",
						Port:         8080,
						Status:       &status,
					},
				},
				TotalCount: 1,
			},
		}, nil)

	state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
	state.ID = modelID
	state.Attributes = map[string]string{
		fieldAIHostedModelClusterID: clusterID,
	}

	res := resourceAIHostedModel()
	data := res.Data(state)

	result := res.ReadContext(context.Background(), data, provider)
	r.Nil(result)
	r.Equal(modelID, data.Id())
	r.Equal("org-1", provider.organizationID)
}

func TestAIHostedModelImporter(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		importID      string
		expectError   bool
		wantClusterID string
		wantModelID   string
	}{
		"valid import ID": {
			importID:      "cluster-xyz/model-abc",
			wantClusterID: "cluster-xyz",
			wantModelID:   "model-abc",
		},
		"missing separator": {
			importID:    "cluster-xyz-model-abc",
			expectError: true,
		},
		"empty parts": {
			importID:    "/model-abc",
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := require.New(t)

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			state.ID = tc.importID

			res := resourceAIHostedModel()
			data := res.Data(state)

			result, err := resourceAIHostedModelImporter(context.Background(), data, nil)

			if tc.expectError {
				r.Error(err)
				return
			}

			r.NoError(err)
			r.Len(result, 1)
			r.Equal(tc.wantModelID, data.Id())
			r.Equal(tc.wantClusterID, data.Get(fieldAIHostedModelClusterID).(string))
		})
	}
}
