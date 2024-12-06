package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestRoleBindingsReadContext(t *testing.T) {
	t.Parallel()

	t.Run("when state is missing role binding ID then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)

		ctx := context.Background()
		provider := &ProviderConfig{}

		stateValue := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("role binding ID is not set", result[0].Summary)
	})

	t.Run("when RbacServiceAPI respond with 404 then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()

		body := io.NopCloser(bytes.NewReader([]byte("")))

		mockClient.EXPECT().
			RbacServiceAPIGetRoleBinding(gomock.Any(), organizationID, roleBindingID).
			Return(&http.Response{StatusCode: http.StatusNotFound, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("getting role binding for read: role binding "+roleBindingID+" not found", result[0].Summary)
	})

	t.Run("when RbacServiceAPI respond with 500 then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()

		body := io.NopCloser(bytes.NewReader([]byte("internal error")))

		mockClient.EXPECT().
			RbacServiceAPIGetRoleBinding(gomock.Any(), organizationID, roleBindingID).
			Return(&http.Response{StatusCode: http.StatusInternalServerError, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("getting role binding for read: retrieving role binding: expected status code 200, received: status=500 body=internal error", result[0].Summary)
	})

	t.Run("when calling RbacServiceAPI throws error then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPIGetRoleBinding(gomock.Any(), organizationID, roleBindingID).
			Return(nil, errors.New("unexpected error"))

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("getting role binding for read: fetching role binding: unexpected error", result[0].Summary)
	})

	t.Run("when RbacServiceAPI respond with 200 then populate the state", func(t *testing.T) {
		t.Parallel()

		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		roleBindingID := "a83b7bf2-5a99-45d9-bcac-b969386e751f"
		roleID := "4df39779-dfb2-48d3-91d8-7ee5bd2bca4b"
		userID := "671b2ebb-f361-42f0-aa2f-3049de93f8c1"
		serviceAccountID := "b11f5945-22ca-4101-a86e-d6e37f44a415"
		groupID := "844d2bf2-870d-42da-a81c-4e19befc78fc"

		body := io.NopCloser(bytes.NewReader([]byte(`{
 "id": "` + roleBindingID + `",
 "organizationId": "` + organizationID + `",
 "name": "role-binding-name",
 "description": "role-binding-description",
 "definition": {
   "roleId": "` + roleID + `",
   "scope": {
     "organization": {
       "id": "` + organizationID + `"
     }
   },
   "subjects": [
     {
       "user": {
         "id": "` + userID + `"
       }
     },
     {
       "serviceAccount": {
         "id": "` + serviceAccountID + `"
		}
	  },
	  {
       "group": {
         "id": "` + groupID + `"
       }
     }
   ]
 }
}`)))

		mockClient.EXPECT().
			RbacServiceAPIGetRoleBinding(gomock.Any(), organizationID, roleBindingID).
			Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Equal(`ID = `+roleBindingID+`
description = role-binding-description
name = role-binding-name
organization_id = `+organizationID+`
role_id = `+roleID+`
scope.# = 1
scope.0.kind = organization
scope.0.resource_id = `+organizationID+`
subjects.# = 1
subjects.0.subject.# = 3
subjects.0.subject.0.group_id = 
subjects.0.subject.0.kind = user
subjects.0.subject.0.service_account_id = 
subjects.0.subject.0.user_id = `+userID+`
subjects.0.subject.1.group_id = 
subjects.0.subject.1.kind = service_account
subjects.0.subject.1.service_account_id = `+serviceAccountID+`
subjects.0.subject.1.user_id = 
subjects.0.subject.2.group_id = `+groupID+`
subjects.0.subject.2.kind = group
subjects.0.subject.2.service_account_id = 
subjects.0.subject.2.user_id = 
Tainted = false
`, data.State().String())
	})
}

