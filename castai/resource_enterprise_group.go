package castai

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/organization_management"
)

// EnterpriseGroupWithRoleBindings represents a group with its associated role bindings
type EnterpriseGroupWithRoleBindings struct {
	Group        Group
	RoleBindings []RoleBinding
}

type Group struct {
	ID             string
	Name           string
	OrganizationID string
	Description    *string
	CreateTime     *time.Time
	ManagedBy      *string
	Members        []Member
}

type Member struct {
	Kind      string
	ID        string
	Email     *string
	AddedTime *time.Time
}

type RoleBinding struct {
	ID         string
	Name       string
	RoleID     string
	Scopes     []Scope
	CreateTime time.Time
}

type Scope struct {
	OrganizationID *string
	ClusterID      *string
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
	FieldEnterpriseGroupMembers        = "members"
	FieldEnterpriseGroupRoleBindings   = "role_bindings"

	// Field names for members
	FieldEnterpriseGroupMemberKind = "kind"
	FieldEnterpriseGroupMemberID   = "id"

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

	tflog.Debug(ctx, "Creating new enterprise groups", map[string]any{"data": data.State()})

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

	groups, err := convertBatchCreateEnterpriseGroupsResponseGroup(*resp.JSON200.Groups...)
	if err != nil {
		return diag.FromErr(fmt.Errorf("converting created groups: %w", err))
	}

	if err = setEnterpriseGroupsData(data, groups); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set created groups data: %w", err))
	}

	data.SetId(enterpriseID)

	return nil
}

func convertBatchCreateEnterpriseGroupsResponseGroup(groups ...organization_management.BatchCreateEnterpriseGroupsResponseGroup) ([]EnterpriseGroupWithRoleBindings, error) {
	out := make([]EnterpriseGroupWithRoleBindings, len(groups))
	for i, g := range groups {
		var members []Member
		if g.Definition != nil && g.Definition.Members != nil && len(*g.Definition.Members) > 0 {
			members = make([]Member, 0, len(*g.Definition.Members))
			for _, member := range *g.Definition.Members {
				m := Member{}
				if member.Kind == nil {
					return nil, fmt.Errorf("member kind is nil for member in group %s", lo.FromPtr(g.Name))
				}

				if *member.Kind == organization_management.DefinitionMemberKindSUBJECTKINDUSER {
					m.Kind = EnterpriseGroupMemberKindUser
				} else if *member.Kind == organization_management.DefinitionMemberKindSUBJECTKINDSERVICEACCOUNT {
					m.Kind = EnterpriseGroupMemberKindServiceAccount
				} else {
					return nil, fmt.Errorf("unsupported member kind %s for member in group %s", *member.Kind, lo.FromPtr(g.Name))
				}
				m.ID = lo.FromPtr(member.Id)
				m.Email = member.Email
				m.AddedTime = member.AddedTime
				members = append(members, m)
			}
		}

		var roleBindings []RoleBinding
		if g.RoleBindings != nil && len(*g.RoleBindings) > 0 {
			roleBindings = make([]RoleBinding, 0, len(*g.RoleBindings))
			for _, rb := range *g.RoleBindings {
				scopes := []Scope{}

				if rb.Definition.Scopes != nil && len(*rb.Definition.Scopes) > 0 {
					for _, scope := range *rb.Definition.Scopes {
						s := Scope{}
						if scope.Organization != nil {
							s.OrganizationID = &scope.Organization.Id
						}

						if scope.Cluster != nil {
							s.ClusterID = &scope.Cluster.Id
						}
						scopes = append(scopes, s)
					}
				}

				r := RoleBinding{
					ID:         rb.Id,
					Name:       rb.Name,
					RoleID:     rb.Definition.RoleId,
					CreateTime: rb.CreateTime,
					Scopes:     scopes,
				}
				roleBindings = append(roleBindings, r)
			}
		}

		out[i] = EnterpriseGroupWithRoleBindings{
			Group: Group{
				ID:             lo.FromPtr(g.Id),
				Name:           lo.FromPtr(g.Name),
				OrganizationID: lo.FromPtr(g.OrganizationId),
				Description:    g.Description,
				CreateTime:     g.CreateTime,
				ManagedBy:      g.ManagedBy,
				Members:        members,
			},
			RoleBindings: roleBindings,
		}
	}

	return out, nil
}

