package castai

import (
	"context"
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

func newModelSpecsProvider(ctrl *gomock.Controller, mockSDK *mock_sdk.MockClientInterface, mockAIClient *mock_ai_optimizer.MockClientWithResponsesInterface) *ProviderConfig {
	return &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockSDK,
		},
		aiOptimizerClient: mockAIClient,
		organizationID:    "org-1",
	}
}

func TestAIModelSpecsCreate(t *testing.T) {
	t.Parallel()

	specsID := "specs-new-1"
	trueBool := true

	tests := map[string]struct {
		body          ai_optimizer.ModelSpecs
		expectedModel string
	}{
		"huggingface model": {
			body: ai_optimizer.ModelSpecs{
				Model:        "llama-3.1-8b",
				RegistryType: "HUGGING_FACE",
				Routable:     &trueBool,
				HuggingFace: &ai_optimizer.HuggingFaceModel{
					ModelName: "meta-llama/Llama-3.1-8B-Instruct",
				},
			},
			expectedModel: "llama-3.1-8b",
		},
		"private registry model": {
			body: ai_optimizer.ModelSpecs{
				Model:        "my-custom-model",
				RegistryType: "PRIVATE",
				PrivateRegistry: &ai_optimizer.PrivateRegistryModel{
					BaseModelId: "c7a7254f-b7c0-43c5-9a09-5c7afe72de92",
					RegistryId:  "reg-abc",
				},
			},
			expectedModel: "my-custom-model",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := require.New(t)
			ctrl := gomock.NewController(t)
			mockSDK := mock_sdk.NewMockClientInterface(gomock.NewController(t))
			mockAIClient := mock_ai_optimizer.NewMockClientWithResponsesInterface(ctrl)

			provider := newModelSpecsProvider(ctrl, mockSDK, mockAIClient)

			created := tc.body
			created.Id = &specsID

			mockAIClient.EXPECT().
				ModelSpecsAPICreateModelSpecsWithResponse(gomock.Any(), "org-1", gomock.Any()).
				Return(&ai_optimizer.ModelSpecsAPICreateModelSpecsResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200:      &created,
				}, nil)

			mockAIClient.EXPECT().
				ModelSpecsAPIGetModelSpecsWithResponse(gomock.Any(), "org-1", specsID).
				Return(&ai_optimizer.ModelSpecsAPIGetModelSpecsResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200:      &created,
				}, nil)

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			res := resourceAIModelSpecs()
			data := res.Data(state)
			_ = data.Set(fieldAIModelSpecsModel, tc.body.Model)
			_ = data.Set(fieldAIModelSpecsRegistryType, string(tc.body.RegistryType))

			result := res.CreateContext(context.Background(), data, provider)
			r.Nil(result)
			r.Equal(specsID, data.Id())
			r.Equal(tc.expectedModel, data.Get(fieldAIModelSpecsModel).(string))
		})
	}
}

func TestAIModelSpecsRead(t *testing.T) {
	t.Parallel()

	hfModelName := "meta-llama/Llama-3.1-8B-Instruct"
	trueBool := true

	tests := map[string]struct {
		specsID       string
		statusCode    int
		json200       *ai_optimizer.ModelSpecs
		expectRemoved bool
		expectedModel string
		expectedType  string
	}{
		"huggingface model": {
			specsID:    "specs-hf-1",
			statusCode: 200,
			json200: &ai_optimizer.ModelSpecs{
				Model:        "llama-3.1-8b",
				RegistryType: "HUGGING_FACE",
				Type:         toPtr("chat"),
				Routable:     &trueBool,
				HuggingFace: &ai_optimizer.HuggingFaceModel{
					ModelName: hfModelName,
				},
			},
			expectedModel: "llama-3.1-8b",
			expectedType:  "chat",
		},
		"private registry model": {
			specsID:    "specs-priv-1",
			statusCode: 200,
			json200: &ai_optimizer.ModelSpecs{
				Model:        "my-custom-model",
				RegistryType: "PRIVATE",
				PrivateRegistry: &ai_optimizer.PrivateRegistryModel{
					BaseModelId: "c7a7254f-b7c0-43c5-9a09-5c7afe72de92",
					RegistryId:  "reg-abc",
				},
			},
			expectedModel: "my-custom-model",
		},
		"not found removes from state": {
			specsID:       "specs-404",
			statusCode:    404,
			expectRemoved: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := require.New(t)
			ctrl := gomock.NewController(t)
			mockSDK := mock_sdk.NewMockClientInterface(gomock.NewController(t))
			mockAIClient := mock_ai_optimizer.NewMockClientWithResponsesInterface(ctrl)

			provider := newModelSpecsProvider(ctrl, mockSDK, mockAIClient)

			var mockResp *ai_optimizer.ModelSpecsAPIGetModelSpecsResponse
			if tc.statusCode == 404 {
				mockResp = &ai_optimizer.ModelSpecsAPIGetModelSpecsResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: 404},
				}
			} else {
				mockResp = &ai_optimizer.ModelSpecsAPIGetModelSpecsResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200:      tc.json200,
				}
			}

			mockAIClient.EXPECT().
				ModelSpecsAPIGetModelSpecsWithResponse(gomock.Any(), "org-1", tc.specsID).
				Return(mockResp, nil)

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			state.ID = tc.specsID

			res := resourceAIModelSpecs()
			data := res.Data(state)

			result := res.ReadContext(context.Background(), data, provider)
			r.Nil(result)

			if tc.expectRemoved {
				r.Equal("", data.Id())
			} else {
				r.Equal(tc.specsID, data.Id())
				r.Equal(tc.expectedModel, data.Get(fieldAIModelSpecsModel).(string))
				if tc.expectedType != "" {
					r.Equal(tc.expectedType, data.Get(fieldAIModelSpecsType).(string))
				}
				if tc.json200.PrivateRegistry != nil {
					pr := data.Get(fieldAIModelSpecsPrivateRegistry).([]interface{})
					r.Len(pr, 1)
					prMap := pr[0].(map[string]interface{})
					r.Equal(tc.json200.PrivateRegistry.BaseModelId, prMap[fieldAIModelSpecsPRBaseModelID].(string))
					r.Equal(tc.json200.PrivateRegistry.RegistryId, prMap[fieldAIModelSpecsPRRegistryID].(string))
				}
			}
		})
	}
}

func TestAIModelSpecsDelete(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specsID     string
		statusCode  int
		expectError bool
	}{
		"successful delete": {
			specsID:    "specs-123",
			statusCode: 200,
		},
		"api error": {
			specsID:     "specs-456",
			statusCode:  500,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := require.New(t)
			ctrl := gomock.NewController(t)
			mockSDK := mock_sdk.NewMockClientInterface(gomock.NewController(t))
			mockAIClient := mock_ai_optimizer.NewMockClientWithResponsesInterface(ctrl)

			provider := newModelSpecsProvider(ctrl, mockSDK, mockAIClient)

			mockAIClient.EXPECT().
				ModelSpecsAPIDeleteModelSpecsWithResponse(gomock.Any(), "org-1", tc.specsID).
				Return(&ai_optimizer.ModelSpecsAPIDeleteModelSpecsResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: tc.statusCode},
				}, nil)

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			state.ID = tc.specsID

			res := resourceAIModelSpecs()
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
