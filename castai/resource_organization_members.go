package castai

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldOrganizationMembersOrganizationID = "organization_id"
	FieldOrganizationMembersOwners         = "owners"
	FieldOrganizationMembersViewers        = "viewers"
	FieldOrganizationMembersMembers        = "members"
)

const OwnerRoleID = "3e1050c7-6593-4298-94bb-154637911d78"
const MemberRoleID = "8c60bd8e-21de-402a-969f-add07fd22c1b"
const ViewerRoleID = "6fc95bd7-6049-4735-80b0-ce5ccde71cb1"

func resourceOrganizationMembers() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceOrganizationMembersRead,
		CreateContext: resourceOrganizationMembersCreate,
		UpdateContext: resourceOrganizationMembersUpdate,
		DeleteContext: resourceOrganizationMembersDelete,
		Description:   "CAST AI organization members resource to manage organization members",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(2 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldOrganizationMembersOrganizationID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "CAST AI organization ID.",
			},
			FieldOrganizationMembersOwners: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of email addresses corresponding to users who should be given owner access to the organization.",
				Deprecated:  "The 'owners' field is deprecated. Use 'castai_role_bindings' resource instead for more granular role management. This field will be removed in a future version.",
			},
			FieldOrganizationMembersViewers: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of email addresses corresponding to users who should be given viewer access to the organization.",
				Deprecated:  "The 'viewers' field is deprecated. Use 'castai_role_bindings' resource instead for more granular role management. This field will be removed in a future version.",
			},
			FieldOrganizationMembersMembers: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of email addresses corresponding to users who should be given member access to the organization.",
				Deprecated:  "The 'members' field is deprecated. Use 'castai_role_bindings' resource instead for more granular role management. This field will be removed in a future version.",
			},
		},
	}
}

func resourceOrganizationMembersCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	organizationID := data.Get(FieldOrganizationMembersOrganizationID).(string)

	tflog.Debug(ctx, "getting current user profile")
	currentUserResp, err := client.UsersAPICurrentUserProfileWithResponse(ctx)
	if err := sdk.CheckOKResponse(currentUserResp, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving current user profile: %w", err))
	}

	var newMemberships []sdk.CastaiUsersV1beta1NewMembershipByEmail

	if owners, ok := data.GetOk(FieldOrganizationMembersOwners); ok {
		emails := toStringList(owners.([]interface{}))

		for _, email := range emails {
			// Person that creates a new organization is automatically the owner
			// of it. That's why when creating this resource we would like to skip
			// re-creating that user because it would fail.
			if lo.FromPtr(currentUserResp.JSON200.Email) == email {
				continue
			}

			newMemberships = append(newMemberships, sdk.CastaiUsersV1beta1NewMembershipByEmail{
				RoleBindings: &[]sdk.CastaiUsersV1beta1InvitationRoleBinding{
					{
						RoleId: lo.ToPtr(OwnerRoleID),
						Scopes: &[]sdk.CastaiUsersV1beta1InvitationRoleBindingScope{
							{
								Id:   lo.ToPtr(organizationID),
								Type: lo.ToPtr(sdk.CastaiRbacV1beta1ScopeTypeORGANIZATION),
							},
						},
					},
				},
				UserEmail: email,
			})
		}
	}

	if viewers, ok := data.GetOk(FieldOrganizationMembersViewers); ok {
		emails := toStringList(viewers.([]interface{}))

		for _, email := range emails {
			newMemberships = append(newMemberships, sdk.CastaiUsersV1beta1NewMembershipByEmail{
				RoleBindings: &[]sdk.CastaiUsersV1beta1InvitationRoleBinding{
					{
						RoleId: lo.ToPtr(ViewerRoleID),
						Scopes: &[]sdk.CastaiUsersV1beta1InvitationRoleBindingScope{
							{
								Id:   lo.ToPtr(organizationID),
								Type: lo.ToPtr(sdk.CastaiRbacV1beta1ScopeTypeORGANIZATION),
							},
						},
					},
				},
				UserEmail: email,
			})
		}
	}

	if members, ok := data.GetOk(FieldOrganizationMembersMembers); ok {
		emails := toStringList(members.([]interface{}))

		for _, email := range emails {
			newMemberships = append(newMemberships, sdk.CastaiUsersV1beta1NewMembershipByEmail{
				RoleBindings: &[]sdk.CastaiUsersV1beta1InvitationRoleBinding{
					{
						RoleId: lo.ToPtr(MemberRoleID),
						Scopes: &[]sdk.CastaiUsersV1beta1InvitationRoleBindingScope{
							{
								Id:   lo.ToPtr(organizationID),
								Type: lo.ToPtr(sdk.CastaiRbacV1beta1ScopeTypeORGANIZATION),
							},
						},
					},
				},
				UserEmail: email,
			})
		}
	}

	tflog.Debug(ctx, "creating invitations")
	resp, err := client.UsersAPICreateInvitationsWithResponse(ctx, sdk.UsersAPICreateInvitationsJSONRequestBody{
		Members: &newMemberships,
	})
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("createMethod: creating invitations: %w, newMemberships: %+v", err, newMemberships))
	}

	data.SetId(organizationID)

	return resourceOrganizationMembersRead(ctx, data, meta)
}

func resourceOrganizationMembersRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "reading organization members")
	var owners, viewers, members []string
	var nextCursorBindings *string

	client := meta.(*ProviderConfig).api
	organizationID := data.Id()

	tflog.Debug(ctx, "listing role bindings")
	roleBindingsResp, err := client.RbacServiceAPIListRoleBindingsWithResponse(ctx, organizationID, &sdk.RbacServiceAPIListRoleBindingsParams{
		SubjectType: &[]sdk.RbacServiceAPIListRoleBindingsParamsSubjectType{sdk.SUBJECTUSER},
		ScopeType:   &[]sdk.RbacServiceAPIListRoleBindingsParamsScopeType{sdk.RbacServiceAPIListRoleBindingsParamsScopeTypeORGANIZATION},
		PageCursor:  nextCursorBindings,
	})

	if err := sdk.CheckOKResponse(roleBindingsResp, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving role bindings: %w", err))
	}

	for _, roleBinding := range *roleBindingsResp.JSON200.RoleBindings {
		for _, subject := range *roleBinding.Definition.Subjects {
			if subject.User == nil {
				continue
			}

			switch roleBinding.Definition.RoleId {
			case OwnerRoleID:
				owners = append(owners, *subject.User.Email)
			case ViewerRoleID:
				viewers = append(viewers, *subject.User.Email)
			case MemberRoleID:
				members = append(members, *subject.User.Email)
			}

		}
	}

	var nextCursorInvitations string

	for {
		tflog.Debug(ctx, "listing invitations")
		invitationsResp, err := client.UsersAPIListInvitationsWithResponse(ctx, &sdk.UsersAPIListInvitationsParams{
			PageCursor: &nextCursorInvitations,
		})

		if err := sdk.CheckOKResponse(invitationsResp, err); err != nil {
			return diag.FromErr(fmt.Errorf("retrieving pending invitations: %w", err))
		}

		for _, invitation := range invitationsResp.JSON200.Invitations {
			for _, roleBinding := range *invitation.RoleBindings {
				switch *roleBinding.RoleId {
				case OwnerRoleID:
					owners = append(owners, invitation.InviteEmail)
				case ViewerRoleID:
					viewers = append(viewers, invitation.InviteEmail)
				case MemberRoleID:
					members = append(members, invitation.InviteEmail)
				}
			}
		}

		nextCursorInvitations = invitationsResp.JSON200.NextCursor
		if nextCursorInvitations == "" {
			break
		}
	}

	tflog.Debug(ctx, "setting state")
	if err := data.Set(FieldOrganizationMembersOwners, owners); err != nil {
		return diag.FromErr(fmt.Errorf("setting owners: %w", err))
	}
	if err := data.Set(FieldOrganizationMembersViewers, viewers); err != nil {
		return diag.FromErr(fmt.Errorf("setting viewers: %w", err))
	}
	if err := data.Set(FieldOrganizationMembersMembers, members); err != nil {
		return diag.FromErr(fmt.Errorf("setting members: %w", err))
	}

	if _, ok := data.GetOk(FieldOrganizationMembersOrganizationID); !ok {
		if err := data.Set(FieldOrganizationMembersOrganizationID, data.Id()); err != nil {
			return diag.FromErr(fmt.Errorf("setting organization_id: %w", err))
		}
	}

	return nil
}

func resourceOrganizationMembersUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	organizationID := data.Id()

	tflog.Debug(ctx, "listing organization users")
	usersResp, err := client.UsersAPIListOrganizationUsersWithResponse(ctx, organizationID, &sdk.UsersAPIListOrganizationUsersParams{})
	if err := sdk.CheckOKResponse(usersResp, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving users: %w", err))
	}

	userIDByEmail := make(map[string]string)
	for _, user := range *usersResp.JSON200.Users {
		userIDByEmail[user.User.Email] = lo.FromPtr(user.User.Id)
	}

	invitationIDByEmail := make(map[string]string)

	var nextCursor string
	for {
		tflog.Debug(ctx, "listing invitations")
		invitationsResp, err := client.UsersAPIListInvitationsWithResponse(ctx, &sdk.UsersAPIListInvitationsParams{
			PageCursor: &nextCursor,
		})
		if err := sdk.CheckOKResponse(usersResp, err); err != nil {
			return diag.FromErr(fmt.Errorf("retrieving pending invitations: %w", err))
		}

		for _, invitation := range invitationsResp.JSON200.Invitations {
			invitationIDByEmail[invitation.InviteEmail] = lo.FromPtr(invitation.Id)
		}

		nextCursor = invitationsResp.JSON200.NextCursor
		if nextCursor == "" {
			break
		}
	}

	tflog.Debug(ctx, "getting members diff")
	diff := getMembersDiff(
		getRoleChange(data.GetChange(FieldOrganizationMembersOwners)),
		getRoleChange(data.GetChange(FieldOrganizationMembersViewers)),
		getRoleChange(data.GetChange(FieldOrganizationMembersMembers)),
	)

	tflog.Debug(ctx, fmt.Sprintf("update diff %+v", diff))

	manipulations := getPendingManipulations(diff, userIDByEmail, invitationIDByEmail)
	tflog.Debug(ctx, fmt.Sprintf("update manipulations %+v", manipulations))

	tflog.Debug(ctx, "getting current user profile")
	currentUserResp, err := client.UsersAPICurrentUserProfileWithResponse(ctx)
	if err := sdk.CheckOKResponse(currentUserResp, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving current user profile: %w", err))
	}
	tflog.Debug(ctx, "checking if current user is in members to delete")
	if contains(manipulations.membersToDelete, *currentUserResp.JSON200.Id) {
		return diag.FromErr(
			fmt.Errorf("can't delete user that is currently managing this organization: %s", lo.FromPtr(currentUserResp.JSON200.Email)),
		)
	}

	tflog.Debug(ctx, "deleting users from organization")
	for _, userID := range manipulations.membersToDelete {
		resp, err := client.UsersAPIRemoveUserFromOrganizationWithResponse(ctx, organizationID, userID)
		if err := sdk.CheckOKResponse(resp, err); err != nil {
			return diag.FromErr(fmt.Errorf("deleting user: %w", err))
		}
	}

	tflog.Debug(ctx, "deleting invitations")
	for _, invitationID := range manipulations.invitationsToDelete {
		resp, err := client.UsersAPIDeleteInvitationWithResponse(ctx, invitationID)
		if err := sdk.CheckOKResponse(resp, err); err != nil {
			return diag.FromErr(fmt.Errorf("deleting invitation: %w", err))
		}
	}

	tflog.Debug(ctx, "creating invitations")
	newMemberships := make([]sdk.CastaiUsersV1beta1NewMembershipByEmail, 0, len(manipulations.membersToAdd))
	for user, role := range manipulations.membersToAdd {
		newMemberships = append(newMemberships, sdk.CastaiUsersV1beta1NewMembershipByEmail{
			RoleBindings: &[]sdk.CastaiUsersV1beta1InvitationRoleBinding{
				{
					RoleId: lo.ToPtr(role),
					Scopes: &[]sdk.CastaiUsersV1beta1InvitationRoleBindingScope{
						{
							Id:   lo.ToPtr(organizationID),
							Type: lo.ToPtr(sdk.CastaiRbacV1beta1ScopeTypeORGANIZATION),
						},
					},
				},
			},
			UserEmail: user,
		})
	}

	for userID, roleID := range manipulations.membersToUpdate {
		rmUserResp, err := client.UsersAPIRemoveUserFromOrganizationWithResponse(ctx, organizationID, userID)
		if err := sdk.CheckOKResponse(rmUserResp, err); err != nil {
			return diag.FromErr(fmt.Errorf("deleting user: %w", err))
		}
		addUserResp, err := client.UsersAPIAddUserToOrganizationWithResponse(ctx, organizationID, sdk.CastaiUsersV1beta1NewMembership{
			UserId: userID,
		})
		if err := sdk.CheckOKResponse(addUserResp, err); err != nil {
			return diag.FromErr(fmt.Errorf("adding user to organization: %w", err))
		}

		createRoleBindingResp, err := client.RbacServiceAPICreateRoleBindingsWithResponse(ctx, organizationID, []sdk.CastaiRbacV1beta1CreateRoleBindingsRequestRoleBinding{
			{
				Name: fmt.Sprintf("organization-binding-%s-%s", roleID, userID),
				Definition: sdk.CastaiRbacV1beta1RoleBindingDefinition{
					RoleId: roleID,
					Subjects: &[]sdk.CastaiRbacV1beta1Subject{
						{
							User: &sdk.CastaiRbacV1beta1UserSubject{
								Id: userID,
							},
						},
					},
					Scopes: &[]sdk.CastaiRbacV1beta1Scope{
						{
							Organization: &sdk.CastaiRbacV1beta1OrganizationScope{
								Id: organizationID,
							},
						},
					},
				},
			},
		})

		if err := sdk.CheckOKResponse(createRoleBindingResp, err); err != nil {
			return diag.FromErr(fmt.Errorf("updating role binding: %w", err))
		}

	}

	rmUserResp, err := client.UsersAPICreateInvitationsWithResponse(ctx, sdk.UsersAPICreateInvitationsJSONRequestBody{
		Members: &newMemberships,
	})
	if err := sdk.CheckOKResponse(rmUserResp, err); err != nil {
		return diag.FromErr(fmt.Errorf("updateMethod: creating invitations: %w, newMemberships: %+v", err, newMemberships))
	}

	return resourceOrganizationMembersRead(ctx, data, meta)
}

func resourceOrganizationMembersDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	organizationID := data.Id()

	tflog.Debug(ctx, "getting current user profile")
	currentUserResp, err := client.UsersAPICurrentUserProfileWithResponse(ctx)
	if err := sdk.CheckOKResponse(currentUserResp, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving current user profile: %w", err))
	}

	tflog.Debug(ctx, "listing organization users")
	usersResp, err := client.UsersAPIListOrganizationUsersWithResponse(ctx, organizationID, &sdk.UsersAPIListOrganizationUsersParams{})
	if err := sdk.CheckOKResponse(usersResp, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving users: %w", err))
	}

	usersIDsToDelete := make([]string, 0, len(*usersResp.JSON200.Users))
	for _, user := range *usersResp.JSON200.Users {
		// When deleting all the members we have to filter out the
		// user that is currently managing this organization.
		// Otherwise, if we have deleted all the members, the organization would
		// get deleted too.
		if lo.FromPtr(user.User.Id) == lo.FromPtr(currentUserResp.JSON200.Id) {
			continue
		}

		usersIDsToDelete = append(usersIDsToDelete, lo.FromPtr(user.User.Id))
	}

	var invitationIDsToDelete []string
	var nextCursor string
	for {
		tflog.Debug(ctx, "listing invitations")
		invitationsResp, err := client.UsersAPIListInvitationsWithResponse(ctx, &sdk.UsersAPIListInvitationsParams{
			PageCursor: &nextCursor,
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("retrieving invitations: %w", err))
		}

		for _, invitation := range invitationsResp.JSON200.Invitations {
			invitationIDsToDelete = append(invitationIDsToDelete, lo.FromPtr(invitation.Id))
		}

		nextCursor = invitationsResp.JSON200.NextCursor
		if nextCursor == "" {
			break
		}
	}

	tflog.Debug(ctx, "deleting invitations")
	for _, invitationID := range invitationIDsToDelete {
		resp, err := client.UsersAPIDeleteInvitationWithResponse(ctx, invitationID)
		if err := sdk.CheckOKResponse(resp, err); err != nil {
			return diag.FromErr(fmt.Errorf("deleting invitation: %w", err))
		}
	}

	tflog.Debug(ctx, "deleting users from organization")
	for _, userID := range usersIDsToDelete {
		resp, err := client.UsersAPIRemoveUserFromOrganizationWithResponse(ctx, organizationID, userID)
		if err := sdk.CheckOKResponse(resp, err); err != nil {
			return diag.FromErr(fmt.Errorf("deleting user: %w", err))
		}
	}

	return nil
}