func resourceEnterpriseGroupsRead(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient
	enterpriseID := data.Id()

	tflog.Debug(ctx, "Reading enterprise groups", map[string]any{"data": data.State()})

	if enterpriseID == "" {
		return diag.FromErr(fmt.Errorf("enterprise ID is not set"))
	}

	managedGroupIDs, err := getManagedGroupIDsFromState(data)
	if err != nil {
		return diag.FromErr(fmt.Errorf("extracting managed group IDs from state: %w", err))
	}

	// Call list groups API to get current state
	resp, err := client.EnterpriseAPIListGroupsWithResponse(ctx, enterpriseID, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("listing enterprise groups: %w", err))
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("list enterprise groups failed with status %d: %s", resp.StatusCode(), string(resp.Body)))
	}

	managedGroupIDsMap := lo.Keyify[string](managedGroupIDs)

	managedGroupIDToGroup := make(map[string]organization_management.ListGroupsResponseGroup, len(managedGroupIDs))
	for _, group := range *resp.JSON200.Items {
		if group.Id == nil {
			continue
		}

		if _, ok := managedGroupIDsMap[*group.Id]; ok {
			managedGroupIDToGroup[*group.Id] = group
		}
	}

	managedGroups := make([]organization_management.ListGroupsResponseGroup, 0, len(managedGroupIDToGroup))
	for _, groupID := range managedGroupIDs {
		if group, exists := managedGroupIDToGroup[groupID]; exists {
			managedGroups = append(managedGroups, group)
		}
	}

	groupsWithRoleBindings, err := getGroupsRoleBindings(ctx, client, enterpriseID, managedGroups)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching role bindings for groups: %w", err))
	}

	if err = setEnterpriseGroupsData(data, groupsWithRoleBindings); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set read groups data: %w", err))
	}

	return nil
}

func resourceEnterpriseGroupsUpdate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient
	enterpriseID := data.Id()

	tflog.Debug(ctx, "Updating enterprise groups", map[string]any{"data": data.State()})

	if enterpriseID == "" {
		return diag.FromErr(fmt.Errorf("enterprise ID is not set"))
	}

	changes, err := getGroupChanges(ctx, data)
	if err != nil {
		return diag.FromErr(fmt.Errorf("analyzing group changes: %w", err))
	}

	tflog.Debug(ctx, "Updating enterprise groups", map[string]any{"changes": changes})

	if len(changes.toDelete) > 0 {
		tflog.Debug(ctx, "Deleting groups", map[string]any{"count": len(changes.toDelete)})
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
		tflog.Debug(ctx, "Creating groups", map[string]any{"count": len(changes.toCreate)})
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

		if resp.JSON200 == nil || resp.JSON200.Groups == nil {
			return diag.FromErr(fmt.Errorf("unexpected empty response from batch create"))
		}
	}

	// Handle modifications
	//if len(changes.toUpdate) > 0 {
	//	tflog.Debug(ctx, "Updating groups", map[string]any{"count": len(changes.toUpdate)})
	//	updateRequest := &organization_management.BatchUpdateEnterpriseGroupsRequest{
	//		EnterpriseId: enterpriseID,
	//		Requests:     changes.toUpdate,
	//	}
	//
	//	resp, err := client.EnterpriseAPIBatchUpdateEnterpriseGroupsWithResponse(ctx, enterpriseID, *updateRequest)
	//	if err != nil {
	//		return diag.FromErr(fmt.Errorf("updating modified groups: %w", err))
	//	}
	//
	//	if resp.StatusCode() != http.StatusOK {
	//		return diag.FromErr(fmt.Errorf("batch update modified groups failed with status %d: %s", resp.StatusCode(), string(resp.Body)))
	//	}
	//}

	return resourceEnterpriseGroupsRead(ctx, data, meta)
}

