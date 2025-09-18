package castai

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk/organization_management"
)

const (
	// Field names for the enterprise groups resource
	FieldEnterpriseGroupsEnterpriseID = "enterprise_id"
	FieldEnterpriseGroupsGroups       = "groups"

	// Field names for individual groups
	FieldEnterpriseGroupID             = "id"
	FieldEnterpriseGroupOrganizationID = "organization_id"
	FieldEnterpriseGroupName           = "name"
	FieldEnterpriseGroupDescription    = "description"
	FieldEnterpriseGroupCreateTime     = "create_time"
	FieldEnterpriseGroupManagedBy      = "managed_by"
	FieldEnterpriseGroupMembers        = "members"
	FieldEnterpriseGroupRoleBindings   = "role_bindings"

	// Field names for members
	FieldEnterpriseGroupMemberKind      = "kind"
	FieldEnterpriseGroupMemberID        = "id"
	FieldEnterpriseGroupMemberEmail     = "email"
	FieldEnterpriseGroupMemberAddedTime = "added_time"

	// Field names for role bindings
	FieldEnterpriseGroupRoleBindingID     = "id"
	FieldEnterpriseGroupRoleBindingName   = "name"
	FieldEnterpriseGroupRoleBindingRoleID = "role_id"
	FieldEnterpriseGroupRoleBindingScopes = "scopes"

	// Field names for scopes
	FieldEnterpriseGroupScopeOrganization = "organization"
	FieldEnterpriseGroupScopeCluster      = "cluster"

	// Member kinds
	EnterpriseGroupMemberKindUser           = "user"
	EnterpriseGroupMemberKindServiceAccount = "service_account"
)

var (
	supportedEnterpriseGroupMemberKinds = []string{
		EnterpriseGroupMemberKindUser,
		EnterpriseGroupMemberKindServiceAccount,
	}
)

func resourceEnterpriseGroups() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEnterpriseGroupsCreate,
		ReadContext:   resourceEnterpriseGroupsRead,
		UpdateContext: resourceEnterpriseGroupsUpdate,
		DeleteContext: resourceEnterpriseGroupsDelete,
		Description:   "CAST AI enterprise groups resource to manage multiple organization groups via batch operations",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldEnterpriseGroupsEnterpriseID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Enterprise organization ID.",
			},
			FieldEnterpriseGroupsGroups: {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of enterprise groups to manage.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldEnterpriseGroupID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Group ID assigned by the API.",
						},
						FieldEnterpriseGroupOrganizationID: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Target organization ID for the group.",
						},
						FieldEnterpriseGroupName: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the group.",
						},
						FieldEnterpriseGroupDescription: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Description of the group.",
						},
						FieldEnterpriseGroupCreateTime: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Timestamp when the group was created.",
						},
						FieldEnterpriseGroupManagedBy: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Method used to create the group (e.g., terraform, console).",
						},
						FieldEnterpriseGroupMembers: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "List of group members.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldEnterpriseGroupMemberKind: {
										Type:             schema.TypeString,
										Required:         true,
										Description:      fmt.Sprintf("Kind of the member. Supported values: %s.", strings.Join(supportedEnterpriseGroupMemberKinds, ", ")),
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(supportedEnterpriseGroupMemberKinds, true)),
										DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
											return strings.EqualFold(oldValue, newValue)
										},
									},
									FieldEnterpriseGroupMemberID: {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Member UUID.",
									},
									FieldEnterpriseGroupMemberEmail: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Member email address.",
									},
									FieldEnterpriseGroupMemberAddedTime: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Timestamp when the member was added to the group.",
									},
								},
							},
						},
						FieldEnterpriseGroupRoleBindings: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "List of role bindings for the group.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldEnterpriseGroupRoleBindingID: {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Role binding ID assigned by the API.",
									},
									FieldEnterpriseGroupRoleBindingName: {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Role binding name.",
									},
									FieldEnterpriseGroupRoleBindingRoleID: {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Role UUID.",
									},
									FieldEnterpriseGroupRoleBindingScopes: {
										Type:        schema.TypeList,
										Required:    true,
										Description: "List of scopes for the role binding.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldEnterpriseGroupScopeOrganization: {
													Type:        schema.TypeString,
													Optional:    true,
													Description: "Organization ID scope.",
												},
												FieldEnterpriseGroupScopeCluster: {
													Type:        schema.TypeString,
													Optional:    true,
													Description: "Cluster ID scope.",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceEnterpriseGroupsCreate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient
	enterpriseID := data.Get(FieldEnterpriseGroupsEnterpriseID).(string)

	createRequest, err := buildBatchCreateRequest(enterpriseID, data)
	if err != nil {
		return diag.FromErr(fmt.Errorf("building create request: %w", err))
	}

	resp, err := client.EnterpriseAPIBatchCreateEnterpriseGroupsWithResponse(ctx, enterpriseID, *createRequest)
	if err != nil {
		return diag.FromErr(fmt.Errorf("calling batch create enterprise groups: %w", err))
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("batch create enterprise groups failed with status %d: %s", resp.StatusCode(), string(resp.Body)))
	}

	if resp.JSON200 == nil || resp.JSON200.Groups == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from batch create"))
	}

	// Set the resource ID to the enterprise ID for tracking
	data.SetId(enterpriseID)

	// Update state with created groups
	if err = setEnterpriseGroupsData(data, *resp.JSON200.Groups); err != nil {
		return diag.FromErr(fmt.Errorf("setting created groups data: %w", err))
	}

	return nil
}

