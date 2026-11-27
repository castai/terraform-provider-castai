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
	FieldOrgGroupOrganizationID = "organization_id"
	FieldOrgGroupName           = "name"
)

func dataSourceOrganizationGroup() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOrganizationGroupRead,
		Description: "Looks up a single group in a Cast AI organization by its exact, case-sensitive name. " +
			"Useful for resolving SCIM-synced group UUIDs for castai_role_bindings without hardcoding them. " +
			"The data source errors if no group with the given name exists.",
		Schema: organizationGroupDataSourceSchema(),
	}
}

// organizationGroupDataSourceSchema builds the schema of the singular group
// data source: the lookup inputs plus the computed fields shared with the
// castai_organization_groups list element (organizationGroupSchemaResource).
func organizationGroupDataSourceSchema() map[string]*schema.Schema {
	fields := map[string]*schema.Schema{
		FieldOrgGroupOrganizationID: {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			Description:      "Cast AI organization ID.",
		},
		FieldOrgGroupName: {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			Description:      "Exact, case-sensitive name of the group to look up.",
		},
	}
	// The top-level "id" attribute is not declared here: it is managed
	// implicitly by the SDK from d.SetId, like in the castai_role data source.
	for name, field := range organizationGroupComputedFields() {
		fields[name] = field
	}
	return fields
}

func dataSourceOrganizationGroupRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	organizationID := d.Get(FieldOrgGroupOrganizationID).(string)
	name := d.Get(FieldOrgGroupName).(string)

	// The groups list API has no name filter, so this walks every page and
	// matches in memory. The full scan is intentional: group names are
	// expected to be unique per organization, and stopping at the first match
	// would hide duplicate names from the error below.
	groups, err := listGroups(ctx, client, organizationID, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	matches := make([]sdk.CastaiRbacV1beta1Group, 0)
	for _, group := range groups {
		if lo.FromPtr(group.Name) == name {
			matches = append(matches, group)
		}
	}

	switch {
	case len(matches) == 0:
		return diag.Errorf("no group with name %q found in organization %s", name, organizationID)
	case len(matches) > 1:
		// Group names are expected to be unique per organization; if the
		// backend ever returns more than one, fail instead of guessing.
		return diag.Errorf("expected a single group with name %q in organization %s, got %d", name, organizationID, len(matches))
	}

	group := matches[0]
	d.SetId(lo.FromPtr(group.Id))
	if err := d.Set("description", lo.FromPtr(group.Description)); err != nil {
		return diag.FromErr(fmt.Errorf("setting description: %w", err))
	}
	if err := d.Set("managed_by", lo.FromPtr(group.ManagedBy)); err != nil {
		return diag.FromErr(fmt.Errorf("setting managed_by: %w", err))
	}
	if err := d.Set("sso_provider", lo.FromPtr(group.SsoProvider)); err != nil {
		return diag.FromErr(fmt.Errorf("setting sso_provider: %w", err))
	}

	return nil
}