func resourceEnterpriseGroupsDelete(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient
	enterpriseID := data.Id()

	tflog.Debug(ctx, "Deleting enterprise groups", map[string]any{"data": data.State()})

	if enterpriseID == "" {
		return diag.FromErr(fmt.Errorf("enterprise ID is not set"))
	}

	// Build delete request from current state
	deleteRequest, err := buildBatchDeleteRequest(enterpriseID, data)
	if err != nil {
		return diag.FromErr(fmt.Errorf("building delete request: %w", err))
	}

	if deleteRequest != nil && len(deleteRequest.Requests) > 0 {
		// Call batch delete API
		resp, err := client.EnterpriseAPIBatchDeleteEnterpriseGroupsWithResponse(ctx, enterpriseID, *deleteRequest)
		if err != nil {
			return diag.FromErr(fmt.Errorf("calling batch delete enterprise groups: %w", err))
		}

		if resp.StatusCode() != http.StatusOK {
			return diag.FromErr(fmt.Errorf("batch delete enterprise groups failed with status %d: %s", resp.StatusCode(), string(resp.Body)))
		}
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
			return nil, fmt.Errorf("invalid group configuration: expected object, got %T", groupData)
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
				return nil, fmt.Errorf("invalid nested group configuration: expected object, got %T", groupDataNested)
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
func getManagedGroupIDsFromState(data *schema.ResourceData) ([]string, error) {
	var managedIDs []string

	groupsListAny := data.Get(FieldEnterpriseGroupsGroups)
	groupsList, ok := groupsListAny.([]any)
	if !ok {
		return nil, fmt.Errorf("groups data is not in expected format")
	}

	for _, groupData := range groupsList {
		if groupData == nil {
			continue // nil entries are acceptable in Terraform lists
		}

		groupWrapper, ok := groupData.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid group configuration: expected object, got %T", groupData)
		}

		// Navigate to the nested group objects
		groupsData, ok := groupWrapper[FieldEnterpriseGroupsGroup].([]any)
		if !ok || len(groupsData) == 0 {
			continue // Empty group arrays are acceptable
		}

		// Process all groups in the nested array
		for _, groupDataNested := range groupsData {
			if groupDataNested == nil {
				continue // nil entries are acceptable in nested arrays
			}

			group, ok := groupDataNested.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid nested group configuration: expected object, got %T", groupDataNested)
			}

			if groupID, ok := group[FieldEnterpriseGroupID].(string); ok && groupID != "" {
				managedIDs = append(managedIDs, groupID)
			}
		}
	}

	return managedIDs, nil
}

