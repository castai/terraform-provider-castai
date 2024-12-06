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

func TestOrganizationGroupReadContext(t *testing.T) {
	t.Parallel()

	t.Run("when state is missing group ID then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)

		ctx := context.Background()
		provider := &ProviderConfig{}

		stateValue := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("group ID is not set", result[0].Summary)
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
		groupID := uuid.NewString()

		body := io.NopCloser(bytes.NewReader([]byte("")))

		mockClient.EXPECT().
			RbacServiceAPIGetGroup(gomock.Any(), organizationID, groupID).
			Return(&http.Response{StatusCode: http.StatusNotFound, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = groupID

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("getting group for read: group "+groupID+" not found", result[0].Summary)
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
		groupID := uuid.NewString()

		body := io.NopCloser(bytes.NewReader([]byte("internal error")))

		mockClient.EXPECT().
			RbacServiceAPIGetGroup(gomock.Any(), organizationID, groupID).
			Return(&http.Response{StatusCode: http.StatusInternalServerError, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = groupID

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("getting group for read: retrieving group: expected status code 200, received: status=500 body=internal error", result[0].Summary)
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
		groupID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPIGetGroup(gomock.Any(), organizationID, groupID).
			Return(nil, errors.New("unexpected error"))

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = groupID

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("getting group for read: fetching group: unexpected error", result[0].Summary)
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

		organizationID := uuid.NewString()
		groupID := uuid.NewString()

		firstUserID := uuid.NewString()
		secondUserID := uuid.NewString()

		body := io.NopCloser(bytes.NewReader([]byte(`{	
		"id": "` + groupID + `",
		"organizationId": "` + organizationID + `",
		"name": "test group",
		"description": "test group description",
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
			RbacServiceAPIGetGroup(gomock.Any(), organizationID, groupID).
			Return(&http.Response{StatusCode: http.StatusOK, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = groupID

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal(`ID = `+groupID+`
description = test group description
members.# = 1
members.0.member.# = 2
members.0.member.0.email = test-user-1@test.com
members.0.member.0.id = `+firstUserID+`
members.0.member.0.kind = 
members.0.member.1.email = test-user-2@test.com
members.0.member.1.id = `+secondUserID+`
members.0.member.1.kind = 
name = test group
organization_id = `+organizationID+`
Tainted = false
`, data.State().String())
	})

	t.Run("when organization is not defined, use default one for the token", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)

		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "b6bfc024-a267-400f-b8f1-db0850c369b4"
		groupID := "e9a3f787-15d4-4850-ae7c-b4864809aa55"

		organizationsBody := io.NopCloser(bytes.NewReader([]byte(`{
  "organizations": [
    {
      "id": "b6bfc024-a267-400f-b8f1-db0850c369b4",
      "name": "Test 1",
      "createdAt": "2023-04-18T16:03:18.800099Z",
      "role": "owner"
    }
  ]
}`)))

		mockClient.EXPECT().
			UsersAPIListOrganizations(gomock.Any(), gomock.Any()).
			Return(&http.Response{StatusCode: http.StatusOK, Body: organizationsBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		body := io.NopCloser(bytes.NewReader([]byte(`{
		"id": "e9a3f787-15d4-4850-ae7c-b4864809aa55",
		"organizationId": "b6bfc024-a267-400f-b8f1-db0850c369b4",
		"name": "test group",
		"description": "test group description",
		"definition": {
			"members": [
				{
					"id": "5d832285-c263-4d27-9ba5-7d8cf5759782",
					"email": "test-user-1@test.com"
				},
				{
					"id": "5d832285-c263-4d27-9ba5-7d8cf5759783",
					"email": "test-user-2@test.com"
				}
			]
		}
	}`)))

		mockClient.EXPECT().
			RbacServiceAPIGetGroup(gomock.Any(), organizationID, groupID).
			Return(&http.Response{StatusCode: http.StatusOK, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = groupID

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal(`ID = e9a3f787-15d4-4850-ae7c-b4864809aa55
description = test group description
members.# = 1
members.0.member.# = 2
members.0.member.0.email = test-user-1@test.com
members.0.member.0.id = 5d832285-c263-4d27-9ba5-7d8cf5759782
members.0.member.0.kind = 
members.0.member.1.email = test-user-2@test.com
members.0.member.1.id = 5d832285-c263-4d27-9ba5-7d8cf5759783
members.0.member.1.kind = 
name = test group
organization_id = b6bfc024-a267-400f-b8f1-db0850c369b4
Tainted = false
`, data.State().String())
	})

}

func TestOrganizationGroupUpdateContext(t *testing.T) {
	t.Parallel()

	t.Run("when state is missing group ID then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)

		ctx := context.Background()
		provider := &ProviderConfig{}

		stateValue := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.UpdateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("group ID is not set", result[0].Summary)
	})

	t.Run("when RbacServiceAPI UpdateGroup respond with 500 then throw error", func(t *testing.T) {
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
		groupID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPIUpdateGroup(gomock.Any(), organizationID, groupID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, reqGroupID string, req sdk.RbacServiceAPIUpdateGroupJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)
				r.Equal(groupID, reqGroupID)

				body := &bytes.Buffer{}
				err := json.NewEncoder(body).Encode(&sdk.CastaiRbacV1beta1Group{})
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = groupID

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.UpdateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Contains(result[0].Summary, "update group: expected status code 200, received: status=500")
	})

	t.Run("when RbacServiceAPI UpdateGroup respond with 200 then no errors", func(t *testing.T) {
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
		groupID := uuid.NewString()

		firstUserID := uuid.NewString()
		secondUserID := uuid.NewString()

		body := io.NopCloser(bytes.NewReader([]byte(`{
				"id": "` + groupID + `",
				"organizationId": "` + organizationID + `",
				"name": "test group",
				"description": "test group description changed",
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
			RbacServiceAPIGetGroup(gomock.Any(), organizationID, groupID).
			Return(&http.Response{StatusCode: http.StatusOK, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		mockClient.EXPECT().
			RbacServiceAPIUpdateGroup(gomock.Any(), organizationID, groupID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, reqGroupID string, req sdk.RbacServiceAPIUpdateGroupJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)
				r.Equal(groupID, reqGroupID)

				body := &bytes.Buffer{}
				err := json.NewEncoder(body).Encode(&sdk.CastaiRbacV1beta1Group{})
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = groupID

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.UpdateContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
	})
}

func TestOrganizationGroupCreateContext(t *testing.T) {
	t.Parallel()

	t.Run("when RbacServiceAPI CreateGroup respond with 500 then throw error", func(t *testing.T) {
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
			RbacServiceAPICreateGroup(gomock.Any(), organizationID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, req sdk.RbacServiceAPICreateGroupJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)

				body := &bytes.Buffer{}
				err := json.NewEncoder(body).Encode(&sdk.CastaiRbacV1beta1Group{})
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Contains(result[0].Summary, "create group: expected status code 200, received: status=500")
	})

	t.Run("when RbacServiceAPI CreateGroup respond with 200 then assume group was created", func(t *testing.T) {
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
		groupID := uuid.NewString()

		firstUserID := uuid.NewString()
		secondUserID := uuid.NewString()

		body := io.NopCloser(bytes.NewReader([]byte(`{
					"id": "` + groupID + `",
					"organizationId": "` + organizationID + `",
					"name": "test group",
					"description": "test group description changed",
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
			RbacServiceAPIGetGroup(gomock.Any(), organizationID, groupID).
			Return(&http.Response{StatusCode: http.StatusOK, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		mockClient.EXPECT().
			RbacServiceAPICreateGroup(gomock.Any(), organizationID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, req sdk.RbacServiceAPICreateGroupJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)

				body := bytes.NewBuffer([]byte(""))
				err := json.NewEncoder(body).Encode(&sdk.CastaiRbacV1beta1Group{
					Id: lo.ToPtr(groupID),
				})
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = groupID

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
	})
}

func TestOrganizationGroupDeleteContext(t *testing.T) {
	t.Parallel()

	t.Run("when state is missing group ID then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)

		ctx := context.Background()
		provider := &ProviderConfig{}

		stateValue := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("group ID is not set", result[0].Summary)
	})

	t.Run("when RbacServiceAPI DeleteGroup respond with 500 then throw error", func(t *testing.T) {
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
		groupID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPIDeleteGroup(gomock.Any(), organizationID, groupID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, reqGroupID string) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)
				r.Equal(groupID, reqGroupID)

				body := &bytes.Buffer{}
				err := json.NewEncoder(body).Encode(&sdk.CastaiRbacV1beta1Group{})
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = groupID

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Contains(result[0].Summary, "destroy group: expected status code 200, received: status=500")
	})

	t.Run("when RbacServiceAPI DeleteGroup respond with 200 then no errors", func(t *testing.T) {
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
		groupID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPIDeleteGroup(gomock.Any(), organizationID, groupID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, reqGroupID string) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)
				r.Equal(groupID, reqGroupID)

				body := &bytes.Buffer{}
				err := json.NewEncoder(body).Encode(&sdk.CastaiRbacV1beta1Group{})
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = groupID

		resource := resourceOrganizationGroup()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
	})
}
