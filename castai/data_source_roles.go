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
	FieldRolesOrganizationID = "organization_id"
	FieldRolesName           = "name"
	FieldRolesType           = "type"
	FieldRolesRoles          = "roles"

	roleTypeAll        = "ALL"
	roleTypeDefault    = "DEFAULT"
	roleTypeEnterprise = "ENTERPRISE"

	// rolesListMaxPages caps pagination
	rolesListMaxPages = 500
)

func dataSourceRoles() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRolesRead,
		Description: "Lists RBAC roles in a Cast AI organization. To look up a single role by its name, " +
			"use the castai_role data source.",
		Schema: map[string]*schema.Schema{
			FieldRolesOrganizationID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Cast AI organization ID.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldRolesType: {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          roleTypeAll,
				Description:      "Server-side role type filter. One of: ALL (default), DEFAULT, ENTERPRISE.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{roleTypeAll, roleTypeDefault, roleTypeEnterprise}, false)),
			},
			FieldRolesRoles: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of roles matching the filters.",
				Elem:        roleSchemaResource(),
			},
		},
	}
}

func dataSourceRolesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	organizationID, ok := d.GetOk(FieldRolesOrganizationID)
	if !ok {
		return diag.Errorf("%q is not set", FieldRolesOrganizationID)
	}
	roleType := d.Get(FieldRolesType).(string)

	roles, err := listRoles(ctx, client, organizationID.(string), "", roleType)
	if err != nil {
		return diag.FromErr(err)
	}

	mapped := make([]map[string]any, 0, len(roles))
	for _, role := range roles {
		mapped = append(mapped, roleToMap(role))
	}

	d.SetId(organizationID.(string))
	if err := d.Set(FieldRolesRoles, mapped); err != nil {
		return diag.FromErr(fmt.Errorf("setting roles: %w", err))
	}

	return nil
}

func listRoles(ctx context.Context, client sdk.ClientWithResponsesInterface, organizationID, nameFilter, roleType string) ([]sdk.CastaiRbacV1beta1Role, error) {
	roles := make([]sdk.CastaiRbacV1beta1Role, 0)
	seenRoleIDs := make(map[string]bool)
	var cursor *string
	pages := 0
	for {
		pages++
		if pages > rolesListMaxPages {
			return nil, fmt.Errorf("listing roles: exceeded maximum of %d pages", rolesListMaxPages)
		}
		resp, err := client.RbacServiceAPIListRolesWithResponse(ctx, organizationID, &sdk.RbacServiceAPIListRolesParams{
			PageCursor: cursor,
			Type:       lo.ToPtr(sdk.RbacServiceAPIListRolesParamsType(roleType)),
		})
		if e := sdk.CheckOKResponse(resp, err); e != nil {
			return nil, fmt.Errorf("listing roles: %w", e)
		}
		if resp.JSON200 == nil {
			return nil, fmt.Errorf("unexpected empty response from listing roles")
		}

		added := 0
		for _, role := range lo.FromPtr(resp.JSON200.Roles) {
			if nameFilter != "" && lo.FromPtr(role.Name) != nameFilter {
				continue
			}
			roleID := lo.FromPtr(role.Id)
			if seenRoleIDs[roleID] {
				continue
			}
			seenRoleIDs[roleID] = true
			added++
			roles = append(roles, role)
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

	return roles, nil
}

func roleSchemaResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Role UUID.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Role name.",
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

func rolePermissionsSchemaResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"effect": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Effect of the permission, e.g. EFFECT_ALLOW.",
			},
			"endpoints": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "API endpoints the permission applies to.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"groups": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "API groups the permission applies to.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func roleToMap(role sdk.CastaiRbacV1beta1Role) map[string]any {
	permissions := make([]map[string]any, 0)
	for _, permission := range lo.FromPtr(role.Definition.Permissions) {
		permissions = append(permissions, map[string]any{
			"effect":    string(permission.Effect),
			"endpoints": lo.ToAnySlice(lo.FromPtr(permission.Endpoints)),
			"groups":    lo.ToAnySlice(lo.FromPtr(permission.Groups)),
		})
	}

	return map[string]any{
		"id":          lo.FromPtr(role.Id),
		"name":        lo.FromPtr(role.Name),
		"description": lo.FromPtr(role.Description),
		"default":     lo.FromPtr(role.Default),
		"deprecated":  lo.FromPtr(role.Deprecated),
		"level":       int(lo.FromPtr(role.Level)),
		"permissions": permissions,
	}
}
