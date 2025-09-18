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

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/organization_management"
)

// EnterpriseGroupWithRoleBindings represents a group with its associated role bindings
type EnterpriseGroupWithRoleBindings struct {
	Group        organization_management.ListGroupsResponseGroup
	RoleBindings []organization_management.RoleBinding
}

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

	// Fetch role bindings for managed groups since they are not included in the list response
	// We track ALL role bindings assigned to our groups since we own them
	groupsWithRoleBindings, err := getGroupsRoleBindings(ctx, client, enterpriseID, managedGroups)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching role bindings for groups: %w", err))
	}

	// Update state with only the groups we are managing
	if err := setEnterpriseGroupsDataFromListResponseWithRoleBindings(data, groupsWithRoleBindings); err != nil {
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

	if resp.StatusCode() != http.StatusOK {
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
			return nil, fmt.Errorf("group in state is missing valid ID - this indicates state corruption")
		}

		// Organization ID is also required for deletes
		organizationID, ok := group[FieldEnterpriseGroupOrganizationID].(string)
		if !ok || organizationID == "" {
			return nil, fmt.Errorf("group %s in state is missing valid organization_id - this indicates state corruption", groupID)
		}

		requests = append(requests, organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest{
			Id:             groupID,
			OrganizationId: organizationID,
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

// getGroupsRoleBindings fetches role bindings for each group and merges them into group data
func getGroupsRoleBindings(
	ctx context.Context,
	client organization_management.ClientWithResponsesInterface,
	enterpriseID string,
	groups []organization_management.ListGroupsResponseGroup,
) ([]EnterpriseGroupWithRoleBindings, error) {
	if len(groups) == 0 {
		return nil, nil
	}

	// Collect all group IDs for batch fetching role bindings
	var groupIDs []string
	for _, group := range groups {
		if group.Id != nil {
			groupIDs = append(groupIDs, *group.Id)
		}
	}

	if len(groupIDs) == 0 {
		// Return groups without role bindings
		groupsWithRoleBindings := make([]EnterpriseGroupWithRoleBindings, len(groups))
		for i, group := range groups {
			groupsWithRoleBindings[i] = EnterpriseGroupWithRoleBindings{
				Group:        group,
				RoleBindings: []organization_management.RoleBinding{},
			}
		}
		return groupsWithRoleBindings, nil
	}

	// Fetch role bindings for all groups in one API call
	params := &organization_management.EnterpriseAPIListRoleBindingsParams{
		SubjectId: &groupIDs,
	}

	resp, err := client.EnterpriseAPIListRoleBindingsWithResponse(ctx, enterpriseID, params)
	if err != nil {
		return nil, fmt.Errorf("fetching role bindings: %w", err)
	}

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return nil, fmt.Errorf("role bindings API response: %w", err)
	}

	if resp.JSON200 == nil || resp.JSON200.Items == nil {
		// No role bindings found, return groups without role bindings
		groupsWithRoleBindings := make([]EnterpriseGroupWithRoleBindings, len(groups))
		for i, group := range groups {
			groupsWithRoleBindings[i] = EnterpriseGroupWithRoleBindings{
				Group:        group,
				RoleBindings: []organization_management.RoleBinding{},
			}
		}
		return groupsWithRoleBindings, nil
	}

	roleBindingsByGroupID := make(map[string][]organization_management.RoleBinding)
	for _, roleBinding := range *resp.JSON200.Items {
		// Role bindings in enterprise groups are associated with subjects (groups)
		// We need to check the definition to find which group this role binding belongs to
		if roleBinding.Definition != nil && roleBinding.Definition.Subjects != nil {
			for _, subject := range *roleBinding.Definition.Subjects {
				if subject.Group != nil {
					roleBindingsByGroupID[subject.Group.Id] = append(roleBindingsByGroupID[subject.Group.Id], roleBinding)
				}
			}
		}
	}

	// Create groups with role bindings
	groupsWithRoleBindings := make([]EnterpriseGroupWithRoleBindings, len(groups))
	for i, group := range groups {
		roleBindings := []organization_management.RoleBinding{}
		if group.Id != nil {
			if bindings, exists := roleBindingsByGroupID[*group.Id]; exists {
				roleBindings = bindings
			}
		}

		groupsWithRoleBindings[i] = EnterpriseGroupWithRoleBindings{
			Group:        group,
			RoleBindings: roleBindings,
		}
	}

	return groupsWithRoleBindings, nil
}

func sortByField(items []map[string]any, field string) {
	sort.Slice(items, func(i, j int) bool {
		idI, okI := items[i][field].(string)
		idJ, okJ := items[j][field].(string)
		if !okI || !okJ {
			return false
		}
		return idI < idJ
	})
}

func sortScopesByType(scopes []map[string]any) {
	sort.Slice(scopes, func(i, j int) bool {
		// Sort by type first (cluster before organization), then by ID
		orgI, hasOrgI := scopes[i][FieldEnterpriseGroupScopeOrganization].(string)
		orgJ, hasOrgJ := scopes[j][FieldEnterpriseGroupScopeOrganization].(string)
		clusterI, hasClusterI := scopes[i][FieldEnterpriseGroupScopeCluster].(string)
		clusterJ, hasClusterJ := scopes[j][FieldEnterpriseGroupScopeCluster].(string)

		// If both have cluster scopes, sort by cluster ID
		if hasClusterI && clusterI != "" && hasClusterJ && clusterJ != "" {
			return clusterI < clusterJ
		}

		// If both have org scopes, sort by org ID
		if hasOrgI && orgI != "" && hasOrgJ && orgJ != "" {
			return orgI < orgJ
		}

		// Cluster scopes come before organization scopes
		if hasClusterI && clusterI != "" {
			return true
		}
		if hasClusterJ && clusterJ != "" {
			return false
		}

		return false
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
	sortByField(result, FieldEnterpriseGroupMemberID)
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
		sortScopesByType(scopes)
		bindingData[FieldEnterpriseGroupRoleBindingScopes] = scopes
		result = append(result, bindingData)
	}
	sortByField(result, FieldEnterpriseGroupRoleBindingID)
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
	sortByField(groupsData, FieldEnterpriseGroupID)

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
			sortByField(members, FieldEnterpriseGroupMemberID)
			groupData[FieldEnterpriseGroupMembers] = members
		}

		// Note: ListGroupsResponseGroup doesn't include role bindings in the current API
		// They would need to be fetched separately if needed

		groupsData = append(groupsData, groupData)
	}

	// Sort groups by ID for consistent ordering
	sortByField(groupsData, FieldEnterpriseGroupID)

	return data.Set(FieldEnterpriseGroupsGroups, groupsData)
}

// convertRoleBindingsForState converts API role bindings to Terraform state format
func convertRoleBindingsForState(roleBindings []organization_management.RoleBinding) []map[string]any {
	var roleBindingsData []map[string]any

	for _, roleBinding := range roleBindings {
		roleBindingData := map[string]any{}

		if roleBinding.Id != nil {
			roleBindingData[FieldEnterpriseGroupRoleBindingID] = *roleBinding.Id
		}

		if roleBinding.Name != nil {
			roleBindingData[FieldEnterpriseGroupRoleBindingName] = *roleBinding.Name
		}

		if roleBinding.Definition != nil {
			if roleBinding.Definition.RoleId != nil {
				roleBindingData[FieldEnterpriseGroupRoleBindingRoleID] = *roleBinding.Definition.RoleId
			}

			// Convert scopes
			if roleBinding.Definition.Scopes != nil {
				var scopesData []map[string]any
				for _, scope := range *roleBinding.Definition.Scopes {
					scopeData := map[string]any{}

					if scope.Organization != nil {
						scopeData[FieldEnterpriseGroupScopeOrganization] = scope.Organization.Id
					}

					if scope.Cluster != nil {
						scopeData[FieldEnterpriseGroupScopeCluster] = scope.Cluster.Id
					}

					scopesData = append(scopesData, scopeData)
				}
				// Sort scopes for consistent ordering
				sortScopesByType(scopesData)
				roleBindingData[FieldEnterpriseGroupRoleBindingScopes] = scopesData
			}
		}

		roleBindingsData = append(roleBindingsData, roleBindingData)
	}

	// Sort role bindings by ID for consistent ordering
	sortByField(roleBindingsData, FieldEnterpriseGroupRoleBindingID)

	return roleBindingsData
}

// setEnterpriseGroupsDataFromListResponseWithRoleBindings sets the Terraform state from list API response enriched with role bindings
func setEnterpriseGroupsDataFromListResponseWithRoleBindings(data *schema.ResourceData, groupsWithRoleBindings []EnterpriseGroupWithRoleBindings) error {
	var groupsData []map[string]any

	for _, groupWithRoleBindings := range groupsWithRoleBindings {
		group := groupWithRoleBindings.Group
		groupData := map[string]any{}

		if group.Id != nil {
			groupData[FieldEnterpriseGroupID] = *group.Id
		}

		if group.Name != nil {
			groupData[FieldEnterpriseGroupName] = *group.Name
		}

		if group.Description != nil {
			groupData[FieldEnterpriseGroupDescription] = *group.Description
		}

		if group.OrganizationId != nil {
			groupData[FieldEnterpriseGroupOrganizationID] = *group.OrganizationId
		}

		// Add computed fields
		if group.CreateTime != nil {
			groupData[FieldEnterpriseGroupCreateTime] = group.CreateTime.Format(time.RFC3339)
		}

		if group.ManagedBy != nil {
			groupData[FieldEnterpriseGroupManagedBy] = *group.ManagedBy
		}

		// Convert members
		var members []map[string]any
		if group.Definition != nil && group.Definition.Members != nil {
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
			sortByField(members, FieldEnterpriseGroupMemberID)
			groupData[FieldEnterpriseGroupMembers] = members
		}

		// Convert role bindings
		roleBindings := convertRoleBindingsForState(groupWithRoleBindings.RoleBindings)
		groupData[FieldEnterpriseGroupRoleBindings] = roleBindings

		groupsData = append(groupsData, groupData)
	}

	// Sort groups by ID for consistent ordering
	sortByField(groupsData, FieldEnterpriseGroupID)

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
			sortByField(members, FieldEnterpriseGroupMemberID)
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
				sortScopesByType(scopes)
				bindingData[FieldEnterpriseGroupRoleBindingScopes] = scopes

				roleBindings = append(roleBindings, bindingData)
			}
			sortByField(roleBindings, FieldEnterpriseGroupRoleBindingID)
			groupData[FieldEnterpriseGroupRoleBindings] = roleBindings
		}

		groupsData = append(groupsData, groupData)
	}

	// Sort groups by ID for consistent ordering
	sortByField(groupsData, FieldEnterpriseGroupID)

	return data.Set(FieldEnterpriseGroupsGroups, groupsData)
}
