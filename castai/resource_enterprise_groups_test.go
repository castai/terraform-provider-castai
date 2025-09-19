package castai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
		organizationID1 := "e" + uuid.NewString()
		organizationID2 := "f" + uuid.NewString()
		groupID1 := "bbbb1111-1111-1111-1111-111111111111"
		groupID2 := "aaaa2222-2222-2222-2222-222222222222"
		memberID1 := "b" + uuid.NewString()
		memberID2 := "a" + uuid.NewString()
		memberID3 := "c" + uuid.NewString()
		roleBindingID1 := "c" + uuid.NewString()
		roleBindingID2 := "a" + uuid.NewString()
		roleBindingID3 := "b" + uuid.NewString()
		roleID1 := "b" + uuid.NewString()
		roleID2 := "a" + uuid.NewString()
		roleID3 := "c" + uuid.NewString()
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

		// Verify first group after sorting (groupID2 starts with "aaaa", comes first)
		group1 := groups[0].(map[string]any)
		r.Equal(groupID2, group1[FieldEnterpriseGroupID])
		r.Equal("security-team", group1[FieldEnterpriseGroupName])
		r.Equal(organizationID2, group1[FieldEnterpriseGroupOrganizationID])
		r.Equal("Security team group", group1[FieldEnterpriseGroupDescription])
		r.Equal(createTime.Format(time.RFC3339), group1[FieldEnterpriseGroupCreateTime])
		r.Equal("terraform", group1[FieldEnterpriseGroupManagedBy])

		// Verify first group members (group2/security-team has 1 member)
		members1 := group1[FieldEnterpriseGroupMembers].([]any)
		r.Len(members1, 1)

		// Single member (memberID2)
		member1 := members1[0].(map[string]any)
		r.Equal(memberID2, member1[FieldEnterpriseGroupMemberID])
		r.Equal("security@example.com", member1[FieldEnterpriseGroupMemberEmail])
		r.Equal("user", member1[FieldEnterpriseGroupMemberKind])
		r.Equal(createTime.Format(time.RFC3339), member1[FieldEnterpriseGroupMemberAddedTime])

		// Verify first group role bindings (security-team has 1 role binding)
		roleBindings1 := group1[FieldEnterpriseGroupRoleBindings].([]any)
		r.Len(roleBindings1, 1)

		// Single role binding (security-auditor)
		roleBinding1 := roleBindings1[0].(map[string]any)
		r.Equal(roleBindingID2, roleBinding1[FieldEnterpriseGroupRoleBindingID])
		r.Equal("security-auditor", roleBinding1[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID2, roleBinding1[FieldEnterpriseGroupRoleBindingRoleID])

		// Verify role binding scopes (2 organization scopes)
		scopes1 := roleBinding1[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes1, 2)

		// First scope (organizationID1)
		scope1a := scopes1[0].(map[string]any)
		r.Equal(organizationID1, scope1a[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope1a[FieldEnterpriseGroupScopeCluster])

		// Second scope (organizationID2)
		scope1b := scopes1[1].(map[string]any)
		r.Equal(organizationID2, scope1b[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope1b[FieldEnterpriseGroupScopeCluster])

		// Verify second group after sorting (groupID1 starts with "bbbb", comes second)
		group2 := groups[1].(map[string]any)
		r.Equal(groupID1, group2[FieldEnterpriseGroupID])
		r.Equal("engineering-team", group2[FieldEnterpriseGroupName])
		r.Equal(organizationID1, group2[FieldEnterpriseGroupOrganizationID])
		r.Equal("Engineering team group", group2[FieldEnterpriseGroupDescription])
		r.Equal(createTime.Format(time.RFC3339), group2[FieldEnterpriseGroupCreateTime])
		r.Equal("terraform", group2[FieldEnterpriseGroupManagedBy])

		// Verify second group members (engineering-team has 3 members)
		members2 := group2[FieldEnterpriseGroupMembers].([]any)
		r.Len(members2, 3)

		// Members should be sorted by ID: memberID2 ("a"...), memberID1 ("b"...), memberID3 ("c"...)
		member2First := members2[0].(map[string]any)
		r.Equal(memberID1, member2First[FieldEnterpriseGroupMemberID])
		r.Equal("engineer@example.com", member2First[FieldEnterpriseGroupMemberEmail])
		r.Equal("user", member2First[FieldEnterpriseGroupMemberKind])
		r.Equal(createTime.Format(time.RFC3339), member2First[FieldEnterpriseGroupMemberAddedTime])

		member2Second := members2[1].(map[string]any)
		r.Equal(memberID2, member2Second[FieldEnterpriseGroupMemberID])
		r.Equal("security@example.com", member2Second[FieldEnterpriseGroupMemberEmail])
		r.Equal("user", member2Second[FieldEnterpriseGroupMemberKind])
		r.Equal(createTime.Format(time.RFC3339), member2Second[FieldEnterpriseGroupMemberAddedTime])

		member2Third := members2[2].(map[string]any)
		r.Equal(memberID3, member2Third[FieldEnterpriseGroupMemberID])
		r.Empty(member2Third[FieldEnterpriseGroupMemberEmail]) // Service account has no email
		r.Equal("service_account", member2Third[FieldEnterpriseGroupMemberKind])
		r.Equal(createTime.Format(time.RFC3339), member2Third[FieldEnterpriseGroupMemberAddedTime])

		// Verify second group role bindings (engineering-team has 2 role bindings)
		roleBindings2 := group2[FieldEnterpriseGroupRoleBindings].([]any)
		r.Len(roleBindings2, 2)

		// First role binding after sorting (roleBindingID3 - starts with "b")
		roleBinding2a := roleBindings2[0].(map[string]any)
		r.Equal(roleBindingID3, roleBinding2a[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-editor", roleBinding2a[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID3, roleBinding2a[FieldEnterpriseGroupRoleBindingRoleID])

		// Verify scopes are sorted correctly for first role binding (2 organization scopes)
		scopes2a := roleBinding2a[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes2a, 2)

		// First scope: organization with organizationID1 ("e"...)
		scope2a1 := scopes2a[0].(map[string]any)
		r.Equal(organizationID1, scope2a1[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope2a1[FieldEnterpriseGroupScopeCluster])

		// Second scope: organization with organizationID2 ("f"...)
		scope2a2 := scopes2a[1].(map[string]any)
		r.Equal(organizationID2, scope2a2[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope2a2[FieldEnterpriseGroupScopeCluster])

		// Second role binding after sorting (roleBindingID1 - starts with "c")
		roleBinding2b := roleBindings2[1].(map[string]any)
		r.Equal(roleBindingID1, roleBinding2b[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-viewer", roleBinding2b[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID1, roleBinding2b[FieldEnterpriseGroupRoleBindingRoleID])

		// Verify scopes are sorted correctly (cluster scopes first, then by ID)
		scopes2b := roleBinding2b[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes2b, 3)

		scope2b1 := scopes2b[0].(map[string]any)
		r.Equal(clusterID1, scope2b1[FieldEnterpriseGroupScopeCluster])
		r.Empty(scope2b1[FieldEnterpriseGroupScopeOrganization])

		scope2b2 := scopes2b[1].(map[string]any)
		r.Equal(organizationID1, scope2b2[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope2b2[FieldEnterpriseGroupScopeCluster])

		scope2b3 := scopes2b[2].(map[string]any)
		r.Equal(clusterID2, scope2b3[FieldEnterpriseGroupScopeCluster])
		r.Empty(scope2b3[FieldEnterpriseGroupScopeOrganization])
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
		createTime := time.Now()

		// API returns both groups, but we should only keep the one in our state
		apiResponse := &organization_management.ListGroupsResponse{
			Items: &[]organization_management.ListGroupsResponseGroup{
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

	t.Run("when API returns groups with multiple role bindings then include all role bindings in state with proper sorting", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID1 := "e" + uuid.NewString()
		organizationID2 := "f" + uuid.NewString()
		groupID1 := "bbbb1111-1111-1111-1111-111111111111"
		groupID2 := "aaaa2222-2222-2222-2222-222222222222"
		memberID1 := "b" + uuid.NewString()
		memberID2 := "a" + uuid.NewString()
		memberID3 := "c" + uuid.NewString()
		roleBindingID1 := "c" + uuid.NewString()
		roleBindingID2 := "a" + uuid.NewString()
		roleBindingID3 := "b" + uuid.NewString()
		roleID1 := "b" + uuid.NewString()
		roleID2 := "a" + uuid.NewString()
		roleID3 := "c" + uuid.NewString()
		clusterID1 := "b" + uuid.NewString()
		clusterID2 := "a" + uuid.NewString()

		createTime := time.Now()

		// API returns groups
		apiGroupsResponse := &organization_management.ListGroupsResponse{
			Items: &[]organization_management.ListGroupsResponseGroup{
				{
					Id:             lo.ToPtr(groupID1),
					Name:           lo.ToPtr("engineering-team"),
					OrganizationId: lo.ToPtr(organizationID1),
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
								Id:        lo.ToPtr(memberID2),
								Email:     lo.ToPtr("security@example.com"),
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
				{
					Id:             lo.ToPtr(groupID2),
					Name:           lo.ToPtr("security-team"),
					OrganizationId: lo.ToPtr(organizationID2),
					Description:    lo.ToPtr("Security team group"),
					CreateTime:     lo.ToPtr(createTime),
					ManagedBy:      lo.ToPtr("terraform"),
					Definition: &organization_management.ListGroupsResponseGroupDefinition{
						Members: &[]organization_management.GroupDefinitionMember{
							{
								Id:        lo.ToPtr(memberID2),
								Email:     lo.ToPtr("security@example.com"),
								AddedTime: lo.ToPtr(createTime),
								Kind:      lo.ToPtr(organization_management.GroupDefinitionMemberKindKINDUSER),
							},
						},
					},
				},
			},
		}

		// API returns role bindings for both groups - we should track ALL of them
		apiRoleBindingsResponse := &organization_management.ListRoleBindingsResponse{
			Items: &[]organization_management.RoleBinding{
				{
					Id:         lo.ToPtr(roleBindingID1),
					Name:       lo.ToPtr("engineering-viewer"),
					CreateTime: lo.ToPtr(createTime),
					ManagedBy:  lo.ToPtr("terraform"),
					Definition: &organization_management.RoleBindingDefinition{
						RoleId: lo.ToPtr(roleID1),
						Subjects: &[]organization_management.Subject{
							{
								Group: &organization_management.GroupSubject{
									Id:   groupID1,
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
					Id:         lo.ToPtr(roleBindingID2),
					Name:       lo.ToPtr("security-auditor"),
					CreateTime: lo.ToPtr(createTime),
					ManagedBy:  lo.ToPtr("terraform"),
					Definition: &organization_management.RoleBindingDefinition{
						RoleId: lo.ToPtr(roleID2),
						Subjects: &[]organization_management.Subject{
							{
								Group: &organization_management.GroupSubject{
									Id:   groupID2,
									Name: lo.ToPtr("security-team"),
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
									Id:   groupID1,
									Name: lo.ToPtr("engineering-team"),
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

		mockClient.EXPECT().
			EnterpriseAPIListRoleBindingsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIListRoleBindingsResponse{
				Body:         nil,
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiRoleBindingsResponse,
			}, nil)

		// State includes minimal group data - role bindings will be discovered
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupID:             cty.StringVal(groupID1),
					FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID1),
					FieldEnterpriseGroupName:           cty.StringVal("engineering-team"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupID:             cty.StringVal(groupID2),
					FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID2),
					FieldEnterpriseGroupName:           cty.StringVal("security-team"),
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

		// Verify both groups are returned with proper sorting by ID (aaaa comes before bbbb)
		groups := data.Get(FieldEnterpriseGroupsGroups).([]any)
		r.Len(groups, 2)

		// Groups should be sorted by ID: groupID2 (aaaa), groupID1 (bbbb)
		group1 := groups[0].(map[string]any)
		group2 := groups[1].(map[string]any)

		// First group (groupID2 - security-team)
		r.Equal(groupID2, group1[FieldEnterpriseGroupID])
		r.Equal("security-team", group1[FieldEnterpriseGroupName])
		r.Equal(organizationID2, group1[FieldEnterpriseGroupOrganizationID])

		// First group should have 1 member and 1 role binding
		members1 := group1[FieldEnterpriseGroupMembers].([]any)
		r.Len(members1, 1)
		member1 := members1[0].(map[string]any)
		r.Equal(memberID2, member1[FieldEnterpriseGroupMemberID])
		r.Equal("security@example.com", member1[FieldEnterpriseGroupMemberEmail])
		r.Equal("user", member1[FieldEnterpriseGroupMemberKind])

		roleBindings1 := group1[FieldEnterpriseGroupRoleBindings].([]any)
		r.Len(roleBindings1, 1)
		roleBinding1 := roleBindings1[0].(map[string]any)
		r.Equal(roleBindingID2, roleBinding1[FieldEnterpriseGroupRoleBindingID])
		r.Equal("security-auditor", roleBinding1[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID2, roleBinding1[FieldEnterpriseGroupRoleBindingRoleID])

		// Second group (groupID1 - engineering-team)
		r.Equal(groupID1, group2[FieldEnterpriseGroupID])
		r.Equal("engineering-team", group2[FieldEnterpriseGroupName])
		r.Equal(organizationID1, group2[FieldEnterpriseGroupOrganizationID])

		// Second group should have 3 members and 2 role bindings
		members2 := group2[FieldEnterpriseGroupMembers].([]any)
		r.Len(members2, 3)

		// Members should be sorted by ID: memberID2 (a), memberID1 (b), memberID3 (c)
		member2A := members2[0].(map[string]any)
		member2B := members2[1].(map[string]any)
		member2C := members2[2].(map[string]any)

		r.Equal(memberID1, member2A[FieldEnterpriseGroupMemberID])
		r.Equal("engineer@example.com", member2A[FieldEnterpriseGroupMemberEmail])
		r.Equal("user", member2A[FieldEnterpriseGroupMemberKind])

		r.Equal(memberID2, member2B[FieldEnterpriseGroupMemberID])
		r.Equal("security@example.com", member2B[FieldEnterpriseGroupMemberEmail])
		r.Equal("user", member2B[FieldEnterpriseGroupMemberKind])

		r.Equal(memberID3, member2C[FieldEnterpriseGroupMemberID])
		r.Equal("service_account", member2C[FieldEnterpriseGroupMemberKind])

		roleBindings2 := group2[FieldEnterpriseGroupRoleBindings].([]any)
		r.Len(roleBindings2, 2)

		// Role bindings should be sorted by ID: roleBindingID3 (b), roleBindingID1 (c)
		roleBinding2A := roleBindings2[0].(map[string]any)
		roleBinding2B := roleBindings2[1].(map[string]any)

		r.Equal(roleBindingID3, roleBinding2A[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-editor", roleBinding2A[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID3, roleBinding2A[FieldEnterpriseGroupRoleBindingRoleID])

		r.Equal(roleBindingID1, roleBinding2B[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-viewer", roleBinding2B[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID1, roleBinding2B[FieldEnterpriseGroupRoleBindingRoleID])
	})
}

func TestResourceEnterpriseGroupsDelete(t *testing.T) {
	t.Run("when API successfully deletes groups then clear state", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID1 := "e" + uuid.NewString()
		organizationID2 := "f" + uuid.NewString()
		groupID1 := "bbbb1111-1111-1111-1111-111111111111"
		groupID2 := "aaaa2222-2222-2222-2222-222222222222"

		// Expected delete request
		expectedRequest := organization_management.BatchDeleteEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest{
				{
					Id:             groupID1,
					OrganizationId: organizationID1,
				},
				{
					Id:             groupID2,
					OrganizationId: organizationID2,
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
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupID:             cty.StringVal(groupID1),
					FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID1),
					FieldEnterpriseGroupName:           cty.StringVal("engineering-team"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupID:             cty.StringVal(groupID2),
					FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID2),
					FieldEnterpriseGroupName:           cty.StringVal("security-team"),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = enterpriseID

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Empty(data.Id(), "Resource ID should be cleared after successful delete")
	})

	t.Run("when enterprise ID is empty then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		// State with no enterprise ID
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsGroups: cty.ListValEmpty(cty.Object(map[string]cty.Type{})),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = "" // Empty enterprise ID

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.True(result.HasError())
		r.Contains(result[0].Summary, "enterprise ID is not set")
	})

	t.Run("when group missing ID then return state corruption error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		organizationID1 := "e" + uuid.NewString()

		// State with group missing ID
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupID:             cty.StringVal(""), // Empty ID
					FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID1),
					FieldEnterpriseGroupName:           cty.StringVal("engineering-team"),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = enterpriseID

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.True(result.HasError())
		r.Contains(result[0].Summary, "group in state is missing valid ID - this indicates state corruption")
	})

	t.Run("when group missing organization_id then return state corruption error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		groupID1 := "bbbb1111-1111-1111-1111-111111111111"

		// State with group missing organization_id
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupID:             cty.StringVal(groupID1),
					FieldEnterpriseGroupOrganizationID: cty.StringVal(""), // Empty organization ID
					FieldEnterpriseGroupName:           cty.StringVal("engineering-team"),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = enterpriseID

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.True(result.HasError())
		r.Contains(result[0].Summary, fmt.Sprintf("group %s in state is missing valid organization_id - this indicates state corruption", groupID1))
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
		organizationID1 := "e" + uuid.NewString()
		groupID1 := "bbbb1111-1111-1111-1111-111111111111"

		mockClient.EXPECT().
			EnterpriseAPIBatchDeleteEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(nil, errors.New("network error"))

		// State with 1 group
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseGroupsGroups: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldEnterpriseGroupID:             cty.StringVal(groupID1),
					FieldEnterpriseGroupOrganizationID: cty.StringVal(organizationID1),
					FieldEnterpriseGroupName:           cty.StringVal("engineering-team"),
				}),
			}),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = enterpriseID

		resource := resourceEnterpriseGroups()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.True(result.HasError())
		r.Contains(result[0].Summary, "calling batch delete enterprise groups")
		r.Contains(result[0].Summary, "network error")
		r.NotEmpty(data.Id(), "Resource ID should not be cleared when delete fails")
	})
}

func TestResourceEnterpriseGroupsUpdate(t *testing.T) {
	t.Parallel()

	enterpriseID := uuid.NewString()
	orgID1 := uuid.NewString()
	orgID2 := uuid.NewString()
	existingGroupID1 := uuid.NewString()
	existingGroupID2 := uuid.NewString()
	ctx := context.Background()

	t.Run("when groups are added then call create API with exact parameters", func(t *testing.T) {
		r := require.New(t)
		ctrl := gomock.NewController(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(ctrl)

		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		memberID1 := uuid.NewString()
		memberID2 := uuid.NewString()
		roleID := uuid.NewString()
		clusterID := uuid.NewString()
		newGroupID := uuid.NewString()
		createTime := time.Now()

		resource := resourceEnterpriseGroups()

		// Create old state with one group
		oldState := &terraform.InstanceState{
			ID: enterpriseID,
			Attributes: map[string]string{
				FieldEnterpriseGroupsEnterpriseID: enterpriseID,
				"groups.#":                        "1",
				"groups.0.id":                     existingGroupID1,
				"groups.0.name":                   "existing-group",
				"groups.0.organization_id":        orgID1,
				"groups.0.members.#":              "0",
				"groups.0.role_bindings.#":        "0",
			},
		}

		// Create diff that adds a new group
		diff := &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"groups.#": {
					Old: "1",
					New: "2",
				},
				"groups.1.name": {
					Old: "",
					New: "new-engineering-team",
				},
				"groups.1.organization_id": {
					Old: "",
					New: orgID2,
				},
				"groups.1.description": {
					Old: "",
					New: "New engineering team",
				},
				"groups.1.members.#": {
					Old: "",
					New: "2",
				},
				"groups.1.members.0.kind": {
					Old: "",
					New: "user",
				},
				"groups.1.members.0.id": {
					Old: "",
					New: memberID1,
				},
				"groups.1.members.1.kind": {
					Old: "",
					New: "service_account",
				},
				"groups.1.members.1.id": {
					Old: "",
					New: memberID2,
				},
				"groups.1.role_bindings.#": {
					Old: "",
					New: "1",
				},
				"groups.1.role_bindings.0.name": {
					Old: "",
					New: "engineering-viewer",
				},
				"groups.1.role_bindings.0.role_id": {
					Old: "",
					New: roleID,
				},
				"groups.1.role_bindings.0.scopes.#": {
					Old: "",
					New: "2",
				},
				"groups.1.role_bindings.0.scopes.0.cluster": {
					Old: "",
					New: clusterID,
				},
				"groups.1.role_bindings.0.scopes.1.organization": {
					Old: "",
					New: orgID2,
				},
			},
		}

		// Use the schemaMap.Data method you suggested
		schemaMap := make(map[string]*schema.Schema)
		for k, v := range resource.Schema {
			schemaMap[k] = v
		}
		data, err := schema.InternalMap(schemaMap).Data(oldState, diff)
		r.NoError(err)

		// Expected create request for new group
		expectedCreateRequest := organization_management.BatchCreateEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchCreateEnterpriseGroupsRequestGroup{
				{
					Name:           "new-engineering-team",
					OrganizationId: orgID2,
					Description:    lo.ToPtr("New engineering team"),
					Members: []organization_management.BatchCreateEnterpriseGroupsRequestMember{
						{
							Kind: lo.ToPtr(organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDUSER),
							Id:   lo.ToPtr(memberID1),
						},
						{
							Kind: lo.ToPtr(organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDSERVICEACCOUNT),
							Id:   lo.ToPtr(memberID2),
						},
					},
					RoleBindings: &[]organization_management.BatchCreateEnterpriseGroupsRequestRoleBinding{
						{
							Name:   "engineering-viewer",
							RoleId: roleID,
							Scopes: []organization_management.Scope{
								{
									Cluster: &organization_management.ClusterScope{
										Id: clusterID,
									},
								},
								{
									Organization: &organization_management.OrganizationScope{
										Id: orgID2,
									},
								},
							},
						},
					},
				},
			},
		}

		// Mock create API call with exact parameters
		mockClient.EXPECT().
			EnterpriseAPIBatchCreateEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, expectedCreateRequest).
			Return(&organization_management.EnterpriseAPIBatchCreateEnterpriseGroupsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &organization_management.BatchCreateEnterpriseGroupsResponse{
					Groups: &[]organization_management.BatchCreateEnterpriseGroupsResponseGroup{
						{
							Id:             lo.ToPtr(newGroupID),
							Name:           lo.ToPtr("new-engineering-team"),
							OrganizationId: lo.ToPtr(orgID2),
							Description:    lo.ToPtr("New engineering team"),
							CreateTime:     lo.ToPtr(createTime),
							ManagedBy:      lo.ToPtr("terraform"),
						},
					},
				},
			}, nil)

		// Expected update request for existing group
		expectedUpdateRequest := organization_management.BatchUpdateEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest{
				{
					Id:             existingGroupID1,
					Name:           "existing-group",
					OrganizationId: orgID1,
					Description:    "",
					Members:        nil,
					RoleBindings:   nil,
				},
			},
		}

		// Mock update API call with exact parameters
		mockClient.EXPECT().
			EnterpriseAPIBatchUpdateEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, expectedUpdateRequest).
			Return(&organization_management.EnterpriseAPIBatchUpdateEnterpriseGroupsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
			}, nil)

		// Mock read calls at the end
		mockClient.EXPECT().
			EnterpriseAPIListGroupsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIListGroupsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &organization_management.ListGroupsResponse{
					Items: &[]organization_management.ListGroupsResponseGroup{
						{
							Id:             &existingGroupID1,
							Name:           lo.ToPtr("existing-group"),
							OrganizationId: &orgID1,
						},
						{
							Id:             &newGroupID,
							Name:           lo.ToPtr("new-engineering-team"),
							OrganizationId: &orgID2,
							Description:    lo.ToPtr("New engineering team"),
							CreateTime:     lo.ToPtr(createTime),
							ManagedBy:      lo.ToPtr("terraform"),
						},
					},
				},
			}, nil)

		mockClient.EXPECT().
			EnterpriseAPIListRoleBindingsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIListRoleBindingsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200:      &organization_management.ListRoleBindingsResponse{Items: &[]organization_management.RoleBinding{}},
			}, nil)

		// Execute update
		result := resource.UpdateContext(ctx, data, provider)

		// Verify no errors
		r.False(result.HasError(), "Update should succeed when adding groups")
		if result.HasError() {
			for _, diag := range result {
				t.Logf("Error: %s - %s", diag.Summary, diag.Detail)
			}
		}
	})

	t.Run("when groups are deleted then call delete API with exact parameters", func(t *testing.T) {
		r := require.New(t)
		ctrl := gomock.NewController(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(ctrl)

		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		resource := resourceEnterpriseGroups()

		// Create old state with two groups
		oldState := &terraform.InstanceState{
			ID: enterpriseID,
			Attributes: map[string]string{
				FieldEnterpriseGroupsEnterpriseID:                enterpriseID,
				"groups.#":                                       "2",
				"groups.0.id":                                    existingGroupID1,
				"groups.0.name":                                  "group-to-keep",
				"groups.0.organization_id":                       orgID1,
				"groups.0.description":                           "Old description",
				"groups.0.members.#":                             "1",
				"groups.0.members.0.kind":                        "user",
				"groups.0.members.0.id":                          "old-member-id",
				"groups.0.role_bindings.#":                       "1",
				"groups.0.role_bindings.0.name":                  "old-role-binding",
				"groups.0.role_bindings.0.role_id":               "old-role-id",
				"groups.0.role_bindings.0.scopes.#":              "1",
				"groups.0.role_bindings.0.scopes.0.organization": orgID1,
				"groups.1.id":                                    existingGroupID2,
				"groups.1.name":                                  "group-to-delete",
				"groups.1.organization_id":                       orgID2,
				"groups.1.description":                           "Will be deleted",
				"groups.1.members.#":                             "0",
				"groups.1.role_bindings.#":                       "0",
			},
		}

		// Create diff that removes the second group and updates the first
		diff := &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"groups.#": {
					Old: "2",
					New: "1",
				},
				"groups.0.description": {
					Old: "Old description",
					New: "Updated description",
				},
				"groups.0.members.0.id": {
					Old: "old-member-id",
					New: "new-member-id",
				},
				"groups.0.role_bindings.0.name": {
					Old: "old-role-binding",
					New: "updated-role-binding",
				},
				"groups.0.role_bindings.0.role_id": {
					Old: "old-role-id",
					New: "new-role-id",
				},
				"groups.1.id": {
					Old:        existingGroupID2,
					New:        "",
					NewRemoved: true,
				},
				"groups.1.name": {
					Old:        "group-to-delete",
					New:        "",
					NewRemoved: true,
				},
				"groups.1.organization_id": {
					Old:        orgID2,
					New:        "",
					NewRemoved: true,
				},
				"groups.1.description": {
					Old:        "Will be deleted",
					New:        "",
					NewRemoved: true,
				},
				"groups.1.members.#": {
					Old:        "0",
					New:        "",
					NewRemoved: true,
				},
				"groups.1.role_bindings.#": {
					Old:        "0",
					New:        "",
					NewRemoved: true,
				},
			},
		}

		schemaMap := make(map[string]*schema.Schema)
		for k, v := range resource.Schema {
			schemaMap[k] = v
		}
		data, err := schema.InternalMap(schemaMap).Data(oldState, diff)
		r.NoError(err)

		// Expected delete request
		expectedDeleteRequest := organization_management.BatchDeleteEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest{
				{
					Id:             existingGroupID2,
					OrganizationId: orgID2,
				},
			},
		}

		// Expected update request for remaining group
		expectedUpdateRequest := organization_management.BatchUpdateEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest{
				{
					Id:             existingGroupID1,
					Name:           "group-to-keep",
					OrganizationId: orgID1,
					Description:    "Updated description",
					Members: []organization_management.BatchUpdateEnterpriseGroupsRequestMember{
						{
							Kind: organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindUSER,
							Id:   "new-member-id",
						},
					},
					RoleBindings: []organization_management.BatchUpdateEnterpriseGroupsRequestRoleBinding{
						{
							Id:     existingGroupID1 + "-updated-role-binding",
							Name:   "updated-role-binding",
							RoleId: "new-role-id",
							Scopes: []organization_management.Scope{
								{
									Organization: &organization_management.OrganizationScope{
										Id: orgID1,
									},
								},
							},
						},
					},
				},
			},
		}

		// Mock delete API call with exact parameters
		mockClient.EXPECT().
			EnterpriseAPIBatchDeleteEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, expectedDeleteRequest).
			Return(&organization_management.EnterpriseAPIBatchDeleteEnterpriseGroupsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
			}, nil)

		// Mock update API call with exact parameters
		mockClient.EXPECT().
			EnterpriseAPIBatchUpdateEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, expectedUpdateRequest).
			Return(&organization_management.EnterpriseAPIBatchUpdateEnterpriseGroupsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
			}, nil)

		// Mock read calls at the end
		mockClient.EXPECT().
			EnterpriseAPIListGroupsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIListGroupsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &organization_management.ListGroupsResponse{
					Items: &[]organization_management.ListGroupsResponseGroup{
						{
							Id:             &existingGroupID1,
							Name:           lo.ToPtr("group-to-keep"),
							OrganizationId: &orgID1,
							Description:    lo.ToPtr("Updated description"),
						},
					},
				},
			}, nil)

		mockClient.EXPECT().
			EnterpriseAPIListRoleBindingsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIListRoleBindingsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &organization_management.ListRoleBindingsResponse{
					Items: &[]organization_management.RoleBinding{
						{
							Id:   lo.ToPtr(existingGroupID1 + "-updated-role-binding"),
							Name: lo.ToPtr("updated-role-binding"),
							Definition: &organization_management.RoleBindingDefinition{
								RoleId: lo.ToPtr("new-role-id"),
								Scopes: &[]organization_management.Scope{
									{
										Organization: &organization_management.OrganizationScope{
											Id: orgID1,
										},
									},
								},
							},
						},
					},
				},
			}, nil)

		// Execute update
		result := resource.UpdateContext(ctx, data, provider)

		// Verify no errors
		r.False(result.HasError(), "Update should succeed when deleting groups")
	})

	t.Run("when groups are updated then call update API with exact parameters", func(t *testing.T) {
		r := require.New(t)
		ctrl := gomock.NewController(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(ctrl)

		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		memberID := uuid.NewString()
		roleID := uuid.NewString()
		orgScopeID := uuid.NewString()

		resource := resourceEnterpriseGroups()

		// Create old state with one group
		oldState := &terraform.InstanceState{
			ID: enterpriseID,
			Attributes: map[string]string{
				FieldEnterpriseGroupsEnterpriseID: enterpriseID,
				"groups.#":                        "1",
				"groups.0.id":                     existingGroupID1,
				"groups.0.name":                   "old-name",
				"groups.0.organization_id":        orgID1,
				"groups.0.description":            "Old description",
				"groups.0.members.#":              "0",
				"groups.0.role_bindings.#":        "0",
			},
		}

		// Create diff that updates the group
		diff := &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"groups.0.name": {
					Old: "old-name",
					New: "updated-name",
				},
				"groups.0.description": {
					Old: "Old description",
					New: "Updated description",
				},
				"groups.0.members.#": {
					Old: "0",
					New: "1",
				},
				"groups.0.members.0.kind": {
					Old: "",
					New: "user",
				},
				"groups.0.members.0.id": {
					Old: "",
					New: memberID,
				},
				"groups.0.role_bindings.#": {
					Old: "0",
					New: "1",
				},
				"groups.0.role_bindings.0.name": {
					Old: "",
					New: "updated-role-binding",
				},
				"groups.0.role_bindings.0.role_id": {
					Old: "",
					New: roleID,
				},
				"groups.0.role_bindings.0.scopes.#": {
					Old: "",
					New: "1",
				},
				"groups.0.role_bindings.0.scopes.0.organization": {
					Old: "",
					New: orgScopeID,
				},
			},
		}

		schemaMap := make(map[string]*schema.Schema)
		for k, v := range resource.Schema {
			schemaMap[k] = v
		}
		data, err := schema.InternalMap(schemaMap).Data(oldState, diff)
		r.NoError(err)

		// Expected update request
		expectedUpdateRequest := organization_management.BatchUpdateEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest{
				{
					Id:             existingGroupID1,
					Name:           "updated-name",
					OrganizationId: orgID1,
					Description:    "Updated description",
					Members: []organization_management.BatchUpdateEnterpriseGroupsRequestMember{
						{
							Kind: organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindUSER,
							Id:   memberID,
						},
					},
					RoleBindings: []organization_management.BatchUpdateEnterpriseGroupsRequestRoleBinding{
						{
							Id:     existingGroupID1 + "-updated-role-binding",
							Name:   "updated-role-binding",
							RoleId: roleID,
							Scopes: []organization_management.Scope{
								{
									Organization: &organization_management.OrganizationScope{
										Id: orgScopeID,
									},
								},
							},
						},
					},
				},
			},
		}

		// Mock update API call with exact parameters
		mockClient.EXPECT().
			EnterpriseAPIBatchUpdateEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, expectedUpdateRequest).
			Return(&organization_management.EnterpriseAPIBatchUpdateEnterpriseGroupsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
			}, nil)

		// Mock read calls at the end
		mockClient.EXPECT().
			EnterpriseAPIListGroupsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIListGroupsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200: &organization_management.ListGroupsResponse{
					Items: &[]organization_management.ListGroupsResponseGroup{
						{
							Id:             &existingGroupID1,
							Name:           lo.ToPtr("updated-name"),
							OrganizationId: &orgID1,
							Description:    lo.ToPtr("Updated description"),
						},
					},
				},
			}, nil)

		mockClient.EXPECT().
			EnterpriseAPIListRoleBindingsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIListRoleBindingsResponse{
				HTTPResponse: &http.Response{StatusCode: 200},
				JSON200:      &organization_management.ListRoleBindingsResponse{Items: &[]organization_management.RoleBinding{}},
			}, nil)

		// Execute update
		result := resource.UpdateContext(ctx, data, provider)

		// Verify no errors
		r.False(result.HasError(), "Update should succeed when updating groups")
		if result.HasError() {
			for _, diag := range result {
				t.Logf("Error: %s - %s", diag.Summary, diag.Detail)
			}
		}
	})

	t.Run("resource update error generic propagated", func(t *testing.T) {
		r := require.New(t)
		ctrl := gomock.NewController(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(ctrl)

		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		resource := resourceEnterpriseGroups()

		// Create old state with one group
		oldState := &terraform.InstanceState{
			ID: enterpriseID,
			Attributes: map[string]string{
				FieldEnterpriseGroupsEnterpriseID: enterpriseID,
				"groups.#":                        "1",
				"groups.0.id":                     existingGroupID1,
				"groups.0.name":                   "old-name",
				"groups.0.organization_id":        orgID1,
				"groups.0.members.#":              "0",
				"groups.0.role_bindings.#":        "0",
			},
		}

		// Create diff that updates the group name
		diff := &terraform.InstanceDiff{
			Attributes: map[string]*terraform.ResourceAttrDiff{
				"groups.0.name": {
					Old: "old-name",
					New: "updated-name",
				},
			},
		}

		schemaMap := make(map[string]*schema.Schema)
		for k, v := range resource.Schema {
			schemaMap[k] = v
		}
		data, err := schema.InternalMap(schemaMap).Data(oldState, diff)
		r.NoError(err)

		// Expected update request for error test
		expectedUpdateRequest := organization_management.BatchUpdateEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest{
				{
					Id:             existingGroupID1,
					Name:           "updated-name",
					OrganizationId: orgID1,
					Description:    "",
					Members:        []organization_management.BatchUpdateEnterpriseGroupsRequestMember{},
					RoleBindings:   []organization_management.BatchUpdateEnterpriseGroupsRequestRoleBinding{},
				},
			},
		}

		// Mock update API to return error with exact parameters
		mockClient.EXPECT().
			EnterpriseAPIBatchUpdateEnterpriseGroupsWithResponse(gomock.Any(), enterpriseID, expectedUpdateRequest).
			Return(&organization_management.EnterpriseAPIBatchUpdateEnterpriseGroupsResponse{
				HTTPResponse: &http.Response{StatusCode: 400, Body: io.NopCloser(bytes.NewBufferString(`{"message":"Bad Request", "fieldViolations":[{"field":"name","description":"invalid name"}]}`))},
			}, nil)

		result := resource.UpdateContext(ctx, data, provider)

		r.True(result.HasError(), "Should return error when API call fails")
		r.Contains(result[0].Summary, "batch update modified groups failed")
		r.Contains(result[0].Summary, "status 400")
	})
}
