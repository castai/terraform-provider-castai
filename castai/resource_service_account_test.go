package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestServiceAccount_ReadContext(t *testing.T) {
	t.Parallel()

	t.Run("when state is missing service account ID then return error", func(t *testing.T) {
		r := require.New(t)

		ctx := context.Background()
		provider := &ProviderConfig{}

		stateValue := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceServiceAccount()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("service account ID is not set", result[0].Summary)
	})

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

		mockClient.EXPECT().
			ServiceAccountsAPIGetServiceAccount(gomock.Any(), organizationID, serviceAccountID).
			Return(&http.Response{StatusCode: http.StatusNotFound, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = serviceAccountID

		resource := resourceServiceAccount()
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

		mockClient.EXPECT().
			ServiceAccountsAPIGetServiceAccount(gomock.Any(), organizationID, serviceAccountID).
			Return(&http.Response{StatusCode: http.StatusInternalServerError, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = serviceAccountID

		resource := resourceServiceAccount()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("getting service account: expected status code 200, received: status=500 body=panic", result[0].Summary)
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
		userID := "671b2ebb-f361-42f0-aa2f-3049de93f8c1"

		body := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{
  "serviceAccount": {
    "id": %q,
    "name": "service-account-name",
    "email": "service-account-email",
    "description": "service-account-description",
    "createdAt": "2024-12-01T15:19:40.384Z",
    "author": {
      "id": %q,
      "kind": "user",
      "email": "user-email"
    },
    "keys": [
      {
        "id": "id",
        "name": "test",
        "prefix": "prefix",
        "lastUsedAt": "2024-12-01T15:19:40.384Z",
        "expiresAt": "2024-12-01T15:19:40.384Z",
        "active": true
      }
    ]
  }
}`, serviceAccountID, userID))))

		mockClient.EXPECT().
			ServiceAccountsAPIGetServiceAccount(gomock.Any(), organizationID, serviceAccountID).
			Return(&http.Response{StatusCode: http.StatusOK, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = serviceAccountID

		resource := resourceServiceAccount()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Equal(fmt.Sprintf(`ID = %s
author.# = 1
author.0.email = user-email
author.0.id = %s
author.0.kind = user
description = service-account-description
email = service-account-email
name = service-account-name
organization_id = %s
Tainted = false
`, serviceAccountID, userID, organizationID), data.State().String())
	})
}

func TestServiceAccount_CreateContext(t *testing.T) {
	t.Parallel()

	t.Run("when ServiceAccountsAPI responds with 201 then populate the state", func(t *testing.T) {
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
		serviceAccountEmail := "service-account-email"
		name := "service-account-name"
		description := "service-account-description"
		userID := "671b2ebb-f361-42f0-aa2f-3049de93f8c1"
		userEmail := "user-email"
		createdAt := time.Date(2024, 12, 1, 15, 19, 40, 384000000, time.UTC)

		mockClient.EXPECT().
			ServiceAccountsAPICreateServiceAccount(gomock.Any(), organizationID, gomock.Any()).
			DoAndReturn(func(_ context.Context, orgID string, req sdk.ServiceAccountsAPICreateServiceAccountJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, orgID)
				r.Equal(name, req.Name)
				r.Equal(description, *req.Description)

				resp := &sdk.CastaiServiceaccountsV1beta1CreateServiceAccountResponse{
					Id:          &serviceAccountID,
					Name:        &name,
					Description: &description,
					Email:       &serviceAccountEmail,
					Author: &sdk.CastaiServiceaccountsV1beta1CreateServiceAccountResponseAuthor{
						Id:    &userID,
						Email: &userEmail,
					},
					CreatedAt: &createdAt,
				}
				body := bytes.NewBuffer([]byte(""))
				err := json.NewEncoder(body).Encode(resp)
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})
		mockClient.EXPECT().
			ServiceAccountsAPIGetServiceAccount(gomock.Any(), organizationID, serviceAccountID).
			DoAndReturn(func(_ context.Context, orgID string, serviceAccountID string) (*http.Response, error) {
				resp := &sdk.CastaiServiceaccountsV1beta1GetServiceAccountResponse{
					ServiceAccount: sdk.CastaiServiceaccountsV1beta1ServiceAccount{
						Id:          &serviceAccountID,
						Name:        &name,
						Description: &description,
						Email:       &serviceAccountEmail,
						Author: &sdk.CastaiServiceaccountsV1beta1ServiceAccountAuthor{
							Id:    &userID,
							Email: &userEmail,
						},
						CreatedAt: &createdAt,
					},
				}
				body := bytes.NewBuffer([]byte(""))
				err := json.NewEncoder(body).Encode(resp)
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})
		resource := resourceServiceAccount()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
			"name":            cty.StringVal(name),
			"description":     cty.StringVal(description),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Equal(serviceAccountID, data.Id())
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
		name := "service-account-name"
		description := "service-account-description"

		mockClient.EXPECT().
			ServiceAccountsAPICreateServiceAccount(gomock.Any(), organizationID, gomock.Any()).
			Return(nil, fmt.Errorf("mock network error"))

		resource := resourceServiceAccount()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
			"name":            cty.StringVal(name),
			"description":     cty.StringVal(description),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("creating service account: mock network error", result[0].Summary)
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
		name := "service-account-name"
		description := "service-account-description"

		body := io.NopCloser(bytes.NewReader([]byte("mock error response")))

		mockClient.EXPECT().
			ServiceAccountsAPICreateServiceAccount(gomock.Any(), organizationID, gomock.Any()).
			Return(&http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       body,
			}, nil)

		resource := resourceServiceAccount()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
			"name":            cty.StringVal(name),
			"description":     cty.StringVal(description),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("creating service account: expected status code 201, received: status=500 body=mock error response", result[0].Summary)
	})
}

func TestServiceAccount_DeleteContext(t *testing.T) {
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

		mockClient.EXPECT().
			ServiceAccountsAPIDeleteServiceAccount(gomock.Any(), organizationID, serviceAccountID).
			Return(nil, fmt.Errorf("mock network error"))

		resource := resourceServiceAccount()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = serviceAccountID
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("deleting service account: mock network error", result[0].Summary)
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

		body := io.NopCloser(bytes.NewReader([]byte("mock error response")))

		mockClient.EXPECT().
			ServiceAccountsAPIDeleteServiceAccount(gomock.Any(), organizationID, gomock.Any()).
			Return(&http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       body,
			}, nil)

		resource := resourceServiceAccount()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = serviceAccountID
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("deleting service account: expected status code 204, received: status=500 body=mock error response", result[0].Summary)
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

		mockClient.EXPECT().
			ServiceAccountsAPIDeleteServiceAccount(gomock.Any(), organizationID, gomock.Any()).
			Return(&http.Response{
				StatusCode: http.StatusNoContent,
			}, nil)

		resource := resourceServiceAccount()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = serviceAccountID
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.True(data.State().Empty())
	})
}

func TestServiceAccount_UpdateContext(t *testing.T) {
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

		mockClient.EXPECT().
			ServiceAccountsAPIUpdateServiceAccount(gomock.Any(), organizationID, serviceAccountID, gomock.Any()).
			Return(nil, fmt.Errorf("mock network error"))

		resource := resourceServiceAccount()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
			"name":            cty.StringVal("new name"),
			"description":     cty.StringVal("new description"),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = serviceAccountID
		data := resource.Data(state)

		result := resource.UpdateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("updating service account: mock network error", result[0].Summary)
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

		body := io.NopCloser(bytes.NewReader([]byte("mock error response")))

		mockClient.EXPECT().
			ServiceAccountsAPIUpdateServiceAccount(gomock.Any(), organizationID, serviceAccountID, gomock.Any()).
			Return(&http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       body,
			}, nil)

		resource := resourceServiceAccount()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
			"name":            cty.StringVal("new name"),
			"description":     cty.StringVal("new description"),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = serviceAccountID
		data := resource.Data(state)

		result := resource.UpdateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("updating service account: expected status code 200, received: status=500 body=mock error response", result[0].Summary)
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

		userID := "4e4cd9eb-82eb-407e-a926-e5fef81cab49"
		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		serviceAccountID := "4e4cd9eb-82eb-407e-a926-e5fef81cab51"
		name := "name"
		description := "description"

		body := io.NopCloser(bytes.NewReader([]byte(`{
  "serviceAccount": {
    "name": "new",
    "description": "new description"
  }}`)))
		readBody := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{
  "serviceAccount": {
    "id": %q,
    "name": "new",
    "email": "service-account-email",
    "description": "new description",
    "createdAt": "2024-12-01T15:19:40.384Z",
    "author": {
      "id": %q,
      "kind": "user",
      "email": "user-email"
    },
    "keys": []
  }}`, serviceAccountID, userID))))

		mockClient.EXPECT().
			ServiceAccountsAPIUpdateServiceAccount(gomock.Any(), organizationID, serviceAccountID, gomock.Any()).
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       body,
			}, nil)
		mockClient.EXPECT().
			ServiceAccountsAPIGetServiceAccount(gomock.Any(), organizationID, serviceAccountID).
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       readBody,
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)

		resource := resourceServiceAccount()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
			"name":            cty.StringVal(name),
			"description":     cty.StringVal(description),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = serviceAccountID
		data := resource.Data(state)

		result := resource.UpdateContext(ctx, data, provider)

		r.Nil(result)
		r.Equal("new", data.Get("name"))
		r.Equal("new description", data.Get("description"))
	})
}
