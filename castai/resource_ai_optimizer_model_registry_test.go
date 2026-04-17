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

func newModelRegistryProvider(ctrl *gomock.Controller, mockSDK *mock_sdk.MockClientInterface, mockAIClient *mock_ai_optimizer.MockClientWithResponsesInterface) *ProviderConfig {
	return &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockSDK,
		},
		aiOptimizerClient: mockAIClient,
		organizationID:    "org-1",
	}
}

func TestAIModelRegistryCreateS3(t *testing.T) {
	t.Parallel()

	prefix := "models/"
	userName := "castai-abc123"
	registryID := "reg-new-1"
	status := ai_optimizer.ModelRegistryStatus("ACTIVE")

	tests := map[string]struct {
		bucket      string
		region      string
		prefix      string
		credentials string
	}{
		"without prefix": {
			bucket:      "my-bucket",
			region:      "us-east-1",
			credentials: `{"access_key_id":"AKIA123","secret_access_key":"secret"}`,
		},
		"with prefix": {
			bucket:      "my-bucket",
			region:      "eu-west-1",
			prefix:      prefix,
			credentials: `{"access_key_id":"AKIA456","secret_access_key":"secret2"}`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := require.New(t)
			ctrl := gomock.NewController(t)
			mockSDK := mock_sdk.NewMockClientInterface(ctrl)
			mockAIClient := mock_ai_optimizer.NewMockClientWithResponsesInterface(ctrl)

			provider := newModelRegistryProvider(ctrl, mockSDK, mockAIClient)

			created := &ai_optimizer.ModelRegistry{
				Id: &registryID,
				Provider: ai_optimizer.Provider{
					Type: "S3",
					S3: &ai_optimizer.ProviderS3Config{
						Bucket:   tc.bucket,
						Region:   tc.region,
						UserName: &userName,
					},
				},
				Status: &status,
			}
			if tc.prefix != "" {
				created.Provider.S3.Prefix = &tc.prefix
			}

			mockAIClient.EXPECT().
				ModelRegistriesAPICreateModelRegistryWithResponse(gomock.Any(), "org-1", gomock.Any()).
				Return(&ai_optimizer.ModelRegistriesAPICreateModelRegistryResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200:      created,
				}, nil)

			mockAIClient.EXPECT().
				ModelRegistriesAPIGetModelRegistryWithResponse(gomock.Any(), "org-1", registryID).
				Return(&ai_optimizer.ModelRegistriesAPIGetModelRegistryResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200:      created,
				}, nil)

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			res := resourceAIModelRegistry()
			data := res.Data(state)
			_ = data.Set(fieldAIModelRegistryProviderType, "S3")
			_ = data.Set(fieldAIModelRegistryCredentials, tc.credentials)
			s3Block := []interface{}{
				map[string]interface{}{
					fieldAIModelRegistryBucket: tc.bucket,
					fieldAIModelRegistryRegion: tc.region,
					fieldAIModelRegistryPrefix: tc.prefix,
				},
			}
			_ = data.Set(fieldAIModelRegistryS3, s3Block)

			result := res.CreateContext(context.Background(), data, provider)
			r.Nil(result)
			r.Equal(registryID, data.Id())

			s3List := data.Get(fieldAIModelRegistryS3).([]interface{})
			r.Len(s3List, 1)
			s3Map := s3List[0].(map[string]interface{})
			r.Equal(tc.bucket, s3Map[fieldAIModelRegistryBucket].(string))
			r.Equal(userName, s3Map[fieldAIModelRegistryUserName].(string))
			r.Equal("ACTIVE", data.Get(fieldAIModelRegistryStatus).(string))
		})
	}
}

