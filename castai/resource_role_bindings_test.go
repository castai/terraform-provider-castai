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

	t.Run("when RbacServiceAPI respond with 200 then populate the state with scopes", func(t *testing.T) {
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
			  "scopes": [
				{
				  "organization": {
					"id": "` + organizationID + `"
				  }
				}
			  ],
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
scopes.# = 1
scopes.0.kind = organization
scopes.0.resource_id = `+organizationID+`
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

				body := io.NopCloser(bytes.NewReader([]byte(`{
					"message": "Internal server error",
					"error": "Something went wrong on the server",
					"code": 500
				}`)))
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
			"name":            cty.StringVal("test group"),
			"description":     cty.StringVal("test role binding description"),
			"role_id":         cty.StringVal(uuid.NewString()),
			"scopes": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"kind":        cty.StringVal("organization"),
					"resource_id": cty.StringVal(organizationID),
				}),
			}),
			"subjects": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"subject": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"kind":    cty.StringVal("user"),
							"user_id": cty.StringVal(uuid.NewString()),
						}),
					}),
				}),
			}),
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
		roleID := uuid.NewString()
		firstUserID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPIGetRoleBinding(gomock.Any(), organizationID, roleBindingID).
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"id": "` + roleBindingID + `",
					"organizationId": "` + organizationID + `",
					"name": "test group",
					"description": "test role binding description",
					"definition": {
						"roleId": "` + roleID + `",
						"scopes": [{
							"organization": {
								"id": "` + organizationID + `"
							}
						}],
						"subjects": [
							{
								"user": {
									"id": "` + firstUserID + `"
								}
							}
						]
					}
				}`))),
				Header: map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		mockClient.EXPECT().
			RbacServiceAPIUpdateRoleBinding(gomock.Any(), organizationID, roleBindingID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, reqRoleBindingID string, req sdk.RbacServiceAPIUpdateRoleBindingJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)
				r.Equal(roleBindingID, reqRoleBindingID)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewReader([]byte(`{
						"id": "` + roleBindingID + `",
						"organizationId": "` + organizationID + `",
						"name": "` + req.Name + `",
						"description": "` + *req.Description + `",
						"definition": {
							"roleId": "` + req.Definition.RoleId + `",
							"scopes": [{
								"organization": {
									"id": "` + organizationID + `"
								}
							}]
						}
					}`))),
					Header: map[string][]string{"Content-Type": {"application/json"}},
				}, nil
			})

		// Create a resource state with all required fields
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
			"name":            cty.StringVal("test group"),
			"description":     cty.StringVal("test role binding description"),
			"role_id":         cty.StringVal(roleID),
			"scopes": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"kind":        cty.StringVal("organization"),
					"resource_id": cty.StringVal(organizationID),
				}),
			}),
			"subjects": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"subject": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"kind":    cty.StringVal("user"),
							"user_id": cty.StringVal(firstUserID),
						}),
					}),
				}),
			}),
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
		roleID := uuid.NewString()
		userID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPICreateRoleBindings(gomock.Any(), organizationID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, req sdk.RbacServiceAPICreateRoleBindingsJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)

				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"message":"Internal server error"}`))),
					Header:     map[string][]string{"Content-Type": {"application/json"}},
				}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
			"name":            cty.StringVal("test role binding"),
			"description":     cty.StringVal("test role binding description"),
			"role_id":         cty.StringVal(roleID),
			"scopes": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"kind":        cty.StringVal("organization"),
					"resource_id": cty.StringVal(organizationID),
				}),
			}),
			"subjects": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"subject": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"kind":    cty.StringVal("user"),
							"user_id": cty.StringVal(userID),
						}),
					}),
				}),
			}),
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
		roleID := uuid.NewString()
		firstUserID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPICreateRoleBindings(gomock.Any(), organizationID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, req sdk.RbacServiceAPICreateRoleBindingsJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewReader([]byte(`[{
                        "id": "` + roleBindingID + `",
                        "organizationId": "` + organizationID + `",
                        "name": "test role binding",
                        "description": "test role binding description"
                    }]`))),
					Header: map[string][]string{"Content-Type": {"application/json"}},
				}, nil
			})

		mockClient.EXPECT().
			RbacServiceAPIGetRoleBinding(gomock.Any(), organizationID, roleBindingID).
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
                    "id": "` + roleBindingID + `",
                    "organizationId": "` + organizationID + `",
                    "name": "test role binding",
                    "description": "test role binding description",
                    "definition": {
                        "roleId": "` + roleID + `",
                        "scopes": [{
                            "organization": {
                                "id": "` + organizationID + `"
                            }
                        }],
                        "subjects": [
                            {
                                "user": {
                                    "id": "` + firstUserID + `"
                                }
                            }
                        ]
                    }
                }`))),
				Header: map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
			"name":            cty.StringVal("test role binding"),
			"description":     cty.StringVal("test role binding description"),
			"role_id":         cty.StringVal(roleID),
			"scopes": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"kind":        cty.StringVal("organization"),
					"resource_id": cty.StringVal(organizationID),
				}),
			}),
			"subjects": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"subject": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"kind":    cty.StringVal("user"),
							"user_id": cty.StringVal(firstUserID),
						}),
					}),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

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

func TestRoleBindingsReadContext_MultipleScopes(t *testing.T) {
	t.Parallel()

	t.Run("when RbacServiceAPI respond with 200 and multiple scopes then populate the state correctly", func(t *testing.T) {
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
		cluster1ID := "7063d31c-897e-48ef-a322-bdfda6fdbcfb"
		cluster2ID := "9872e54f-1122-33ef-b456-cdef98765432"

		body := io.NopCloser(bytes.NewReader([]byte(`{
			"id": "` + roleBindingID + `",
			"organizationId": "` + organizationID + `",
			"name": "role-binding-with-multiple-scopes",
			"description": "role binding with two cluster scopes",
			"definition": {
			  "roleId": "` + roleID + `",
			  "scopes": [
				{
				  "cluster": {
					"id": "` + cluster1ID + `"
				  }
				},
				{
				  "cluster": {
					"id": "` + cluster2ID + `"
				  }
				}
			  ],
			  "subjects": [
				{
				  "user": {
					"id": "` + userID + `"
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
description = role binding with two cluster scopes
name = role-binding-with-multiple-scopes
organization_id = `+organizationID+`
role_id = `+roleID+`
scopes.# = 2
scopes.0.kind = cluster
scopes.0.resource_id = `+cluster1ID+`
scopes.1.kind = cluster
scopes.1.resource_id = `+cluster2ID+`
subjects.# = 1
subjects.0.subject.# = 1
subjects.0.subject.0.group_id = 
subjects.0.subject.0.kind = user
subjects.0.subject.0.service_account_id = 
subjects.0.subject.0.user_id = `+userID+`
Tainted = false
`, data.State().String())
	})
}

func TestRoleBindingsUpdateContext_MultipleScopes(t *testing.T) {
	t.Parallel()

	t.Run("when updating role binding with multiple scopes then update successfully", func(t *testing.T) {
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
		roleID := uuid.NewString()
		userID := uuid.NewString()
		cluster1ID := uuid.NewString()
		cluster2ID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPIGetRoleBinding(gomock.Any(), organizationID, roleBindingID).
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"id": "` + roleBindingID + `",
					"organizationId": "` + organizationID + `",
					"name": "multi-scope role binding",
					"description": "role binding with multiple scopes",
					"definition": {
						"roleId": "` + roleID + `",
						"scopes": [
							{
								"cluster": {
									"id": "` + cluster1ID + `"
								}
							},
							{
								"cluster": {
									"id": "` + cluster2ID + `"
								}
							}
						],
						"subjects": [
							{
								"user": {
									"id": "` + userID + `"
								}
							}
						]
					}
				}`))),
				Header: map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		mockClient.EXPECT().
			RbacServiceAPIUpdateRoleBinding(gomock.Any(), organizationID, roleBindingID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, reqRoleBindingID string, req sdk.RbacServiceAPIUpdateRoleBindingJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)
				r.Equal(roleBindingID, reqRoleBindingID)

				// Verify the request contains both scopes
				r.NotNil(req.Definition.Scopes)
				r.Equal(2, len(*req.Definition.Scopes))

				// Check that both cluster scopes are included
				clusterIDsFound := []string{}
				for _, scope := range *req.Definition.Scopes {
					if scope.Cluster != nil {
						clusterIDsFound = append(clusterIDsFound, scope.Cluster.Id)
					}
				}
				r.Contains(clusterIDsFound, cluster1ID)
				r.Contains(clusterIDsFound, cluster2ID)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewReader([]byte(`{
						"id": "` + roleBindingID + `",
						"organizationId": "` + organizationID + `",
						"name": "` + req.Name + `",
						"description": "` + *req.Description + `",
						"definition": {
							"roleId": "` + req.Definition.RoleId + `",
							"scopes": [
								{
									"cluster": {
										"id": "` + cluster1ID + `"
									}
								},
								{
									"cluster": {
										"id": "` + cluster2ID + `"
									}
								}
							]
						}
					}`))),
					Header: map[string][]string{"Content-Type": {"application/json"}},
				}, nil
			})

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
			"name":            cty.StringVal("multi-scope role binding"),
			"description":     cty.StringVal("role binding with multiple scopes"),
			"role_id":         cty.StringVal(roleID),
			"scopes": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"kind":        cty.StringVal("cluster"),
					"resource_id": cty.StringVal(cluster1ID),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"kind":        cty.StringVal("cluster"),
					"resource_id": cty.StringVal(cluster2ID),
				}),
			}),
			"subjects": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"subject": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"kind":    cty.StringVal("user"),
							"user_id": cty.StringVal(userID),
						}),
					}),
				}),
			}),
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

