package castai

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldOrgUsersOrganizationID = "organization_id"
	FieldOrgUsersEmail          = "email"
	FieldOrgUsersAuthProvider   = "auth_provider"
	FieldOrgUsersScimActive     = "scim_active"
	FieldOrgUsersRoleIDs        = "role_ids"
	FieldOrgUsersIncludeGroups  = "include_groups"
	FieldOrgUsersUsers          = "users"
)

func dataSourceOrganizationUsers() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOrganizationUsersRead,
		Description: "Lists users that are members of a Cast AI organization. The API has no email filter and " +
			"one email can map to several identities (e.g. a password login and an SSO login), so filters are " +
			"applied client-side and a lookup can return multiple users; pair it with a lifecycle precondition " +
			"when a single subject id is needed, e.g. for castai_role_bindings.",
		Schema: map[string]*schema.Schema{
			FieldOrgUsersOrganizationID: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "Cast AI organization ID.",
			},
			FieldOrgUsersEmail: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Client-side, case-insensitive filter by user email. A single email can map to several identities, so this can still match multiple users.",
			},
			FieldOrgUsersAuthProvider: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Client-side filter by auth provider label (the SSO connection the identity comes from), exactly as returned in the user payload.",
			},
			FieldOrgUsersScimActive: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Client-side filter for users managed by SCIM provisioning. Only an explicitly set value filters; when the attribute is absent from the configuration no filtering happens.",
			},
			FieldOrgUsersRoleIDs: {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Server-side filter by role IDs; only members holding any of the listed roles are returned.",
			},
			FieldOrgUsersIncludeGroups: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Server-side flag controlling whether group memberships are fetched and exposed in group_ids. Defaults to true.",
			},
			FieldOrgUsersUsers: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of users matching the filters, ordered by user ID. Empty when nothing matches; this is not an error.",
				Elem:        organizationUsersUserSchema(),
			},
		},
	}
}

func organizationUsersUserSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User UUID, e.g. to reference a subject in castai_role_bindings.",
			},
			"email": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User email.",
			},
			"username": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Username, corresponds to the auth0 user id (SSO subject).",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Readable user name.",
			},
			"auth_provider": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Auth provider label for identities coming from an SSO connection; empty otherwise.",
			},
			"scim_active": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the user is managed by SCIM provisioning.",
			},
			"group_ids": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "IDs of the groups the user belongs to; empty when include_groups is false.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceOrganizationUsersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	organizationID := d.Get(FieldOrgUsersOrganizationID).(string)

	params := &sdk.UsersAPIListOrganizationUsersParams{
		IncludeGroups: lo.ToPtr(d.Get(FieldOrgUsersIncludeGroups).(bool)),
	}
	if roleIDs := organizationUsersRoleIDs(d); len(roleIDs) > 0 {
		params.RoleId = &roleIDs
	}

	resp, err := client.UsersAPIListOrganizationUsersWithResponse(ctx, organizationID, params)
	if err != nil {
		return diag.FromErr(fmt.Errorf("listing organization users: %w", err))
	}
	// Guard before touching the response: its methods have value receivers, so a
	// nil response would panic rather than return an error.
	if resp == nil {
		return diag.FromErr(errors.New("unexpected empty response from listing organization users"))
	}
	if e := sdk.StatusOk(resp); e != nil {
		return diag.FromErr(fmt.Errorf("listing organization users: %w", e))
	}
	if resp.JSON200 == nil {
		return diag.FromErr(errors.New("unexpected empty response from listing organization users"))
	}

	emailFilter := d.Get(FieldOrgUsersEmail).(string)
	authProviderFilter := d.Get(FieldOrgUsersAuthProvider).(string)
	scimActiveFilter := organizationUsersScimActiveFilter(d.GetRawConfig())

	matched := lo.Filter(lo.FromPtr(resp.JSON200.Users), func(membership sdk.CastaiUsersV1beta1Membership, _ int) bool {
		return organizationUsersMatchFilters(membership.User, emailFilter, authProviderFilter, scimActiveFilter)
	})
	// The API does not promise an order, so sort by user ID to keep ids and
	// indexes stable across reads. The sort is stable, so identities sharing a
	// missing id keep their API order.
	slices.SortStableFunc(matched, func(a, b sdk.CastaiUsersV1beta1Membership) int {
		return strings.Compare(lo.FromPtr(a.User.Id), lo.FromPtr(b.User.Id))
	})

	users := make([]map[string]any, 0, len(matched))
	for _, membership := range matched {
		users = append(users, organizationUserToMap(membership.User, lo.FromPtr(membership.Groups)))
	}

	d.SetId(organizationID)
	if err := d.Set(FieldOrgUsersUsers, users); err != nil {
		return diag.FromErr(fmt.Errorf("setting users: %w", err))
	}

	return nil
}

// organizationUsersRoleIDs reads the role_ids list as a Go slice; an empty or
// unset list is returned as nil so no role filter reaches the API.
func organizationUsersRoleIDs(d *schema.ResourceData) []string {
	raw := d.Get(FieldOrgUsersRoleIDs).([]any)
	if len(raw) == 0 {
		return nil
	}
	roleIDs := make([]string, 0, len(raw))
	for _, id := range raw {
		roleIDs = append(roleIDs, id.(string))
	}
	return roleIDs
}

// organizationUsersScimActiveFilter resolves the scim_active filter from the
// raw configuration: d.Get reports an absent optional bool the same way as an
// explicit false, so the distinction has to come from the config value. It
// returns nil when scim_active is not set in the configuration, meaning no
// filtering should happen. A nil raw config (e.g. during tests) is treated as
// "not set" as well, and so is an unknown value: reading it would panic.
func organizationUsersScimActiveFilter(raw cty.Value) *bool {
	if !rawConfigHasField(raw, FieldOrgUsersScimActive) {
		return nil
	}
	attr := raw.GetAttr(FieldOrgUsersScimActive)
	if !attr.IsKnown() || attr.Type() != cty.Bool {
		return nil
	}
	active := attr.True()
	return &active
}

// organizationUsersMatchFilters applies the client-side filters to a single
// membership user. An empty filter value means "no filtering" for that filter;
// scimActiveFilter nil likewise means no filtering.
func organizationUsersMatchFilters(
	user sdk.CastaiUsersV1beta1User,
	emailFilter, authProviderFilter string,
	scimActiveFilter *bool,
) bool {
	if emailFilter != "" && !strings.EqualFold(user.Email, emailFilter) {
		return false
	}
	if authProviderFilter != "" && lo.FromPtr(user.AuthProvider) != authProviderFilter {
		return false
	}
	if scimActiveFilter != nil && lo.FromPtr(user.ScimActive) != *scimActiveFilter {
		return false
	}
	return true
}

func organizationUserToMap(user sdk.CastaiUsersV1beta1User, groups []sdk.CastaiUsersV1beta1GroupRef) map[string]any {
	return map[string]any{
		"id":            lo.FromPtr(user.Id),
		"email":         user.Email,
		"username":      user.Username,
		"name":          user.Name,
		"auth_provider": lo.FromPtr(user.AuthProvider),
		"scim_active":   lo.FromPtr(user.ScimActive),
		"group_ids": lo.Map(groups, func(group sdk.CastaiUsersV1beta1GroupRef, _ int) string {
			return group.Id
		}),
	}
}