func TestRoleBindingsUpdateContext(t *testing.T) {
	t.Parallel()

	t.Run("when state is missing role binding ID then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)

		ctx := context.Background()
		provider := &ProviderConfig{}

		stateValue := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.UpdateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("role binding ID is not set", result[0].Summary)
	})

	t.Run("when RbacServiceAPI UpdateRoleBinding respond with 500 then throw error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPIUpdateRoleBinding(gomock.Any(), organizationID, roleBindingID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, reqRoleBindingID string, req sdk.RbacServiceAPIUpdateRoleBindingJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)
				r.Equal(roleBindingID, reqRoleBindingID)

				body := &bytes.Buffer{}
				err := json.NewEncoder(body).Encode(&sdk.CastaiRbacV1beta1RoleBinding{})
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.UpdateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Contains(result[0].Summary, "update role binding: expected status code 200, received: status=500")
	})

	t.Run("when RbacServiceAPI UpdateRoleBinding respond with 200 then no errors", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()

		firstUserID := uuid.NewString()
		secondUserID := uuid.NewString()

		body := io.NopCloser(bytes.NewReader([]byte(`{
				"id": "` + roleBindingID + `",
				"organizationId": "` + organizationID + `",
				"name": "test group",
				"description": "test role binding description changed",
				"definition": {
					"members": [
						{
							"id": "` + firstUserID + `",
							"email": "test-user-1@test.com"
						},
						{
							"id": "` + secondUserID + `",
							"email": "test-user-2@test.com"
						}
					]
				}
			}`)))

		mockClient.EXPECT().
			RbacServiceAPIGetRoleBinding(gomock.Any(), organizationID, roleBindingID).
			Return(&http.Response{StatusCode: http.StatusOK, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		mockClient.EXPECT().
			RbacServiceAPIUpdateRoleBinding(gomock.Any(), organizationID, roleBindingID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, reqRoleBindingID string, req sdk.RbacServiceAPIUpdateRoleBindingJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)
				r.Equal(roleBindingID, reqRoleBindingID)

				body := &bytes.Buffer{}
				err := json.NewEncoder(body).Encode(&sdk.CastaiRbacV1beta1RoleBinding{})
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.UpdateContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
	})
}

func TestRoleBindingsCreateContext(t *testing.T) {
	t.Parallel()

	t.Run("when RbacServiceAPI CreateRoleBindings respond with 500 then throw error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPICreateRoleBindings(gomock.Any(), organizationID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, req sdk.RbacServiceAPICreateRoleBindingsJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)

				body := &bytes.Buffer{}
				err := json.NewEncoder(body).Encode(&sdk.CastaiRbacV1beta1RoleBinding{})
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Contains(result[0].Summary, "create role binding: expected status code 200, received: status=500")
	})

	t.Run("when RbacServiceAPI CreateRoleBindings respond with 200 then assume role binding was created", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()

		firstUserID := uuid.NewString()
		secondUserID := uuid.NewString()

		body := io.NopCloser(bytes.NewReader([]byte(`{
					"id": "` + roleBindingID + `",
					"organizationId": "` + organizationID + `",
					"name": "test role binding",
					"description": "test role binding description changed",
					"definition": {
						"members": [
							{
								"id": "` + firstUserID + `",
								"email": "test-user-1@test.com"
							},
							{
								"id": "` + secondUserID + `",
								"email": "test-user-2@test.com"
							}
						]
					}
				}`)))

		mockClient.EXPECT().
			RbacServiceAPIGetRoleBinding(gomock.Any(), organizationID, roleBindingID).
			Return(&http.Response{StatusCode: http.StatusOK, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		mockClient.EXPECT().
			RbacServiceAPICreateRoleBindings(gomock.Any(), organizationID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, req sdk.RbacServiceAPICreateRoleBindingsJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)

				body := bytes.NewBuffer([]byte(""))
				err := json.NewEncoder(body).Encode(&[]sdk.CastaiRbacV1beta1RoleBinding{
					{
						Id: lo.ToPtr(roleBindingID),
					},
				})
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
	})
}

func TestRoleBindingsDeleteContext(t *testing.T) {
	t.Parallel()

	t.Run("when state is missing role binding ID then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)

		ctx := context.Background()
		provider := &ProviderConfig{}

		stateValue := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("role binding ID is not set", result[0].Summary)
	})

	t.Run("when RbacServiceAPI DeleteRoleBinding respond with 500 then throw error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPIDeleteRoleBinding(gomock.Any(), organizationID, roleBindingID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, reqRoleBindingID string) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)
				r.Equal(roleBindingID, reqRoleBindingID)

				body := &bytes.Buffer{}
				err := json.NewEncoder(body).Encode(&sdk.CastaiRbacV1beta1RoleBinding{})
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Contains(result[0].Summary, "destroy role binding: expected status code 200, received: status=500")
	})

	t.Run("when RbacServiceAPI DeleteRoleBinding respond with 200 then no errors", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPIDeleteRoleBinding(gomock.Any(), organizationID, roleBindingID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, reqRoleBindingID string) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)
				r.Equal(roleBindingID, reqRoleBindingID)

				body := &bytes.Buffer{}
				err := json.NewEncoder(body).Encode(&sdk.CastaiRbacV1beta1RoleBinding{})
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
	})
}
