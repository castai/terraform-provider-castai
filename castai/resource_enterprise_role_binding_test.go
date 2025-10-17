package castai

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/organization_management"
	mockOrganizationManagement "github.com/castai/terraform-provider-castai/castai/sdk/organization_management/mock"
)

func TestResourceEnterpriseRoleBindingCreateContext(t *testing.T) {
	t.Parallel()

	t.Run("when API call fails then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		roleID := uuid.NewString()
		userID := uuid.NewString()

		// Mock API to return error
		mockClient.EXPECT().
			EnterpriseAPIBatchCreateEnterpriseRoleBindingsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIBatchCreateEnterpriseRoleBindingsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError},
				JSON200:      nil,
			}, nil)

		// Minimal input state
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseRoleBindingEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseRoleBindingOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseRoleBindingName:           cty.StringVal("test-role-binding"),
			FieldEnterpriseRoleBindingDescription:    cty.StringVal("Test description"),
			FieldEnterpriseRoleBindingRoleID:         cty.StringVal(roleID),
			FieldEnterpriseRoleBindingSubjects: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingSubjectUser: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(userID),
						}),
					}),
					FieldEnterpriseRoleBindingSubjectServiceAccount: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
					FieldEnterpriseRoleBindingSubjectGroup: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
				}),
			}),
			FieldEnterpriseRoleBindingScopes: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingScopeOrganization: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingScopeID: cty.StringVal(organizationID),
						}),
					}),
					FieldEnterpriseRoleBindingScopeCluster: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingScopeID: cty.String,
					})),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseRoleBinding()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		// Verify error is returned
		r.NotNil(result)
		r.True(result.HasError())
		r.Contains(result[0].Summary, "batch create enterprise role bindings failed")
	})

	t.Run("when API successfully creates role binding then set state correctly", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID1 := uuid.NewString()
		organizationID2 := uuid.NewString()
		roleBindingID := uuid.NewString()
		roleID := uuid.NewString()
		userID1 := uuid.NewString()
		userID2 := uuid.NewString()
		serviceAccountID1 := uuid.NewString()
		serviceAccountID2 := uuid.NewString()
		groupID1 := uuid.NewString()
		groupID2 := uuid.NewString()
		clusterID1 := uuid.NewString()
		clusterID2 := uuid.NewString()

		createTime := time.Now()

		// Expected create request
		expectedCreateRequest := organization_management.BatchCreateEnterpriseRoleBindingsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchCreateEnterpriseRoleBindingsRequestCreateRoleBindingRequest{
				{
					OrganizationId: organizationID1,
					RoleBinding: organization_management.BatchCreateEnterpriseRoleBindingsRequestRoleBinding{
						Name:        "engineering-viewer",
						Description: lo.ToPtr("Engineering viewer role binding"),
						Definition: organization_management.RoleBindingDefinition{
							RoleId: lo.ToPtr(roleID),
							Subjects: &[]organization_management.Subject{
								{
									User: &organization_management.UserSubject{
										Id: userID1,
									},
								},
								{
									User: &organization_management.UserSubject{
										Id: userID2,
									},
								},
								{
									ServiceAccount: &organization_management.ServiceAccountSubject{
										Id: serviceAccountID1,
									},
								},
								{
									ServiceAccount: &organization_management.ServiceAccountSubject{
										Id: serviceAccountID2,
									},
								},
								{
									Group: &organization_management.GroupSubject{
										Id: groupID1,
									},
								},
								{
									Group: &organization_management.GroupSubject{
										Id: groupID2,
									},
								},
							},
							Scopes: &[]organization_management.Scope{
								{
									Organization: &organization_management.OrganizationScope{
										Id: organizationID1,
									},
								},
								{
									Organization: &organization_management.OrganizationScope{
										Id: organizationID2,
									},
								},
								{
									Cluster: &organization_management.ClusterScope{
										Id: clusterID1,
									},
								},
								{
									Cluster: &organization_management.ClusterScope{
										Id: clusterID2,
									},
								},
							},
						},
					},
				},
			},
		}

		// Mock API response
		apiResponse := &organization_management.BatchCreateEnterpriseRoleBindingsResponse{
			RoleBindings: &[]organization_management.RoleBinding{
				{
					Id:             lo.ToPtr(roleBindingID),
					Name:           lo.ToPtr("engineering-viewer"),
					Description:    lo.ToPtr("Engineering viewer role binding"),
					OrganizationId: lo.ToPtr(organizationID1),
					CreateTime:     lo.ToPtr(createTime),
					ManagedBy:      lo.ToPtr("terraform"),
					Definition: &organization_management.RoleBindingDefinition{
						RoleId: lo.ToPtr(roleID),
						Subjects: &[]organization_management.Subject{
							{
								User: &organization_management.UserSubject{
									Id:    userID1,
									Email: lo.ToPtr("user1@example.com"),
									Name:  lo.ToPtr("User One"),
								},
							},
							{
								User: &organization_management.UserSubject{
									Id:    userID2,
									Email: lo.ToPtr("user2@example.com"),
									Name:  lo.ToPtr("User Two"),
								},
							},
							{
								ServiceAccount: &organization_management.ServiceAccountSubject{
									Id:   serviceAccountID1,
									Name: lo.ToPtr("Service Account 1"),
								},
							},
							{
								ServiceAccount: &organization_management.ServiceAccountSubject{
									Id:   serviceAccountID2,
									Name: lo.ToPtr("Service Account 2"),
								},
							},
							{
								Group: &organization_management.GroupSubject{
									Id:   groupID1,
									Name: lo.ToPtr("Engineering Group"),
								},
							},
							{
								Group: &organization_management.GroupSubject{
									Id:   groupID2,
									Name: lo.ToPtr("DevOps Group"),
								},
							},
						},
						Scopes: &[]organization_management.Scope{
							{
								Organization: &organization_management.OrganizationScope{
									Id: organizationID1,
								},
							},
							{
								Organization: &organization_management.OrganizationScope{
									Id: organizationID2,
								},
							},
							{
								Cluster: &organization_management.ClusterScope{
									Id: clusterID1,
								},
							},
							{
								Cluster: &organization_management.ClusterScope{
									Id: clusterID2,
								},
							},
						},
					},
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIBatchCreateEnterpriseRoleBindingsWithResponse(gomock.Any(), enterpriseID, expectedCreateRequest).
			Return(&organization_management.EnterpriseAPIBatchCreateEnterpriseRoleBindingsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiResponse,
			}, nil)

		// Input state - what user defined in Terraform
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseRoleBindingEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseRoleBindingOrganizationID: cty.StringVal(organizationID1),
			FieldEnterpriseRoleBindingName:           cty.StringVal("engineering-viewer"),
			FieldEnterpriseRoleBindingDescription:    cty.StringVal("Engineering viewer role binding"),
			FieldEnterpriseRoleBindingRoleID:         cty.StringVal(roleID),
			FieldEnterpriseRoleBindingSubjects: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingSubjectUser: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(userID1),
						}),
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(userID2),
						}),
					}),
					FieldEnterpriseRoleBindingSubjectServiceAccount: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(serviceAccountID1),
						}),
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(serviceAccountID2),
						}),
					}),
					FieldEnterpriseRoleBindingSubjectGroup: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(groupID1),
						}),
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(groupID2),
						}),
					}),
				}),
			}),
			FieldEnterpriseRoleBindingScopes: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingScopeOrganization: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingScopeID: cty.StringVal(organizationID1),
						}),
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingScopeID: cty.StringVal(organizationID2),
						}),
					}),
					FieldEnterpriseRoleBindingScopeCluster: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingScopeID: cty.StringVal(clusterID1),
						}),
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingScopeID: cty.StringVal(clusterID2),
						}),
					}),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseRoleBinding()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		// Verify no errors
		r.Nil(result)
		r.False(result.HasError())

		// Verify resource ID is set
		r.Equal(roleBindingID, data.Id())

		// Verify basic fields
		r.Equal(enterpriseID, data.Get(FieldEnterpriseRoleBindingEnterpriseID).(string))
		r.Equal(organizationID1, data.Get(FieldEnterpriseRoleBindingOrganizationID).(string))
		r.Equal("engineering-viewer", data.Get(FieldEnterpriseRoleBindingName).(string))
		r.Equal("Engineering viewer role binding", data.Get(FieldEnterpriseRoleBindingDescription).(string))
		r.Equal(roleID, data.Get(FieldEnterpriseRoleBindingRoleID).(string))

		// Verify subjects
		subjects := data.Get(FieldEnterpriseRoleBindingSubjects).([]any)
		r.Len(subjects, 1) // Single wrapper
		subjectsBlock := subjects[0].(map[string]any)

		// Verify users
		users := subjectsBlock[FieldEnterpriseRoleBindingSubjectUser].([]any)
		r.Len(users, 2)
		r.Equal(userID1, users[0].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])
		r.Equal(userID2, users[1].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])

		// Verify service accounts
		serviceAccounts := subjectsBlock[FieldEnterpriseRoleBindingSubjectServiceAccount].([]any)
		r.Len(serviceAccounts, 2)
		r.Equal(serviceAccountID1, serviceAccounts[0].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])
		r.Equal(serviceAccountID2, serviceAccounts[1].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])

		// Verify groups
		groups := subjectsBlock[FieldEnterpriseRoleBindingSubjectGroup].([]any)
		r.Len(groups, 2)
		r.Equal(groupID1, groups[0].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])
		r.Equal(groupID2, groups[1].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])

		// Verify scopes
		scopes := data.Get(FieldEnterpriseRoleBindingScopes).([]any)
		r.Len(scopes, 1) // Single wrapper
		scopesBlock := scopes[0].(map[string]any)

		// Verify organization scopes
		organizations := scopesBlock[FieldEnterpriseRoleBindingScopeOrganization].([]any)
		r.Len(organizations, 2)
		r.Equal(organizationID1, organizations[0].(map[string]any)[FieldEnterpriseRoleBindingScopeID])
		r.Equal(organizationID2, organizations[1].(map[string]any)[FieldEnterpriseRoleBindingScopeID])

		// Verify cluster scopes
		clusters := scopesBlock[FieldEnterpriseRoleBindingScopeCluster].([]any)
		r.Len(clusters, 2)
		r.Equal(clusterID1, clusters[0].(map[string]any)[FieldEnterpriseRoleBindingScopeID])
		r.Equal(clusterID2, clusters[1].(map[string]any)[FieldEnterpriseRoleBindingScopeID])
	})

	t.Run("when subjects is empty then returns an error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		roleID := uuid.NewString()

		// Create state with empty subjects list
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseRoleBindingEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseRoleBindingOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseRoleBindingName:           cty.StringVal("test-role-binding"),
			FieldEnterpriseRoleBindingDescription:    cty.StringVal("Test description"),
			FieldEnterpriseRoleBindingRoleID:         cty.StringVal(roleID),
			FieldEnterpriseRoleBindingSubjects: cty.ListValEmpty(cty.Object(map[string]cty.Type{
				FieldEnterpriseRoleBindingSubjectUser: cty.List(cty.Object(map[string]cty.Type{
					FieldEnterpriseRoleBindingSubjectID: cty.String,
				})),
				FieldEnterpriseRoleBindingSubjectServiceAccount: cty.List(cty.Object(map[string]cty.Type{
					FieldEnterpriseRoleBindingSubjectID: cty.String,
				})),
				FieldEnterpriseRoleBindingSubjectGroup: cty.List(cty.Object(map[string]cty.Type{
					FieldEnterpriseRoleBindingSubjectID: cty.String,
				})),
			})),
			FieldEnterpriseRoleBindingScopes: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingScopeOrganization: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingScopeID: cty.StringVal(organizationID),
						}),
					}),
					FieldEnterpriseRoleBindingScopeCluster: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingScopeID: cty.String,
					})),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseRoleBinding()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		// Verify error is returned
		r.NotNil(result)
		r.True(result.HasError())
		r.Contains(result[0].Summary, "at least one subject (user, service account, or group) must be defined")
	})

	t.Run("when scopes is empty then returns an error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		roleID := uuid.NewString()
		userID := uuid.NewString()

		// Create state with empty scopes list
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseRoleBindingEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseRoleBindingOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseRoleBindingName:           cty.StringVal("test-role-binding"),
			FieldEnterpriseRoleBindingDescription:    cty.StringVal("Test description"),
			FieldEnterpriseRoleBindingRoleID:         cty.StringVal(roleID),
			FieldEnterpriseRoleBindingSubjects: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingSubjectUser: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(userID),
						}),
					}),
					FieldEnterpriseRoleBindingSubjectServiceAccount: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
					FieldEnterpriseRoleBindingSubjectGroup: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
				}),
			}),
			FieldEnterpriseRoleBindingScopes: cty.ListValEmpty(cty.Object(map[string]cty.Type{
				FieldEnterpriseRoleBindingScopeOrganization: cty.List(cty.Object(map[string]cty.Type{
					FieldEnterpriseRoleBindingScopeID: cty.String,
				})),
				FieldEnterpriseRoleBindingScopeCluster: cty.List(cty.Object(map[string]cty.Type{
					FieldEnterpriseRoleBindingScopeID: cty.String,
				})),
			})),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseRoleBinding()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		// Verify error is returned
		r.NotNil(result)
		r.True(result.HasError())
		r.Contains(result[0].Summary, "at least one scope (organization or cluster) must be defined")
	})
}