func TestAIModelRegistryRead(t *testing.T) {
	t.Parallel()

	prefix := "models/"
	userName := "castai-abc123"
	statusReason := "bucket not accessible"

	tests := map[string]struct {
		registryID           string
		statusCode           int
		json200              *ai_optimizer.ModelRegistry
		expectRemoved        bool
		expectedBucket       string
		expectedRegion       string
		expectedPrefix       string
		expectedUserName     string
		expectedStatus       string
		expectedStatusReason string
	}{
		"full S3 response": {
			registryID: "reg-123",
			statusCode: 200,
			json200: &ai_optimizer.ModelRegistry{
				Provider: ai_optimizer.Provider{
					Type: "S3",
					S3: &ai_optimizer.ProviderS3Config{
						Bucket:   "my-bucket",
						Region:   "us-east-1",
						Prefix:   &prefix,
						UserName: &userName,
					},
				},
				Status: func() *ai_optimizer.ModelRegistryStatus {
					s := ai_optimizer.ModelRegistryStatus("ACTIVE")
					return &s
				}(),
			},
			expectedBucket:   "my-bucket",
			expectedRegion:   "us-east-1",
			expectedPrefix:   prefix,
			expectedUserName: userName,
			expectedStatus:   "ACTIVE",
		},
		"error status with reason": {
			registryID: "reg-456",
			statusCode: 200,
			json200: &ai_optimizer.ModelRegistry{
				Provider: ai_optimizer.Provider{
					Type: "S3",
					S3: &ai_optimizer.ProviderS3Config{
						Bucket: "other-bucket",
						Region: "eu-west-1",
					},
				},
				Status: func() *ai_optimizer.ModelRegistryStatus {
					s := ai_optimizer.ModelRegistryStatus("ERROR")
					return &s
				}(),
				StatusReason: &statusReason,
			},
			expectedBucket:       "other-bucket",
			expectedRegion:       "eu-west-1",
			expectedStatus:       "ERROR",
			expectedStatusReason: statusReason,
		},
		"not found removes from state": {
			registryID:    "reg-404",
			statusCode:    404,
			expectRemoved: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := require.New(t)
			ctrl := gomock.NewController(t)
			mockSDK := mock_sdk.NewMockClientInterface(ctrl)
			mockAIClient := mock_ai_optimizer.NewMockClientWithResponsesInterface(ctrl)

			provider := newModelRegistryProvider(ctrl, mockSDK, mockAIClient)

			var mockResp *ai_optimizer.ModelRegistriesAPIGetModelRegistryResponse
			if tc.statusCode == 404 {
				mockResp = &ai_optimizer.ModelRegistriesAPIGetModelRegistryResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: 404},
				}
			} else {
				mockResp = &ai_optimizer.ModelRegistriesAPIGetModelRegistryResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: 200},
					JSON200:      tc.json200,
				}
			}

			mockAIClient.EXPECT().
				ModelRegistriesAPIGetModelRegistryWithResponse(gomock.Any(), "org-1", tc.registryID).
				Return(mockResp, nil)

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			state.ID = tc.registryID

			res := resourceAIModelRegistry()
			data := res.Data(state)

			result := res.ReadContext(context.Background(), data, provider)
			r.Nil(result)

			if tc.expectRemoved {
				r.Equal("", data.Id())
			} else {
				r.Equal(tc.registryID, data.Id())
				s3List := data.Get(fieldAIModelRegistryS3).([]interface{})
				r.Len(s3List, 1)
				s3Map := s3List[0].(map[string]interface{})
				r.Equal(tc.expectedBucket, s3Map[fieldAIModelRegistryBucket].(string))
				r.Equal(tc.expectedRegion, s3Map[fieldAIModelRegistryRegion].(string))
				r.Equal(tc.expectedPrefix, s3Map[fieldAIModelRegistryPrefix].(string))
				r.Equal(tc.expectedUserName, s3Map[fieldAIModelRegistryUserName].(string))
				r.Equal(tc.expectedStatus, data.Get(fieldAIModelRegistryStatus).(string))
				r.Equal(tc.expectedStatusReason, data.Get(fieldAIModelRegistryStatusReason).(string))
			}
		})
	}
}

func TestAIModelRegistryDelete(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		registryID  string
		statusCode  int
		expectError bool
	}{
		"successful delete": {
			registryID: "reg-123",
			statusCode: 200,
		},
		"api error": {
			registryID:  "reg-456",
			statusCode:  500,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := require.New(t)
			ctrl := gomock.NewController(t)
			mockSDK := mock_sdk.NewMockClientInterface(ctrl)
			mockAIClient := mock_ai_optimizer.NewMockClientWithResponsesInterface(ctrl)

			provider := newModelRegistryProvider(ctrl, mockSDK, mockAIClient)

			mockAIClient.EXPECT().
				ModelRegistriesAPIDeleteModelRegistryWithResponse(gomock.Any(), "org-1", tc.registryID).
				Return(&ai_optimizer.ModelRegistriesAPIDeleteModelRegistryResponse{
					Body:         []byte(`{}`),
					HTTPResponse: &http.Response{StatusCode: tc.statusCode},
				}, nil)

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			state.ID = tc.registryID

			res := resourceAIModelRegistry()
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
