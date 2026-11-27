package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldOrgGroupsOrganizationID = "organization_id"
	FieldOrgGroupsManagedBy      = "managed_by"
	FieldOrgGroupsGroups         = "groups"

	groupManagedByConsole    = "console"
	groupManagedByTerraform  = "terraform"
	groupManagedByIdpSync    = "idp-sync"
	groupManagedBySSODefault = "sso-default"

	// groupsListMaxPages caps pagination.
	groupsListMaxPages = 500
)

func dataSourceOrganizationGroups() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOrganizationGroupsRead,
		Description: "Lists groups in a Cast AI organization, e.g. SCIM-synced groups created by the IdP that " +
			"never enter Terraform state. To look up a single group by its exact name, use the " +
			"castai_organization_group data source.",
		Schema: map[string]*schema.Schema{
			FieldOrgGroupsOrganizationID: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "Cast AI organization ID.",
			},
			FieldOrgGroupsManagedBy: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{groupManagedByConsole, groupManagedByTerraform, groupManagedByIdpSync, groupManagedBySSODefault}, false)),
				},
				Description: "Server-side filter by how the groups are managed. One or more of: console, terraform, idp-sync, sso-default. Unset returns all groups.",
			},
			FieldOrgGroupsGroups: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of groups matching the filters. Empty when nothing matches; this is not an error.",
				Elem:        organizationGroupSchemaResource(),
			},
		},
	}
}

// organizationGroupSchemaResource holds the fields describing a single group,
// shared by the castai_organization_group and castai_organization_groups data
// sources.
func organizationGroupSchemaResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Group UUID, e.g. to reference a subject in castai_role_bindings.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Group name.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Group description.",
			},
			"managed_by": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "How the group is managed: console, terraform, idp-sync or sso-default; empty for legacy groups.",
			},
			"sso_provider": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SSO connection that owns the group, when the group is provisioned by SSO/SCIM.",
			},
		},
	}
}

func dataSourceOrganizationGroupsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	organizationID := d.Get(FieldOrgGroupsOrganizationID).(string)

	groups, err := listGroups(ctx, client, organizationID, organizationGroupsManagedByFilter(d))
	if err != nil {
		return diag.FromErr(err)
	}

	mapped := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		mapped = append(mapped, organizationGroupToMap(group))
	}

	d.SetId(organizationID)
	if err := d.Set(FieldOrgGroupsGroups, mapped); err != nil {
		return diag.FromErr(fmt.Errorf("setting groups: %w", err))
	}

	return nil
}

// organizationGroupsManagedByFilter reads the managed_by list as a Go slice;
// an empty or unset list is returned as nil so no filter reaches the API.
func organizationGroupsManagedByFilter(d *schema.ResourceData) []string {
	raw := d.Get(FieldOrgGroupsManagedBy).([]any)
	if len(raw) == 0 {
		return nil
	}
	managedBy := make([]string, 0, len(raw))
	for _, v := range raw {
		managedBy = append(managedBy, v.(string))
	}
	return managedBy
}

// listGroups fetches all groups of an organization, following the pagination
// cursor until every page is retrieved. A nil managedBy returns all groups.
func listGroups(ctx context.Context, client sdk.ClientWithResponsesInterface, organizationID string, managedBy []string) ([]sdk.CastaiRbacV1beta1Group, error) {
	groups := make([]sdk.CastaiRbacV1beta1Group, 0)
	seenGroupIDs := make(map[string]bool)
	var cursor *string
	pages := 0
	for {
		pages++
		if pages > groupsListMaxPages {
			return nil, fmt.Errorf("listing groups: exceeded maximum of %d pages", groupsListMaxPages)
		}
		params := &sdk.RbacServiceAPIListGroupsParams{
			PageCursor: cursor,
		}
		if len(managedBy) > 0 {
			params.ManagedBy = &managedBy
		}
		resp, err := client.RbacServiceAPIListGroupsWithResponse(ctx, organizationID, params)
		if e := sdk.CheckOKResponse(resp, err); e != nil {
			return nil, fmt.Errorf("listing groups: %w", e)
		}
		if resp.JSON200 == nil {
			return nil, fmt.Errorf("unexpected empty response from listing groups")
		}

		added := 0
		for _, group := range lo.FromPtr(resp.JSON200.Groups) {
			groupID := lo.FromPtr(group.Id)
			if seenGroupIDs[groupID] {
				continue
			}
			seenGroupIDs[groupID] = true
			added++
			groups = append(groups, group)
		}

		next := ""
		if resp.JSON200.NextPage != nil && resp.JSON200.NextPage.Cursor != nil {
			next = *resp.JSON200.NextPage.Cursor
		}
		if next == "" || added == 0 {
			break
		}
		cursor = &next
	}

	return groups, nil
}

func organizationGroupToMap(group sdk.CastaiRbacV1beta1Group) map[string]any {
	return map[string]any{
		"id":           lo.FromPtr(group.Id),
		"name":         lo.FromPtr(group.Name),
		"description":  lo.FromPtr(group.Description),
		"managed_by":   lo.FromPtr(group.ManagedBy),
		"sso_provider": lo.FromPtr(group.SsoProvider),
	}
}