func getGroupsRoleBindings(
	ctx context.Context,
	client organization_management.ClientWithResponsesInterface,
	enterpriseID string,
	groups []organization_management.ListGroupsResponseGroup,
) ([]EnterpriseGroupWithRoleBindings, error) {
	if len(groups) == 0 {
		return nil, nil
	}

	groupsWithRoleBindings := make([]EnterpriseGroupWithRoleBindings, 0, len(groups))
	for _, group := range groups {
		resp, err := client.EnterpriseAPIListRoleBindingsWithResponse(
			ctx,
			enterpriseID,
			&organization_management.EnterpriseAPIListRoleBindingsParams{
				SubjectId: &[]string{*group.Id},
			})
		if err != nil {
			return nil, fmt.Errorf("listing role bindings for group %s: %w", *group.Id, err)
		}

		if err = sdk.CheckOKResponse(resp, err); err != nil {
			return nil, fmt.Errorf("list role bindings for group %s failed: %w", *group.Id, err)
		}

		var members []Member
		if group.Definition != nil && group.Definition.Members != nil && len(*group.Definition.Members) > 0 {
			members = make([]Member, 0, len(*group.Definition.Members))
			for _, member := range *group.Definition.Members {
				m := Member{}
				if member.Kind == nil {
					return nil, fmt.Errorf("member kind is nil for member in group %s", lo.FromPtr(group.Name))
				}

				if *member.Kind == "KIND_USER" {
					m.Kind = EnterpriseGroupMemberKindUser
				} else if *member.Kind == "KIND_SERVICE_ACCOUNT" {
					m.Kind = EnterpriseGroupMemberKindServiceAccount
				} else {
					return nil, fmt.Errorf("unsupported member kind %s for member in group %s", *member.Kind, lo.FromPtr(group.Name))
				}
				m.ID = lo.FromPtr(member.Id)
				m.Email = member.Email
				m.AddedTime = member.AddedTime
				members = append(members, m)
			}
		}

		g := Group{
			ID:             lo.FromPtr(group.Id),
			Name:           lo.FromPtr(group.Name),
			OrganizationID: lo.FromPtr(group.OrganizationId),
			Description:    group.Description,
			CreateTime:     group.CreateTime,
			ManagedBy:      group.ManagedBy,
			Members:        members,
		}

		var roleBindings []RoleBinding
		if resp.JSON200 != nil && resp.JSON200.Items != nil {
			roleBindings = make([]RoleBinding, 0, len(*resp.JSON200.Items))

			for _, rb := range *resp.JSON200.Items {
				scopes := []Scope{}
				var roleID string

				if rb.Definition != nil {
					if rb.Definition.Scopes != nil && len(*rb.Definition.Scopes) > 0 {
						for _, scope := range *rb.Definition.Scopes {
							s := Scope{}
							if scope.Organization != nil {
								s.OrganizationID = &scope.Organization.Id
							}

							if scope.Cluster != nil {
								s.ClusterID = &scope.Cluster.Id
							}
							scopes = append(scopes, s)
						}
					}

					roleID = lo.FromPtr(rb.Definition.RoleId)
				}

				r := RoleBinding{
					ID:         lo.FromPtr(rb.Id),
					Name:       lo.FromPtr(rb.Name),
					RoleID:     roleID,
					CreateTime: lo.FromPtr(rb.CreateTime),
					Scopes:     scopes,
				}
				roleBindings = append(roleBindings, r)
			}
		}

		groupsWithRoleBindings = append(groupsWithRoleBindings, EnterpriseGroupWithRoleBindings{
			Group:        g,
			RoleBindings: roleBindings,
		})
	}

	return groupsWithRoleBindings, nil
}

