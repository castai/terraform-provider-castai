package castai

import (
	"context"
	"fmt"
	"log"
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

	// Field names for nested objects
	FieldEnterpriseGroupsGroup      = "group"
	FieldEnterpriseGroupsMember     = "member"
	FieldEnterpriseGroupRoleBinding = "role_binding"
	FieldEnterpriseGroupScope       = "scope"

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
						FieldEnterpriseGroupsGroup: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Enterprise group configuration.",
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
										ForceNew:    true,
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
												FieldEnterpriseGroupsMember: {
													Type:        schema.TypeList,
													Optional:    true,
													Description: "Group member configuration.",
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
											},
										},
									},
									FieldEnterpriseGroupRoleBindings: {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "List of role bindings for the group.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldEnterpriseGroupRoleBinding: {
													Type:        schema.TypeList,
													Optional:    true,
													Description: "Role binding configuration.",
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
																		FieldEnterpriseGroupScope: {
																			Type:        schema.TypeList,
																			Optional:    true,
																			Description: "Scope configuration.",
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

	if err = setEnterpriseCreatedGroupsData(data, *resp.JSON200.Groups); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set created groups data: %w", err))
	}

	data.SetId(enterpriseID)

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

	log.Printf("[INFO] Reading enterprise groups for enterprise ID %s, managing %d groups", enterpriseID, len(managedGroupIDs))

	// Call list groups API to get current state
	resp, err := client.EnterpriseAPIListGroupsWithResponse(ctx, enterpriseID, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("listing enterprise groups: %w", err))
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

	log.Printf("[INFO] Found %d managed groups in enterprise ID %s", len(managedGroups), enterpriseID)

	// Sort groups to match the configuration order to prevent Terraform drift
	// We need to match the order in configuration, not sort by ID or name
	managedGroups = sortGroupsByConfigOrder(data, managedGroups)

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

	changes, err := getGroupChanges(data)
	if err != nil {
		return diag.FromErr(fmt.Errorf("analyzing group changes: %w", err))
	}

	if len(changes.toDelete) > 0 {
		deleteRequest := &organization_management.BatchDeleteEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests:     changes.toDelete,
		}

		resp, err := client.EnterpriseAPIBatchDeleteEnterpriseGroupsWithResponse(ctx, enterpriseID, *deleteRequest)
		if err != nil {
			return diag.FromErr(fmt.Errorf("deleting removed groups: %w", err))
		}

		if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNotFound {
			return diag.FromErr(fmt.Errorf("batch delete removed groups failed with status %d: %s", resp.StatusCode(), string(resp.Body)))
		}
	}

	if len(changes.toCreate) > 0 {
		createRequest := &organization_management.BatchCreateEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests:     changes.toCreate,
		}

		resp, err := client.EnterpriseAPIBatchCreateEnterpriseGroupsWithResponse(ctx, enterpriseID, *createRequest)
		if err != nil {
			return diag.FromErr(fmt.Errorf("creating new groups: %w", err))
		}

		if resp.StatusCode() != http.StatusOK {
			return diag.FromErr(fmt.Errorf("batch create new groups failed with status %d: %s", resp.StatusCode(), string(resp.Body)))
		}

		if resp.JSON200 != nil && resp.JSON200.Groups != nil {
			if err = setEnterpriseCreatedGroupsData(data, *resp.JSON200.Groups); err != nil {
				return diag.FromErr(fmt.Errorf("failed to set created groups data: %w", err))
			}
		}
	}

	// Handle modifications
	if len(changes.toUpdate) > 0 {
		updateRequest := &organization_management.BatchUpdateEnterpriseGroupsRequest{
			EnterpriseId: enterpriseID,
			Requests:     changes.toUpdate,
		}

		resp, err := client.EnterpriseAPIBatchUpdateEnterpriseGroupsWithResponse(ctx, enterpriseID, *updateRequest)
		if err != nil {
			return diag.FromErr(fmt.Errorf("updating modified groups: %w", err))
		}

		if resp.StatusCode() != http.StatusOK {
			return diag.FromErr(fmt.Errorf("batch update modified groups failed with status %d: %s", resp.StatusCode(), string(resp.Body)))
		}
	}

	// Refresh the entire state by reading current data
	return resourceEnterpriseGroupsRead(ctx, data, meta)
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
	groupsListAny := data.Get(FieldEnterpriseGroupsGroups)
	groupsList, ok := groupsListAny.([]any)
	if !ok {
		return nil, fmt.Errorf("groups data is not in expected format")
	}

	var requests []organization_management.BatchCreateEnterpriseGroupsRequestGroup

	for i, groupData := range groupsList {
		if groupData == nil {
			continue // nil entries are acceptable in Terraform lists
		}

		groupWrapper, ok := groupData.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid group configuration at index %d: expected object, got %T", i, groupData)
		}

		// Navigate to the nested group objects
		groupsData, ok := groupWrapper[FieldEnterpriseGroupsGroup].([]any)
		if !ok {
			return nil, fmt.Errorf("group configuration missing required 'group' field at index %d", i)
		}
		if len(groupsData) == 0 {
			continue // Empty group arrays are acceptable
		}

		// Process all groups in the nested array
		for j, groupDataNested := range groupsData {
			if groupDataNested == nil {
				continue // nil entries are acceptable in nested arrays
			}

			group, ok := groupDataNested.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid nested group configuration at index %d.%d: expected object, got %T", i, j, groupDataNested)
			}

			// Use the already-fixed buildCreateRequestForGroup function
			createRequest, err := buildCreateRequestForGroup(group)
			if err != nil {
				return nil, fmt.Errorf("building create request for group at index %d.%d: %w", i, j, err)
			}

			requests = append(requests, *createRequest)
		}
	}

	return &organization_management.BatchCreateEnterpriseGroupsRequest{
		EnterpriseId: enterpriseID,
		Requests:     requests,
	}, nil
}

