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
		"private model": {
			specsID:    "specs-priv-1",
			statusCode: 200,
			json200: &ai_optimizer.ModelSpecs{
				Model:        "my-custom-model",
				RegistryType: "PRIVATE",
				PrivateRegistry: &ai_optimizer.PrivateRegistryModel{
					BaseModelId: "my-custom-model",
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

			ctx := context.Background()
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

			result := res.ReadContext(ctx, data, provider)
			r.Nil(result)

			if tc.expectRemoved {
				r.Equal("", data.Id())
			} else {
				r.Equal(tc.specsID, data.Id())
				r.Equal(tc.expectedModel, data.Get(fieldAIModelSpecsModel).(string))
				if tc.expectedType != "" {
					r.Equal(tc.expectedType, data.Get(fieldAIModelSpecsType).(string))
				}
			}
		})
	}
}
