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

func TestAIModelRegistryRead(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		registryID     string
		apiResponse    string
		statusCode     int
		expectedBucket string
		expectedRegion string
		expectedStatus string
		expectRemoved  bool
	}{
		"successful read": {
			registryID:     "reg-123",
			statusCode:     200,
			apiResponse:    `{"id":"reg-123","provider":{"type":"S3","s3":{"bucket":"my-bucket","region":"us-east-1"}},"status":"ACTIVE"}`,
			expectedBucket: "my-bucket",
			expectedRegion: "us-east-1",
			expectedStatus: "ACTIVE",
		},
		"not found removes from state": {
			registryID:    "reg-404",
			statusCode:    404,
			apiResponse:   `{}`,
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

			body := []byte(tc.apiResponse)
			var mockResp *ai_optimizer.ModelRegistriesAPIGetModelRegistryResponse
			if tc.statusCode == 404 {
				mockResp = &ai_optimizer.ModelRegistriesAPIGetModelRegistryResponse{
					Body:         body,
					HTTPResponse: &http.Response{StatusCode: 404},
				}
			} else {
				mockResp = &ai_optimizer.ModelRegistriesAPIGetModelRegistryResponse{
					Body:         body,
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200: &ai_optimizer.ModelRegistry{
						Provider: ai_optimizer.Provider{
							Type: "S3",
							S3: &ai_optimizer.ProviderS3Config{
								Bucket: tc.expectedBucket,
								Region: tc.expectedRegion,
							},
						},
						Status: func() *ai_optimizer.ModelRegistryStatus {
							s := ai_optimizer.ModelRegistryStatus(tc.expectedStatus)
							return &s
						}(),
					},
				}
			}

			mockAIClient.EXPECT().
				ModelRegistriesAPIGetModelRegistryWithResponse(gomock.Any(), "org-1", tc.registryID).
				Return(mockResp, nil)

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			state.ID = tc.registryID

			res := resourceAIModelRegistry()
			data := res.Data(state)

			result := res.ReadContext(ctx, data, provider)
			r.Nil(result)

			if tc.expectRemoved {
				r.Equal("", data.Id())
			} else {
				r.Equal(tc.registryID, data.Id())
				r.Equal(tc.expectedBucket, data.Get(fieldAIModelRegistryBucket).(string))
				r.Equal(tc.expectedRegion, data.Get(fieldAIModelRegistryRegion).(string))
				r.Equal(tc.expectedStatus, data.Get(fieldAIModelRegistryStatus).(string))
			}
		})
	}
}
