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
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestImpersonationServiceAccountDataSourceRead(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		clusterType   string
		responseBody  string
		statusCode    int
		expectedID    string
		expectedError bool
		errorContains string
	}{
		{
			name:        "successful response with id and email",
			clusterType: "aks",
			responseBody: `{
  "id": "test-service-account-id",
  "email": "test@example.com"
}`,
			statusCode:    200,
			expectedID:    "test-service-account-id",
			expectedError: false,
		},
		{
			name:        "successful response with id only",
			clusterType: "gke",
			responseBody: `{
  "id": "another-service-account-id"
}`,
			statusCode:    200,
			expectedID:    "another-service-account-id",
			expectedError: false,
		},
		{
			name:          "api error response",
			clusterType:   "aks",
			responseBody:  `{"message": "Internal server error"}`,
			statusCode:    500,
			expectedError: true,
			errorContains: "retrieving impersonation service account",
		},
		{
			name:          "unauthorized response",
			clusterType:   "gke",
			responseBody:  `{"message": "Unauthorized"}`,
			statusCode:    401,
			expectedError: true,
			errorContains: "retrieving impersonation service account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

			ctx := context.Background()
			provider := &ProviderConfig{
				api: &sdk.ClientWithResponses{
					ClientInterface: mockClient,
				},
			}

			body := io.NopCloser(bytes.NewReader([]byte(tt.responseBody)))

			mockClient.EXPECT().
				ExternalClusterAPIImpersonationServiceAccount(gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(_ context.Context, body sdk.ExternalClusterAPIImpersonationServiceAccountJSONRequestBody, _ ...sdk.RequestEditorFn) {
					r.NotNil(body.Provider)
					r.Equal(tt.clusterType, *body.Provider)
				}).
				Return(&http.Response{
					StatusCode: tt.statusCode,
					Body:       body,
					Header:     map[string][]string{"Content-Type": {"json"}},
				}, nil)

			state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
			state.Attributes = map[string]string{FieldImpersonationServiceAccountClusterType: tt.clusterType}

			resource := dataSourceImpersonationServiceAccount()
			data := resource.Data(state)

			result := resource.ReadContext(ctx, data, provider)

			if tt.expectedError {
				r.True(result.HasError())
				if tt.errorContains != "" {
					r.Contains(result[0].Summary, tt.errorContains)
				}
			} else {
				r.Nil(result)
				r.False(result.HasError())
				r.Equal(tt.expectedID, data.Id())
				r.Equal(tt.expectedID, data.Get(FieldImpersonationServiceAccountId))
			}
		})
	}
}