func TestRoleBindingsCreateContext_MultipleScopes(t *testing.T) {
	t.Parallel()

	t.Run("when creating role binding with multiple cluster scopes then create successfully", func(t *testing.T) {
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
		roleID := uuid.NewString()
		userID := uuid.NewString()
		cluster1ID := uuid.NewString()
		cluster2ID := uuid.NewString()

		mockClient.EXPECT().
			RbacServiceAPICreateRoleBindings(gomock.Any(), organizationID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, reqOrgID string, req sdk.RbacServiceAPICreateRoleBindingsJSONRequestBody) (*http.Response, error) {
				r.Equal(organizationID, reqOrgID)

				// Verify the request has 2 scopes
				r.NotNil(req[0].Definition.Scopes)
				r.Equal(2, len(*req[0].Definition.Scopes))

				// Check that both cluster scopes are included
				clusterIDsFound := []string{}
				for _, scope := range *req[0].Definition.Scopes {
					if scope.Cluster != nil {
						clusterIDsFound = append(clusterIDsFound, scope.Cluster.Id)
					}
				}
				r.Contains(clusterIDsFound, cluster1ID)
				r.Contains(clusterIDsFound, cluster2ID)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewReader([]byte(`[{
                        "id": "` + roleBindingID + `",
                        "organizationId": "` + organizationID + `",
                        "name": "multi-cluster role binding",
                        "description": "role binding with multiple cluster scopes"
                    }]`))),
					Header: map[string][]string{"Content-Type": {"application/json"}},
				}, nil
			})

		mockClient.EXPECT().
			RbacServiceAPIGetRoleBinding(gomock.Any(), organizationID, roleBindingID).
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
                    "id": "` + roleBindingID + `",
                    "organizationId": "` + organizationID + `",
                    "name": "multi-cluster role binding",
                    "description": "role binding with multiple cluster scopes",
                    "definition": {
                        "roleId": "` + roleID + `",
                        "scopes": [
                            {
                                "cluster": {
                                    "id": "` + cluster1ID + `"
                                }
                            },
                            {
                                "cluster": {
                                    "id": "` + cluster2ID + `"
                                }
                            }
                        ],
                        "subjects": [
                            {
                                "user": {
                                    "id": "` + userID + `"
                                }
                            }
                        ]
                    }
                }`))),
				Header: map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(organizationID),
			"name":            cty.StringVal("multi-cluster role binding"),
			"description":     cty.StringVal("role binding with multiple cluster scopes"),
			"role_id":         cty.StringVal(roleID),
			"scopes": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"kind":        cty.StringVal("cluster"),
					"resource_id": cty.StringVal(cluster1ID),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"kind":        cty.StringVal("cluster"),
					"resource_id": cty.StringVal(cluster2ID),
				}),
			}),
			"subjects": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"subject": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"kind":    cty.StringVal("user"),
							"user_id": cty.StringVal(userID),
						}),
					}),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceRoleBindings()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
	})
}
