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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestServiceAccountKey_CreateContext(t *testing.T) {
	t.Parallel()

	t.Run("when ServiceAccountsAPI responds with non-201 status then return error", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		name := "service-account-key-name"
		serviceAccountID := "4e4cd9eb-82eb-407e-a926-e5fef81cab51"
		expiresAt := "2024-12-01T15:19:40.384Z"

		body := io.NopCloser(bytes.NewReader([]byte("mock error response")))

		mockClient.EXPECT().
			ServiceAccountsAPICreateServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, gomock.Any()).
			Return(&http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       body,
			}, nil)

		resource := resourceServiceAccountKey()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id":    cty.StringVal(organizationID),
			"name":               cty.StringVal(name),
			"service_account_id": cty.StringVal(serviceAccountID),
			"expires_at":         cty.StringVal(expiresAt),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("creating service account key: expected status code 201, received: status=500 body=mock error response", result[0].Summary)
	})

	t.Run("when ServiceAccountsAPI responds with an error then return error", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		name := "service-account-key-name"
		serviceAccountID := "4e4cd9eb-82eb-407e-a926-e5fef81cab51"
		expiresAt := "2024-12-01T15:19:40.384Z"

		mockClient.EXPECT().
			ServiceAccountsAPICreateServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, gomock.Any()).
			Return(nil, fmt.Errorf("mock network error"))

		resource := resourceServiceAccountKey()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id":    cty.StringVal(organizationID),
			"name":               cty.StringVal(name),
			"service_account_id": cty.StringVal(serviceAccountID),
			"expires_at":         cty.StringVal(expiresAt),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("creating service account key: mock network error", result[0].Summary)
	})

	t.Run("when ServiceAccountAPI respond with 201 then populate the state", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		serviceAccountID := "b11f5945-22ca-4101-a86e-d6e37f44a415"
		keyID := "da5664b3-87bf-4e03-9d1c-ec26049991b7"
		expiresAt := "2024-12-01T15:19:40.384Z"

		body := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{
  "key": {
    "id": %q,
    "name": "test-key",
    "prefix": "123q",
    "lastUsedAt": "2024-12-11T13:47:20.927Z",
    "expiresAt": %q,
    "active": true
  }
}`, keyID, expiresAt))))

		createBody := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{
    "id": %q,
    "name": "test-key",
    "token": "secret",
    "lastUsedAt": "2024-12-11T13:47:20.927Z",
    "expiresAt": %q,
    "active": true
}`, keyID, expiresAt))))

		mockClient.EXPECT().
			ServiceAccountsAPICreateServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, gomock.Any()).
			Return(&http.Response{StatusCode: http.StatusCreated, Body: createBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		mockClient.EXPECT().
			ServiceAccountsAPIGetServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, keyID).
			Return(&http.Response{StatusCode: http.StatusOK, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id":    cty.StringVal(organizationID),
			"service_account_id": cty.StringVal(serviceAccountID),
			"expires_at":         cty.StringVal(expiresAt),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = keyID

		resource := resourceServiceAccountKey()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Equal(`ID = da5664b3-87bf-4e03-9d1c-ec26049991b7
active = true
expires_at = 2024-12-01T15:19:40Z
last_used_at = 2024-12-11T13:47:20Z
name = test-key
organization_id = 4e4cd9eb-82eb-407e-a926-e5fef81cab50
prefix = 123q
service_account_id = b11f5945-22ca-4101-a86e-d6e37f44a415
token = secret
Tainted = false
`, data.State().String())
	})

	// TODO: what happens if i dont provide expiresAt nor active
}

