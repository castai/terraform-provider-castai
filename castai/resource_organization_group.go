package castai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldOrganizationGroupOrganizationID = "organization_id"
	FieldOrganizationGroupName           = "name"
	FieldOrganizationGroupDescription    = "description"
	FieldOrganizationGroupMembers        = "members"
	FieldOrganizationGroupMember         = "member"
	FieldOrganizationGroupMemberKind     = "kind"
	FieldOrganizationGroupMemberID       = "id"
	FieldOrganizationGroupMemberEmail    = "email"

	GroupMemberKindUser           = "user"
	GroupMemberKindServiceAccount = "service_account"
)

var (
	supportedMemberKinds = []string{GroupMemberKindUser, GroupMemberKindServiceAccount}
)

func resourceOrganizationGroup() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceOrganizationGroupRead,
		CreateContext: resourceOrganizationGroupCreate,
		UpdateContext: resourceOrganizationGroupUpdate,
		DeleteContext: resourceOrganizationGroupDelete,
		Description:   "CAST AI organization group resource to manage organization groups",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(2 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldOrganizationGroupOrganizationID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "CAST AI organization ID.",
			},
			FieldOrganizationGroupName: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the group.",
			},
			FieldOrganizationGroupDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the group.",
			},
			FieldOrganizationGroupMembers: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldOrganizationGroupMember: {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldOrganizationGroupMemberKind: {
										Type:             schema.TypeString,
										Required:         true,
										Description:      fmt.Sprintf("Kind of the member. Supported values include: %s.", strings.Join(supportedMemberKinds, ", ")),
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(supportedMemberKinds, true)),
										DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
											return strings.EqualFold(oldValue, newValue)
										},
									},
									FieldOrganizationGroupMemberID: {
										Type:     schema.TypeString,
										Required: true,
									},
									FieldOrganizationGroupMemberEmail: {
										Type:     schema.TypeString,
										Required: true,
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

func resourceOrganizationGroupCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	organizationID := data.Get(FieldOrganizationGroupOrganizationID).(string)
	if organizationID == "" {
		var err error
		organizationID, err = getDefaultOrganizationId(ctx, meta)
		if err != nil {
			return diag.Errorf("getting default organization: %v", err)
		}
	}

	client := meta.(*ProviderConfig).api

	members := convertMembersToSDK(data)

	resp, err := client.RbacServiceAPICreateGroupWithResponse(ctx, organizationID, sdk.RbacServiceAPICreateGroupJSONRequestBody{
		Definition: sdk.CastaiRbacV1beta1CreateGroupRequestGroupDefinition{
			Members: &members,
		},
		Description: lo.ToPtr(data.Get(FieldOrganizationGroupDescription).(string)),
		Name:        data.Get(FieldOrganizationName).(string),
	})

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("create group: %w", err))
	}

	data.SetId(*resp.JSON200.Id)

	return resourceOrganizationGroupRead(ctx, data, meta)
}

func resourceOrganizationGroupRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	groupID := data.Id()
	if groupID == "" {
		return diag.Errorf("group ID is not set")
	}

	organizationID := data.Get(FieldOrganizationGroupOrganizationID).(string)
	if organizationID == "" {
		var err error
		organizationID, err = getDefaultOrganizationId(ctx, meta)
		if err != nil {
			return diag.FromErr(fmt.Errorf("getting default organization: %w", err))
		}
	}

	client := meta.(*ProviderConfig).api

	group, err := getGroup(client, ctx, organizationID, groupID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("getting group for read: %w", err))
	}

	if err := assignGroupData(group, data); err != nil {
		return diag.FromErr(fmt.Errorf("assigning group data for read: %w", err))
	}

	return nil
}

func resourceOrganizationGroupUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	groupID := data.Id()
	if groupID == "" {
		return diag.Errorf("group ID is not set")
	}

	organizationID := data.Get(FieldOrganizationGroupOrganizationID).(string)
	if organizationID == "" {
		var err error
		organizationID, err = getDefaultOrganizationId(ctx, meta)
		if err != nil {
			return diag.FromErr(fmt.Errorf("getting default organization: %w", err))
		}
	}

	client := meta.(*ProviderConfig).api

	members := convertMembersToSDK(data)

	resp, err := client.RbacServiceAPIUpdateGroupWithResponse(ctx, organizationID, groupID, sdk.RbacServiceAPIUpdateGroupJSONRequestBody{
		Definition: sdk.CastaiRbacV1beta1UpdateGroupRequestGroupDefinition{
			Members: members,
		},
		Description: lo.ToPtr(data.Get(FieldOrganizationGroupDescription).(string)),
		Name:        data.Get(FieldOrganizationName).(string),
	})
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("update group: %w", err))
	}

	return resourceOrganizationGroupRead(ctx, data, meta)
}

func resourceOrganizationGroupDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	groupID := data.Id()
	if groupID == "" {
		return diag.Errorf("group ID is not set")
	}

	organizationID := data.Get(FieldOrganizationGroupOrganizationID).(string)
	if organizationID == "" {
		var err error
		organizationID, err = getDefaultOrganizationId(ctx, meta)
		if err != nil {
			return diag.FromErr(fmt.Errorf("getting default organization: %w", err))
		}
	}

	client := meta.(*ProviderConfig).api

	resp, err := client.RbacServiceAPIDeleteGroupWithResponse(ctx, organizationID, groupID)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("destroy group: %w", err))
	}

	return nil
}

func getGroup(client *sdk.ClientWithResponses, ctx context.Context, organizationID, groupID string) (*sdk.CastaiRbacV1beta1Group, error) {
	groupsResp, err := client.RbacServiceAPIGetGroupWithResponse(ctx, organizationID, groupID)
	if err != nil {
		return nil, fmt.Errorf("fetching group: %w", err)
	}

	if groupsResp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("group %s not found", groupID)
	}
	if err := sdk.CheckOKResponse(groupsResp, err); err != nil {
		return nil, fmt.Errorf("retrieving group: %w", err)
	}
	if groupsResp.JSON200 == nil {
		return nil, errors.New("group not found")
	}
	return groupsResp.JSON200, nil
}

func assignGroupData(group *sdk.CastaiRbacV1beta1Group, data *schema.ResourceData) error {
	if err := data.Set(FieldOrganizationGroupOrganizationID, group.OrganizationId); err != nil {
		return fmt.Errorf("setting organization_id: %w", err)
	}
	if err := data.Set(FieldOrganizationGroupDescription, group.Description); err != nil {
		return fmt.Errorf("setting description: %w", err)
	}
	if err := data.Set(FieldOrganizationGroupName, group.Name); err != nil {
		return fmt.Errorf("setting group name: %w", err)
	}

	if group.Definition.Members != nil {
		var members []map[string]string
		for _, member := range *group.Definition.Members {
			var kind string
			switch member.Kind {
			case sdk.USER:
				kind = GroupMemberKindUser
			case sdk.SERVICEACCOUNT:
				kind = GroupMemberKindServiceAccount
			}
			members = append(members, map[string]string{
				FieldOrganizationGroupMemberKind:  kind,
				FieldOrganizationGroupMemberID:    member.Id,
				FieldOrganizationGroupMemberEmail: member.Email,
			})
		}
		err := data.Set(FieldOrganizationGroupMembers, []any{
			map[string]any{
				FieldOrganizationGroupMember: members,
			},
		})
		if err != nil {
			return fmt.Errorf("parsing group members: %w", err)
		}
	}

	return nil
}

func convertMembersToSDK(data *schema.ResourceData) []sdk.CastaiRbacV1beta1Member {
	var members []sdk.CastaiRbacV1beta1Member

	for _, dataMembersDef := range data.Get(FieldOrganizationGroupMembers).([]any) {
		for _, dataMember := range dataMembersDef.(map[string]any)[FieldOrganizationGroupMember].([]any) {
			var kind sdk.CastaiRbacV1beta1MemberKind
			switch dataMember.(map[string]any)[FieldOrganizationGroupMemberKind].(string) {
			case GroupMemberKindUser:
				kind = sdk.USER
			case GroupMemberKindServiceAccount:
				kind = sdk.SERVICEACCOUNT
			}
			members = append(members, sdk.CastaiRbacV1beta1Member{
				Kind:  kind,
				Email: dataMember.(map[string]any)[FieldOrganizationGroupMemberEmail].(string),
				Id:    dataMember.(map[string]any)[FieldOrganizationGroupMemberID].(string),
			})
		}
	}

	return members
}
