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

		// First role binding (roleBindingID1 - engineering-viewer)
		roleBinding2a := roleBindings2[0].(map[string]any)
		r.Equal(roleBindingID1, roleBinding2a[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-viewer", roleBinding2a[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID1, roleBinding2a[FieldEnterpriseGroupRoleBindingRoleID])

		scopes2a := roleBinding2a[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes2a, 3)

		// Second role binding (roleBindingID3 - engineering-editor)
		roleBinding2b := roleBindings2[1].(map[string]any)
		r.Equal(roleBindingID3, roleBinding2b[FieldEnterpriseGroupRoleBindingID])
		r.Equal("engineering-editor", roleBinding2b[FieldEnterpriseGroupRoleBindingName])
		r.Equal(roleID3, roleBinding2b[FieldEnterpriseGroupRoleBindingRoleID])

		scopes2b := roleBinding2b[FieldEnterpriseGroupRoleBindingScopes].([]any)
		r.Len(scopes2b, 2)
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
