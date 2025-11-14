package castai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCreateClusterToken(t *testing.T) {
	ctx := context.Background()
	clusterID := "test-cluster-id"
	expectedToken := "test-token-123"

	tests := []struct {
		name          string
		setupMock     func(*mock_sdk.MockClientInterface)
		expectError   bool
		errorContains string
		expectedToken string
	}{
		{
			name: "successful token creation",
			setupMock: func(mockClient *mock_sdk.MockClientInterface) {
				body := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"token": "%s"}`, expectedToken))))
				mockClient.EXPECT().
					ExternalClusterAPICreateClusterToken(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
			},
			expectError:   false,
			expectedToken: expectedToken,
		},
		{
			name: "error is nil but response is 400",
			setupMock: func(mockClient *mock_sdk.MockClientInterface) {
				body := io.NopCloser(bytes.NewReader([]byte(`{"message": "bad request"}`)))
				mockClient.EXPECT().
					ExternalClusterAPICreateClusterToken(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 400, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
			},
			expectError:   true,
			errorContains: "creating cluster token",
		},
		{
			name: "error is nil but response is 401",
			setupMock: func(mockClient *mock_sdk.MockClientInterface) {
				body := io.NopCloser(bytes.NewReader([]byte(`{"message": "unauthorized"}`)))
				mockClient.EXPECT().
					ExternalClusterAPICreateClusterToken(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 401, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
			},
			expectError:   true,
			errorContains: "creating cluster token",
		},
		{
			name: "error is nil but response is 404",
			setupMock: func(mockClient *mock_sdk.MockClientInterface) {
				body := io.NopCloser(bytes.NewReader([]byte(`{"message": "cluster not found"}`)))
				mockClient.EXPECT().
					ExternalClusterAPICreateClusterToken(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 404, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
			},
			expectError:   true,
			errorContains: "creating cluster token",
		},
		{
			name: "error is nil but response is 500",
			setupMock: func(mockClient *mock_sdk.MockClientInterface) {
				body := io.NopCloser(bytes.NewReader([]byte(`{"message": "internal server error"}`)))
				mockClient.EXPECT().
					ExternalClusterAPICreateClusterToken(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 500, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
			},
			expectError:   true,
			errorContains: "creating cluster token",
		},
		{
			name: "network error with nil response",
			setupMock: func(mockClient *mock_sdk.MockClientInterface) {
				mockClient.EXPECT().
					ExternalClusterAPICreateClusterToken(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 0, Body: http.NoBody}, fmt.Errorf("network error"))
			},
			expectError:   true,
			errorContains: "creating cluster token",
		},
		{
			name: "response body is empty with 200 status",
			setupMock: func(mockClient *mock_sdk.MockClientInterface) {
				mockClient.EXPECT().
					ExternalClusterAPICreateClusterToken(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 200, Body: http.NoBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
			},
			expectError:   true,
			errorContains: "unexpected end of JSON input",
		},
		{
			name: "response with null token in JSON",
			setupMock: func(mockClient *mock_sdk.MockClientInterface) {
				body := io.NopCloser(bytes.NewReader([]byte(`{"token": null}`)))
				mockClient.EXPECT().
					ExternalClusterAPICreateClusterToken(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
			},
			expectError:   true,
			errorContains: "response was empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			mockctrl := gomock.NewController(t)
			defer mockctrl.Finish()

			mockClient := mock_sdk.NewMockClientInterface(mockctrl)
			tt.setupMock(mockClient)

			client := &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			}

			token, err := createClusterToken(ctx, client, clusterID)

			if tt.expectError {
				r.Error(err)
				if tt.errorContains != "" {
					r.Contains(err.Error(), tt.errorContains)
				}
				r.Empty(token)
			} else {
				r.NoError(err)
				r.Equal(tt.expectedToken, token)
			}
		})
	}
}
