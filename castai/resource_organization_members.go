package castai

import (
	"context"
	"fmt"
	"time"

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

const (
	ownerRole  = "owner"
	viewerRole = "viewer"
	memberRole = "member"
)

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
			},
			FieldOrganizationMembersViewers: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of email addresses corresponding to users who should be given viewer access to the organization.",
			},
			FieldOrganizationMembersMembers: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of email addresses corresponding to users who should be given member access to the organization.",
			},
		},
	}
}

func resourceOrganizationMembersCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

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
				Role:      ownerRole,
				UserEmail: email,
			})
		}
	}

	if viewers, ok := data.GetOk(FieldOrganizationMembersViewers); ok {
		emails := toStringList(viewers.([]interface{}))

		for _, email := range emails {
			newMemberships = append(newMemberships, sdk.CastaiUsersV1beta1NewMembershipByEmail{
				Role:      viewerRole,
				UserEmail: email,
			})
		}
	}

	if members, ok := data.GetOk(FieldOrganizationMembersMembers); ok {
		emails := toStringList(members.([]interface{}))

		for _, email := range emails {
			newMemberships = append(newMemberships, sdk.CastaiUsersV1beta1NewMembershipByEmail{
				Role:      memberRole,
				UserEmail: email,
			})
		}
	}

	organizationID := data.Get(FieldOrganizationMembersOrganizationID).(string)

	resp, err := client.UsersAPICreateInvitationsWithResponse(ctx, sdk.UsersAPICreateInvitationsJSONRequestBody{
		Members: &newMemberships,
	})
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("creating invitations: %w", err))
	}

	data.SetId(organizationID)

	return resourceOrganizationMembersRead(ctx, data, meta)
}

func resourceOrganizationMembersRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	organizationID := data.Id()
	usersResp, err := client.UsersAPIListOrganizationUsersWithResponse(ctx, organizationID, &sdk.UsersAPIListOrganizationUsersParams{})
	if err := sdk.CheckOKResponse(usersResp, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving users: %w", err))
	}

	var owners, viewers, members []string
	for _, user := range *usersResp.JSON200.Users {
		switch user.Role {
		case ownerRole:
			owners = append(owners, user.User.Email)
		case viewerRole:
			viewers = append(viewers, user.User.Email)
		case memberRole:
			members = append(members, user.User.Email)
		}
	}

	var nextCursor string
	for {
		invitationsResp, err := client.UsersAPIListInvitationsWithResponse(ctx, &sdk.UsersAPIListInvitationsParams{
			PageCursor: &nextCursor,
		})
		if err := sdk.CheckOKResponse(usersResp, err); err != nil {
			return diag.FromErr(fmt.Errorf("retrieving pending invitations: %w", err))
		}

		for _, invitation := range invitationsResp.JSON200.Invitations {
			switch invitation.Role {
			case ownerRole:
				owners = append(owners, invitation.InviteEmail)
			case viewerRole:
				viewers = append(viewers, invitation.InviteEmail)
			case memberRole:
				members = append(members, invitation.InviteEmail)
			}
		}

		nextCursor = invitationsResp.JSON200.NextCursor
		if nextCursor == "" {
			break
		}
	}

	if err := data.Set(FieldOrganizationMembersOwners, owners); err != nil {
		return diag.FromErr(fmt.Errorf("setting owners: %w", err))
	}
	if err := data.Set(FieldOrganizationMembersViewers, viewers); err != nil {
		return diag.FromErr(fmt.Errorf("setting viewers);: %w", err))
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

	diff := getMembersDiff(
		getRoleChange(data.GetChange(FieldOrganizationMembersOwners)),
		getRoleChange(data.GetChange(FieldOrganizationMembersViewers)),
		getRoleChange(data.GetChange(FieldOrganizationMembersMembers)),
	)

	manipulations := getPendingManipulations(diff, userIDByEmail, invitationIDByEmail)

	currentUserResp, err := client.UsersAPICurrentUserProfileWithResponse(ctx)
	if err := sdk.CheckOKResponse(currentUserResp, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving current user profile: %w", err))
	}
	if contains(manipulations.membersToDelete, *currentUserResp.JSON200.Id) {
		return diag.FromErr(
			fmt.Errorf("can't delete user that is currently managing this organization: %s", lo.FromPtr(currentUserResp.JSON200.Email)),
		)
	}

	for userID, role := range manipulations.membersToUpdate {
		role := role
		resp, err := client.UsersAPIUpdateOrganizationUserWithResponse(ctx, organizationID, userID, sdk.UsersAPIUpdateOrganizationUserJSONRequestBody{
			Role: &role,
		})
		if err := sdk.CheckOKResponse(resp, err); err != nil {
			return diag.FromErr(fmt.Errorf("updating user: %w", err))
		}
	}

	for _, userID := range manipulations.membersToDelete {
		resp, err := client.UsersAPIRemoveUserFromOrganizationWithResponse(ctx, organizationID, userID)
		if err := sdk.CheckOKResponse(resp, err); err != nil {
			return diag.FromErr(fmt.Errorf("deleting user: %w", err))
		}
	}

	for _, invitationID := range manipulations.invitationsToDelete {
		resp, err := client.UsersAPIDeleteInvitationWithResponse(ctx, invitationID)
		if err := sdk.CheckOKResponse(resp, err); err != nil {
			return diag.FromErr(fmt.Errorf("deleting invitation: %w", err))
		}
	}

	newMemberships := make([]sdk.CastaiUsersV1beta1NewMembershipByEmail, 0, len(manipulations.membersToAdd))
	for user, role := range manipulations.membersToAdd {
		newMemberships = append(newMemberships, sdk.CastaiUsersV1beta1NewMembershipByEmail{
			Role:      role,
			UserEmail: user,
		})
	}

	resp, err := client.UsersAPICreateInvitationsWithResponse(ctx, sdk.UsersAPICreateInvitationsJSONBody{
		Members: &newMemberships,
	})
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("creating invitations: %w", err))
	}

	return resourceOrganizationMembersRead(ctx, data, meta)
}

func resourceOrganizationMembersDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	organizationID := data.Id()

	currentUserResp, err := client.UsersAPICurrentUserProfileWithResponse(ctx)
	if err := sdk.CheckOKResponse(currentUserResp, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving current user profile: %w", err))
	}

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

	for _, invitationID := range invitationIDsToDelete {
		resp, err := client.UsersAPIDeleteInvitationWithResponse(ctx, invitationID)
		if err := sdk.CheckOKResponse(resp, err); err != nil {
			return diag.FromErr(fmt.Errorf("deleting invitation: %w", err))
		}
	}

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
			out.membersToMove[owner] = memberRole
			continue
		}
		if contains(viewers.addedMembers, owner) {
			out.membersToMove[owner] = viewerRole
			continue
		}

		out.membersToDelete[owner] = struct{}{}
	}

	for _, viewer := range viewers.deletedMembers {
		if contains(owners.addedMembers, viewer) {
			out.membersToMove[viewer] = ownerRole
			continue
		}
		if contains(members.addedMembers, viewer) {
			out.membersToMove[viewer] = memberRole
			continue
		}

		out.membersToDelete[viewer] = struct{}{}
	}

	for _, member := range members.deletedMembers {
		if contains(owners.addedMembers, member) {
			out.membersToMove[member] = ownerRole
			continue
		}
		if contains(viewers.addedMembers, member) {
			out.membersToMove[member] = viewerRole
			continue
		}

		out.membersToDelete[member] = struct{}{}
	}

	for _, owner := range owners.addedMembers {
		if _, ok := out.membersToMove[owner]; !ok {
			out.membersToAdd[owner] = ownerRole
		}
	}

	for _, viewer := range viewers.addedMembers {
		if _, ok := out.membersToMove[viewer]; !ok {
			out.membersToAdd[viewer] = viewerRole
		}
	}

	for _, member := range members.addedMembers {
		if _, ok := out.membersToMove[member]; !ok {
			out.membersToAdd[member] = memberRole
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

	for email, role := range input.membersToAdd {
		out.membersToAdd[email] = role
	}

	return out
}

func contains(array []string, elem string) bool {
	for _, v := range array {
		if v == elem {
			return true
		}
	}
	return false
}