func resourceEnterpriseGroupsRead(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient
	enterpriseID := data.Id()

	if enterpriseID == "" {
		return diag.FromErr(fmt.Errorf("enterprise ID is not set"))
	}

	// Get the group IDs from current state to determine which groups we should be managing
	managedGroupIDs := getManagedGroupIDsFromState(data)
	if len(managedGroupIDs) == 0 {
		// No groups to manage, set empty state
		if err := data.Set(FieldEnterpriseGroupsGroups, []any{}); err != nil {
			return diag.FromErr(fmt.Errorf("setting empty groups: %w", err))
		}
		return nil
	}

	// Call list groups API to get current state
	resp, err := client.EnterpriseAPIListGroupsWithResponse(ctx, enterpriseID, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("listing enterprise groups: %w", err))
	}

	if resp.StatusCode() == http.StatusNotFound {
		// Enterprise not found, remove from state
		data.SetId("")
		return nil
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("list enterprise groups failed with status %d: %s", resp.StatusCode(), string(resp.Body)))
	}

	if resp.JSON200 == nil || resp.JSON200.Items == nil {
		// No groups found in API, but we have groups in state - they might have been deleted
		if err := data.Set(FieldEnterpriseGroupsGroups, []any{}); err != nil {
			return diag.FromErr(fmt.Errorf("setting empty groups: %w", err))
		}
		return nil
	}

	// Filter API response to only include groups we are managing
	var managedGroups []organization_management.ListGroupsResponseGroup
	for _, group := range *resp.JSON200.Items {
		if group.Id != nil && managedGroupIDs[*group.Id] {
			managedGroups = append(managedGroups, group)
		}
	}

	// TODO: Fetch role bindings for each of groups since they are not included in the list response

	// Update state with only the groups we are managing
	if err := setEnterpriseGroupsDataFromListResponse(data, managedGroups); err != nil {
		return diag.FromErr(fmt.Errorf("setting groups data from list response: %w", err))
	}

	return nil
}

func resourceEnterpriseGroupsUpdate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient
	enterpriseID := data.Id()

	if enterpriseID == "" {
		return diag.FromErr(fmt.Errorf("enterprise ID is not set"))
	}

	// Build update request from schema data
	updateRequest, err := buildBatchUpdateRequest(enterpriseID, data)
	if err != nil {
		return diag.FromErr(fmt.Errorf("building update request: %w", err))
	}

	// Call batch update API
	resp, err := client.EnterpriseAPIBatchUpdateEnterpriseGroupsWithResponse(ctx, enterpriseID, *updateRequest)
	if err != nil {
		return diag.FromErr(fmt.Errorf("calling batch update enterprise groups: %w", err))
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("batch update enterprise groups failed with status %d: %s", resp.StatusCode(), string(resp.Body)))
	}

	if resp.JSON200 == nil || resp.JSON200.Groups == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from batch update"))
	}

	// Update state with updated groups
	if err := setEnterpriseGroupsDataFromUpdateResponse(data, *resp.JSON200.Groups); err != nil {
		return diag.FromErr(fmt.Errorf("setting updated groups data: %w", err))
	}

	return nil
}

