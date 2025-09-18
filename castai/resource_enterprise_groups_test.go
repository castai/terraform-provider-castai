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
		r.Equal(memberID2, member2First[FieldEnterpriseGroupMemberID])
		r.Equal("security@example.com", member2First[FieldEnterpriseGroupMemberEmail])
		r.Equal("user", member2First[FieldEnterpriseGroupMemberKind])
		r.Equal(createTime.Format(time.RFC3339), member2First[FieldEnterpriseGroupMemberAddedTime])

		member2Second := members2[1].(map[string]any)
		r.Equal(memberID1, member2Second[FieldEnterpriseGroupMemberID])
		r.Equal("engineer@example.com", member2Second[FieldEnterpriseGroupMemberEmail])
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

		// First scope: cluster with clusterID2 ("a"...)
		scope2b1 := scopes2b[0].(map[string]any)
		r.Equal(clusterID2, scope2b1[FieldEnterpriseGroupScopeCluster])
		r.Empty(scope2b1[FieldEnterpriseGroupScopeOrganization])

		// Second scope: cluster with clusterID1 ("b"...)
		scope2b2 := scopes2b[1].(map[string]any)
		r.Equal(clusterID1, scope2b2[FieldEnterpriseGroupScopeCluster])
		r.Empty(scope2b2[FieldEnterpriseGroupScopeOrganization])

		// Third scope: organization with organizationID1
		scope2b3 := scopes2b[2].(map[string]any)
		r.Equal(organizationID1, scope2b3[FieldEnterpriseGroupScopeOrganization])
		r.Empty(scope2b3[FieldEnterpriseGroupScopeCluster])
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

		r.Equal(memberID2, member2A[FieldEnterpriseGroupMemberID])
		r.Equal("security@example.com", member2A[FieldEnterpriseGroupMemberEmail])
		r.Equal("user", member2A[FieldEnterpriseGroupMemberKind])

		r.Equal(memberID1, member2B[FieldEnterpriseGroupMemberID])
		r.Equal("engineer@example.com", member2B[FieldEnterpriseGroupMemberEmail])
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
