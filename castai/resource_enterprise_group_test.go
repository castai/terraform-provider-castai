package castai

import (
	"context"
	"errors"
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

func TestResourceEnterpriseGroupCreateContext(t *testing.T) {
	t.Parallel()

	t.Run("when cluster id and organization id provided for the same scope return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseGroupName:           cty.StringVal("test-group"),
			FieldEnterpriseGroupDescription:    cty.StringVal("Test description"),
			FieldEnterpriseGroupMembers: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupsMember: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupMemberKind: cty.StringVal("user"),
							FieldEnterpriseGroupMemberID:   cty.StringVal("a" + uuid.NewString()),
						}),
					}),
				}),
			}),
			FieldEnterpriseGroupRoleBindings: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupRoleBinding: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupRoleBindingName:   cty.StringVal("test-binding"),
							FieldEnterpriseGroupRoleBindingRoleID: cty.StringVal(uuid.NewString()),
							FieldEnterpriseGroupRoleBindingScopes: cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									FieldEnterpriseGroupScope: cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											FieldEnterpriseGroupScopeCluster:      cty.StringVal(uuid.NewString()),
											FieldEnterpriseGroupScopeOrganization: cty.StringVal(organizationID),
										}),
									}),
								}),
							}),
						}),
					}),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseGroup()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("building create request: reading group data: scope cannot have both 'organization' and 'cluster' set simultaneously", result[0].Summary)
	})

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

		mockClient.EXPECT().
			EnterpriseAPIBatchCreateEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(nil, errors.New("network error"))

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseGroupName:           cty.StringVal("test-group"),
			FieldEnterpriseGroupDescription:    cty.StringVal("Test description"),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseGroup()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("batch create enterprise groups failed: network error", result[0].Summary)
	})

	t.Run("when API successfully creates group then set state correctly", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		groupID := uuid.NewString()
		memberID1 := uuid.NewString()
		memberID2 := uuid.NewString()
		memberID3 := uuid.NewString()
		roleBindingID1 := uuid.NewString()
		roleBindingID3 := uuid.NewString()
		roleID1 := uuid.NewString()
		roleID3 := uuid.NewString()
		clusterID1 := uuid.NewString()
		clusterID2 := uuid.NewString()

		createTime := time.Now()

		// Expected create request for new group
		expectedCreateRequest := organization_management.BatchCreateEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchCreateEnterpriseGroupsRequestGroup{
				{
					OrganizationId: organizationID,
					Name:           "engineering-team",
					Description:    lo.ToPtr("Engineering team group"),
					Members: []organization_management.BatchCreateEnterpriseGroupsRequestMember{
						{
							Kind: lo.ToPtr(organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDUSER),
							Id:   lo.ToPtr(memberID1),
						},
						{
							Kind: lo.ToPtr(organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDUSER),
							Id:   lo.ToPtr(memberID2),
						},
						{
							Kind: lo.ToPtr(organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDSERVICEACCOUNT),
							Id:   lo.ToPtr(memberID3),
						},
					},
					RoleBindings: &[]organization_management.BatchCreateEnterpriseGroupsRequestRoleBinding{
						{
							Name:   "engineering-viewer",
							RoleId: roleID1,
							Scopes: []organization_management.Scope{
								{
									Cluster: &organization_management.ClusterScope{
										Id: clusterID1,
									},
								},
								{
									Organization: &organization_management.OrganizationScope{
										Id: organizationID,
									},
								},
								{
									Cluster: &organization_management.ClusterScope{
										Id: clusterID2,
									},
								},
							},
						},
						{
							Name:   "engineering-editor",
							RoleId: roleID3,
							Scopes: []organization_management.Scope{
								{
									Organization: &organization_management.OrganizationScope{
										Id: organizationID,
									},
								},
							},
						},
					},
				},
			},
		}

		// Mock API response
		apiResponse := &organization_management.BatchCreateEnterpriseGroupsResponse{
			Groups: &[]organization_management.BatchCreateEnterpriseGroupsResponseGroup{
				{
					Id:             lo.ToPtr(groupID),
					Name:           lo.ToPtr("engineering-team"),
					OrganizationId: lo.ToPtr(organizationID),
					Description:    lo.ToPtr("Engineering team group"),
					CreateTime:     lo.ToPtr(createTime),
					ManagedBy:      lo.ToPtr("terraform"),
					Definition: &organization_management.GroupDefinition{
						Members: &[]organization_management.DefinitionMember{
							{
								Id:        lo.ToPtr(memberID1),
								Email:     lo.ToPtr("engineer@example.com"),
								AddedTime: lo.ToPtr(createTime),
								Kind:      lo.ToPtr(organization_management.DefinitionMemberKindSUBJECTKINDUSER),
							},
							{
								Id:        lo.ToPtr(memberID2),
								Email:     lo.ToPtr("security@example.com"),
								AddedTime: lo.ToPtr(createTime),
								Kind:      lo.ToPtr(organization_management.DefinitionMemberKindSUBJECTKINDUSER),
							},
							{
								Id:        lo.ToPtr(memberID3),
								AddedTime: lo.ToPtr(createTime),
								Kind:      lo.ToPtr(organization_management.DefinitionMemberKindSUBJECTKINDSERVICEACCOUNT),
							},
						},
					},
					RoleBindings: &[]organization_management.GroupRoleBinding{
						{
							Id:             roleBindingID1,
							Name:           "engineering-viewer",
							Description:    "Engineering viewer role binding",
							ManagedBy:      "terraform",
							CreateTime:     createTime,
							OrganizationId: organizationID,
							Definition: organization_management.RoleBindingRoleBindingDefinition{
								RoleId: roleID1,
								Scopes: &[]organization_management.Scope{
									{
										Cluster: &organization_management.ClusterScope{
											Id: clusterID1,
										},
									},
									{
										Organization: &organization_management.OrganizationScope{
											Id: organizationID,
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
						{
							Id:             roleBindingID3,
							Name:           "engineering-editor",
							Description:    "Engineering editor role binding",
							ManagedBy:      "terraform",
							CreateTime:     createTime,
							OrganizationId: organizationID,
							Status:         []organization_management.RoleBindingRoleBindingStatus{},
							Definition: organization_management.RoleBindingRoleBindingDefinition{
								RoleId: roleID3,
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
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIBatchCreateEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, expectedCreateRequest).
			Return(&organization_management.EnterpriseAPIBatchCreateEnterpriseGroupsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiResponse,
			}, nil)

		// Input state - what user defined in Terraform
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseGroupName:           cty.StringVal("engineering-team"),
			FieldEnterpriseGroupDescription:    cty.StringVal("Engineering team group"),
			FieldEnterpriseGroupMembers: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupsMember: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupMemberKind: cty.StringVal("user"),
							FieldEnterpriseGroupMemberID:   cty.StringVal(memberID1),
						}),
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupMemberKind: cty.StringVal("user"),
							FieldEnterpriseGroupMemberID:   cty.StringVal(memberID2),
						}),
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupMemberKind: cty.StringVal("service_account"),
							FieldEnterpriseGroupMemberID:   cty.StringVal(memberID3),
						}),
					}),
				}),
			}),
			FieldEnterpriseGroupRoleBindings: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupRoleBinding: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupRoleBindingName:   cty.StringVal("engineering-viewer"),
							FieldEnterpriseGroupRoleBindingRoleID: cty.StringVal(roleID1),
							FieldEnterpriseGroupRoleBindingScopes: cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									FieldEnterpriseGroupScope: cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											FieldEnterpriseGroupScopeCluster:      cty.StringVal(clusterID1),
											FieldEnterpriseGroupScopeOrganization: cty.StringVal(""),
										}),
										cty.ObjectVal(map[string]cty.Value{
											FieldEnterpriseGroupScopeCluster:      cty.StringVal(""),
											FieldEnterpriseGroupScopeOrganization: cty.StringVal(organizationID),
										}),
										cty.ObjectVal(map[string]cty.Value{
											FieldEnterpriseGroupScopeCluster:      cty.StringVal(clusterID2),
											FieldEnterpriseGroupScopeOrganization: cty.StringVal(""),
										}),
									}),
								}),
							}),
						}),
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupRoleBindingName:   cty.StringVal("engineering-editor"),
							FieldEnterpriseGroupRoleBindingRoleID: cty.StringVal(roleID3),
							FieldEnterpriseGroupRoleBindingScopes: cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									FieldEnterpriseGroupScope: cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											FieldEnterpriseGroupScopeCluster:      cty.StringVal(""),
											FieldEnterpriseGroupScopeOrganization: cty.StringVal(organizationID),
										}),
									}),
								}),
							}),
						}),
					}),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseGroup()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		// Verify no errors
		r.Nil(result)
		r.False(result.HasError())

		// Verify resource ID is set to enterprise ID
		r.Equal(groupID, data.Id())

		// Verify groups state
		actualEnterpriseID := data.Get(FieldEnterpriseGroupEnterpriseID).(string)
		r.Equal(enterpriseID, actualEnterpriseID)

		actualOrganizationID := data.Get(FieldEnterpriseGroupOrganizationID).(string)
		r.Equal(organizationID, actualOrganizationID)

		actualGroupName := data.Get(FieldEnterpriseGroupName).(string)
		r.Equal("engineering-team", actualGroupName)

		actualGroupDescription := data.Get(FieldEnterpriseGroupDescription).(string)
		r.Equal("Engineering team group", actualGroupDescription)

		// Verify members
		members := data.Get(FieldEnterpriseGroupMembers).([]any)
		r.Len(members, 1) // Single wrapper
		memberWrapper := members[0].(map[string]any)
		memberList := memberWrapper[FieldEnterpriseGroupsMember].([]any)
		r.Len(memberList, 3)
		r.Equal(memberID1, memberList[0].(map[string]any)[FieldEnterpriseGroupMemberID])
		r.Equal("user", memberList[0].(map[string]any)[FieldEnterpriseGroupMemberKind])
		r.Equal(memberID2, memberList[1].(map[string]any)[FieldEnterpriseGroupMemberID])
		r.Equal("user", memberList[1].(map[string]any)[FieldEnterpriseGroupMemberKind])
		r.Equal(memberID3, memberList[2].(map[string]any)[FieldEnterpriseGroupMemberID])
		r.Equal("service_account", memberList[2].(map[string]any)[FieldEnterpriseGroupMemberKind])

		// Verify role bindings
		roleBindings := data.Get(FieldEnterpriseGroupRoleBindings).([]any)
		r.Len(roleBindings, 1) // Single wrapper
		roleBindingWrapper := roleBindings[0].(map[string]any)
		roleBindingList := roleBindingWrapper[FieldEnterpriseGroupRoleBinding].([]any)
		r.Len(roleBindingList, 2)
		r.Equal(roleBindingID1, roleBindingList[0].(map[string]any)[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-viewer", roleBindingList[0].(map[string]any)[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID1, roleBindingList[0].(map[string]any)[FieldEnterpriseGroupRoleBindingRoleID])
		// Check scopes of first role binding
		scopes := roleBindingList[0].(map[string]any)[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes, 1) // Single wrapper
		scopeWrapper := scopes[0].(map[string]any)
		scopeList := scopeWrapper[FieldEnterpriseGroupScope].([]any)
		r.Len(scopeList, 3)
		r.Equal(clusterID1, scopeList[0].(map[string]any)[FieldEnterpriseGroupScopeCluster])
		r.Equal("", scopeList[0].(map[string]any)[FieldEnterpriseGroupScopeOrganization])
		r.Equal("", scopeList[1].(map[string]any)[FieldEnterpriseGroupScopeCluster])
		r.Equal(organizationID, scopeList[1].(map[string]any)[FieldEnterpriseGroupScopeOrganization])
		r.Equal(clusterID2, scopeList[2].(map[string]any)[FieldEnterpriseGroupScopeCluster])
		r.Equal("", scopeList[2].(map[string]any)[FieldEnterpriseGroupScopeOrganization])

		r.Equal(roleBindingID3, roleBindingList[1].(map[string]any)[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-editor", roleBindingList[1].(map[string]any)[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID3, roleBindingList[1].(map[string]any)[FieldEnterpriseGroupRoleBindingRoleID])

		// Check scopes of second role binding
		scopes2 := roleBindingList[1].(map[string]any)[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes2, 1) // Single wrapper
		scopeWrapper2 := scopes2[0].(map[string]any)
		scopeList2 := scopeWrapper2[FieldEnterpriseGroupScope].([]any)
		r.Len(scopeList2, 1)
		r.Equal("", scopeList2[0].(map[string]any)[FieldEnterpriseGroupScopeCluster])
		r.Equal(organizationID, scopeList2[0].(map[string]any)[FieldEnterpriseGroupScopeOrganization])
	})
}

func TestResourceEnterpriseGroupReadContext(t *testing.T) {
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
		groupID := uuid.NewString()
		organizationID := uuid.NewString()

		mockClient.EXPECT().
			EnterpriseAPIListGroupsWithResponse(gomock.Any(), enterpriseID, nil).
			Return(nil, errors.New("network error"))

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupID:             cty.StringVal(groupID),
			FieldEnterpriseGroupEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseGroupName:           cty.StringVal("test-group"),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseGroup()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("list enterprise groups failed: network error", result[0].Summary)
	})

	t.Run("when API returns groups then filter and update state with managed group only", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		groupID1 := uuid.NewString()
		groupID2 := uuid.NewString() // Group not in our state
		organizationID1 := uuid.NewString()
		organizationID2 := uuid.NewString()
		memberID1 := uuid.NewString()

		createTime := time.Now()

		// API returns both groups, but we should only keep the one in our state
		apiResponse := &organization_management.ListGroupsResponse{
			Items: &[]organization_management.ListGroupsResponseGroup{
				{
					Id:             lo.ToPtr(groupID2), // This group is NOT in our state
					Name:           lo.ToPtr("other-group"),
					OrganizationId: lo.ToPtr(organizationID2),
					Description:    lo.ToPtr("Not our group"),
				},
				{
					Id:             lo.ToPtr(groupID1),
					Name:           lo.ToPtr("managed-group"),
					OrganizationId: lo.ToPtr(organizationID1),
					Description:    lo.ToPtr("A managed group"),
					CreateTime:     lo.ToPtr(createTime),
					ManagedBy:      lo.ToPtr("terraform"),
					Definition: &organization_management.ListGroupsResponseGroupDefinition{
						Members: &[]organization_management.GroupDefinitionMember{
							{
								Id:        lo.ToPtr(memberID1),
								Email:     lo.ToPtr("test@example.com"),
								AddedTime: lo.ToPtr(createTime),
								Kind:      lo.ToPtr(organization_management.GroupDefinitionMemberKindKINDUSER),
							},
						},
					},
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIListGroupsWithResponse(gomock.Any(), enterpriseID, nil).
			Return(&organization_management.EnterpriseAPIListGroupsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiResponse,
			}, nil)

		// Mock role bindings API call
		mockClient.EXPECT().
			EnterpriseAPIListRoleBindingsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIListRoleBindingsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &organization_management.ListRoleBindingsResponse{
					Items: &[]organization_management.RoleBinding{}, // Empty role bindings for this test
				},
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupID:             cty.StringVal(groupID1), // Only this group is managed by us
			FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID1),
			FieldEnterpriseGroupName:           cty.StringVal("managed-group"),
			FieldEnterpriseGroupEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseGroupDescription:    cty.StringVal("A managed group"),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseGroup()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())

		r.Equal(groupID1, data.Id())
		r.Equal(enterpriseID, data.Get(FieldEnterpriseGroupEnterpriseID).(string))
		r.Equal(organizationID1, data.Get(FieldEnterpriseGroupOrganizationID).(string))
		r.Equal("managed-group", data.Get(FieldEnterpriseGroupName).(string))
		r.Equal("A managed group", data.Get(FieldEnterpriseGroupDescription).(string))
		r.Equal(1, len(data.Get(FieldEnterpriseGroupMembers).([]any)))
		memberWrapper := data.Get(FieldEnterpriseGroupMembers).([]any)[0].(map[string]any)
		memberList := memberWrapper[FieldEnterpriseGroupsMember].([]any)
		r.Len(memberList, 1)
		r.Equal(memberID1, memberList[0].(map[string]any)[FieldEnterpriseGroupMemberID])
		r.Equal("user", memberList[0].(map[string]any)[FieldEnterpriseGroupMemberKind])
	})

	t.Run("when API returns groups with multiple role bindings then include only tracked role bindings in state", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		groupID := uuid.NewString()
		memberID1 := uuid.NewString()
		memberID3 := uuid.NewString()
		roleBindingID1 := uuid.NewString()
		roleBindingID3 := uuid.NewString()
		roleID1 := uuid.NewString()
		roleID3 := uuid.NewString()
		clusterID1 := uuid.NewString()
		clusterID2 := uuid.NewString()

		createTime := time.Now()

		// API returns groups
		apiGroupsResponse := &organization_management.ListGroupsResponse{
			Items: &[]organization_management.ListGroupsResponseGroup{
				{
					Id:             lo.ToPtr(groupID),
					Name:           lo.ToPtr("engineering-team"),
					OrganizationId: lo.ToPtr(organizationID),
					Description:    lo.ToPtr("Engineering team group"),
					CreateTime:     lo.ToPtr(createTime),
					ManagedBy:      lo.ToPtr("terraform"),
					Definition: &organization_management.ListGroupsResponseGroupDefinition{
						Members: &[]organization_management.GroupDefinitionMember{
							{
								Id:        lo.ToPtr(memberID1),
								Email:     lo.ToPtr("engineer@example.com"),
								AddedTime: lo.ToPtr(createTime),
								Kind:      lo.ToPtr(organization_management.GroupDefinitionMemberKindKINDUSER),
							},
							{
								Id:        lo.ToPtr(memberID3),
								AddedTime: lo.ToPtr(createTime),
								Kind:      lo.ToPtr(organization_management.GroupDefinitionMemberKindKINDSERVICEACCOUNT),
							},
						},
					},
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIListGroupsWithResponse(gomock.Any(), enterpriseID, nil).
			Return(&organization_management.EnterpriseAPIListGroupsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiGroupsResponse,
			}, nil)

		apiRoleBindingsGroupResponse := &organization_management.ListRoleBindingsResponse{
			Items: &[]organization_management.RoleBinding{
				{
					Id:         lo.ToPtr(roleBindingID1),
					Name:       lo.ToPtr("engineering-viewer-1"),
					CreateTime: lo.ToPtr(createTime),
					ManagedBy:  lo.ToPtr("terraform-1"),
					Definition: &organization_management.RoleBindingDefinition{
						RoleId: lo.ToPtr(roleID1),
						Subjects: &[]organization_management.Subject{
							{
								Group: &organization_management.GroupSubject{
									Id:   groupID,
									Name: lo.ToPtr("engineering-team"),
								},
							},
						},
						Scopes: &[]organization_management.Scope{
							{
								Cluster: &organization_management.ClusterScope{
									Id: clusterID1,
								},
							},
							{
								Organization: &organization_management.OrganizationScope{
									Id: organizationID,
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
				{
					Id:         lo.ToPtr(roleBindingID3),
					Name:       lo.ToPtr("engineering-editor"),
					CreateTime: lo.ToPtr(createTime),
					ManagedBy:  lo.ToPtr("terraform"),
					Definition: &organization_management.RoleBindingDefinition{
						RoleId: lo.ToPtr(roleID3),
						Subjects: &[]organization_management.Subject{
							{
								Group: &organization_management.GroupSubject{
									Id:   groupID,
									Name: lo.ToPtr("engineering-team"),
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
			EnterpriseAPIListRoleBindingsWithResponse(
				gomock.Any(),
				enterpriseID,
				&organization_management.EnterpriseAPIListRoleBindingsParams{
					SubjectId: &[]string{groupID},
				},
			).
			Return(&organization_management.EnterpriseAPIListRoleBindingsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiRoleBindingsGroupResponse,
			}, nil)

		// State includes minimal group data - role bindings will be discovered
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupID:             cty.StringVal(groupID),
			FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseGroupName:           cty.StringVal("engineering-team"),
			FieldEnterpriseGroupEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseGroupMembers: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupsMember: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupMemberKind: cty.StringVal("user"),
							FieldEnterpriseGroupMemberID:   cty.StringVal(memberID1),
						}),
					}),
				}),
			}),
			FieldEnterpriseGroupRoleBindings: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupRoleBinding: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupRoleBindingID:     cty.StringVal(roleBindingID3),
							FieldEnterpriseGroupRoleBindingName:   cty.StringVal("engineering-editor"),
							FieldEnterpriseGroupRoleBindingRoleID: cty.StringVal(roleID3),
							FieldEnterpriseGroupRoleBindingScopes: cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									FieldEnterpriseGroupScope: cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											FieldEnterpriseGroupScopeOrganization: cty.StringVal(organizationID),
											FieldEnterpriseGroupScopeCluster:      cty.StringVal(""),
										}),
									}),
								}),
							}),
						}),
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupRoleBindingID:     cty.StringVal(roleBindingID1),
							FieldEnterpriseGroupRoleBindingName:   cty.StringVal("engineering-viewer"),
							FieldEnterpriseGroupRoleBindingRoleID: cty.StringVal(roleID1),
							FieldEnterpriseGroupRoleBindingScopes: cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									FieldEnterpriseGroupScope: cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											FieldEnterpriseGroupScopeCluster:      cty.StringVal(clusterID1),
											FieldEnterpriseGroupScopeOrganization: cty.StringVal(""),
										}),
									}),
								}),
							}),
						}),
					}),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseGroup()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())

		r.Equal(groupID, data.Id())
		r.Equal(enterpriseID, data.Get(FieldEnterpriseGroupEnterpriseID).(string))
		r.Equal(organizationID, data.Get(FieldEnterpriseGroupOrganizationID).(string))
		r.Equal("engineering-team", data.Get(FieldEnterpriseGroupName).(string))
		r.Equal("Engineering team group", data.Get(FieldEnterpriseGroupDescription).(string))

		members := data.Get(FieldEnterpriseGroupMembers).([]any)
		r.Len(members, 1) // Single group
		memberWrapper := members[0].(map[string]any)
		memberList := memberWrapper[FieldEnterpriseGroupsMember].([]any)
		r.Len(memberList, 2)
		member1 := memberList[0].(map[string]any)
		r.Equal(memberID1, member1[FieldEnterpriseGroupMemberID])
		r.Equal("user", member1[FieldEnterpriseGroupMemberKind])
		member2 := memberList[1].(map[string]any)
		r.Equal(memberID3, member2[FieldEnterpriseGroupMemberID])
		r.Equal("service_account", member2[FieldEnterpriseGroupMemberKind])

		roleBindings := data.Get(FieldEnterpriseGroupRoleBindings).([]any)
		r.Len(roleBindings, 1) // Single group
		roleBindingWrapper := roleBindings[0].(map[string]any)
		roleBindingList := roleBindingWrapper[FieldEnterpriseGroupRoleBinding].([]any)
		r.Len(roleBindingList, 2)
		roleBinding1 := roleBindingList[0].(map[string]any)
		r.Equal(roleBindingID3, roleBinding1[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-editor", roleBinding1[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID3, roleBinding1[FieldEnterpriseGroupRoleBindingRoleID])
		scopes1 := roleBinding1[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes1, 1)
		scope1a := scopes1[0].(map[string]any)
		scopeList1a := scope1a[FieldEnterpriseGroupScope].([]any)
		r.Len(scopeList1a, 1)
		scope1a1 := scopeList1a[0].(map[string]any)
		r.Equal(organizationID, scope1a1[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope1a1[FieldEnterpriseGroupScopeCluster])
		roleBinding2 := roleBindingList[1].(map[string]any)
		r.Equal(roleBindingID1, roleBinding2[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-viewer-1", roleBinding2[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID1, roleBinding2[FieldEnterpriseGroupRoleBindingRoleID])
		scopes2 := roleBinding2[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes2, 1)
		scope2a := scopes2[0].(map[string]any)
		scopeList2a := scope2a[FieldEnterpriseGroupScope].([]any)
		r.Len(scopeList2a, 3)
		scope2a1 := scopeList2a[0].(map[string]any)
		r.Equal(clusterID1, scope2a1[FieldEnterpriseGroupScopeCluster])
		r.Empty(scope2a1[FieldEnterpriseGroupScopeOrganization])
		scope2a2 := scopeList2a[1].(map[string]any)
		r.Equal(organizationID, scope2a2[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope2a2[FieldEnterpriseGroupScopeCluster])
		scope2a3 := scopeList2a[2].(map[string]any)
		r.Equal(clusterID2, scope2a3[FieldEnterpriseGroupScopeCluster])
		r.Empty(scope2a3[FieldEnterpriseGroupScopeOrganization])
	})
}

func TestResourceEnterpriseGroupDeleteContext(t *testing.T) {
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
		groupID := uuid.NewString()

		mockClient.EXPECT().
			EnterpriseAPIBatchDeleteEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(nil, errors.New("network error"))

		// State with 1 group
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseGroupID:             cty.StringVal(groupID),
			FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseGroupName:           cty.StringVal("engineering-team"),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = enterpriseID

		resource := resourceEnterpriseGroup()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.True(result.HasError())
		r.Equal("batch delete enterprise groups failed: network error", result[0].Summary)
		r.NotEmpty(data.Id(), "Resource ID should not be cleared when delete fails")
	})

	t.Run("when API successfully deletes groups then clear state", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID := uuid.NewString()
		groupID := uuid.NewString()

		// Expected delete request
		expectedRequest := organization_management.BatchDeleteEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest{
				{
					Id:             groupID,
					OrganizationId: organizationID,
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIBatchDeleteEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, expectedRequest).
			Return(&organization_management.EnterpriseAPIBatchDeleteEnterpriseGroupsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil)

		// State with 2 groups
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseGroupID:             cty.StringVal(groupID),
			FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID),
			FieldEnterpriseGroupName:           cty.StringVal("engineering-team"),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = enterpriseID

		resource := resourceEnterpriseGroup()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Empty(data.Id(), "Resource ID should be cleared after successful delete")
	})
}

