package castai

import (
	"context"
	"errors"
	"fmt"
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
	FieldEnterpriseGroupEnterpriseID = "enterprise_id"

	// Field names for nested objects
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

func resourceEnterpriseGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEnterpriseGroupCreate,
		ReadContext:   resourceEnterpriseGroupRead,
		UpdateContext: resourceEnterpriseGroupUpdate,
		DeleteContext: resourceEnterpriseGroupDelete,
		Description:   "CAST AI Enterprise Group resource.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldEnterpriseGroupEnterpriseID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Enterprise organization ID.",
			},
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
				Default:     "",
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
	}
}

func resourceEnterpriseGroupCreate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient
	enterpriseID := data.Get(FieldEnterpriseGroupEnterpriseID).(string)

	tflog.Debug(ctx, "Creating enterprise group", map[string]any{})

	createRequest, err := buildBatchCreateRequest(enterpriseID, data)
	if err != nil {
		return diag.FromErr(fmt.Errorf("building create request: %w", err))
	}

	resp, err := client.EnterpriseAPIBatchCreateEnterpriseGroupsWithResponse(ctx, enterpriseID, *createRequest)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("batch create enterprise groups failed: %w", err))
	}

	if resp.JSON200 == nil || resp.JSON200.Groups == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from batch create"))
	}

	if len(*resp.JSON200.Groups) != 1 {
		return diag.FromErr(fmt.Errorf("unexpected number of groups created: expected 1, got %d", len(*resp.JSON200.Groups)))
	}

	groupRsp := (*resp.JSON200.Groups)[0]

	group, err := convertBatchCreateEnterpriseGroupsResponseGroup(groupRsp)
	if err != nil {
		return diag.FromErr(fmt.Errorf("converting created group data: %w", err))
	}

	if err = setEnterpriseGroupsData(ctx, data, group); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set created group data: %w", err))
	}

	data.SetId(group.Group.ID)

	tflog.Debug(ctx, "Created enterprise group", map[string]any{"group_id": group.Group.ID})

	return nil
}

func buildBatchCreateRequest(
	enterpriseID string,
	data *schema.ResourceData,
) (*organization_management.BatchCreateEnterpriseGroupsRequest, error) {
	groupWithRBs, err := readEnterpriseGroupData(data)
	if err != nil {
		return nil, fmt.Errorf("reading group data: %w", err)
	}

	members := lo.Map(groupWithRBs.Group.Members, func(m Member, _ int) organization_management.BatchCreateEnterpriseGroupsRequestMember {
		var kind organization_management.BatchCreateEnterpriseGroupsRequestMemberKind

		switch m.Kind {
		case EnterpriseGroupMemberKindUser:
			kind = organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDUSER
		case EnterpriseGroupMemberKindServiceAccount:
			kind = organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDSERVICEACCOUNT
		default:
			kind = organization_management.BatchCreateEnterpriseGroupsRequestMemberKindSUBJECTKINDUNSPECIFIED
		}

		return organization_management.BatchCreateEnterpriseGroupsRequestMember{
			Id:   &m.ID,
			Kind: &kind,
		}
	})

	roleBindings := lo.Map(
		groupWithRBs.RoleBindings,
		func(rb RoleBinding, _ int) organization_management.BatchCreateEnterpriseGroupsRequestRoleBinding {
			scopes := lo.Map(rb.Scopes, func(s Scope, _ int) organization_management.Scope {
				scope := organization_management.Scope{}
				if s.OrganizationID != nil {
					scope.Organization = &organization_management.OrganizationScope{
						Id: *s.OrganizationID,
					}
				}
				if s.ClusterID != nil {
					scope.Cluster = &organization_management.ClusterScope{
						Id: *s.ClusterID,
					}
				}
				return scope
			})

			return organization_management.BatchCreateEnterpriseGroupsRequestRoleBinding{
				Name:   rb.Name,
				RoleId: rb.RoleID,
				Scopes: scopes,
			}
		},
	)

	groupRequest := organization_management.BatchCreateEnterpriseGroupsRequestGroup{
		Name:           groupWithRBs.Group.Name,
		OrganizationId: groupWithRBs.Group.OrganizationID,
		Members:        members,
		RoleBindings:   &roleBindings,
	}

	if desc := data.Get(FieldEnterpriseGroupDescription).(string); desc != "" {
		groupRequest.Description = &desc
	}

	return &organization_management.BatchCreateEnterpriseGroupsRequest{
		EnterpriseId: enterpriseID,
		Requests:     []organization_management.BatchCreateEnterpriseGroupsRequestGroup{groupRequest},
	}, nil
}