// buildBatchDeleteRequest constructs the batch delete request from Terraform schema data
func buildBatchDeleteRequest(enterpriseID string, data *schema.ResourceData) (*organization_management.BatchDeleteEnterpriseGroupsRequest, error) {
	groupsListAny := data.Get(FieldEnterpriseGroupsGroups)
	groupsList, ok := groupsListAny.([]any)
	if !ok {
		return nil, fmt.Errorf("groups data is not in expected format")
	}

	var requests []organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest

	for _, groupData := range groupsList {
		if groupData == nil {
			continue
		}

		groupWrapper, ok := groupData.(map[string]any)
		if !ok {
			continue
		}

		// Navigate to the nested group objects
		groupsData, ok := groupWrapper[FieldEnterpriseGroupsGroup].([]any)
		if !ok || len(groupsData) == 0 {
			continue // Skip if no group data
		}

		// Process all groups in the nested array
		for _, groupDataNested := range groupsData {
			if groupDataNested == nil {
				continue
			}

			group, ok := groupDataNested.(map[string]any)
			if !ok {
				continue
			}

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
	}

	return &organization_management.BatchDeleteEnterpriseGroupsRequest{
		EnterpriseId: enterpriseID,
		Requests:     requests,
	}, nil
}

// getManagedGroupIDsFromState extracts the group IDs from Terraform state that this resource should manage
func getManagedGroupIDsFromState(data *schema.ResourceData) map[string]bool {
	managedIDs := make(map[string]bool)

	groupsListAny := data.Get(FieldEnterpriseGroupsGroups)
	groupsList, ok := groupsListAny.([]any)
	if !ok {
		return managedIDs
	}

	for _, groupData := range groupsList {
		if groupData == nil {
			continue
		}

		groupWrapper, ok := groupData.(map[string]any)
		if !ok {
			continue
		}

		// Navigate to the nested group objects
		groupsData, ok := groupWrapper[FieldEnterpriseGroupsGroup].([]any)
		if !ok || len(groupsData) == 0 {
			continue
		}

		// Process all groups in the nested array
		for _, groupDataNested := range groupsData {
			if groupDataNested == nil {
				continue
			}

			group, ok := groupDataNested.(map[string]any)
			if !ok {
				continue
			}

			if groupID, ok := group[FieldEnterpriseGroupID].(string); ok && groupID != "" {
				managedIDs[groupID] = true
			}
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

// sortGroupsByConfigOrder sorts API groups to match the order they appear in Terraform configuration
func sortGroupsByConfigOrder(data *schema.ResourceData, apiGroups []organization_management.ListGroupsResponseGroup) []organization_management.ListGroupsResponseGroup {
	// Get the configuration order by extracting group names in order
	configOrder := getGroupNamesFromConfig(data)
	if len(configOrder) == 0 {
		return apiGroups
	}

	// Create a map from group name to its position in config
	nameToOrder := make(map[string]int)
	for i, name := range configOrder {
		nameToOrder[name] = i
	}

	// Sort API groups by their configuration order
	sort.Slice(apiGroups, func(i, j int) bool {
		nameI := ""
		nameJ := ""
		if apiGroups[i].Name != nil {
			nameI = *apiGroups[i].Name
		}
		if apiGroups[j].Name != nil {
			nameJ = *apiGroups[j].Name
		}

		orderI, foundI := nameToOrder[nameI]
		orderJ, foundJ := nameToOrder[nameJ]

		// If both found, sort by config order
		if foundI && foundJ {
			return orderI < orderJ
		}
		// If only one found, prioritize the one in config
		if foundI {
			return true
		}
		if foundJ {
			return false
		}
		// If neither found, sort alphabetically as fallback
		return nameI < nameJ
	})

	return apiGroups
}

// getGroupNamesFromConfig extracts group names in the order they appear in configuration
func getGroupNamesFromConfig(data *schema.ResourceData) []string {
	var names []string

	groupsListAny := data.Get(FieldEnterpriseGroupsGroups)
	groupsList, ok := groupsListAny.([]any)
	if !ok {
		return names
	}

	for _, groupData := range groupsList {
		if groupData == nil {
			continue
		}

		groupWrapper, ok := groupData.(map[string]any)
		if !ok {
			continue
		}

		// Navigate to the nested group objects
		groupsData, ok := groupWrapper[FieldEnterpriseGroupsGroup].([]any)
		if !ok || len(groupsData) == 0 {
			continue
		}

		// Process all groups in the nested array
		for _, groupDataNested := range groupsData {
			if groupDataNested == nil {
				continue
			}

			group, ok := groupDataNested.(map[string]any)
			if !ok {
				continue
			}

			if groupName, ok := group[FieldEnterpriseGroupName].(string); ok && groupName != "" {
				names = append(names, groupName)
			}
		}
	}

	return names
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

// sortByFieldNested sorts items by a field that is nested one level deep
func sortByFieldNested(items []map[string]any, wrapperField, field string) {
	sort.Slice(items, func(i, j int) bool {
		// Navigate to nested data
		wrapperI, okI := items[i][wrapperField].([]map[string]any)
		wrapperJ, okJ := items[j][wrapperField].([]map[string]any)
		if !okI || !okJ || len(wrapperI) == 0 || len(wrapperJ) == 0 {
			return false
		}

		idI, okI := wrapperI[0][field].(string)
		idJ, okJ := wrapperJ[0][field].(string)
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

	// Collect all member data
	var allMemberData []map[string]any
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
		allMemberData = append(allMemberData, memberData)
	}

	// Create single members wrapper containing all members
	if len(allMemberData) > 0 {
		memberWrapper := map[string]any{
			FieldEnterpriseGroupsMember: allMemberData,
		}
		return []map[string]any{memberWrapper}
	}

	return nil
}

// convertRoleBindingsForBatch converts role bindings from batch response
func convertRoleBindingsForBatch(roleBindings *[]organization_management.GroupRoleBinding) []map[string]any {
	if roleBindings == nil {
		return nil
	}

	// Collect all role binding data
	var allRoleBindingData []map[string]any
	for _, binding := range *roleBindings {
		bindingData := map[string]any{
			FieldEnterpriseGroupRoleBindingID:     binding.Id,
			FieldEnterpriseGroupRoleBindingName:   binding.Name,
			FieldEnterpriseGroupRoleBindingRoleID: binding.Definition.RoleId,
		}
		// Collect all scope data
		var allScopeData []map[string]any
		if binding.Definition.Scopes != nil {
			for _, scope := range *binding.Definition.Scopes {
				scopeData := map[string]any{}
				if scope.Organization != nil {
					scopeData[FieldEnterpriseGroupScopeOrganization] = scope.Organization.Id
				}
				if scope.Cluster != nil {
					scopeData[FieldEnterpriseGroupScopeCluster] = scope.Cluster.Id
				}
				allScopeData = append(allScopeData, scopeData)
			}
		}

		// Create single scopes wrapper containing all scopes
		var scopes []map[string]any
		if len(allScopeData) > 0 {
			scopeWrapper := map[string]any{
				FieldEnterpriseGroupScope: allScopeData,
			}
			scopes = []map[string]any{scopeWrapper}
		}
		bindingData[FieldEnterpriseGroupRoleBindingScopes] = scopes
		allRoleBindingData = append(allRoleBindingData, bindingData)
	}

	// Create single role bindings wrapper containing all role bindings
	if len(allRoleBindingData) > 0 {
		sortByField(allRoleBindingData, FieldEnterpriseGroupRoleBindingID)
		roleBindingWrapper := map[string]any{
			FieldEnterpriseGroupRoleBinding: allRoleBindingData,
		}
		return []map[string]any{roleBindingWrapper}
	}

	return nil
}

// setEnterpriseCreatedGroupsData sets the Terraform state from SDK response data
func setEnterpriseCreatedGroupsData(
	data *schema.ResourceData,
	groups []organization_management.BatchCreateEnterpriseGroupsResponseGroup,
) error {
	type group struct {
		organizationID string
		name           string
	}

	groupIDToGroup := lo.SliceToMap(groups,
		func(item organization_management.BatchCreateEnterpriseGroupsResponseGroup) (group, organization_management.BatchCreateEnterpriseGroupsResponseGroup) {
			g := group{
				organizationID: *item.OrganizationId,
				name:           *item.Name,
			}

			return g, item
		},
	)

	configGroups := data.Get(FieldEnterpriseGroupsGroups).([]any)
	var groupsData []map[string]any

	for _, configGroup := range configGroups {
		configGroupWrapper := configGroup.(map[string]any)
		configGroupsData := configGroupWrapper[FieldEnterpriseGroupsGroup].([]any)

		for i, configGroupNested := range configGroupsData {
			configGroupMap := configGroupNested.(map[string]any)
			configGroupName := configGroupMap[FieldEnterpriseGroupName].(string)
			configOrganizationID := configGroupMap[FieldEnterpriseGroupOrganizationID].(string)

			g := group{
				organizationID: configOrganizationID,
				name:           configGroupName,
			}

			matchedGroup, ok := groupIDToGroup[g]
			if !ok {
				return fmt.Errorf("created group not found in response: name=%s, organization_id=%s", configGroupName, configOrganizationID)
			}

			groupData := map[string]any{
				FieldEnterpriseGroupName:           matchedGroup.Name,
				FieldEnterpriseGroupOrganizationID: matchedGroup.OrganizationId,
			}

			if matchedGroup.Id != nil {
				groupData[FieldEnterpriseGroupID] = *matchedGroup.Id
			}

			if matchedGroup.Description != nil {
				groupData[FieldEnterpriseGroupDescription] = *matchedGroup.Description
			}

			if matchedGroup.CreateTime != nil {
				groupData[FieldEnterpriseGroupCreateTime] = matchedGroup.CreateTime.Format(time.RFC3339)
			}

			if matchedGroup.ManagedBy != nil {
				groupData[FieldEnterpriseGroupManagedBy] = *matchedGroup.ManagedBy
			}

			if matchedGroup.Definition != nil {
				groupData[FieldEnterpriseGroupMembers] = convertMembersForBatchCreate(matchedGroup.Definition.Members)
			}

			groupData[FieldEnterpriseGroupRoleBindings] = convertRoleBindingsForBatch(matchedGroup.RoleBindings)

			// Wrap the group data in nested structure
			groupWrapper := map[string]any{
				FieldEnterpriseGroupsGroup: []map[string]any{groupData},
			}

			configGroupsData[i] = groupWrapper
		}
	}

	return data.Set(FieldEnterpriseGroupsGroups, groupsData)
}

// convertRoleBindingsForState converts API role bindings to Terraform state format
func convertRoleBindingsForState(roleBindings []organization_management.RoleBinding) []map[string]any {
	var allRoleBindingData []map[string]any

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
				var allScopeData []map[string]any
				for _, scope := range *roleBinding.Definition.Scopes {
					scopeData := map[string]any{}

					if scope.Organization != nil {
						scopeData[FieldEnterpriseGroupScopeOrganization] = scope.Organization.Id
					}

					if scope.Cluster != nil {
						scopeData[FieldEnterpriseGroupScopeCluster] = scope.Cluster.Id
					}

					allScopeData = append(allScopeData, scopeData)
				}

				// Create single scopes wrapper containing all scopes
				if len(allScopeData) > 0 {
					scopeWrapper := map[string]any{
						FieldEnterpriseGroupScope: allScopeData,
					}
					roleBindingData[FieldEnterpriseGroupRoleBindingScopes] = []map[string]any{scopeWrapper}
				}
			}
		}

		allRoleBindingData = append(allRoleBindingData, roleBindingData)
	}

	// Create single role bindings wrapper containing all role bindings
	if len(allRoleBindingData) > 0 {
		// Sort role bindings by ID for consistent ordering
		sortByField(allRoleBindingData, FieldEnterpriseGroupRoleBindingID)
		roleBindingWrapper := map[string]any{
			FieldEnterpriseGroupRoleBinding: allRoleBindingData,
		}
		return []map[string]any{roleBindingWrapper}
	}

	return []map[string]any{}
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

		// Convert members with nested structure
		if group.Definition != nil && group.Definition.Members != nil {
			// Collect all member data
			var allMemberData []map[string]any
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

				allMemberData = append(allMemberData, memberData)
			}

			// Create single members wrapper containing all members
			if len(allMemberData) > 0 {
				memberWrapper := map[string]any{
					FieldEnterpriseGroupsMember: allMemberData,
				}
				groupData[FieldEnterpriseGroupMembers] = []map[string]any{memberWrapper}
			}
		}

		// Convert role bindings
		roleBindings := convertRoleBindingsForState(groupWithRoleBindings.RoleBindings)
		groupData[FieldEnterpriseGroupRoleBindings] = roleBindings

		// Wrap the group data in nested structure
		groupWrapper := map[string]any{
			FieldEnterpriseGroupsGroup: []map[string]any{groupData},
		}

		groupsData = append(groupsData, groupWrapper)
	}

	// Groups are already sorted by config order from the Read function
	// No need to re-sort here to avoid disrupting the configuration order

	return data.Set(FieldEnterpriseGroupsGroups, groupsData)
}

// EnterpriseGroupsChanges represents the changes needed during an update operation
type EnterpriseGroupsChanges struct {
	toCreate []organization_management.BatchCreateEnterpriseGroupsRequestGroup
	toUpdate []organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest
	toDelete []organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest
}

func getGroupChanges(data *schema.ResourceData) (*EnterpriseGroupsChanges, error) {
	changes := &EnterpriseGroupsChanges{
		toCreate: []organization_management.BatchCreateEnterpriseGroupsRequestGroup{},
		toUpdate: []organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest{},
		toDelete: []organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest{},
	}

	if !data.HasChange(FieldEnterpriseGroupsGroups) {
		// No changes to groups
		return changes, nil
	}

	oldValue, newValue := data.GetChange(FieldEnterpriseGroupsGroups)

	oldGroups, ok := oldValue.([]any)
	if !ok {
		oldGroups = []any{}
	}

	newGroups, ok := newValue.([]any)
	if !ok {
		newGroups = []any{}
	}

	oldGroupIDs := []string{}
	oldGroupIDToGroup := make(map[string]map[string]any)
	newGroupIDs := []string{}
	newGroupIDToGroup := make(map[string]map[string]any)

	for _, groupData := range oldGroups {
		if groupData == nil {
			continue
		}

		groupWrapper, ok := groupData.(map[string]any)
		if !ok {
			continue
		}

		// Navigate to the nested group objects
		groupsData, ok := groupWrapper[FieldEnterpriseGroupsGroup].([]any)
		if !ok || len(groupsData) == 0 {
			continue
		}

		// Process all groups in the nested array
		for _, groupDataNested := range groupsData {
			if groupDataNested == nil {
				continue
			}

			group, ok := groupDataNested.(map[string]any)
			if !ok {
				continue
			}

			if groupID, ok := group[FieldEnterpriseGroupID].(string); ok && groupID != "" {
				oldGroupIDs = append(oldGroupIDs, groupID)
				oldGroupIDToGroup[groupID] = group
			}
		}
	}

	for _, groupData := range newGroups {
		if groupData == nil {
			continue
		}

		groupWrapper, ok := groupData.(map[string]any)
		if !ok {
			continue
		}

		// Navigate to the nested group objects
		groupsData, ok := groupWrapper[FieldEnterpriseGroupsGroup].([]any)
		if !ok || len(groupsData) == 0 {
			continue
		}

		// Process all groups in the nested array
		for _, groupDataNested := range groupsData {
			if groupDataNested == nil {
				continue
			}

			group, ok := groupDataNested.(map[string]any)
			if !ok {
				continue
			}

			if groupID, ok := group[FieldEnterpriseGroupID].(string); ok && groupID != "" {
				newGroupIDs = append(newGroupIDs, groupID)
				newGroupIDToGroup[groupID] = group
			} else {
				// New group without ID - needs to be created
				createRequest, err := buildCreateRequestForGroup(group)
				if err != nil {
					return nil, fmt.Errorf("building create request for new group: %w", err)
				}
				changes.toCreate = append(changes.toCreate, *createRequest)
			}
		}
	}

	for _, groupID := range lo.Intersect(oldGroupIDs, newGroupIDs) {
		updateRequest, err := buildUpdateRequestForGroup(groupID, newGroupIDToGroup[groupID])
		if err != nil {
			return nil, fmt.Errorf("building update request for group %s: %w", groupID, err)
		}
		changes.toUpdate = append(changes.toUpdate, *updateRequest)
	}

	toDeleteGroupIDs, _ := lo.Difference(oldGroupIDs, newGroupIDs)
	for _, groupID := range toDeleteGroupIDs {
		orgID, ok := oldGroupIDToGroup[groupID][FieldEnterpriseGroupOrganizationID].(string)
		if !ok || orgID == "" {
			return nil, fmt.Errorf("group %s in old state is missing organization_id", groupID)
		}

		changes.toDelete = append(changes.toDelete, organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest{
			Id:             groupID,
			OrganizationId: orgID,
		})
	}

	return changes, nil
}

// buildUpdateRequestForGroup creates an update request for a single group
func buildUpdateRequestForGroup(groupID string, group map[string]any) (*organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest, error) {
	// Convert members with nested structure
	var members []organization_management.BatchUpdateEnterpriseGroupsRequestMember
	if membersData, ok := group[FieldEnterpriseGroupMembers].([]any); ok {
		for _, memberWrapper := range membersData {
			if memberWrapper == nil {
				continue
			}

			memberWrapperMap, ok := memberWrapper.(map[string]any)
			if !ok {
				continue
			}

			// Navigate to the nested member object
			membersDataNested, ok := memberWrapperMap[FieldEnterpriseGroupsMember].([]any)
			if !ok || len(membersDataNested) == 0 {
				continue
			}

			// Process all members in the nested array
			for _, memberData := range membersDataNested {
				if memberData == nil {
					continue
				}

				member, ok := memberData.(map[string]any)
				if !ok {
					continue
				}

				memberKind, ok := member[FieldEnterpriseGroupMemberKind].(string)
				if !ok {
					continue
				}

				memberID, ok := member[FieldEnterpriseGroupMemberID].(string)
				if !ok {
					continue
				}

				var kind organization_management.BatchUpdateEnterpriseGroupsRequestMemberKind
				switch memberKind {
				case EnterpriseGroupMemberKindUser:
					kind = organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindUSER
				case EnterpriseGroupMemberKindServiceAccount:
					kind = organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindSERVICEACCOUNT
				default:
					kind = organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindSUBJECTKINDUNSPECIFIED
				}

				members = append(members, organization_management.BatchUpdateEnterpriseGroupsRequestMember{
					Kind: kind,
					Id:   memberID,
				})
			}
		}
	}

	// Convert role bindings
	var roleBindings []organization_management.BatchUpdateEnterpriseGroupsRequestRoleBinding
	if bindingsData, ok := group[FieldEnterpriseGroupRoleBindings].([]any); ok {
		for _, bindingWrapper := range bindingsData {
			if bindingWrapper == nil {
				continue
			}

			bindingWrapperMap, ok := bindingWrapper.(map[string]any)
			if !ok {
				continue
			}

			// Navigate to the nested role binding object
			bindingsDataNested, ok := bindingWrapperMap[FieldEnterpriseGroupRoleBinding].([]any)
			if !ok || len(bindingsDataNested) == 0 {
				continue
			}

			// Process all role bindings in the nested array
			for _, bindingData := range bindingsDataNested {
				if bindingData == nil {
					continue
				}

				binding, ok := bindingData.(map[string]any)
				if !ok {
					continue
				}

				bindingName, ok := binding[FieldEnterpriseGroupRoleBindingName].(string)
				if !ok {
					continue
				}

				bindingRoleID, ok := binding[FieldEnterpriseGroupRoleBindingRoleID].(string)
				if !ok {
					continue
				}

				var scopes []organization_management.Scope
				if scopesData, ok := binding[FieldEnterpriseGroupRoleBindingScopes].([]any); ok {
					for _, scopeWrapper := range scopesData {
						if scopeWrapper == nil {
							continue
						}

						scopeWrapperMap, ok := scopeWrapper.(map[string]any)
						if !ok {
							continue
						}

						// Navigate to the nested scope object
						scopesDataNested, ok := scopeWrapperMap[FieldEnterpriseGroupScope].([]any)
						if !ok || len(scopesDataNested) == 0 {
							continue
						}

						// Process all scopes in the nested array
						for _, scopeData := range scopesDataNested {
							if scopeData == nil {
								continue
							}

							scope, ok := scopeData.(map[string]any)
							if !ok {
								continue
							}

							orgID, _ := scope[FieldEnterpriseGroupScopeOrganization].(string)
							clusterID, _ := scope[FieldEnterpriseGroupScopeCluster].(string)

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
				}

				bindingID := ""
				if id, ok := binding[FieldEnterpriseGroupRoleBindingID].(string); ok && id != "" {
					bindingID = id
				}

				roleBindings = append(roleBindings, organization_management.BatchUpdateEnterpriseGroupsRequestRoleBinding{
					Id:     bindingID,
					Name:   bindingName,
					RoleId: bindingRoleID,
					Scopes: scopes,
				})
			}
		}
	}

	// Safe extraction of required fields with error handling
	groupName, ok := group[FieldEnterpriseGroupName].(string)
	if !ok {
		return nil, fmt.Errorf("group name is required for group %s", groupID)
	}

	orgID, ok := group[FieldEnterpriseGroupOrganizationID].(string)
	if !ok {
		return nil, fmt.Errorf("organization ID is required for group %s", groupID)
	}

	groupDesc, _ := group[FieldEnterpriseGroupDescription].(string)

	return &organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest{
		Id:             groupID,
		Name:           groupName,
		OrganizationId: orgID,
		Description:    groupDesc,
		Members:        members,
		RoleBindings:   roleBindings,
	}, nil
}

// buildCreateRequestForGroup creates a create request for a single new group
func buildCreateRequestForGroup(group map[string]any) (*organization_management.BatchCreateEnterpriseGroupsRequestGroup, error) {
	var members []organization_management.BatchCreateEnterpriseGroupsRequestMember
	if membersData, ok := group[FieldEnterpriseGroupMembers].([]any); ok {
		for _, memberWrapper := range membersData {
			if memberWrapper == nil {
				continue
			}

			memberWrapperMap, ok := memberWrapper.(map[string]any)
			if !ok {
				continue
			}

			// Navigate to the nested member object
			membersDataNested, ok := memberWrapperMap[FieldEnterpriseGroupsMember].([]any)
			if !ok || len(membersDataNested) == 0 {
				continue
			}

			// Process all members in the nested array
			for _, memberData := range membersDataNested {
				if memberData == nil {
					continue
				}

				member, ok := memberData.(map[string]any)
				if !ok {
					continue
				}

				memberKind, ok := member[FieldEnterpriseGroupMemberKind].(string)
				if !ok {
					continue
				}

				memberID, ok := member[FieldEnterpriseGroupMemberID].(string)
				if !ok {
					continue
				}

				var kind organization_management.BatchCreateEnterpriseGroupsRequestMemberKind
				switch memberKind {
				case EnterpriseGroupMemberKindUser:
					kind = organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDUSER
				case EnterpriseGroupMemberKindServiceAccount:
					kind = organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDSERVICEACCOUNT
				default:
					kind = organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDUNSPECIFIED
				}

				members = append(members, organization_management.BatchCreateEnterpriseGroupsRequestMember{
					Kind: &kind,
					Id:   lo.ToPtr(memberID),
				})
			}
		}
	}

	var roleBindings *[]organization_management.BatchCreateEnterpriseGroupsRequestRoleBinding
	if bindingsData, ok := group[FieldEnterpriseGroupRoleBindings].([]any); ok && len(bindingsData) > 0 {
		var bindings []organization_management.BatchCreateEnterpriseGroupsRequestRoleBinding

		for _, bindingWrapper := range bindingsData {
			if bindingWrapper == nil {
				continue
			}

			bindingWrapperMap, ok := bindingWrapper.(map[string]any)
			if !ok {
				continue
			}

			// Navigate to the nested role binding object
			bindingsDataNested, ok := bindingWrapperMap[FieldEnterpriseGroupRoleBinding].([]any)
			if !ok || len(bindingsDataNested) == 0 {
				continue
			}

			// Process all role bindings in the nested array
			for _, bindingData := range bindingsDataNested {
				if bindingData == nil {
					continue
				}

				binding, ok := bindingData.(map[string]any)
				if !ok {
					continue
				}

				bindingName, ok := binding[FieldEnterpriseGroupRoleBindingName].(string)
				if !ok {
					continue
				}

				bindingRoleID, ok := binding[FieldEnterpriseGroupRoleBindingRoleID].(string)
				if !ok {
					continue
				}

				var scopes []organization_management.Scope
				if scopesData, ok := binding[FieldEnterpriseGroupRoleBindingScopes].([]any); ok {
					for _, scopeWrapper := range scopesData {
						if scopeWrapper == nil {
							continue
						}

						scopeWrapperMap, ok := scopeWrapper.(map[string]any)
						if !ok {
							continue
						}

						// Navigate to the nested scope object
						scopesDataNested, ok := scopeWrapperMap[FieldEnterpriseGroupScope].([]any)
						if !ok || len(scopesDataNested) == 0 {
							continue
						}

						// Process all scopes in the nested array
						for _, scopeData := range scopesDataNested {
							if scopeData == nil {
								continue
							}

							scope, ok := scopeData.(map[string]any)
							if !ok {
								continue
							}

							orgID, _ := scope[FieldEnterpriseGroupScopeOrganization].(string)
							clusterID, _ := scope[FieldEnterpriseGroupScopeCluster].(string)

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
				}

				bindings = append(bindings, organization_management.BatchCreateEnterpriseGroupsRequestRoleBinding{
					Name:   bindingName,
					RoleId: bindingRoleID,
					Scopes: scopes,
				})
			}
		}

		roleBindings = &bindings
	}

	groupName, ok := group[FieldEnterpriseGroupName].(string)
	if !ok {
		return nil, fmt.Errorf("group name is required")
	}

	orgID, ok := group[FieldEnterpriseGroupOrganizationID].(string)
	if !ok {
		return nil, fmt.Errorf("organization ID is required")
	}

	groupRequest := organization_management.BatchCreateEnterpriseGroupsRequestGroup{
		Name:           groupName,
		OrganizationId: orgID,
		Members:        members,
		RoleBindings:   roleBindings,
	}

	if desc, ok := group[FieldEnterpriseGroupDescription].(string); ok && desc != "" {
		groupRequest.Description = &desc
	}

	return &groupRequest, nil
}