func resourceEnterpriseGroupsDelete(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient
	enterpriseID := data.Id()

	if enterpriseID == "" {
		return diag.FromErr(fmt.Errorf("enterprise ID is not set"))
	}

	// Build delete request from current state
	deleteRequest, err := buildBatchDeleteRequest(enterpriseID, data)
	if err != nil {
		return diag.FromErr(fmt.Errorf("building delete request: %w", err))
	}

	// Call batch delete API
	resp, err := client.EnterpriseAPIBatchDeleteEnterpriseGroupsWithResponse(ctx, enterpriseID, *deleteRequest)
	if err != nil {
		return diag.FromErr(fmt.Errorf("calling batch delete enterprise groups: %w", err))
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNotFound {
		return diag.FromErr(fmt.Errorf("batch delete enterprise groups failed with status %d: %s", resp.StatusCode(), string(resp.Body)))
	}

	// Clear the resource ID
	data.SetId("")

	return nil
}

// buildBatchCreateRequest constructs the batch create request from Terraform schema data
func buildBatchCreateRequest(enterpriseID string, data *schema.ResourceData) (*organization_management.BatchCreateEnterpriseGroupsRequest, error) {
	groupsList := data.Get(FieldEnterpriseGroupsGroups).([]any)

	var requests []organization_management.BatchCreateEnterpriseGroupsRequestGroup

	for _, groupData := range groupsList {
		group := groupData.(map[string]any)

		var members []organization_management.BatchCreateEnterpriseGroupsRequestMember
		if membersData, ok := group[FieldEnterpriseGroupMembers].([]any); ok {
			for _, memberData := range membersData {
				member := memberData.(map[string]any)

				var kind organization_management.BatchCreateEnterpriseGroupsRequestMemberKind
				switch member[FieldEnterpriseGroupMemberKind].(string) {
				case EnterpriseGroupMemberKindUser:
					kind = organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDUSER
				case EnterpriseGroupMemberKindServiceAccount:
					kind = organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDSERVICEACCOUNT
				default:
					kind = organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDUNSPECIFIED
				}

				members = append(members, organization_management.BatchCreateEnterpriseGroupsRequestMember{
					Kind: &kind,
					Id:   lo.ToPtr(member[FieldEnterpriseGroupMemberID].(string)),
				})
			}
		}

		var roleBindings *[]organization_management.BatchCreateEnterpriseGroupsRequestRoleBinding
		if bindingsData, ok := group[FieldEnterpriseGroupRoleBindings].([]any); ok && len(bindingsData) > 0 {
			var bindings []organization_management.BatchCreateEnterpriseGroupsRequestRoleBinding

			for _, bindingData := range bindingsData {
				binding := bindingData.(map[string]any)

				var scopes []organization_management.Scope
				if scopesData, ok := binding[FieldEnterpriseGroupRoleBindingScopes].([]any); ok {
					for _, scopeData := range scopesData {
						scope := scopeData.(map[string]any)

						orgID := scope[FieldEnterpriseGroupScopeOrganization].(string)
						clusterID := scope[FieldEnterpriseGroupScopeCluster].(string)

						if orgID != "" {
							scopes = append(scopes, organization_management.Scope{
								Organization: &organization_management.OrganizationScope{
									Id: orgID,
								},
							})
						}

						if clusterID != "" {
							scopes = append(scopes, organization_management.Scope{
								Cluster: &organization_management.ClusterScope{
									Id: clusterID,
								},
							})
						}
					}
				}

				bindings = append(bindings, organization_management.BatchCreateEnterpriseGroupsRequestRoleBinding{
					Name:   binding[FieldEnterpriseGroupRoleBindingName].(string),
					RoleId: binding[FieldEnterpriseGroupRoleBindingRoleID].(string),
					Scopes: scopes,
				})
			}

			roleBindings = &bindings
		}

		groupRequest := organization_management.BatchCreateEnterpriseGroupsRequestGroup{
			Name:           group[FieldEnterpriseGroupName].(string),
			OrganizationId: group[FieldEnterpriseGroupOrganizationID].(string),
			Members:        members,
			RoleBindings:   roleBindings,
		}

		if desc, ok := group[FieldEnterpriseGroupDescription].(string); ok && desc != "" {
			groupRequest.Description = &desc
		}

		requests = append(requests, groupRequest)
	}

	return &organization_management.BatchCreateEnterpriseGroupsRequest{
		EnterpriseId: enterpriseID,
		Requests:     requests,
	}, nil
}

// buildBatchUpdateRequest constructs the batch update request from Terraform schema data
func buildBatchUpdateRequest(enterpriseID string, data *schema.ResourceData) (*organization_management.BatchUpdateEnterpriseGroupsRequest, error) {
	groupsList := data.Get(FieldEnterpriseGroupsGroups).([]any)

	var requests []organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest

	for _, groupData := range groupsList {
		group := groupData.(map[string]any)

		// Group ID is required for updates
		groupID, ok := group[FieldEnterpriseGroupID].(string)
		if !ok || groupID == "" {
			return nil, fmt.Errorf("group ID is required for update operations")
		}

		// Convert members
		var members []organization_management.BatchUpdateEnterpriseGroupsRequestMember
		if membersData, ok := group[FieldEnterpriseGroupMembers].([]any); ok {
			for _, memberData := range membersData {
				member := memberData.(map[string]any)

				// Convert member kind
				var kind organization_management.BatchUpdateEnterpriseGroupsRequestMemberKind
				switch member[FieldEnterpriseGroupMemberKind].(string) {
				case EnterpriseGroupMemberKindUser:
					kind = organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindUSER
				case EnterpriseGroupMemberKindServiceAccount:
					kind = organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindSERVICEACCOUNT
				default:
					kind = organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindSUBJECTKINDUNSPECIFIED
				}

				members = append(members, organization_management.BatchUpdateEnterpriseGroupsRequestMember{
					Kind: kind,
					Id:   member[FieldEnterpriseGroupMemberID].(string),
				})
			}
		}

		// Convert role bindings
		var roleBindings []organization_management.BatchUpdateEnterpriseGroupsRequestRoleBinding
		if bindingsData, ok := group[FieldEnterpriseGroupRoleBindings].([]any); ok {
			for _, bindingData := range bindingsData {
				binding := bindingData.(map[string]any)

				// Convert scopes
				var scopes []organization_management.Scope
				if scopesData, ok := binding[FieldEnterpriseGroupRoleBindingScopes].([]any); ok {
					for _, scopeData := range scopesData {
						scope := scopeData.(map[string]any)

						if orgID := scope[FieldEnterpriseGroupScopeOrganization].(string); orgID != "" {
							scopes = append(scopes, organization_management.Scope{
								Organization: &organization_management.OrganizationScope{
									Id: orgID,
								},
							})
						}

						if clusterID := scope[FieldEnterpriseGroupScopeCluster].(string); clusterID != "" {
							scopes = append(scopes, organization_management.Scope{
								Cluster: &organization_management.ClusterScope{
									Id: clusterID,
								},
							})
						}
					}
				}

				// For updates, we need a role binding ID, but since this is not available in create,
				// we'll generate a synthetic ID based on the role binding name
				bindingID := fmt.Sprintf("%s-%s", groupID, binding[FieldEnterpriseGroupRoleBindingName].(string))

				roleBindings = append(roleBindings, organization_management.BatchUpdateEnterpriseGroupsRequestRoleBinding{
					Id:     bindingID,
					Name:   binding[FieldEnterpriseGroupRoleBindingName].(string),
					RoleId: binding[FieldEnterpriseGroupRoleBindingRoleID].(string),
					Scopes: scopes,
				})
			}
		}

		// Build the update request
		updateRequest := organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest{
			Id:             groupID,
			Name:           group[FieldEnterpriseGroupName].(string),
			OrganizationId: group[FieldEnterpriseGroupOrganizationID].(string),
			Description:    group[FieldEnterpriseGroupDescription].(string),
			Members:        members,
			RoleBindings:   roleBindings,
		}

		requests = append(requests, updateRequest)
	}

	return &organization_management.BatchUpdateEnterpriseGroupsRequest{
		EnterpriseId: enterpriseID,
		Requests:     requests,
	}, nil
}

// buildBatchDeleteRequest constructs the batch delete request from Terraform schema data
func buildBatchDeleteRequest(enterpriseID string, data *schema.ResourceData) (*organization_management.BatchDeleteEnterpriseGroupsRequest, error) {
	groupsList := data.Get(FieldEnterpriseGroupsGroups).([]any)

	var requests []organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest

	for _, groupData := range groupsList {
		group := groupData.(map[string]any)

		// Group ID is required for deletes
		groupID, ok := group[FieldEnterpriseGroupID].(string)
		if !ok || groupID == "" {
			// Skip groups without IDs (may not have been created yet)
			continue
		}

		requests = append(requests, organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest{
			Id:             groupID,
			OrganizationId: group[FieldEnterpriseGroupOrganizationID].(string),
		})
	}

	return &organization_management.BatchDeleteEnterpriseGroupsRequest{
		EnterpriseId: enterpriseID,
		Requests:     requests,
	}, nil
}

// getManagedGroupIDsFromState extracts the group IDs from Terraform state that this resource should manage
func getManagedGroupIDsFromState(data *schema.ResourceData) map[string]bool {
	managedIDs := make(map[string]bool)

	groupsList := data.Get(FieldEnterpriseGroupsGroups).([]any)
	for _, groupData := range groupsList {
		group := groupData.(map[string]any)
		if groupID, ok := group[FieldEnterpriseGroupID].(string); ok && groupID != "" {
			managedIDs[groupID] = true
		}
	}

	return managedIDs
}

// sortGroupsByID sorts a slice of group data by group ID for consistent ordering
func sortGroupsByID(groupsData []map[string]any) {
	sort.Slice(groupsData, func(i, j int) bool {
		idI, okI := groupsData[i][FieldEnterpriseGroupID].(string)
		idJ, okJ := groupsData[j][FieldEnterpriseGroupID].(string)
		if !okI || !okJ {
			return false
		}
		return idI < idJ
	})
}

func sortMembersByID(members []map[string]any) {
	sort.Slice(members, func(i, j int) bool {
		idI, okI := members[i][FieldEnterpriseGroupMemberID].(string)
		idJ, okJ := members[j][FieldEnterpriseGroupMemberID].(string)
		if !okI || !okJ {
			return false
		}
		return idI < idJ
	})
}

// convertMembersForBatchCreate converts members from batch create response
func convertMembersForBatchCreate(members *[]organization_management.DefinitionMember) []map[string]any {
	if members == nil {
		return nil
	}
	var result []map[string]any
	for _, member := range *members {
		memberData := map[string]any{}
		if member.Id != nil {
			memberData[FieldEnterpriseGroupMemberID] = *member.Id
		}
		if member.Email != nil {
			memberData[FieldEnterpriseGroupMemberEmail] = *member.Email
		}
		if member.AddedTime != nil {
			memberData[FieldEnterpriseGroupMemberAddedTime] = member.AddedTime.Format(time.RFC3339)
		}
		if member.Kind != nil {
			switch *member.Kind {
			case organization_management.DefinitionMemberKindSUBJECTKINDUSER:
				memberData[FieldEnterpriseGroupMemberKind] = EnterpriseGroupMemberKindUser
			case organization_management.DefinitionMemberKindSUBJECTKINDSERVICEACCOUNT:
				memberData[FieldEnterpriseGroupMemberKind] = EnterpriseGroupMemberKindServiceAccount
			}
		}
		result = append(result, memberData)
	}
	sortMembersByID(result)
	return result
}

// convertRoleBindingsForBatch converts role bindings from batch response
func convertRoleBindingsForBatch(roleBindings *[]organization_management.GroupRoleBinding) []map[string]any {
	if roleBindings == nil {
		return nil
	}
	var result []map[string]any
	for _, binding := range *roleBindings {
		bindingData := map[string]any{
			FieldEnterpriseGroupRoleBindingID:     binding.Id,
			FieldEnterpriseGroupRoleBindingName:   binding.Name,
			FieldEnterpriseGroupRoleBindingRoleID: binding.Definition.RoleId,
		}
		var scopes []map[string]any
		if binding.Definition.Scopes != nil {
			for _, scope := range *binding.Definition.Scopes {
				scopeData := map[string]any{}
				if scope.Organization != nil {
					scopeData[FieldEnterpriseGroupScopeOrganization] = scope.Organization.Id
				}
				if scope.Cluster != nil {
					scopeData[FieldEnterpriseGroupScopeCluster] = scope.Cluster.Id
				}
				scopes = append(scopes, scopeData)
			}
		}
		bindingData[FieldEnterpriseGroupRoleBindingScopes] = scopes
		result = append(result, bindingData)
	}
	return result
}

// setEnterpriseGroupsData sets the Terraform state from SDK response data
func setEnterpriseGroupsData(data *schema.ResourceData, groups []organization_management.BatchCreateEnterpriseGroupsResponseGroup) error {
	var groupsData []map[string]any

	for _, group := range groups {
		groupData := map[string]any{
			FieldEnterpriseGroupName:           group.Name,
			FieldEnterpriseGroupOrganizationID: group.OrganizationId,
		}

		if group.Id != nil {
			groupData[FieldEnterpriseGroupID] = *group.Id
		}

		if group.Description != nil {
			groupData[FieldEnterpriseGroupDescription] = *group.Description
		}

		if group.CreateTime != nil {
			groupData[FieldEnterpriseGroupCreateTime] = group.CreateTime.Format(time.RFC3339)
		}

		if group.ManagedBy != nil {
			groupData[FieldEnterpriseGroupManagedBy] = *group.ManagedBy
		}

		if group.Definition != nil {
			groupData[FieldEnterpriseGroupMembers] = convertMembersForBatchCreate(group.Definition.Members)
		}

		groupData[FieldEnterpriseGroupRoleBindings] = convertRoleBindingsForBatch(group.RoleBindings)

		groupsData = append(groupsData, groupData)
	}

	// Sort groups by ID for consistent ordering
	sortGroupsByID(groupsData)

	return data.Set(FieldEnterpriseGroupsGroups, groupsData)
}

// setEnterpriseGroupsDataFromListResponse sets the Terraform state from list API response
func setEnterpriseGroupsDataFromListResponse(data *schema.ResourceData, groups []organization_management.ListGroupsResponseGroup) error {
	var groupsData []map[string]any

	for _, group := range groups {
		groupData := map[string]any{}

		if group.Id != nil {
			groupData[FieldEnterpriseGroupID] = *group.Id
		}

		if group.Name != nil {
			groupData[FieldEnterpriseGroupName] = *group.Name
		}

		if group.OrganizationId != nil {
			groupData[FieldEnterpriseGroupOrganizationID] = *group.OrganizationId
		}

		if group.Description != nil {
			groupData[FieldEnterpriseGroupDescription] = *group.Description
		}

		if group.CreateTime != nil {
			groupData[FieldEnterpriseGroupCreateTime] = group.CreateTime.Format(time.RFC3339)
		}

		if group.ManagedBy != nil {
			groupData[FieldEnterpriseGroupManagedBy] = *group.ManagedBy
		}

		// Convert members
		if group.Definition != nil && group.Definition.Members != nil {
			var members []map[string]any
			for _, member := range *group.Definition.Members {
				memberData := map[string]any{}

				if member.Id != nil {
					memberData[FieldEnterpriseGroupMemberID] = *member.Id
				}

				if member.Email != nil {
					memberData[FieldEnterpriseGroupMemberEmail] = *member.Email
				}

				if member.AddedTime != nil {
					memberData[FieldEnterpriseGroupMemberAddedTime] = member.AddedTime.Format(time.RFC3339)
				}

				if member.Kind != nil {
					switch *member.Kind {
					case organization_management.GroupDefinitionMemberKindKINDUSER:
						memberData[FieldEnterpriseGroupMemberKind] = EnterpriseGroupMemberKindUser
					case organization_management.GroupDefinitionMemberKindKINDSERVICEACCOUNT:
						memberData[FieldEnterpriseGroupMemberKind] = EnterpriseGroupMemberKindServiceAccount
					}
				}

				members = append(members, memberData)
			}
			sortMembersByID(members)
			groupData[FieldEnterpriseGroupMembers] = members
		}

		// Note: ListGroupsResponseGroup doesn't include role bindings in the current API
		// They would need to be fetched separately if needed

		groupsData = append(groupsData, groupData)
	}

	// Sort groups by ID for consistent ordering
	sortGroupsByID(groupsData)

	return data.Set(FieldEnterpriseGroupsGroups, groupsData)
}

// setEnterpriseGroupsDataFromUpdateResponse sets the Terraform state from batch update response data
func setEnterpriseGroupsDataFromUpdateResponse(data *schema.ResourceData, groups []organization_management.BatchUpdateEnterpriseGroupsResponseGroup) error {
	var groupsData []map[string]any

	for _, group := range groups {
		groupData := map[string]any{
			FieldEnterpriseGroupName:           group.Name,
			FieldEnterpriseGroupOrganizationID: group.OrganizationId,
		}

		if group.Id != nil {
			groupData[FieldEnterpriseGroupID] = *group.Id
		}

		if group.Description != nil {
			groupData[FieldEnterpriseGroupDescription] = *group.Description
		}

		if group.CreateTime != nil {
			groupData[FieldEnterpriseGroupCreateTime] = group.CreateTime.Format(time.RFC3339)
		}

		if group.ManagedBy != nil {
			groupData[FieldEnterpriseGroupManagedBy] = *group.ManagedBy
		}

		// Convert members
		if group.Definition != nil && group.Definition.Members != nil {
			var members []map[string]any
			for _, member := range *group.Definition.Members {
				memberData := map[string]any{}

				if member.Id != nil {
					memberData[FieldEnterpriseGroupMemberID] = *member.Id
				}

				if member.Email != nil {
					memberData[FieldEnterpriseGroupMemberEmail] = *member.Email
				}

				if member.AddedTime != nil {
					memberData[FieldEnterpriseGroupMemberAddedTime] = member.AddedTime.Format(time.RFC3339)
				}

				if member.Kind != nil {
					switch *member.Kind {
					case organization_management.DefinitionMemberKindSUBJECTKINDUSER:
						memberData[FieldEnterpriseGroupMemberKind] = EnterpriseGroupMemberKindUser
					case organization_management.DefinitionMemberKindSUBJECTKINDSERVICEACCOUNT:
						memberData[FieldEnterpriseGroupMemberKind] = EnterpriseGroupMemberKindServiceAccount
					}
				}

				members = append(members, memberData)
			}
			sortMembersByID(members)
			groupData[FieldEnterpriseGroupMembers] = members
		}

		// Convert role bindings
		if group.RoleBindings != nil {
			var roleBindings []map[string]any
			for _, binding := range *group.RoleBindings {
				bindingData := map[string]any{
					FieldEnterpriseGroupRoleBindingID:     binding.Id,
					FieldEnterpriseGroupRoleBindingName:   binding.Name,
					FieldEnterpriseGroupRoleBindingRoleID: binding.Definition.RoleId,
				}

				// Convert scopes if they exist
				var scopes []map[string]any
				if binding.Definition.Scopes != nil {
					for _, scope := range *binding.Definition.Scopes {
						scopeData := map[string]any{}

						if scope.Organization != nil {
							scopeData[FieldEnterpriseGroupScopeOrganization] = scope.Organization.Id
						}
						if scope.Cluster != nil {
							scopeData[FieldEnterpriseGroupScopeCluster] = scope.Cluster.Id
						}

						scopes = append(scopes, scopeData)
					}
				}
				bindingData[FieldEnterpriseGroupRoleBindingScopes] = scopes

				roleBindings = append(roleBindings, bindingData)
			}
			groupData[FieldEnterpriseGroupRoleBindings] = roleBindings
		}

		groupsData = append(groupsData, groupData)
	}

	// Sort groups by ID for consistent ordering
	sortGroupsByID(groupsData)

	return data.Set(FieldEnterpriseGroupsGroups, groupsData)
}