func TestResourceEnterpriseRoleBindingReadContext(t *testing.T) {
	t.Parallel()

	t.Run("when API call throws error then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()
		roleID := uuid.NewString()
		userID := uuid.NewString()

		// Mock API to return error
		mockClient.EXPECT().
			EnterpriseAPIListRoleBindingsWithResponse(
				gomock.Any(),
				enterpriseID,
				&organization_management.EnterpriseAPIListRoleBindingsParams{
					OrganizationId: lo.ToPtr([]string{organizationID}),
				},
			).
			Return(&organization_management.EnterpriseAPIListRoleBindingsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError},
				JSON200:      nil,
			}, nil)

		// Create existing state
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseRoleBindingID:             cty.StringVal(roleBindingID),
			FieldEnterpriseRoleBindingEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseRoleBindingOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseRoleBindingName:           cty.StringVal("test-role-binding"),
			FieldEnterpriseRoleBindingDescription:    cty.StringVal("Test description"),
			FieldEnterpriseRoleBindingRoleID:         cty.StringVal(roleID),
			FieldEnterpriseRoleBindingSubjects: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingSubjectUser: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(userID),
						}),
					}),
					FieldEnterpriseRoleBindingSubjectServiceAccount: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
					FieldEnterpriseRoleBindingSubjectGroup: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
				}),
			}),
			FieldEnterpriseRoleBindingScopes: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingScopeOrganization: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingScopeID: cty.StringVal(organizationID),
						}),
					}),
					FieldEnterpriseRoleBindingScopeCluster: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingScopeID: cty.String,
					})),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceEnterpriseRoleBinding()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		// Verify error is returned
		r.NotNil(result)
		r.True(result.HasError())
		r.Contains(result[0].Summary, "list enterprise role bindings failed")
	})

	t.Run("when role binding not found then remove from state", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()
		otherRoleBindingID := uuid.NewString()
		roleID := uuid.NewString()
		userID := uuid.NewString()

		// Mock API to return list without our role binding
		mockClient.EXPECT().
			EnterpriseAPIListRoleBindingsWithResponse(
				gomock.Any(),
				enterpriseID,
				&organization_management.EnterpriseAPIListRoleBindingsParams{
					OrganizationId: lo.ToPtr([]string{organizationID}),
				},
			).
			Return(&organization_management.EnterpriseAPIListRoleBindingsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &organization_management.ListRoleBindingsResponse{
					Items: &[]organization_management.RoleBinding{
						{
							Id:             lo.ToPtr(otherRoleBindingID), // Different ID
							Name:           lo.ToPtr("other-role-binding"),
							OrganizationId: lo.ToPtr(organizationID),
							Definition: &organization_management.RoleBindingDefinition{
								RoleId: lo.ToPtr(roleID),
							},
						},
					},
				},
			}, nil)

		// Create existing state
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseRoleBindingID:             cty.StringVal(roleBindingID),
			FieldEnterpriseRoleBindingEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseRoleBindingOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseRoleBindingName:           cty.StringVal("test-role-binding"),
			FieldEnterpriseRoleBindingDescription:    cty.StringVal("Test description"),
			FieldEnterpriseRoleBindingRoleID:         cty.StringVal(roleID),
			FieldEnterpriseRoleBindingSubjects: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingSubjectUser: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(userID),
						}),
					}),
					FieldEnterpriseRoleBindingSubjectServiceAccount: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
					FieldEnterpriseRoleBindingSubjectGroup: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
				}),
			}),
			FieldEnterpriseRoleBindingScopes: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingScopeOrganization: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingScopeID: cty.StringVal(organizationID),
						}),
					}),
					FieldEnterpriseRoleBindingScopeCluster: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingScopeID: cty.String,
					})),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceEnterpriseRoleBinding()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		// Verify no error but ID is cleared (resource removed from state)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal("", data.Id())
	})

	t.Run("when API returns role binding then update state correctly", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID1 := uuid.NewString()
		organizationID2 := uuid.NewString()
		roleBindingID := uuid.NewString()
		roleID := uuid.NewString()
		userID1 := uuid.NewString()
		userID2 := uuid.NewString()
		serviceAccountID1 := uuid.NewString()
		groupID1 := uuid.NewString()
		clusterID1 := uuid.NewString()
		clusterID2 := uuid.NewString()

		createTime := time.Now()

		// Mock API response
		mockClient.EXPECT().
			EnterpriseAPIListRoleBindingsWithResponse(
				gomock.Any(),
				enterpriseID,
				&organization_management.EnterpriseAPIListRoleBindingsParams{
					OrganizationId: lo.ToPtr([]string{organizationID1}),
				},
			).
			Return(&organization_management.EnterpriseAPIListRoleBindingsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &organization_management.ListRoleBindingsResponse{
					Items: &[]organization_management.RoleBinding{
						{
							Id:             lo.ToPtr(roleBindingID),
							Name:           lo.ToPtr("engineering-viewer"),
							Description:    lo.ToPtr("Engineering viewer role binding"),
							OrganizationId: lo.ToPtr(organizationID1),
							CreateTime:     lo.ToPtr(createTime),
							ManagedBy:      lo.ToPtr("terraform"),
							Definition: &organization_management.RoleBindingDefinition{
								RoleId: lo.ToPtr(roleID),
								Subjects: &[]organization_management.Subject{
									{
										User: &organization_management.UserSubject{
											Id: userID1,
										},
									},
									{
										User: &organization_management.UserSubject{
											Id: userID2,
										},
									},
									{
										ServiceAccount: &organization_management.ServiceAccountSubject{
											Id: serviceAccountID1,
										},
									},
									{
										Group: &organization_management.GroupSubject{
											Id: groupID1,
										},
									},
								},
								Scopes: &[]organization_management.Scope{
									{
										Organization: &organization_management.OrganizationScope{
											Id: organizationID1,
										},
									},
									{
										Organization: &organization_management.OrganizationScope{
											Id: organizationID2,
										},
									},
									{
										Cluster: &organization_management.ClusterScope{
											Id: clusterID1,
										},
									},
									{
										Cluster: &organization_management.ClusterScope{
											Id: clusterID2,
										},
									},
								},
							},
						},
					},
				},
			}, nil)

		// Create existing state (simulating what exists before refresh)
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseRoleBindingID:             cty.StringVal(roleBindingID),
			FieldEnterpriseRoleBindingEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseRoleBindingOrganizationID: cty.StringVal(organizationID1),
			FieldEnterpriseRoleBindingName:           cty.StringVal("old-name"),
			FieldEnterpriseRoleBindingDescription:    cty.StringVal("old description"),
			FieldEnterpriseRoleBindingRoleID:         cty.StringVal(roleID),
			FieldEnterpriseRoleBindingSubjects: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingSubjectUser: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(userID1),
						}),
					}),
					FieldEnterpriseRoleBindingSubjectServiceAccount: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
					FieldEnterpriseRoleBindingSubjectGroup: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
				}),
			}),
			FieldEnterpriseRoleBindingScopes: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingScopeOrganization: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingScopeID: cty.StringVal(organizationID1),
						}),
					}),
					FieldEnterpriseRoleBindingScopeCluster: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingScopeID: cty.String,
					})),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceEnterpriseRoleBinding()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		// Verify no errors
		r.Nil(result)
		r.False(result.HasError())

		// Verify resource ID is maintained
		r.Equal(roleBindingID, data.Id())

		// Verify basic fields are updated
		r.Equal(enterpriseID, data.Get(FieldEnterpriseRoleBindingEnterpriseID).(string))
		r.Equal(organizationID1, data.Get(FieldEnterpriseRoleBindingOrganizationID).(string))
		r.Equal("engineering-viewer", data.Get(FieldEnterpriseRoleBindingName).(string))
		r.Equal("Engineering viewer role binding", data.Get(FieldEnterpriseRoleBindingDescription).(string))
		r.Equal(roleID, data.Get(FieldEnterpriseRoleBindingRoleID).(string))

		// Verify subjects
		subjects := data.Get(FieldEnterpriseRoleBindingSubjects).([]any)
		r.Len(subjects, 1)
		subjectsBlock := subjects[0].(map[string]any)

		// Verify users
		users := subjectsBlock[FieldEnterpriseRoleBindingSubjectUser].([]any)
		r.Len(users, 2)
		r.Equal(userID1, users[0].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])
		r.Equal(userID2, users[1].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])

		// Verify service accounts
		serviceAccounts := subjectsBlock[FieldEnterpriseRoleBindingSubjectServiceAccount].([]any)
		r.Len(serviceAccounts, 1)
		r.Equal(serviceAccountID1, serviceAccounts[0].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])

		// Verify groups
		groups := subjectsBlock[FieldEnterpriseRoleBindingSubjectGroup].([]any)
		r.Len(groups, 1)
		r.Equal(groupID1, groups[0].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])

		// Verify scopes
		scopes := data.Get(FieldEnterpriseRoleBindingScopes).([]any)
		r.Len(scopes, 1)
		scopesBlock := scopes[0].(map[string]any)

		// Verify organization scopes
		organizations := scopesBlock[FieldEnterpriseRoleBindingScopeOrganization].([]any)
		r.Len(organizations, 2)
		r.Equal(organizationID1, organizations[0].(map[string]any)[FieldEnterpriseRoleBindingScopeID])
		r.Equal(organizationID2, organizations[1].(map[string]any)[FieldEnterpriseRoleBindingScopeID])

		// Verify cluster scopes
		clusters := scopesBlock[FieldEnterpriseRoleBindingScopeCluster].([]any)
		r.Len(clusters, 2)
		r.Equal(clusterID1, clusters[0].(map[string]any)[FieldEnterpriseRoleBindingScopeID])
		r.Equal(clusterID2, clusters[1].(map[string]any)[FieldEnterpriseRoleBindingScopeID])
	})
}

