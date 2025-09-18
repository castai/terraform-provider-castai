package castai

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/organization_management"
	mockOrganizationManagement "github.com/castai/terraform-provider-castai/castai/sdk/organization_management/mock"
)

func TestResourceEnterpriseGroupsCreate(t *testing.T) {
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

		mockClient.EXPECT().
			EnterpriseAPIBatchCreateEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(nil, errors.New("network error"))

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsEnterpriseID: cty.StringVal(enterpriseID),
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID),
					FieldEnterpriseGroupName:           cty.StringVal("test-group"),
					FieldEnterpriseGroupDescription:    cty.StringVal("Test description"),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("calling batch create enterprise groups: network error", result[0].Summary)
	})

	t.Run("when API returns empty response then return error", func(t *testing.T) {
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
			Return(&organization_management.EnterpriseAPIBatchCreateEnterpriseGroupsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil, // Empty response
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsEnterpriseID: cty.StringVal(enterpriseID),
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID),
					FieldEnterpriseGroupName:           cty.StringVal("test-group"),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("unexpected empty response from batch create", result[0].Summary)
	})

	t.Run("when API successfully creates groups then set state correctly", func(t *testing.T) {
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
		groupID1 := "aaaa1111-1111-1111-1111-111111111111" // Will sort first
		groupID2 := "bbbb2222-2222-2222-2222-222222222222" // Will sort second
		memberID1 := "b" + uuid.NewString()
		memberID2 := "a" + uuid.NewString()
		memberID3 := "c" + uuid.NewString()
		roleBindingID1 := uuid.NewString()
		roleBindingID2 := uuid.NewString()
		roleBindingID3 := uuid.NewString() // Second role binding for first group
		roleID1 := "b" + uuid.NewString()
		roleID2 := "a" + uuid.NewString()
		roleID3 := "c" + uuid.NewString() // Role for second role binding
		clusterID1 := "b" + uuid.NewString()
		clusterID2 := "a" + uuid.NewString()

		createTime := time.Now()

		// Mock API response
		apiResponse := &organization_management.BatchCreateEnterpriseGroupsResponse{
			Groups: &[]organization_management.BatchCreateEnterpriseGroupsResponseGroup{
				{
					Id:             lo.ToPtr(groupID1),
					Name:           lo.ToPtr("engineering-team"),
					OrganizationId: lo.ToPtr(organizationID1),
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
							OrganizationId: organizationID1,
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
											Id: organizationID1,
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
							OrganizationId: organizationID1,
							Statuses:       []organization_management.RoleBindingRoleBindingStatus{},
							Definition: organization_management.RoleBindingRoleBindingDefinition{
								RoleId: roleID3,
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
								},
							},
						},
					},
				},
				{
					Id:             lo.ToPtr(groupID2),
					Name:           lo.ToPtr("security-team"),
					OrganizationId: lo.ToPtr(organizationID2),
					Description:    lo.ToPtr("Security team group"),
					CreateTime:     lo.ToPtr(createTime),
					ManagedBy:      lo.ToPtr("terraform"),
					Definition: &organization_management.GroupDefinition{
						Members: &[]organization_management.DefinitionMember{
							{
								Id:        lo.ToPtr(memberID2),
								Email:     lo.ToPtr("security@example.com"),
								AddedTime: lo.ToPtr(createTime),
								Kind:      lo.ToPtr(organization_management.DefinitionMemberKindSUBJECTKINDUSER),
							},
						},
					},
					RoleBindings: &[]organization_management.GroupRoleBinding{
						{
							Id:             roleBindingID2,
							Name:           "security-auditor",
							Description:    "Security auditor role binding",
							ManagedBy:      "terraform",
							CreateTime:     createTime,
							OrganizationId: organizationID2,
							Statuses:       []organization_management.RoleBindingRoleBindingStatus{},
							Definition: organization_management.RoleBindingRoleBindingDefinition{
								RoleId: roleID2,
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
								},
							},
						},
					},
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIBatchCreateEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIBatchCreateEnterpriseGroupsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiResponse,
			}, nil)

		// Input state - what user defined in Terraform
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsEnterpriseID: cty.StringVal(enterpriseID),
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID1),
					FieldEnterpriseGroupName:           cty.StringVal("engineering-team"),
					FieldEnterpriseGroupDescription:    cty.StringVal("Engineering team group"),
					FieldEnterpriseGroupMembers: cty.ListVal([]cty.Value{
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
					FieldEnterpriseGroupRoleBindings: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupRoleBindingName:   cty.StringVal("engineering-viewer"),
							FieldEnterpriseGroupRoleBindingRoleID: cty.StringVal(roleID1),
							FieldEnterpriseGroupRoleBindingScopes: cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									FieldEnterpriseGroupScopeCluster:      cty.StringVal(clusterID1),
									FieldEnterpriseGroupScopeOrganization: cty.StringVal(""),
								}),
								cty.ObjectVal(map[string]cty.Value{
									FieldEnterpriseGroupScopeCluster:      cty.StringVal(""),
									FieldEnterpriseGroupScopeOrganization: cty.StringVal(organizationID1),
								}),
								cty.ObjectVal(map[string]cty.Value{
									FieldEnterpriseGroupScopeCluster:      cty.StringVal(clusterID2),
									FieldEnterpriseGroupScopeOrganization: cty.StringVal(""),
								}),
							}),
						}),
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupRoleBindingName:   cty.StringVal("engineering-editor"),
							FieldEnterpriseGroupRoleBindingRoleID: cty.StringVal(roleID3),
							FieldEnterpriseGroupRoleBindingScopes: cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									FieldEnterpriseGroupScopeCluster:      cty.StringVal(""),
									FieldEnterpriseGroupScopeOrganization: cty.StringVal(organizationID1),
								}),
								cty.ObjectVal(map[string]cty.Value{
									FieldEnterpriseGroupScopeCluster:      cty.StringVal(""),
									FieldEnterpriseGroupScopeOrganization: cty.StringVal(organizationID2),
								}),
							}),
						}),
					}),
				}),
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID2),
					FieldEnterpriseGroupName:           cty.StringVal("security-team"),
					FieldEnterpriseGroupDescription:    cty.StringVal("Security team group"),
					FieldEnterpriseGroupMembers: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupMemberKind: cty.StringVal("user"),
							FieldEnterpriseGroupMemberID:   cty.StringVal(memberID2),
						}),
					}),
					FieldEnterpriseGroupRoleBindings: cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							FieldEnterpriseGroupRoleBindingName:   cty.StringVal("security-auditor"),
							FieldEnterpriseGroupRoleBindingRoleID: cty.StringVal(roleID2),
							FieldEnterpriseGroupRoleBindingScopes: cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									FieldEnterpriseGroupScopeCluster:      cty.StringVal(""),
									FieldEnterpriseGroupScopeOrganization: cty.StringVal(organizationID1),
								}),
								cty.ObjectVal(map[string]cty.Value{
									FieldEnterpriseGroupScopeCluster:      cty.StringVal(""),
									FieldEnterpriseGroupScopeOrganization: cty.StringVal(organizationID2),
								}),
							}),
						}),
					}),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		// Verify no errors
		r.Nil(result)
		r.False(result.HasError())

		// Verify resource ID is set to enterprise ID
		r.Equal(enterpriseID, data.Id())

		// Verify groups state
		groups := data.Get(FieldEnterpriseGroupsGroups).([]any)
		r.Len(groups, 2)

		// Verify first group (should be sorted by ID)
		group1 := groups[0].(map[string]any)
		r.Equal(groupID1, group1[FieldEnterpriseGroupID])
		r.Equal("engineering-team", group1[FieldEnterpriseGroupName])
		r.Equal(organizationID1, group1[FieldEnterpriseGroupOrganizationID])
		r.Equal("Engineering team group", group1[FieldEnterpriseGroupDescription])
		r.Equal(createTime.Format(time.RFC3339), group1[FieldEnterpriseGroupCreateTime])
		r.Equal("terraform", group1[FieldEnterpriseGroupManagedBy])

		// Verify first group members (should be 3 members, in API response order)
		members1 := group1[FieldEnterpriseGroupMembers].([]any)
		r.Len(members1, 3)

		// Members should match the order they were provided in the API response
		// API response has: memberID1, memberID2, memberID3

		// First member (memberID1)
		member1 := members1[0].(map[string]any)
		r.Equal(memberID1, member1[FieldEnterpriseGroupMemberID])
		r.Equal("engineer@example.com", member1[FieldEnterpriseGroupMemberEmail])
		r.Equal("user", member1[FieldEnterpriseGroupMemberKind])
		r.Equal(createTime.Format(time.RFC3339), member1[FieldEnterpriseGroupMemberAddedTime])

		// Second member (memberID2)
		member2 := members1[1].(map[string]any)
		r.Equal(memberID2, member2[FieldEnterpriseGroupMemberID])
		r.Equal("security@example.com", member2[FieldEnterpriseGroupMemberEmail])
		r.Equal("user", member2[FieldEnterpriseGroupMemberKind])
		r.Equal(createTime.Format(time.RFC3339), member2[FieldEnterpriseGroupMemberAddedTime])

		// Third member (memberID3 - service account with no email)
		member3 := members1[2].(map[string]any)
		r.Equal(memberID3, member3[FieldEnterpriseGroupMemberID])
		r.Empty(member3[FieldEnterpriseGroupMemberEmail]) // Service account has no email
		r.Equal("service_account", member3[FieldEnterpriseGroupMemberKind])
		r.Equal(createTime.Format(time.RFC3339), member3[FieldEnterpriseGroupMemberAddedTime])

		// Verify first group role bindings (should be 2 role bindings)
		roleBindings1 := group1[FieldEnterpriseGroupRoleBindings].([]any)
		r.Len(roleBindings1, 2)

		// First role binding (in API response order)
		roleBinding1a := roleBindings1[0].(map[string]any)
		r.Equal(roleBindingID1, roleBinding1a[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-viewer", roleBinding1a[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID1, roleBinding1a[FieldEnterpriseGroupRoleBindingRoleID])

		// Second role binding
		roleBinding1b := roleBindings1[1].(map[string]any)
		r.Equal(roleBindingID3, roleBinding1b[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-editor", roleBinding1b[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID3, roleBinding1b[FieldEnterpriseGroupRoleBindingRoleID])

		// Verify first role binding scopes (3 scopes: cluster, org, cluster)
		scopes1a := roleBinding1a[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes1a, 3)

		// First scope (cluster scope with clusterID1)
		scope1a1 := scopes1a[0].(map[string]any)
		r.Equal(clusterID1, scope1a1[FieldEnterpriseGroupScopeCluster])
		r.Empty(scope1a1[FieldEnterpriseGroupScopeOrganization])

		// Second scope (organization scope with organizationID1)
		scope1a2 := scopes1a[1].(map[string]any)
		r.Equal(organizationID1, scope1a2[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope1a2[FieldEnterpriseGroupScopeCluster])

		// Third scope (cluster scope with clusterID2)
		scope1a3 := scopes1a[2].(map[string]any)
		r.Equal(clusterID2, scope1a3[FieldEnterpriseGroupScopeCluster])
		r.Empty(scope1a3[FieldEnterpriseGroupScopeOrganization])

		// Verify second role binding scopes (2 organization scopes)
		scopes1b := roleBinding1b[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes1b, 2)

		// First scope (organizationID1)
		scope1b1 := scopes1b[0].(map[string]any)
		r.Equal(organizationID1, scope1b1[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope1b1[FieldEnterpriseGroupScopeCluster])

		// Second scope (organizationID2)
		scope1b2 := scopes1b[1].(map[string]any)
		r.Equal(organizationID2, scope1b2[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope1b2[FieldEnterpriseGroupScopeCluster])

		// Verify second group
		group2 := groups[1].(map[string]any)
		r.Equal(groupID2, group2[FieldEnterpriseGroupID])
		r.Equal("security-team", group2[FieldEnterpriseGroupName])
		r.Equal(organizationID2, group2[FieldEnterpriseGroupOrganizationID])
		r.Equal("Security team group", group2[FieldEnterpriseGroupDescription])
		r.Equal(createTime.Format(time.RFC3339), group2[FieldEnterpriseGroupCreateTime])
		r.Equal("terraform", group2[FieldEnterpriseGroupManagedBy])

		// Verify second group members
		members2 := group2[FieldEnterpriseGroupMembers].([]any)
		r.Len(members2, 1)
		member2Second := members2[0].(map[string]any)
		r.Equal(memberID2, member2Second[FieldEnterpriseGroupMemberID])
		r.Equal("security@example.com", member2Second[FieldEnterpriseGroupMemberEmail])
		r.Equal("user", member2Second[FieldEnterpriseGroupMemberKind])
		r.Equal(createTime.Format(time.RFC3339), member2Second[FieldEnterpriseGroupMemberAddedTime])

		// Verify second group role bindings
		roleBindings2 := group2[FieldEnterpriseGroupRoleBindings].([]any)
		r.Len(roleBindings2, 1)
		roleBinding2 := roleBindings2[0].(map[string]any)
		r.Equal(roleBindingID2, roleBinding2[FieldEnterpriseGroupRoleBindingID])
		r.Equal("security-auditor", roleBinding2[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID2, roleBinding2[FieldEnterpriseGroupRoleBindingRoleID])

		scopes2 := roleBinding2[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes2, 2)

		// First scope (organizationID1)
		scope2a := scopes2[0].(map[string]any)
		r.Equal(organizationID1, scope2a[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope2a[FieldEnterpriseGroupScopeCluster]) // Should be empty

		// Second scope (organizationID2)
		scope2b := scopes2[1].(map[string]any)
		r.Equal(organizationID2, scope2b[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope2b[FieldEnterpriseGroupScopeCluster]) // Should be empty
	})
}

func TestEnterpriseGroupsResourceReadContext(t *testing.T) {
	t.Parallel()

	t.Run("when state is missing enterprise ID then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)

		ctx := context.Background()
		provider := &ProviderConfig{}

		stateValue := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("enterprise ID is not set", result[0].Summary)
	})

	t.Run("when no groups in state then return empty state", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)

		ctx := context.Background()
		provider := &ProviderConfig{}

		enterpriseID := uuid.NewString()

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsGroups: cty.ListValEmpty(cty.Object(map[string]cty.Type{
				FieldEnterpriseGroupID:             cty.String,
				FieldEnterpriseGroupOrganizationID: cty.String,
				FieldEnterpriseGroupName:           cty.String,
			})),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = enterpriseID

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())

		groups := data.Get(FieldEnterpriseGroupsGroups).([]any)
		r.Empty(groups)
	})

	t.Run("when API returns 404 then remove from state", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		groupID1 := uuid.NewString()

		body := io.NopCloser(bytes.NewReader([]byte("")))

		mockClient.EXPECT().
			EnterpriseAPIListGroupsWithResponse(gomock.Any(), enterpriseID, nil).
			Return(&organization_management.EnterpriseAPIListGroupsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusNotFound, Body: body},
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupID:             cty.StringVal(groupID1),
					FieldEnterpriseGroupOrganizationID: cty.StringVal(uuid.NewString()),
					FieldEnterpriseGroupName:           cty.StringVal("test-group"),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = enterpriseID

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Empty(data.Id()) // Should clear the resource ID
	})

	t.Run("when API returns 500 then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		groupID1 := uuid.NewString()

		body := io.NopCloser(bytes.NewReader([]byte("internal error")))

		mockClient.EXPECT().
			EnterpriseAPIListGroupsWithResponse(gomock.Any(), enterpriseID, nil).
			Return(&organization_management.EnterpriseAPIListGroupsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError, Body: body},
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupID:             cty.StringVal(groupID1),
					FieldEnterpriseGroupOrganizationID: cty.StringVal(uuid.NewString()),
					FieldEnterpriseGroupName:           cty.StringVal("test-group"),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = enterpriseID

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("list enterprise groups failed with status 500: internal error", result[0].Summary)
	})

	t.Run("when API call throws error then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		groupID1 := uuid.NewString()

		mockClient.EXPECT().
			EnterpriseAPIListGroupsWithResponse(gomock.Any(), enterpriseID, nil).
			Return(nil, errors.New("network error"))

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupID:             cty.StringVal(groupID1),
					FieldEnterpriseGroupOrganizationID: cty.StringVal(uuid.NewString()),
					FieldEnterpriseGroupName:           cty.StringVal("test-group"),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = enterpriseID

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Len(result, 1)
		r.Equal("listing enterprise groups: network error", result[0].Summary)
	})

	t.Run("when API returns empty response then clear state", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		groupID1 := uuid.NewString()

		mockClient.EXPECT().
			EnterpriseAPIListGroupsWithResponse(gomock.Any(), enterpriseID, nil).
			Return(&organization_management.EnterpriseAPIListGroupsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil, // Empty response
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupID:             cty.StringVal(groupID1),
					FieldEnterpriseGroupOrganizationID: cty.StringVal(uuid.NewString()),
					FieldEnterpriseGroupName:           cty.StringVal("test-group"),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = enterpriseID

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())

		groups := data.Get(FieldEnterpriseGroupsGroups).([]any)
		r.Empty(groups)
	})

	t.Run("when API returns groups then filter and update state with managed groups only", func(t *testing.T) {
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

		addedTime := time.Now()

		// API returns both groups, but we should only keep the one in our state
		apiResponse := &organization_management.ListGroupsResponse{
			Items: &[]organization_management.ListGroupsResponseGroup{
				{
					Id:             lo.ToPtr(groupID1),
					Name:           lo.ToPtr("managed-group"),
					OrganizationId: lo.ToPtr(organizationID1),
					Description:    lo.ToPtr("A managed group"),
					CreateTime:     lo.ToPtr(addedTime),
					ManagedBy:      lo.ToPtr("terraform"),
					Definition: &organization_management.ListGroupsResponseGroupDefinition{
						Members: &[]organization_management.GroupDefinitionMember{
							{
								Id:        lo.ToPtr(memberID1),
								Email:     lo.ToPtr("test@example.com"),
								AddedTime: lo.ToPtr(addedTime),
								Kind:      lo.ToPtr(organization_management.GroupDefinitionMemberKindKINDUSER),
							},
						},
					},
				},
				{
					Id:             lo.ToPtr(groupID2), // This group is NOT in our state
					Name:           lo.ToPtr("other-group"),
					OrganizationId: lo.ToPtr(organizationID2),
					Description:    lo.ToPtr("Not our group"),
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

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupID:             cty.StringVal(groupID1), // Only this group is managed by us
					FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID1),
					FieldEnterpriseGroupName:           cty.StringVal("managed-group"),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = enterpriseID

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())

		// Verify only the managed group is in state
		groups := data.Get(FieldEnterpriseGroupsGroups).([]any)
		r.Len(groups, 1)

		group := groups[0].(map[string]any)
		r.Equal(groupID1, group[FieldEnterpriseGroupID])
		r.Equal("managed-group", group[FieldEnterpriseGroupName])
		r.Equal(organizationID1, group[FieldEnterpriseGroupOrganizationID])
		r.Equal("A managed group", group[FieldEnterpriseGroupDescription])
		r.Equal("terraform", group[FieldEnterpriseGroupManagedBy])

		// Verify members are included
		members := group[FieldEnterpriseGroupMembers].([]any)
		r.Len(members, 1)
		member := members[0].(map[string]any)
		r.Equal(memberID1, member[FieldEnterpriseGroupMemberID])
		r.Equal("test@example.com", member[FieldEnterpriseGroupMemberEmail])
		r.Equal("user", member[FieldEnterpriseGroupMemberKind])
		r.Equal(addedTime.Format(time.RFC3339), member[FieldEnterpriseGroupMemberAddedTime])
	})
}
