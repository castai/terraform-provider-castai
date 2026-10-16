package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"
)

// dataSourceRole looks up a single RBAC role by its exact name.
func dataSourceRole() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRoleRead,
		Description: "Looks up a single RBAC role in a Cast AI organization by its exact, case-sensitive name. " +
			"Role names are unique per organization, so the name identifies the role; the data source errors " +
			"if no role with the given name exists. Useful for resolving role IDs (Owner, Member, Viewer) " +
			"without maintaining a hardcoded UUID map.",
		Schema: map[string]*schema.Schema{
			FieldRolesOrganizationID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Cast AI organization ID.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldRolesName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Exact, case-sensitive name of the role to look up.",
			},
			FieldRolesType: {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          roleTypeAll,
				Description:      "Server-side role type filter. One of: ALL (default), DEFAULT, ENTERPRISE.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{roleTypeAll, roleTypeDefault, roleTypeEnterprise}, false)),
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Role description.",
			},
			"default": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this is a default (built-in) role.",
			},
			"deprecated": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this role is deprecated.",
			},
			"level": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Priority level of the role (higher is more important).",
			},
			"permissions": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Permissions granted by the role.",
				Elem:        rolePermissionsSchemaResource(),
			},
		},
	}
}

func dataSourceRoleRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	organizationID, ok := d.GetOk(FieldRolesOrganizationID)
	if !ok {
		return diag.Errorf("%q is not set", FieldRolesOrganizationID)
	}
	name, ok := d.GetOk(FieldRolesName)
	if !ok {
		return diag.Errorf("%q is not set", FieldRolesName)
	}
	roleType := d.Get(FieldRolesType).(string)

	roles, err := listRoles(ctx, client, organizationID.(string), name.(string), roleType)
	if err != nil {
		return diag.FromErr(err)
	}

	switch {
	case len(roles) == 0:
		return diag.Errorf("no %s role with name %q found in organization %s", roleType, name, organizationID)
	case len(roles) > 1:
		return diag.Errorf("expected a single role with name %q in organization %s, got %d", name, organizationID, len(roles))
	}

	role := roles[0]
	d.SetId(lo.FromPtr(role.Id))
	for key, value := range roleToMap(role) {
		if key == "id" || key == FieldRolesName {
			continue
		}
		if err := d.Set(key, value); err != nil {
			return diag.FromErr(fmt.Errorf("setting %s: %w", key, err))
		}
	}

	return nil
}