func convertMembers(members []Member) []map[string]any {
	if members == nil {
		return nil
	}

	var allMemberData []map[string]any
	for _, member := range members {
		memberData := map[string]any{}
		memberData[FieldEnterpriseGroupMemberID] = member.ID
		memberData[FieldEnterpriseGroupMemberKind] = member.Kind
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

// convertRoleBindings converts the list of role bindings into the format expected by Terraform schema
// Role bindings added to the group outside of Terraform are ignored, as we cannot track ordering for them.
func convertRoleBindings(currentRoleBindings, newRoleBinding []RoleBinding) []map[string]any {
	if len(newRoleBinding) == 0 {
		return nil
	}

	var allRoleBindingData []map[string]any
	newRoleBindingIDToRoleBinding := lo.SliceToMap(newRoleBinding, func(item RoleBinding) (string, RoleBinding) {
		return item.ID, item
	})
	// When role binding is not yet created, it won't have ID, so also index by Name
	newRoleBindingNameToRoleBinding := lo.SliceToMap(newRoleBinding, func(item RoleBinding) (string, RoleBinding) {
		return item.Name, item
	})

	for _, rb := range currentRoleBindings {
		var newRb RoleBinding
		var ok bool
		if rb.ID == "" {
			newRb, ok = newRoleBindingNameToRoleBinding[rb.Name]
			if !ok {
				// Something is wrong, role binding without ID should be found by Name
				continue
			}
		} else {
			newRb, ok = newRoleBindingIDToRoleBinding[rb.ID]
			if !ok {
				// Role binding was removed outside of Terraform, skip it
				continue
			}
		}

		rbData := map[string]any{
			FieldEnterpriseGroupRoleBindingID:     newRb.ID,
			FieldEnterpriseGroupRoleBindingName:   newRb.Name,
			FieldEnterpriseGroupRoleBindingRoleID: newRb.RoleID,
		}

		var allScopeData []map[string]any
		if newRb.Scopes != nil {
			for _, scope := range newRb.Scopes {
				scopeData := map[string]any{}
				if scope.OrganizationID != nil {
					scopeData[FieldEnterpriseGroupScopeOrganization] = *scope.OrganizationID
				}
				if scope.ClusterID != nil {
					scopeData[FieldEnterpriseGroupScopeCluster] = *scope.ClusterID
				}
				allScopeData = append(allScopeData, scopeData)
			}
		}

		var scopes []map[string]any
		if len(allScopeData) > 0 {
			scopeWrapper := map[string]any{
				FieldEnterpriseGroupScope: allScopeData,
			}
			scopes = []map[string]any{scopeWrapper}
		}
		rbData[FieldEnterpriseGroupRoleBindingScopes] = scopes
		allRoleBindingData = append(allRoleBindingData, rbData)
	}

	if len(allRoleBindingData) > 0 {
		roleBindingWrapper := map[string]any{
			FieldEnterpriseGroupRoleBinding: allRoleBindingData,
		}
		return []map[string]any{roleBindingWrapper}
	}

	return nil
}

// setEnterpriseGroupsData sets the Terraform state from SDK response data
func setEnterpriseGroupsData(
	data *schema.ResourceData,
	groups []EnterpriseGroupWithRoleBindings,
) error {
	type groupKey struct {
		orgID string
		name  string
	}

	groupKeyToGroup := lo.SliceToMap(groups,
		func(item EnterpriseGroupWithRoleBindings) (groupKey, EnterpriseGroupWithRoleBindings) {
			return groupKey{orgID: item.Group.OrganizationID, name: item.Group.Name}, item
		},
	)

	groupIDToGroup := lo.SliceToMap(groups,
		func(item EnterpriseGroupWithRoleBindings) (string, EnterpriseGroupWithRoleBindings) {
			return item.Group.ID, item
		},
	)

	configGroups := data.Get(FieldEnterpriseGroupsGroups).([]any)

	wrappedGroups := make([]map[string]any, 0, len(configGroups))
	for _, configGroup := range configGroups {
		configGroupWrapper := configGroup.(map[string]any)
		configGroupsData := configGroupWrapper[FieldEnterpriseGroupsGroup].([]any)

		updatedGroups := make([]map[string]any, 0, len(configGroupsData))
		for _, configGroupNested := range configGroupsData {
			configGroupMap := configGroupNested.(map[string]any)
			configOrganizationID := configGroupMap[FieldEnterpriseGroupOrganizationID].(string)
			configName := configGroupMap[FieldEnterpriseGroupName].(string)
			configGroupID := configGroupMap[FieldEnterpriseGroupID].(string)

			var matchedGroup EnterpriseGroupWithRoleBindings
			var ok bool
			if configGroupID == "" {
				// New group, match by org ID and name
				matchedGroup, ok = groupKeyToGroup[groupKey{orgID: configOrganizationID, name: configName}]
				if !ok {
					// Group not yet created, skip it
					continue
				}
			} else {
				// Existing group, match by ID
				matchedGroup, ok = groupIDToGroup[configGroupID]
				if !ok {
					// Group was removed outside of Terraform, skip it
					continue
				}
			}

			updatedGroup := map[string]any{
				FieldEnterpriseGroupName:           matchedGroup.Group.Name,
				FieldEnterpriseGroupOrganizationID: matchedGroup.Group.OrganizationID,
			}

			updatedGroup[FieldEnterpriseGroupID] = matchedGroup.Group.ID
			updatedGroup[FieldEnterpriseGroupDescription] = matchedGroup.Group.Description
			updatedGroup[FieldEnterpriseGroupMembers] = convertMembers(matchedGroup.Group.Members)

			currentRBs := []RoleBinding{}
			for _, rbs := range configGroupMap[FieldEnterpriseGroupRoleBindings].([]any) {
				if rbs == nil {
					continue
				}
				rbWrapper, ok := rbs.(map[string]any)
				if !ok {
					return fmt.Errorf("invalid role bindings configuration: expected object, got %T", rbs)
				}

				if rbList, ok := rbWrapper[FieldEnterpriseGroupRoleBinding].([]any); ok && len(rbList) > 0 {
					for _, rbItem := range rbList {
						if rbItem == nil {
							continue
						}
						rbItemMap, ok := rbItem.(map[string]any)
						if !ok {
							return fmt.Errorf("invalid role bindings configuration: expected object, got %T", rbItemMap)
						}

						rbID := rbItemMap[FieldEnterpriseGroupRoleBindingID].(string)
						rbName := rbItemMap[FieldEnterpriseGroupRoleBindingName].(string)

						// Writing only ID and Name as they will be used to force correct ordering
						currentRBs = append(currentRBs, RoleBinding{
							ID:   rbID,
							Name: rbName,
						})
					}
				}
			}
			updatedGroup[FieldEnterpriseGroupRoleBindings] = convertRoleBindings(currentRBs, matchedGroup.RoleBindings)
			updatedGroups = append(updatedGroups, updatedGroup)
		}

		wrappedGroup := map[string]any{
			FieldEnterpriseGroupsGroup: updatedGroups,
		}
		wrappedGroups = append(wrappedGroups, wrappedGroup)
	}

	if err := data.Set(FieldEnterpriseGroupsGroups, wrappedGroups); err != nil {
		return fmt.Errorf("failed to set groups data: %w", err)
	}

	return nil
}

// EnterpriseGroupsChanges represents the changes needed during an update operation
type EnterpriseGroupsChanges struct {
	toCreate []organization_management.BatchCreateEnterpriseGroupsRequestGroup
	toUpdate []organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest
	toDelete []organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest
}

func getGroupChanges(ctx context.Context, data *schema.ResourceData) (*EnterpriseGroupsChanges, error) {
	changes := &EnterpriseGroupsChanges{
		toCreate: []organization_management.BatchCreateEnterpriseGroupsRequestGroup{},
		toUpdate: []organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest{},
		toDelete: []organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest{},
	}

	if !data.HasChange(FieldEnterpriseGroupsGroups) {
		tflog.Debug(ctx, "No changes detected in enterprise groups.")
		return changes, nil
	}

	oldValue, newValue := data.GetChange(FieldEnterpriseGroupsGroups)

	tflog.Info(ctx, "Old groups data", map[string]any{"old": oldValue})
	tflog.Info(ctx, "New groups data", map[string]any{"new": newValue})

	oldGroupsList, ok := oldValue.([]any)
	if !ok {
		oldGroupsList = []any{}
	}

	newGroupsList, ok := newValue.([]any)
	if !ok {
		newGroupsList = []any{}
	}

	type groupKey struct {
		orgID string
		name  string
	}

	// Normalizes a group map for comparison by removing computed fields like 'id'.
	normalizeGroupForComparison := func(group map[string]any) map[string]any {
		// Deep copy to avoid modifying the state maps.
		normalized := make(map[string]any)
		for k, v := range group {
			normalized[k] = v
		}
		delete(normalized, FieldEnterpriseGroupID)

		if rbsList, ok := normalized[FieldEnterpriseGroupRoleBindings].([]any); ok {
			newRbsList := make([]any, 0, len(rbsList))
			for _, rbsItem := range rbsList {
				rbsWrapper, ok := rbsItem.(map[string]any)
				if !ok {
					continue
				}
				rbList, ok := rbsWrapper[FieldEnterpriseGroupRoleBinding].([]any)
				if !ok {
					continue
				}

				newRbWrapper := make(map[string]any)
				newRbListContents := make([]any, 0, len(rbList))
				for _, rbItem := range rbList {
					rbMap, ok := rbItem.(map[string]any)
					if !ok {
						continue
					}
					newRbMap := make(map[string]any)
					for k, v := range rbMap {
						newRbMap[k] = v
					}
					delete(newRbMap, FieldEnterpriseGroupRoleBindingID)
					newRbListContents = append(newRbListContents, newRbMap)
				}
				newRbWrapper[FieldEnterpriseGroupRoleBinding] = newRbListContents
				newRbsList = append(newRbsList, newRbWrapper)
			}
			normalized[FieldEnterpriseGroupRoleBindings] = newRbsList
		}

		return normalized
	}

	extractGroupsToCompositeKeyMap := func(list []any) (map[groupKey]map[string]any, error) {
		groupsMap := make(map[groupKey]map[string]any)
		for _, groupData := range list {
			if groupData == nil {
				continue
			}
			groupWrapper, ok := groupData.(map[string]any)
			if !ok {
				continue
			}
			groupsData, ok := groupWrapper[FieldEnterpriseGroupsGroup].([]any)
			if !ok || len(groupsData) == 0 {
				continue
			}
			for _, groupDataNested := range groupsData {
				if groupDataNested == nil {
					continue
				}
				group, ok := groupDataNested.(map[string]any)
				if !ok {
					continue
				}
				name, ok := group[FieldEnterpriseGroupName].(string)
				if !ok || name == "" {
					return nil, fmt.Errorf("group found in state without a name")
				}
				orgID, ok := group[FieldEnterpriseGroupOrganizationID].(string)
				if !ok || orgID == "" {
					return nil, fmt.Errorf("group '%s' found in state without an organization_id", name)
				}
				key := groupKey{orgID: orgID, name: name}
				groupsMap[key] = group
			}
		}
		return groupsMap, nil
	}

	oldGroupsMap, err := extractGroupsToCompositeKeyMap(oldGroupsList)
	if err != nil {
		return nil, err
	}
	newGroupsMap, err := extractGroupsToCompositeKeyMap(newGroupsList)
	if err != nil {
		return nil, err
	}

	for key, newGroup := range newGroupsMap {
		if oldGroup, exists := oldGroupsMap[key]; exists {
			normalizedOld := normalizeGroupForComparison(oldGroup)
			normalizedNew := normalizeGroupForComparison(newGroup)

			if !reflect.DeepEqual(normalizedOld, normalizedNew) {
				groupID, ok := oldGroup[FieldEnterpriseGroupID].(string)
				if !ok || groupID == "" {
					return nil, fmt.Errorf("group '%s' is present in old state but missing an ID", key.name)
				}
				updateRequest, err := buildUpdateRequestForGroup(groupID, newGroup)
				if err != nil {
					return nil, fmt.Errorf("building update request for group %s: %w", groupID, err)
				}
				changes.toUpdate = append(changes.toUpdate, *updateRequest)
			}
			delete(oldGroupsMap, key)
		} else {
			createRequest, err := buildCreateRequestForGroup(newGroup)
			if err != nil {
				return nil, fmt.Errorf("building create request for new group '%s': %w", key.name, err)
			}
			changes.toCreate = append(changes.toCreate, *createRequest)
		}
	}

	for _, groupToDelete := range oldGroupsMap {
		groupID, ok := groupToDelete[FieldEnterpriseGroupID].(string)
		if !ok || groupID == "" {
			return nil, fmt.Errorf("group to be deleted is missing an ID from state")
		}
		orgID, ok := groupToDelete[FieldEnterpriseGroupOrganizationID].(string)
		if !ok || orgID == "" {
			return nil, fmt.Errorf("group to be deleted (%s) is missing an organization_id from state", groupID)
		}
		changes.toDelete = append(changes.toDelete, organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest{
			Id:             groupID,
			OrganizationId: orgID,
		})
	}

	tflog.Info(ctx, "TO CREATE:", map[string]any{"to_create": changes.toCreate})
	tflog.Info(ctx, "TO UPDATE:", map[string]any{"to_update": changes.toUpdate})
	tflog.Info(ctx, "TO DELETE:", map[string]any{"to_delete": changes.toDelete})

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