func compareRoleMembers(before, after []string) ([]string, []string) {
	var added []string
	var deleted []string

	for _, user := range before {
		if !contains(after, user) {
			deleted = append(deleted, user)
		}
	}

	for _, user := range after {
		if !contains(before, user) {
			added = append(added, user)
		}
	}

	return added, deleted
}

type roleChange struct {
	addedMembers   []string
	deletedMembers []string
}

func getRoleChange(before, after interface{}) roleChange {
	membersBefore := toStringList(before.([]interface{}))
	membersAfter := toStringList(after.([]interface{}))
	addedMembers, deletedMembers := compareRoleMembers(membersBefore, membersAfter)

	return roleChange{
		addedMembers:   addedMembers,
		deletedMembers: deletedMembers,
	}
}

type (
	userRoleByEmail map[string]string
	userRoleByID    map[string]string
)

type membersDiff struct {
	membersToAdd    userRoleByEmail
	membersToMove   userRoleByEmail
	membersToDelete map[string]struct{}
}

func getMembersDiff(owners, viewers, members roleChange) membersDiff {
	out := membersDiff{
		membersToMove:   map[string]string{},
		membersToDelete: map[string]struct{}{},
		membersToAdd:    map[string]string{},
	}

	for _, owner := range owners.deletedMembers {
		if contains(members.addedMembers, owner) {
			out.membersToMove[owner] = MemberRoleID
			continue
		}
		if contains(viewers.addedMembers, owner) {
			out.membersToMove[owner] = ViewerRoleID
			continue
		}

		out.membersToDelete[owner] = struct{}{}
	}

	for _, viewer := range viewers.deletedMembers {
		if contains(owners.addedMembers, viewer) {
			out.membersToMove[viewer] = OwnerRoleID
			continue
		}
		if contains(members.addedMembers, viewer) {
			out.membersToMove[viewer] = MemberRoleID
			continue
		}

		out.membersToDelete[viewer] = struct{}{}
	}

	for _, member := range members.deletedMembers {
		if contains(owners.addedMembers, member) {
			out.membersToMove[member] = OwnerRoleID
			continue
		}
		if contains(viewers.addedMembers, member) {
			out.membersToMove[member] = ViewerRoleID
			continue
		}

		out.membersToDelete[member] = struct{}{}
	}

	for _, owner := range owners.addedMembers {
		if _, ok := out.membersToMove[owner]; !ok {
			out.membersToAdd[owner] = OwnerRoleID
		}
	}

	for _, viewer := range viewers.addedMembers {
		if _, ok := out.membersToMove[viewer]; !ok {
			out.membersToAdd[viewer] = ViewerRoleID
		}
	}

	for _, member := range members.addedMembers {
		if _, ok := out.membersToMove[member]; !ok {
			out.membersToAdd[member] = MemberRoleID
		}
	}

	return out
}

type pendingManipulations struct {
	membersToAdd        userRoleByEmail
	membersToUpdate     userRoleByID
	membersToDelete     []string
	invitationsToDelete []string
}

func getPendingManipulations(input membersDiff, existingUserIDByEmail, invitationIDByEmail map[string]string) pendingManipulations {
	out := pendingManipulations{
		membersToAdd:        userRoleByEmail{},
		membersToUpdate:     userRoleByID{},
		membersToDelete:     []string{},
		invitationsToDelete: []string{},
	}

	for email := range input.membersToDelete {
		if userID, ok := existingUserIDByEmail[email]; ok {
			out.membersToDelete = append(out.membersToDelete, userID)
			continue
		}

		if invitationID, ok := invitationIDByEmail[email]; ok {
			out.invitationsToDelete = append(out.invitationsToDelete, invitationID)
		}
	}

	for user, role := range input.membersToMove {
		if userID, ok := existingUserIDByEmail[user]; ok {
			out.membersToUpdate[userID] = role
			continue
		}

		if invitationID, ok := invitationIDByEmail[user]; ok {
			out.invitationsToDelete = append(out.invitationsToDelete, invitationID)
		}

		out.membersToAdd[user] = role
	}

	maps.Copy(out.membersToAdd, input.membersToAdd)

	return out
}

func contains(array []string, elem string) bool {
	return slices.Contains(array, elem)
}