func convertBatchCreateEnterpriseGroupsResponseGroup(
	g organization_management.BatchCreateEnterpriseGroupsResponseGroup,
) (EnterpriseGroupWithRoleBindings, error) {
	var members []Member
	if g.Definition != nil && g.Definition.Members != nil && len(*g.Definition.Members) > 0 {
		members = make([]Member, 0, len(*g.Definition.Members))
		for _, member := range *g.Definition.Members {
			m := Member{}
			if member.Kind == nil {
				return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("member kind is nil for member in group %s", lo.FromPtr(g.Name))
			}

			switch *member.Kind {
			case organization_management.DefinitionMemberKindSUBJECTKINDUSER:
				m.Kind = EnterpriseGroupMemberKindUser
			case organization_management.DefinitionMemberKindSUBJECTKINDSERVICEACCOUNT:
				m.Kind = EnterpriseGroupMemberKindServiceAccount
			default:
				return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("unsupported member kind %s for member in group %s", *member.Kind, lo.FromPtr(g.Name))
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

	return EnterpriseGroupWithRoleBindings{
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
	}, nil
}

func resourceEnterpriseGroupRead(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient

	tflog.Debug(ctx, "Reading enterprise group", map[string]any{"group_id": data.Id()})

	enterpriseID, ok := data.GetOk(FieldEnterpriseGroupEnterpriseID)
	if !ok {
		return diag.FromErr(fmt.Errorf("enterprise ID is not set"))
	}

	enterpriseIDStr := enterpriseID.(string)

	groupID, ok := data.GetOk(FieldEnterpriseGroupID)
	if !ok {
		return diag.FromErr(fmt.Errorf("group ID is not set"))
	}

	groupIDStr := groupID.(string)

	resp, err := client.EnterpriseAPIListGroupsWithResponse(ctx, enterpriseIDStr, nil)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("list enterprise groups failed: %w", err))
	}

	if resp.JSON200 == nil || resp.JSON200.Items == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from list enterprise groups"))
	}

	var group *organization_management.ListGroupsResponseGroup
	for _, g := range *resp.JSON200.Items {
		if g.Id != nil && *g.Id == groupIDStr {
			group = &g
		}
	}

	if group == nil {
		// Group not found, remove from state
		tflog.Warn(
			ctx,
			"Group not found, removing from state",
			map[string]any{
				"group_id":      groupIDStr,
				"enterprise_id": enterpriseIDStr,
			},
		)
		data.SetId("")
	} else {
		groupWithRoleBindings, err := convertListGroupsResponseGroup(ctx, client, enterpriseIDStr, *group)
		if err != nil {
			return diag.FromErr(fmt.Errorf("fetching role bindings for groups: %w", err))
		}

		if err = setEnterpriseGroupsData(ctx, data, groupWithRoleBindings); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set read groups data: %w", err))
		}
	}

	tflog.Debug(ctx, "Finished reading enterprise group", map[string]any{"group_id": data.Id()})

	return nil
}