func TestServiceAccountKey_ReadContext(t *testing.T) {
	t.Parallel()

	t.Run("when ServiceAccountAPI respond with 404 then remove form the state gracefully", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}
		body := io.NopCloser(bytes.NewReader([]byte("")))

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		serviceAccountID := "b11f5945-22ca-4101-a86e-d6e37f44a415"
		keyID := "da5664b3-87bf-4e03-9d1c-ec26049991b7"

		mockClient.EXPECT().
			ServiceAccountsAPIGetServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, keyID).
			Return(&http.Response{StatusCode: http.StatusNotFound, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id":    cty.StringVal(organizationID),
			"service_account_id": cty.StringVal(serviceAccountID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = keyID

		resource := resourceServiceAccountKey()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Empty(data.Id())
	})

	t.Run("when ServiceAccountAPI respond with 500 then return error", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}
		body := io.NopCloser(bytes.NewReader([]byte("panic")))

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		serviceAccountID := "b11f5945-22ca-4101-a86e-d6e37f44a415"
		keyID := "da5664b3-87bf-4e03-9d1c-ec26049991b7"

		mockClient.EXPECT().
			ServiceAccountsAPIGetServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, keyID).
			Return(&http.Response{StatusCode: http.StatusInternalServerError, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id":    cty.StringVal(organizationID),
			"service_account_id": cty.StringVal(serviceAccountID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = keyID

		resource := resourceServiceAccountKey()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("reading service account key: expected status code 200, received: status=500 body=panic", result[0].Summary)
	})

	t.Run("when ServiceAccountAPI respond with 200 then populate the state", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		serviceAccountID := "b11f5945-22ca-4101-a86e-d6e37f44a415"
		keyID := "da5664b3-87bf-4e03-9d1c-ec26049991b7"

		body := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{
  "key": {
    "id": %q,
    "name": "test-key",
    "prefix": "123q",
    "lastUsedAt": "2024-12-11T13:47:20.927Z",
    "expiresAt": "2024-12-11T13:47:20.927Z",
    "active": true
  }
}`, keyID))))

		mockClient.EXPECT().
			ServiceAccountsAPIGetServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, keyID).
			Return(&http.Response{StatusCode: http.StatusOK, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id":    cty.StringVal(organizationID),
			"service_account_id": cty.StringVal(serviceAccountID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = keyID

		resource := resourceServiceAccountKey()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Equal(`ID = da5664b3-87bf-4e03-9d1c-ec26049991b7
active = true
expires_at = 2024-12-11T13:47:20Z
last_used_at = 2024-12-11T13:47:20Z
name = test-key
organization_id = 4e4cd9eb-82eb-407e-a926-e5fef81cab50
prefix = 123q
service_account_id = b11f5945-22ca-4101-a86e-d6e37f44a415
Tainted = false
`, data.State().String())
	})
}

func TestServiceAccountKey_DeleteContext(t *testing.T) {
	t.Parallel()

	t.Run("when ServiceAccountsAPI responds with an error then return error", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		serviceAccountID := "4e4cd9eb-82eb-407e-a926-e5fef81cab51"
		keyID := "da5664b3-87bf-4e03-9d1c-ec26049991b7"

		mockClient.EXPECT().
			ServiceAccountsAPIDeleteServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, keyID).
			Return(nil, fmt.Errorf("mock network error"))

		resource := resourceServiceAccountKey()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id":    cty.StringVal(organizationID),
			"service_account_id": cty.StringVal(serviceAccountID),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = keyID
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("deleting service account key: mock network error", result[0].Summary)
	})

	t.Run("when ServiceAccountsAPI responds with non-201 status then return error", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		serviceAccountID := "4e4cd9eb-82eb-407e-a926-e5fef81cab51"
		keyID := "da5664b3-87bf-4e03-9d1c-ec26049991b7"

		body := io.NopCloser(bytes.NewReader([]byte("mock error response")))

		mockClient.EXPECT().
			ServiceAccountsAPIDeleteServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, keyID).
			Return(&http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       body,
			}, nil)

		resource := resourceServiceAccountKey()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id":    cty.StringVal(organizationID),
			"service_account_id": cty.StringVal(serviceAccountID),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = keyID
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("deleting service account key: expected status code 204, received: status=500 body=mock error response", result[0].Summary)
	})

	t.Run("when ServiceAccountsAPI responds with 204 then return nil", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		serviceAccountID := "4e4cd9eb-82eb-407e-a926-e5fef81cab51"
		keyID := "da5664b3-87bf-4e03-9d1c-ec26049991b7"

		mockClient.EXPECT().
			ServiceAccountsAPIDeleteServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, keyID).
			Return(&http.Response{
				StatusCode: http.StatusNoContent,
			}, nil)

		resource := resourceServiceAccountKey()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id":    cty.StringVal(organizationID),
			"service_account_id": cty.StringVal(serviceAccountID),
			"name":               cty.StringVal("key name"),
			"expires_at":         cty.StringVal("2022-01-01T00:00:00Z"),
			"last_used_at":       cty.StringVal("2022-01-01T00:00:00Z"),
			"active":             cty.BoolVal(true),
			"prefix":             cty.StringVal("key prefix"),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = keyID
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.True(data.State().Empty())
	})
}

func TestServiceAccountKey_UpdateContext(t *testing.T) {
	t.Parallel()

	t.Run("when ServiceAccountsAPI responds with an error then return error", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		serviceAccountID := "4e4cd9eb-82eb-407e-a926-e5fef81cab51"
		keyID := "da5664b3-87bf-4e03-9d1c-ec26049991b7"

		mockClient.EXPECT().
			ServiceAccountsAPIUpdateServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, keyID, gomock.Any()).
			Return(nil, fmt.Errorf("mock network error"))

		resource := resourceServiceAccountKey()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id":    cty.StringVal(organizationID),
			"name":               cty.StringVal("new name"),
			"active":             cty.BoolVal(true),
			"service_account_id": cty.StringVal(serviceAccountID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = keyID
		data := resource.Data(state)

		result := resource.UpdateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("updating service account key: mock network error", result[0].Summary)
	})

	t.Run("when ServiceAccountsAPI responds with non-200 status then return error", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		serviceAccountID := "4e4cd9eb-82eb-407e-a926-e5fef81cab51"
		keyID := "da5664b3-87bf-4e03-9d1c-ec26049991b7"

		body := io.NopCloser(bytes.NewReader([]byte("mock error response")))

		mockClient.EXPECT().
			ServiceAccountsAPIUpdateServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, keyID, gomock.Any()).
			Return(&http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       body,
			}, nil)

		resource := resourceServiceAccountKey()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id":    cty.StringVal(organizationID),
			"service_account_id": cty.StringVal(serviceAccountID),
			"name":               cty.StringVal("new name"),
			"active":             cty.BoolVal(true),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = keyID
		data := resource.Data(state)

		result := resource.UpdateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("updating service account key: expected status code 200, received: status=500 body=mock error response", result[0].Summary)
	})

	t.Run("when ServiceAccountsAPI responds with 200 then return nil", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		serviceAccountID := "4e4cd9eb-82eb-407e-a926-e5fef81cab51"
		keyID := "da5664b3-87bf-4e03-9d1c-ec26049991b7"
		expiresAt := "2024-12-11T13:47:20.927Z"
		name := "test-key"

		body := io.NopCloser(bytes.NewReader([]byte(`{"active":false}`)))
		readBody := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{
  "key": {
    "id": %q,
    "name": "test-key",
    "prefix": "123q",
    "lastUsedAt": "2024-12-11T13:47:20.927Z",
    "expiresAt": %q,
    "active": false
  }
}`, keyID, expiresAt))))

		mockClient.EXPECT().
			ServiceAccountsAPIUpdateServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, keyID, gomock.Any()).
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       body,
			}, nil)
		mockClient.EXPECT().
			ServiceAccountsAPIGetServiceAccountKey(gomock.Any(), organizationID, serviceAccountID, keyID).
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       readBody,
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)

		resource := resourceServiceAccountKey()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id":    cty.StringVal(organizationID),
			"service_account_id": cty.StringVal(serviceAccountID),
			"active":             cty.BoolVal(true),
			"name":               cty.StringVal(name),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = keyID
		data := resource.Data(state)

		result := resource.UpdateContext(ctx, data, provider)

		r.Nil(result)
		r.Equal(`ID = da5664b3-87bf-4e03-9d1c-ec26049991b7
active = false
expires_at = 2024-12-11T13:47:20Z
last_used_at = 2024-12-11T13:47:20Z
name = test-key
organization_id = 4e4cd9eb-82eb-407e-a926-e5fef81cab50
prefix = 123q
service_account_id = 4e4cd9eb-82eb-407e-a926-e5fef81cab51
Tainted = false
`, data.State().String())
	})
}