func TestResourceEnterpriseRoleBindingUpdateContext(t *testing.T) {
	t.Parallel()

	t.Run("when API call fails then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()
		roleID := uuid.NewString()
		userID := uuid.NewString()

		// Mock API to return error
		mockClient.EXPECT().
			EnterpriseAPIBatchUpdateEnterpriseRoleBindingsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIBatchUpdateEnterpriseRoleBindingsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError},
				JSON200:      nil,
			}, nil)

		resource := resourceEnterpriseRoleBinding()

		// Configuration with updated values
		diff := map[string]any{
			FieldEnterpriseRoleBindingEnterpriseID:   enterpriseID,
			FieldEnterpriseRoleBindingOrganizationID: organizationID,
			FieldEnterpriseRoleBindingName:           "updated-name",
			FieldEnterpriseRoleBindingDescription:    "Updated description",
			FieldEnterpriseRoleBindingRoleID:         roleID,
			FieldEnterpriseRoleBindingSubjects: []any{
				map[string]any{
					FieldEnterpriseRoleBindingSubjectUser: []any{
						map[string]any{FieldEnterpriseRoleBindingSubjectID: userID},
					},
					FieldEnterpriseRoleBindingSubjectServiceAccount: []any{},
					FieldEnterpriseRoleBindingSubjectGroup:          []any{},
				},
			},
			FieldEnterpriseRoleBindingScopes: []any{
				map[string]any{
					FieldEnterpriseRoleBindingScopeOrganization: []any{
						map[string]any{FieldEnterpriseRoleBindingScopeID: organizationID},
					},
					FieldEnterpriseRoleBindingScopeCluster: []any{},
				},
			},
		}
		data := schema.TestResourceDataRaw(t, resource.Schema, diff)
		data.SetId(roleBindingID)

		result := resource.UpdateContext(ctx, data, provider)

		// Verify error is returned
		r.NotNil(result)
		r.True(result.HasError())
		r.Contains(result[0].Summary, "batch update enterprise role bindings failed")
	})

	t.Run("when there are changes then update resource", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()
		newRoleID := uuid.NewString()
		userID1 := uuid.NewString()
		userID2 := uuid.NewString()
		serviceAccountID := uuid.NewString()
		clusterID := uuid.NewString()

		createTime := time.Now()

		// Expected update request
		expectedUpdateRequest := organization_management.BatchUpdateEnterpriseRoleBindingsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchUpdateEnterpriseRoleBindingsRequestUpdateEnterpriseRoleBindingRequest{
				{
					Id:             roleBindingID,
					Name:           "updated-role-binding",
					OrganizationId: organizationID,
					Description:    lo.ToPtr("Updated description"),
					Definition: organization_management.RoleBindingDefinition{
						RoleId: lo.ToPtr(newRoleID),
						Subjects: &[]organization_management.Subject{
							{
								User: &organization_management.UserSubject{
									Id: userID1,
								},
							},
							{
								User: &organization_management.UserSubject{
									Id: userID2,
								},
							},
							{
								ServiceAccount: &organization_management.ServiceAccountSubject{
									Id: serviceAccountID,
								},
							},
						},
						Scopes: &[]organization_management.Scope{
							{
								Cluster: &organization_management.ClusterScope{
									Id: clusterID,
								},
							},
						},
					},
				},
			},
		}

		// Mock API response
		apiResponse := &organization_management.BatchUpdateEnterpriseRoleBindingsResponse{
			RoleBindings: &[]organization_management.RoleBinding{
				{
					Id:             lo.ToPtr(roleBindingID),
					Name:           lo.ToPtr("updated-role-binding"),
					Description:    lo.ToPtr("Updated description"),
					OrganizationId: lo.ToPtr(organizationID),
					CreateTime:     lo.ToPtr(createTime),
					ManagedBy:      lo.ToPtr("terraform"),
					Definition: &organization_management.RoleBindingDefinition{
						RoleId: lo.ToPtr(newRoleID),
						Subjects: &[]organization_management.Subject{
							{
								User: &organization_management.UserSubject{
									Id: userID1,
								},
							},
							{
								User: &organization_management.UserSubject{
									Id: userID2,
								},
							},
							{
								ServiceAccount: &organization_management.ServiceAccountSubject{
									Id: serviceAccountID,
								},
							},
						},
						Scopes: &[]organization_management.Scope{
							{
								Cluster: &organization_management.ClusterScope{
									Id: clusterID,
								},
							},
						},
					},
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIBatchUpdateEnterpriseRoleBindingsWithResponse(gomock.Any(), enterpriseID, expectedUpdateRequest).
			Return(&organization_management.EnterpriseAPIBatchUpdateEnterpriseRoleBindingsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiResponse,
			}, nil)

		resource := resourceEnterpriseRoleBinding()

		// Configuration with updated values
		diff := map[string]any{
			FieldEnterpriseRoleBindingEnterpriseID:   enterpriseID,
			FieldEnterpriseRoleBindingOrganizationID: organizationID,
			FieldEnterpriseRoleBindingName:           "updated-role-binding",
			FieldEnterpriseRoleBindingDescription:    "Updated description",
			FieldEnterpriseRoleBindingRoleID:         newRoleID,
			FieldEnterpriseRoleBindingSubjects: []any{
				map[string]any{
					FieldEnterpriseRoleBindingSubjectUser: []any{
						map[string]any{FieldEnterpriseRoleBindingSubjectID: userID1},
						map[string]any{FieldEnterpriseRoleBindingSubjectID: userID2},
					},
					FieldEnterpriseRoleBindingSubjectServiceAccount: []any{
						map[string]any{FieldEnterpriseRoleBindingSubjectID: serviceAccountID},
					},
					FieldEnterpriseRoleBindingSubjectGroup: []any{},
				},
			},
			FieldEnterpriseRoleBindingScopes: []any{
				map[string]any{
					FieldEnterpriseRoleBindingScopeOrganization: []any{},
					FieldEnterpriseRoleBindingScopeCluster: []any{
						map[string]any{FieldEnterpriseRoleBindingScopeID: clusterID},
					},
				},
			},
		}
		data := schema.TestResourceDataRaw(t, resource.Schema, diff)
		data.SetId(roleBindingID)

		result := resource.UpdateContext(ctx, data, provider)

		// Verify no errors
		r.Nil(result)
		r.False(result.HasError())

		// Verify resource ID is maintained
		r.Equal(roleBindingID, data.Id())

		// Verify updated fields
		r.Equal("updated-role-binding", data.Get(FieldEnterpriseRoleBindingName).(string))
		r.Equal("Updated description", data.Get(FieldEnterpriseRoleBindingDescription).(string))
		r.Equal(newRoleID, data.Get(FieldEnterpriseRoleBindingRoleID).(string))

		// Verify updated subjects
		subjects := data.Get(FieldEnterpriseRoleBindingSubjects).([]any)
		r.Len(subjects, 1)
		subjectsBlock := subjects[0].(map[string]any)

		users := subjectsBlock[FieldEnterpriseRoleBindingSubjectUser].([]any)
		r.Len(users, 2)
		r.Equal(userID1, users[0].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])
		r.Equal(userID2, users[1].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])

		serviceAccounts := subjectsBlock[FieldEnterpriseRoleBindingSubjectServiceAccount].([]any)
		r.Len(serviceAccounts, 1)
		r.Equal(serviceAccountID, serviceAccounts[0].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])

		// Verify updated scopes
		scopes := data.Get(FieldEnterpriseRoleBindingScopes).([]any)
		r.Len(scopes, 1)
		scopesBlock := scopes[0].(map[string]any)

		clusters := scopesBlock[FieldEnterpriseRoleBindingScopeCluster].([]any)
		r.Len(clusters, 1)
		r.Equal(clusterID, clusters[0].(map[string]any)[FieldEnterpriseRoleBindingScopeID])
	})

	t.Run("when no changes detected then skip update", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockCtrl := gomock.NewController(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(mockCtrl)

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()
		roleID := uuid.NewString()
		userID := uuid.NewString()

		// No API call should be made - verify by not setting any expectations
		// If any call is made, gomock will fail the test

		// Create state with no changes
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseRoleBindingID:             cty.StringVal(roleBindingID),
			FieldEnterpriseRoleBindingEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseRoleBindingOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseRoleBindingName:           cty.StringVal("test-role-binding"),
			FieldEnterpriseRoleBindingDescription:    cty.StringVal("Test description"),
			FieldEnterpriseRoleBindingRoleID:         cty.StringVal(roleID),
			FieldEnterpriseRoleBindingSubjects: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingSubjectUser: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(userID),
						}),
					}),
					FieldEnterpriseRoleBindingSubjectServiceAccount: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
					FieldEnterpriseRoleBindingSubjectGroup: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
				}),
			}),
			FieldEnterpriseRoleBindingScopes: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingScopeOrganization: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingScopeID: cty.StringVal(organizationID),
						}),
					}),
					FieldEnterpriseRoleBindingScopeCluster: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingScopeID: cty.String,
					})),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceEnterpriseRoleBinding()
		data := resource.Data(state)

		// Don't make any changes - UpdateContext should detect no changes and skip API call
		result := resource.UpdateContext(ctx, data, provider)

		// Verify no errors
		r.Nil(result)
		r.False(result.HasError())

		// Verify resource ID is maintained
		r.Equal(roleBindingID, data.Id())
	})

	t.Run("when updating subjects then apply changes correctly", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()
		roleID := uuid.NewString()

		// New subjects
		newUserID1 := uuid.NewString()
		newUserID2 := uuid.NewString()
		serviceAccountID := uuid.NewString()
		groupID := uuid.NewString()

		createTime := time.Now()

		// Expected update request with new subjects
		expectedUpdateRequest := organization_management.BatchUpdateEnterpriseRoleBindingsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchUpdateEnterpriseRoleBindingsRequestUpdateEnterpriseRoleBindingRequest{
				{
					Id:             roleBindingID,
					Name:           "test-role-binding",
					OrganizationId: organizationID,
					Description:    lo.ToPtr("Test description"),
					Definition: organization_management.RoleBindingDefinition{
						RoleId: lo.ToPtr(roleID),
						Subjects: &[]organization_management.Subject{
							{
								User: &organization_management.UserSubject{
									Id: newUserID1,
								},
							},
							{
								User: &organization_management.UserSubject{
									Id: newUserID2,
								},
							},
							{
								ServiceAccount: &organization_management.ServiceAccountSubject{
									Id: serviceAccountID,
								},
							},
							{
								Group: &organization_management.GroupSubject{
									Id: groupID,
								},
							},
						},
						Scopes: &[]organization_management.Scope{
							{
								Organization: &organization_management.OrganizationScope{
									Id: organizationID,
								},
							},
						},
					},
				},
			},
		}

		// Mock API response
		apiResponse := &organization_management.BatchUpdateEnterpriseRoleBindingsResponse{
			RoleBindings: &[]organization_management.RoleBinding{
				{
					Id:             lo.ToPtr(roleBindingID),
					Name:           lo.ToPtr("test-role-binding"),
					Description:    lo.ToPtr("Test description"),
					OrganizationId: lo.ToPtr(organizationID),
					CreateTime:     lo.ToPtr(createTime),
					Definition: &organization_management.RoleBindingDefinition{
						RoleId: lo.ToPtr(roleID),
						Subjects: &[]organization_management.Subject{
							{
								User: &organization_management.UserSubject{
									Id: newUserID1,
								},
							},
							{
								User: &organization_management.UserSubject{
									Id: newUserID2,
								},
							},
							{
								ServiceAccount: &organization_management.ServiceAccountSubject{
									Id: serviceAccountID,
								},
							},
							{
								Group: &organization_management.GroupSubject{
									Id: groupID,
								},
							},
						},
						Scopes: &[]organization_management.Scope{
							{
								Organization: &organization_management.OrganizationScope{
									Id: organizationID,
								},
							},
						},
					},
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIBatchUpdateEnterpriseRoleBindingsWithResponse(gomock.Any(), enterpriseID, expectedUpdateRequest).
			Return(&organization_management.EnterpriseAPIBatchUpdateEnterpriseRoleBindingsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiResponse,
			}, nil)

		resource := resourceEnterpriseRoleBinding()

		// Configuration with updated subjects
		diff := map[string]any{
			FieldEnterpriseRoleBindingEnterpriseID:   enterpriseID,
			FieldEnterpriseRoleBindingOrganizationID: organizationID,
			FieldEnterpriseRoleBindingName:           "test-role-binding",
			FieldEnterpriseRoleBindingDescription:    "Test description",
			FieldEnterpriseRoleBindingRoleID:         roleID,
			FieldEnterpriseRoleBindingSubjects: []any{
				map[string]any{
					FieldEnterpriseRoleBindingSubjectUser: []any{
						map[string]any{FieldEnterpriseRoleBindingSubjectID: newUserID1},
						map[string]any{FieldEnterpriseRoleBindingSubjectID: newUserID2},
					},
					FieldEnterpriseRoleBindingSubjectServiceAccount: []any{
						map[string]any{FieldEnterpriseRoleBindingSubjectID: serviceAccountID},
					},
					FieldEnterpriseRoleBindingSubjectGroup: []any{
						map[string]any{FieldEnterpriseRoleBindingSubjectID: groupID},
					},
				},
			},
			FieldEnterpriseRoleBindingScopes: []any{
				map[string]any{
					FieldEnterpriseRoleBindingScopeOrganization: []any{
						map[string]any{FieldEnterpriseRoleBindingScopeID: organizationID},
					},
					FieldEnterpriseRoleBindingScopeCluster: []any{},
				},
			},
		}
		data := schema.TestResourceDataRaw(t, resource.Schema, diff)
		data.SetId(roleBindingID)

		result := resource.UpdateContext(ctx, data, provider)

		// Verify no errors
		r.Nil(result)
		r.False(result.HasError())

		// Verify updated subjects
		subjects := data.Get(FieldEnterpriseRoleBindingSubjects).([]any)
		r.Len(subjects, 1)
		subjectsBlock := subjects[0].(map[string]any)

		users := subjectsBlock[FieldEnterpriseRoleBindingSubjectUser].([]any)
		r.Len(users, 2)
		r.Equal(newUserID1, users[0].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])
		r.Equal(newUserID2, users[1].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])

		serviceAccounts := subjectsBlock[FieldEnterpriseRoleBindingSubjectServiceAccount].([]any)
		r.Len(serviceAccounts, 1)
		r.Equal(serviceAccountID, serviceAccounts[0].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])

		groups := subjectsBlock[FieldEnterpriseRoleBindingSubjectGroup].([]any)
		r.Len(groups, 1)
		r.Equal(groupID, groups[0].(map[string]any)[FieldEnterpriseRoleBindingSubjectID])
	})

	t.Run("when updating scopes then apply changes correctly", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()
		roleID := uuid.NewString()
		userID := uuid.NewString()

		// New scopes
		newClusterID1 := uuid.NewString()
		newClusterID2 := uuid.NewString()
		orgID1 := uuid.NewString()

		createTime := time.Now()

		// Expected update request with new scopes
		expectedUpdateRequest := organization_management.BatchUpdateEnterpriseRoleBindingsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchUpdateEnterpriseRoleBindingsRequestUpdateEnterpriseRoleBindingRequest{
				{
					Id:             roleBindingID,
					Name:           "test-role-binding",
					OrganizationId: organizationID,
					Description:    lo.ToPtr("Test description"),
					Definition: organization_management.RoleBindingDefinition{
						RoleId: lo.ToPtr(roleID),
						Subjects: &[]organization_management.Subject{
							{
								User: &organization_management.UserSubject{
									Id: userID,
								},
							},
						},
						Scopes: &[]organization_management.Scope{
							{
								Organization: &organization_management.OrganizationScope{
									Id: orgID1,
								},
							},
							{
								Cluster: &organization_management.ClusterScope{
									Id: newClusterID1,
								},
							},
							{
								Cluster: &organization_management.ClusterScope{
									Id: newClusterID2,
								},
							},
						},
					},
				},
			},
		}

		// Mock API response
		apiResponse := &organization_management.BatchUpdateEnterpriseRoleBindingsResponse{
			RoleBindings: &[]organization_management.RoleBinding{
				{
					Id:             lo.ToPtr(roleBindingID),
					Name:           lo.ToPtr("test-role-binding"),
					Description:    lo.ToPtr("Test description"),
					OrganizationId: lo.ToPtr(organizationID),
					CreateTime:     lo.ToPtr(createTime),
					Definition: &organization_management.RoleBindingDefinition{
						RoleId: lo.ToPtr(roleID),
						Subjects: &[]organization_management.Subject{
							{
								User: &organization_management.UserSubject{
									Id: userID,
								},
							},
						},
						Scopes: &[]organization_management.Scope{
							{
								Organization: &organization_management.OrganizationScope{
									Id: orgID1,
								},
							},
							{
								Cluster: &organization_management.ClusterScope{
									Id: newClusterID1,
								},
							},
							{
								Cluster: &organization_management.ClusterScope{
									Id: newClusterID2,
								},
							},
						},
					},
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIBatchUpdateEnterpriseRoleBindingsWithResponse(gomock.Any(), enterpriseID, expectedUpdateRequest).
			Return(&organization_management.EnterpriseAPIBatchUpdateEnterpriseRoleBindingsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiResponse,
			}, nil)

		resource := resourceEnterpriseRoleBinding()

		// Configuration with updated scopes
		diff := map[string]any{
			FieldEnterpriseRoleBindingEnterpriseID:   enterpriseID,
			FieldEnterpriseRoleBindingOrganizationID: organizationID,
			FieldEnterpriseRoleBindingName:           "test-role-binding",
			FieldEnterpriseRoleBindingDescription:    "Test description",
			FieldEnterpriseRoleBindingRoleID:         roleID,
			FieldEnterpriseRoleBindingSubjects: []any{
				map[string]any{
					FieldEnterpriseRoleBindingSubjectUser: []any{
						map[string]any{FieldEnterpriseRoleBindingSubjectID: userID},
					},
					FieldEnterpriseRoleBindingSubjectServiceAccount: []any{},
					FieldEnterpriseRoleBindingSubjectGroup:          []any{},
				},
			},
			FieldEnterpriseRoleBindingScopes: []any{
				map[string]any{
					FieldEnterpriseRoleBindingScopeOrganization: []any{
						map[string]any{FieldEnterpriseRoleBindingScopeID: orgID1},
					},
					FieldEnterpriseRoleBindingScopeCluster: []any{
						map[string]any{FieldEnterpriseRoleBindingScopeID: newClusterID1},
						map[string]any{FieldEnterpriseRoleBindingScopeID: newClusterID2},
					},
				},
			},
		}
		data := schema.TestResourceDataRaw(t, resource.Schema, diff)
		data.SetId(roleBindingID)

		result := resource.UpdateContext(ctx, data, provider)

		// Verify no errors
		r.Nil(result)
		r.False(result.HasError())

		// Verify updated scopes
		scopes := data.Get(FieldEnterpriseRoleBindingScopes).([]any)
		r.Len(scopes, 1)
		scopesBlock := scopes[0].(map[string]any)

		organizations := scopesBlock[FieldEnterpriseRoleBindingScopeOrganization].([]any)
		r.Len(organizations, 1)
		r.Equal(orgID1, organizations[0].(map[string]any)[FieldEnterpriseRoleBindingScopeID])

		clusters := scopesBlock[FieldEnterpriseRoleBindingScopeCluster].([]any)
		r.Len(clusters, 2)
		r.Equal(newClusterID1, clusters[0].(map[string]any)[FieldEnterpriseRoleBindingScopeID])
		r.Equal(newClusterID2, clusters[1].(map[string]any)[FieldEnterpriseRoleBindingScopeID])
	})
}