func convertListGroupsResponseGroup(
	ctx context.Context,
	client organization_management.ClientWithResponsesInterface,
	enterpriseID string,
	group organization_management.ListGroupsResponseGroup,
) (EnterpriseGroupWithRoleBindings, error) {
	resp, err := client.EnterpriseAPIListRoleBindingsWithResponse(
		ctx,
		enterpriseID,
		&organization_management.EnterpriseAPIListRoleBindingsParams{
			SubjectId: &[]string{*group.Id},
		},
	)
	if err = sdk.CheckOKResponse(resp, err); err != nil {
		return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("list role bindings for group %s failed: %w", *group.Id, err)
	}

	var members []Member
	if group.Definition != nil && group.Definition.Members != nil && len(*group.Definition.Members) > 0 {
		members = make([]Member, 0, len(*group.Definition.Members))
		for _, member := range *group.Definition.Members {
			m := Member{}
			if member.Kind == nil {
				return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("member kind is nil for member in group %s", lo.FromPtr(group.Name))
			}

			switch *member.Kind {
			case "KIND_USER":
				m.Kind = EnterpriseGroupMemberKindUser
			case "KIND_SERVICE_ACCOUNT":
				m.Kind = EnterpriseGroupMemberKindServiceAccount
			default:
				return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("unsupported member kind %s for member in group %s", *member.Kind, lo.FromPtr(group.Name))
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

	return EnterpriseGroupWithRoleBindings{
		Group:        g,
		RoleBindings: roleBindings,
	}, nil
}

func setEnterpriseGroupsData(
	ctx context.Context,
	data *schema.ResourceData,
	group EnterpriseGroupWithRoleBindings,
) error {
	if err := data.Set(FieldEnterpriseGroupID, group.Group.ID); err != nil {
		return fmt.Errorf("failed to set group ID: %w", err)
	}
	if err := data.Set(FieldEnterpriseGroupName, group.Group.Name); err != nil {
		return fmt.Errorf("failed to set group name: %w", err)
	}
	if err := data.Set(FieldEnterpriseGroupOrganizationID, group.Group.OrganizationID); err != nil {
		return fmt.Errorf("failed to set organization ID: %w", err)
	}
	if err := data.Set(FieldEnterpriseGroupDescription, group.Group.Description); err != nil {
		return fmt.Errorf("failed to set description: %w", err)
	}
	if err := data.Set(FieldEnterpriseGroupMembers, convertMembers(group.Group.Members)); err != nil {
		return fmt.Errorf("failed to set members: %w", err)
	}

	roleBindings, err := convertRoleBindings(ctx, data, group.RoleBindings)
	if err != nil {
		return fmt.Errorf("failed to convert role bindings: %w", err)
	}

	if err = data.Set(FieldEnterpriseGroupRoleBindings, roleBindings); err != nil {
		return fmt.Errorf("failed to set role bindings: %w", err)
	}

	return nil
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

	if len(allMemberData) > 0 {
		memberWrapper := map[string]any{
			FieldEnterpriseGroupsMember: allMemberData,
		}
		return []map[string]any{memberWrapper}
	}

	return nil
}

func convertRoleBindings(
	ctx context.Context,
	data *schema.ResourceData,
	newRoleBinding []RoleBinding,
) ([]map[string]any, error) {
	if len(newRoleBinding) == 0 {
		return nil, nil
	}

	currentRoleBindings := []RoleBinding{}
	for _, rbs := range data.Get(FieldEnterpriseGroupRoleBindings).([]any) {
		if rbs == nil {
			continue
		}
		rbWrapper, ok := rbs.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid role bindings configuration: expected object, got %T", rbs)
		}

		if rbList, ok := rbWrapper[FieldEnterpriseGroupRoleBinding].([]any); ok {
			for _, rbItem := range rbList {
				if rbItem == nil {
					continue
				}
				rbItemMap, ok := rbItem.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("invalid role bindings configuration: expected object, got %T", rbItemMap)
				}

				rbID := rbItemMap[FieldEnterpriseGroupRoleBindingID].(string)
				rbName := rbItemMap[FieldEnterpriseGroupRoleBindingName].(string)

				// Writing only ID and Name as they will be used to force correct ordering
				currentRoleBindings = append(currentRoleBindings, RoleBinding{
					ID:   rbID,
					Name: rbName,
				})
			}
		}
	}

	var allRoleBindingData []map[string]any
	newRoleBindingIDToRoleBinding := lo.SliceToMap(newRoleBinding, func(item RoleBinding) (string, RoleBinding) {
		return item.ID, item
	})
	newRoleBindingNameToRoleBinding := lo.SliceToMap(newRoleBinding, func(item RoleBinding) (string, RoleBinding) {
		return item.Name, item
	})

	for _, rb := range currentRoleBindings {
		var newRb RoleBinding
		var ok bool
		if rb.ID == "" {
			newRb, ok = newRoleBindingNameToRoleBinding[rb.Name]
			if !ok {
				tflog.Warn(ctx, "Role binding not found by Name, skipping", map[string]any{
					"role_binding_name": rb.Name,
				})
				continue
			}
		} else {
			newRb, ok = newRoleBindingIDToRoleBinding[rb.ID]
			if !ok {
				tflog.Warn(ctx, "Role binding not found by ID, trying to find by Name", map[string]any{
					"role_binding_id":   rb.ID,
					"role_binding_name": rb.Name,
				})
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
					scopeData[FieldEnterpriseGroupScopeCluster] = ""
				}
				if scope.ClusterID != nil {
					scopeData[FieldEnterpriseGroupScopeOrganization] = ""
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
		return []map[string]any{roleBindingWrapper}, nil
	}

	return nil, nil
}

func resourceEnterpriseGroupUpdate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient
	enterpriseID := data.Get(FieldEnterpriseGroupEnterpriseID).(string)
	groupID := data.Id()

	tflog.Debug(ctx, "Updating enterprise group", map[string]any{"group_id": groupID})

	if !data.HasChanges(
		FieldEnterpriseGroupName,
		FieldEnterpriseGroupDescription,
		FieldEnterpriseGroupMembers,
		FieldEnterpriseGroupRoleBindings,
	) {
		tflog.Debug(ctx, "No changes detected, skipping update", map[string]any{"group_id": groupID})
		return nil
	}

	groupWithRBs, err := readEnterpriseGroupData(data)
	if err != nil {
		return diag.FromErr(fmt.Errorf("reading group data: %w", err))
	}

	var description string
	if groupWithRBs.Group.Description != nil {
		description = *groupWithRBs.Group.Description
	}

	members := lo.Map(groupWithRBs.Group.Members, func(m Member, _ int) organization_management.BatchUpdateEnterpriseGroupsRequestMember {
		var kind organization_management.BatchUpdateEnterpriseGroupsRequestMemberKind
		switch m.Kind {
		case EnterpriseGroupMemberKindUser:
			kind = organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindUSER
		case EnterpriseGroupMemberKindServiceAccount:
			kind = organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindSERVICEACCOUNT
		default:
			kind = organization_management.BatchUpdateEnterpriseGroupsRequestMemberKindSUBJECTKINDUNSPECIFIED
		}

		return organization_management.BatchUpdateEnterpriseGroupsRequestMember{
			Kind: kind,
			Id:   m.ID,
		}
	})

	roleBindings := lo.Map(groupWithRBs.RoleBindings, func(rb RoleBinding, _ int) organization_management.BatchUpdateEnterpriseGroupsRequestRoleBinding {
		scopes := lo.Map(rb.Scopes, func(s Scope, _ int) organization_management.Scope {
			scope := organization_management.Scope{}
			if s.OrganizationID != nil {
				scope.Organization = &organization_management.OrganizationScope{
					Id: *s.OrganizationID,
				}
			}
			if s.ClusterID != nil {
				scope.Cluster = &organization_management.ClusterScope{
					Id: *s.ClusterID,
				}
			}
			return scope
		})

		return organization_management.BatchUpdateEnterpriseGroupsRequestRoleBinding{
			Id:     rb.ID,
			Name:   rb.Name,
			RoleId: rb.RoleID,
			Scopes: scopes,
		}
	})

	updateRequest := organization_management.BatchUpdateEnterpriseGroupsRequest{
		EnterpriseId: enterpriseID,
		Requests: []organization_management.BatchUpdateEnterpriseGroupsRequestUpdateGroupRequest{
			{
				Id:             groupID,
				Name:           groupWithRBs.Group.Name,
				OrganizationId: groupWithRBs.Group.OrganizationID,
				Description:    description,
				Members:        members,
				RoleBindings:   roleBindings,
			},
		},
	}

	resp, err := client.EnterpriseAPIBatchUpdateEnterpriseGroupsWithResponse(
		ctx,
		enterpriseID,
		updateRequest,
	)
	if err = sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("batch update enterprise groups failed: %w", err))
	}

	if resp.JSON200 == nil || resp.JSON200.Groups == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from batch update"))
	}

	g := (*resp.JSON200.Groups)[0]

	group, err := convertBatchUpdateEnterpriseGroupsResponseGroup(g)
	if err != nil {
		return diag.FromErr(fmt.Errorf("converting updated group data: %w", err))
	}

	if err = setEnterpriseGroupsData(ctx, data, group); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set updated group data: %w", err))
	}

	tflog.Debug(ctx, "Enterprise group updated successfully", map[string]any{"group_id": groupID})

	return nil
}