func TestResourceEnterpriseGroupUpdateContext(t *testing.T) {
	t.Parallel()

	enterpriseID := uuid.NewString()
	organizationID := uuid.NewString()
	existingGroupID := uuid.NewString()
	ctx := context.Background()

	t.Run("when scopes include both organization and cluster scopes, return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		resource := resourceEnterpriseGroup()

		diff := map[string]any{
			FieldEnterpriseGroupEnterpriseID:   enterpriseID,
			FieldEnterpriseGroupOrganizationID: organizationID,
			FieldEnterpriseGroupName:           "engineering-team",
			FieldEnterpriseGroupDescription:    "Engineering team group",

			FieldEnterpriseGroupRoleBindings: []any{
				map[string]any{
					FieldEnterpriseGroupRoleBinding: []any{
						map[string]any{
							FieldEnterpriseGroupRoleBindingName:   "engineering-viewer",
							FieldEnterpriseGroupRoleBindingRoleID: "role-1",
							FieldEnterpriseGroupRoleBindingScopes: []any{
								map[string]any{
									FieldEnterpriseGroupScope: []any{
										map[string]any{
											FieldEnterpriseGroupScopeCluster:      uuid.NewString(),
											FieldEnterpriseGroupScopeOrganization: uuid.NewString(),
										},
									},
								},
							},
						},
					},
				},
			},
		}
		data := schema.TestResourceDataRaw(t, resource.Schema, diff)
		data.SetId(existingGroupID)

		result := resource.UpdateContext(ctx, data, provider)
		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("reading group data: scope cannot have both 'organization' and 'cluster' set simultaneously", result[0].Summary)
	})

	t.Run("when there are changes then update resource", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		member1ID := uuid.NewString()
		member2ID := uuid.NewString()
		member3ID := uuid.NewString()
		roleBinding1ID := uuid.NewString()
		roleBinding1RoleID := uuid.NewString()
		roleBinding2ID := uuid.NewString()
		roleBinding2RoleID := uuid.NewString()
		clusterID := uuid.NewString()

		expectedRequest := organization_management.BatchUpdateEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest{
				{
					Id:             existingGroupID,
					OrganizationId: organizationID,
					Name:           "engineering-team",
					Description:    "Engineering team group",
					Members: []organization_management.BatchUpdateEnterpriseGroupsRequestMember{
						{
							Id:   member1ID,
							Kind: organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindUSER,
						},
						{
							Id:   member2ID,
							Kind: organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindSERVICEACCOUNT,
						},
						{
							Id:   member3ID,
							Kind: organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindSERVICEACCOUNT,
						},
					},
					RoleBindings: []organization_management.BatchUpdateEnterpriseGroupsRequestRoleBinding{
						{
							Name:   "engineering-viewer",
							RoleId: roleBinding1RoleID,
							Scopes: []organization_management.Scope{
								{
									Organization: &organization_management.OrganizationScope{
										Id: organizationID,
									},
								},
								{
									Cluster: &organization_management.ClusterScope{
										Id: clusterID,
									},
								},
							},
						},
						{
							Name:   "engineering-editor",
							RoleId: roleBinding2RoleID,
							Scopes: []organization_management.Scope{
								{
									Organization: &organization_management.OrganizationScope{
										Id: organizationID,
									},
								},
							},
						},
					},
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIBatchUpdateEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Do(func(ctx context.Context, enterpriseID string, req organization_management.BatchUpdateEnterpriseGroupsRequest) {
				// Additional validation of request can be done here if needed
				r.Equal(expectedRequest, req)
			}).
			Return(&organization_management.EnterpriseAPIBatchUpdateEnterpriseGroupsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200: &organization_management.BatchUpdateEnterpriseGroupsResponse{
					Groups: &[]organization_management.BatchUpdateEnterpriseGroupsResponseGroup{
						{
							Id:             lo.ToPtr(existingGroupID),
							OrganizationId: lo.ToPtr(organizationID),
							Name:           lo.ToPtr("engineering-team"),
							Description:    lo.ToPtr("Engineering team group"),
							ManagedBy:      lo.ToPtr("terraform"),
							Definition: &organization_management.GroupDefinition{
								Members: &[]organization_management.DefinitionMember{
									{
										Id:        lo.ToPtr(member1ID),
										Email:     lo.ToPtr("member1@example.com"),
										AddedTime: lo.ToPtr(time.Now()),
										Kind:      lo.ToPtr(organization_management.DefinitionMemberKind("USER")),
									},
									{
										Id:        lo.ToPtr(member2ID),
										AddedTime: lo.ToPtr(time.Now()),
										Kind:      lo.ToPtr(organization_management.DefinitionMemberKind("SERVICE_ACCOUNT")),
									},
									{
										Id:        lo.ToPtr(member3ID),
										AddedTime: lo.ToPtr(time.Now()),
										Kind:      lo.ToPtr(organization_management.DefinitionMemberKind("SERVICE_ACCOUNT")),
									},
								},
							},
							RoleBindings: &[]organization_management.GroupRoleBinding{
								{
									Id:             roleBinding1ID,
									Name:           "engineering-viewer",
									CreateTime:     time.Now(),
									ManagedBy:      "terraform",
									OrganizationId: organizationID,
									Definition: organization_management.RoleBindingRoleBindingDefinition{
										RoleId: roleBinding1RoleID,
										Scopes: &[]organization_management.Scope{
											{
												Organization: &organization_management.OrganizationScope{
													Id: organizationID,
												},
											},
											{
												Cluster: &organization_management.ClusterScope{
													Id: clusterID,
												},
											},
										},
									},
								},
								{
									Id:             roleBinding2ID,
									Name:           "engineering-editor",
									CreateTime:     time.Now(),
									ManagedBy:      "terraform",
									OrganizationId: organizationID,
									Definition: organization_management.RoleBindingRoleBindingDefinition{
										RoleId: roleBinding2RoleID,
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
						},
					},
				},
			}, nil)

		resource := resourceEnterpriseGroup()

		diff := map[string]any{
			FieldEnterpriseGroupEnterpriseID:   enterpriseID,
			FieldEnterpriseGroupOrganizationID: organizationID,
			FieldEnterpriseGroupName:           "engineering-team",
			FieldEnterpriseGroupDescription:    "Engineering team group",
			FieldEnterpriseGroupMembers: []any{
				map[string]any{
					FieldEnterpriseGroupsMember: []any{
						map[string]any{
							FieldEnterpriseGroupMemberID:   member1ID,
							FieldEnterpriseGroupMemberKind: "user",
						},
						map[string]any{
							FieldEnterpriseGroupMemberID:   member2ID,
							FieldEnterpriseGroupMemberKind: "service_account",
						},
						map[string]any{
							FieldEnterpriseGroupMemberID:   member3ID,
							FieldEnterpriseGroupMemberKind: "service_account",
						},
					},
				},
			},
			FieldEnterpriseGroupRoleBindings: []any{
				map[string]any{
					FieldEnterpriseGroupRoleBinding: []any{
						map[string]any{
							FieldEnterpriseGroupRoleBindingName:   "engineering-viewer",
							FieldEnterpriseGroupRoleBindingRoleID: roleBinding1RoleID,
							FieldEnterpriseGroupRoleBindingScopes: []any{
								map[string]any{
									FieldEnterpriseGroupScope: []any{
										map[string]any{
											FieldEnterpriseGroupScopeCluster:      "",
											FieldEnterpriseGroupScopeOrganization: organizationID,
										},
										map[string]any{
											FieldEnterpriseGroupScopeCluster:      clusterID,
											FieldEnterpriseGroupScopeOrganization: "",
										},
									},
								},
							},
						},

						map[string]any{
							FieldEnterpriseGroupRoleBindingName:   "engineering-editor",
							FieldEnterpriseGroupRoleBindingRoleID: roleBinding2RoleID,
							FieldEnterpriseGroupRoleBindingScopes: []any{
								map[string]any{
									FieldEnterpriseGroupScope: []any{
										map[string]any{
											FieldEnterpriseGroupScopeCluster:      "",
											FieldEnterpriseGroupScopeOrganization: organizationID,
										},
									},
								},
							},
						},
					},
				},
			},
		}
		data := schema.TestResourceDataRaw(t, resource.Schema, diff)
		data.SetId(existingGroupID)

		result := resource.UpdateContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal(existingGroupID, data.Id(), "Resource ID should not change")

		// Verify the updated values
		r.Equal("engineering-team", data.Get(FieldEnterpriseGroupName).(string))
		r.Equal("Engineering team group", data.Get(FieldEnterpriseGroupDescription).(string))
		members := data.Get(FieldEnterpriseGroupMembers).([]any)
		r.Len(members, 1)
		memberWrapper := members[0].(map[string]any)
		memberList := memberWrapper[FieldEnterpriseGroupsMember].([]any)
		r.Len(memberList, 3)
		member1 := memberList[0].(map[string]any)
		r.Equal(member1ID, member1[FieldEnterpriseGroupMemberID])
		r.Equal("user", member1[FieldEnterpriseGroupMemberKind])
		member2 := memberList[1].(map[string]any)
		r.Equal(member2ID, member2[FieldEnterpriseGroupMemberID])
		r.Equal("service_account", member2[FieldEnterpriseGroupMemberKind])
		member3 := memberList[2].(map[string]any)
		r.Equal(member3ID, member3[FieldEnterpriseGroupMemberID])
		r.Equal("service_account", member3[FieldEnterpriseGroupMemberKind])
		roleBindings := data.Get(FieldEnterpriseGroupRoleBindings).([]any)
		r.Len(roleBindings, 1)
		roleBindingWrapper := roleBindings[0].(map[string]any)
		roleBindingList := roleBindingWrapper[FieldEnterpriseGroupRoleBinding].([]any)
		r.Len(roleBindingList, 2)
		roleBinding1 := roleBindingList[0].(map[string]any)
		r.Equal(roleBinding1ID, roleBinding1[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-viewer", roleBinding1[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleBinding1RoleID, roleBinding1[FieldEnterpriseGroupRoleBindingRoleID])
		scopes1 := roleBinding1[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes1, 1)
		scope1a := scopes1[0].(map[string]any)
		scopeList1a := scope1a[FieldEnterpriseGroupScope].([]any)
		r.Len(scopeList1a, 2)
		scope1a1 := scopeList1a[0].(map[string]any)
		r.Equal("", scope1a1[FieldEnterpriseGroupScopeCluster])
		r.Equal(organizationID, scope1a1[FieldEnterpriseGroupScopeOrganization])
		scope1a2 := scopeList1a[1].(map[string]any)
		r.Equal(clusterID, scope1a2[FieldEnterpriseGroupScopeCluster])
		r.Equal("", scope1a2[FieldEnterpriseGroupScopeOrganization])
		roleBinding2 := roleBindingList[1].(map[string]any)
		r.Equal(roleBinding2ID, roleBinding2[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-editor", roleBinding2[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleBinding2RoleID, roleBinding2[FieldEnterpriseGroupRoleBindingRoleID])
		scopes2 := roleBinding2[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes2, 1)
		scope2a := scopes2[0].(map[string]any)
		scopeList2a := scope2a[FieldEnterpriseGroupScope].([]any)
		r.Len(scopeList2a, 1)
		scope2a1 := scopeList2a[0].(map[string]any)
		r.Equal("", scope2a1[FieldEnterpriseGroupScopeCluster])
		r.Equal(organizationID, scope2a1[FieldEnterpriseGroupScopeOrganization])
	})
}