func TestResourceEnterpriseRoleBindingDeleteContext(t *testing.T) {
	t.Parallel()

	t.Run("when API call fails then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()
		roleID := uuid.NewString()
		userID := uuid.NewString()

		// Mock API to return error
		mockClient.EXPECT().
			EnterpriseAPIBatchDeleteEnterpriseRoleBindingsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIBatchDeleteEnterpriseRoleBindingsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError},
			}, nil)

		// Create existing state
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseRoleBindingID:             cty.StringVal(roleBindingID),
			FieldEnterpriseRoleBindingEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseRoleBindingOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseRoleBindingName:           cty.StringVal("test-role-binding"),
			FieldEnterpriseRoleBindingDescription:    cty.StringVal("Test description"),
			FieldEnterpriseRoleBindingRoleID:         cty.StringVal(roleID),
			FieldEnterpriseRoleBindingSubjects: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingSubjectUser: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(userID),
						}),
					}),
					FieldEnterpriseRoleBindingSubjectServiceAccount: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
					FieldEnterpriseRoleBindingSubjectGroup: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
				}),
			}),
			FieldEnterpriseRoleBindingScopes: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingScopeOrganization: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingScopeID: cty.StringVal(organizationID),
						}),
					}),
					FieldEnterpriseRoleBindingScopeCluster: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingScopeID: cty.String,
					})),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceEnterpriseRoleBinding()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		// Verify error is returned
		r.NotNil(result)
		r.True(result.HasError())
		r.Contains(result[0].Summary, "batch delete enterprise role bindings failed")
		r.NotEmpty(data.Id(), "Resource ID should not be cleared when delete fails")
	})

	t.Run("when API successfully deletes role binding then clear state", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		roleBindingID := uuid.NewString()
		roleID := uuid.NewString()
		userID := uuid.NewString()

		// Expected delete request
		expectedDeleteRequest := organization_management.BatchDeleteEnterpriseRoleBindingsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchDeleteEnterpriseRoleBindingsRequestDeleteRoleBindingRequest{
				{
					Id:             roleBindingID,
					OrganizationId: organizationID,
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIBatchDeleteEnterpriseRoleBindingsWithResponse(gomock.Any(), enterpriseID, expectedDeleteRequest).
			Return(&organization_management.EnterpriseAPIBatchDeleteEnterpriseRoleBindingsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil)

		// Create existing state
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseRoleBindingID:             cty.StringVal(roleBindingID),
			FieldEnterpriseRoleBindingEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseRoleBindingOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseRoleBindingName:           cty.StringVal("test-role-binding"),
			FieldEnterpriseRoleBindingDescription:    cty.StringVal("Test description"),
			FieldEnterpriseRoleBindingRoleID:         cty.StringVal(roleID),
			FieldEnterpriseRoleBindingSubjects: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingSubjectUser: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingSubjectID: cty.StringVal(userID),
						}),
					}),
					FieldEnterpriseRoleBindingSubjectServiceAccount: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
					FieldEnterpriseRoleBindingSubjectGroup: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingSubjectID: cty.String,
					})),
				}),
			}),
			FieldEnterpriseRoleBindingScopes: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseRoleBindingScopeOrganization: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseRoleBindingScopeID: cty.StringVal(organizationID),
						}),
					}),
					FieldEnterpriseRoleBindingScopeCluster: cty.ListValEmpty(cty.Object(map[string]cty.Type{
						FieldEnterpriseRoleBindingScopeID: cty.String,
					})),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = roleBindingID

		resource := resourceEnterpriseRoleBinding()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		// Verify no errors
		r.Nil(result)
		r.False(result.HasError())
		r.Empty(data.Id(), "Resource ID should be cleared after successful delete")
	})
}