func convertBatchUpdateEnterpriseGroupsResponseGroup(
	g organization_management.BatchUpdateEnterpriseGroupsResponseGroup,
) (EnterpriseGroupWithRoleBindings, error) {
	var members []Member
	if g.Definition != nil && g.Definition.Members != nil && len(*g.Definition.Members) > 0 {
		members = make([]Member, 0, len(*g.Definition.Members))
		for _, member := range *g.Definition.Members {
			m := Member{}
			if member.Kind == nil {
				return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("member kind is nil for member in group %s", lo.FromPtr(g.Name))
			}

			switch *member.Kind {
			case "USER":
				m.Kind = EnterpriseGroupMemberKindUser
			case "SERVICE_ACCOUNT":
				m.Kind = EnterpriseGroupMemberKindServiceAccount
			default:
				return EnterpriseGroupWithRoleBindings{},
					fmt.Errorf("unsupported member kind %s for member in group %s", *member.Kind, lo.FromPtr(g.Name))
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

	return EnterpriseGroupWithRoleBindings{
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
	}, nil
}

func resourceEnterpriseGroupDelete(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient

	tflog.Debug(ctx, "Deleting enterprise group", map[string]any{"group_id": data.Id()})

	deleteRequest, err := buildBatchDeleteRequest(data)
	if err != nil {
		return diag.FromErr(fmt.Errorf("building delete request: %w", err))
	}

	resp, err := client.EnterpriseAPIBatchDeleteEnterpriseGroupsWithResponse(
		ctx,
		deleteRequest.EnterpriseId,
		deleteRequest,
	)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("batch delete enterprise groups failed: %w", err))
	}

	// Clear the resource ID
	data.SetId("")

	tflog.Debug(ctx, "Deleted enterprise group", map[string]any{})

	return nil
}

func buildBatchDeleteRequest(data *schema.ResourceData) (organization_management.BatchDeleteEnterpriseGroupsRequest, error) {
	groupID, ok := data.GetOk(FieldEnterpriseGroupID)
	if !ok {
		return organization_management.BatchDeleteEnterpriseGroupsRequest{}, fmt.Errorf("group ID is not set")
	}

	groupIDStr := groupID.(string)

	enterpriseID, ok := data.GetOk(FieldEnterpriseGroupEnterpriseID)
	if !ok {
		return organization_management.BatchDeleteEnterpriseGroupsRequest{}, fmt.Errorf("enterprise ID is not set")
	}

	enterpriseIDStr := enterpriseID.(string)

	organizationID, ok := data.GetOk(FieldEnterpriseGroupOrganizationID)
	if !ok {
		return organization_management.BatchDeleteEnterpriseGroupsRequest{}, fmt.Errorf("organization ID is not set")
	}

	organizationIDStr := organizationID.(string)

	return organization_management.BatchDeleteEnterpriseGroupsRequest{
		EnterpriseId: enterpriseIDStr,
		Requests: []organization_management.BatchDeleteEnterpriseGroupsRequestDeleteGroupRequest{
			{
				Id:             groupIDStr,
				OrganizationId: organizationIDStr,
			},
		},
	}, nil
}

func readEnterpriseGroupData(data *schema.ResourceData) (EnterpriseGroupWithRoleBindings, error) {
	members := []Member{}
	if membersData, ok := data.Get(FieldEnterpriseGroupMembers).([]any); ok {
		for _, memberWrapper := range membersData {
			if memberWrapper == nil {
				continue
			}

			memberWrapperMap, ok := memberWrapper.(map[string]any)
			if !ok {
				return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("invalid member configuration: expected object, got %T", memberWrapper)
			}

			membersDataNested, ok := memberWrapperMap[FieldEnterpriseGroupsMember].([]any)
			if !ok || len(membersDataNested) == 0 {
				continue
			}

			for _, memberData := range membersDataNested {
				if memberData == nil {
					continue
				}

				member, ok := memberData.(map[string]any)
				if !ok {
					return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("invalid member data: expected object, got %T", memberData)
				}

				memberKind, ok := member[FieldEnterpriseGroupMemberKind].(string)
				if !ok {
					return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("member missing required 'kind' field")
				}

				memberID, ok := member[FieldEnterpriseGroupMemberID].(string)
				if !ok {
					return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("member missing required 'id' field")
				}

				members = append(members, Member{
					Kind: memberKind,
					ID:   memberID,
				})
			}
		}
	}

	roleBindings := []RoleBinding{}
	if bindingsData, ok := data.Get(FieldEnterpriseGroupRoleBindings).([]any); ok {
		for _, bindingWrapper := range bindingsData {
			if bindingWrapper == nil {
				continue
			}

			bindingWrapperMap, ok := bindingWrapper.(map[string]any)
			if !ok {
				return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("invalid role binding configuration: expected object, got %T", bindingWrapper)
			}

			bindingsDataNested, ok := bindingWrapperMap[FieldEnterpriseGroupRoleBinding].([]any)
			if !ok || len(bindingsDataNested) == 0 {
				continue
			}

			for _, bindingData := range bindingsDataNested {
				if bindingData == nil {
					continue
				}

				binding, ok := bindingData.(map[string]any)
				if !ok {
					return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("invalid role binding data: expected object, got %T", bindingData)
				}

				rbName, ok := binding[FieldEnterpriseGroupRoleBindingName].(string)
				if !ok {
					return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("role binding missing required 'name' field")
				}

				rbRoleID, ok := binding[FieldEnterpriseGroupRoleBindingRoleID].(string)
				if !ok {
					return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("role binding missing required 'role_id' field")
				}

				rbScopes := []Scope{}
				if scopesData, ok := binding[FieldEnterpriseGroupRoleBindingScopes].([]any); ok {
					for _, scopeWrapper := range scopesData {
						if scopeWrapper == nil {
							continue
						}

						scopeWrapperMap, ok := scopeWrapper.(map[string]any)
						if !ok {
							return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("invalid scope configuration: expected object, got %T", scopeWrapper)
						}

						scopesDataNested, ok := scopeWrapperMap[FieldEnterpriseGroupScope].([]any)
						if !ok || len(scopesDataNested) == 0 {
							continue
						}

						for _, scopeData := range scopesDataNested {
							if scopeData == nil {
								continue
							}

							scope, ok := scopeData.(map[string]any)
							if !ok {
								return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("invalid scope data: expected object, got %T", scopeData)
							}

							orgID, _ := scope[FieldEnterpriseGroupScopeOrganization].(string)
							clusterID, _ := scope[FieldEnterpriseGroupScopeCluster].(string)

							if orgID != "" && clusterID != "" {
								return EnterpriseGroupWithRoleBindings{}, fmt.Errorf("scope cannot have both 'organization' and 'cluster' set simultaneously")
							}

							if orgID != "" {
								rbScopes = append(rbScopes, Scope{
									OrganizationID: &orgID,
								})
							} else if clusterID != "" {
								rbScopes = append(rbScopes, Scope{
									ClusterID: &clusterID,
								})
							}
						}
					}
				}

				rbID := ""
				if id, ok := binding[FieldEnterpriseGroupRoleBindingID].(string); ok && id != "" {
					rbID = id
				}

				roleBindings = append(roleBindings, RoleBinding{
					ID:     rbID,
					Name:   rbName,
					RoleID: rbRoleID,
					Scopes: rbScopes,
				})
			}
		}
	}

	groupName, ok := data.Get(FieldEnterpriseGroupName).(string)
	if !ok {
		return EnterpriseGroupWithRoleBindings{}, errors.New("group name is required")
	}

	orgID, ok := data.Get(FieldEnterpriseGroupOrganizationID).(string)
	if !ok {
		return EnterpriseGroupWithRoleBindings{}, errors.New("organization ID is required")
	}

	description, ok := data.GetOk(FieldEnterpriseGroupDescription)
	if !ok {
		description = ""
	}
	descriptionStr := description.(string)

	return EnterpriseGroupWithRoleBindings{
		Group: Group{
			ID:             data.Id(),
			Name:           groupName,
			OrganizationID: orgID,
			Description:    &descriptionStr,
			Members:        members,
		},
		RoleBindings: roleBindings,
	}, nil
}
