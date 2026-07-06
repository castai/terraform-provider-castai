package castai

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/organization_management"
)

const (
	FieldEnterpriseServiceAccountEnterpriseID   = "enterprise_id"
	FieldEnterpriseServiceAccountOrganizationID = "organization_id"
	FieldEnterpriseServiceAccountName           = "name"
	FieldEnterpriseServiceAccountDescription    = "description"
	FieldEnterpriseServiceAccountEmail          = "email"
)

func resourceEnterpriseServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEnterpriseServiceAccountCreate,
		ReadContext:   resourceEnterpriseServiceAccountRead,
		UpdateContext: resourceEnterpriseServiceAccountUpdate,
		DeleteContext: resourceEnterpriseServiceAccountDelete,
		Description:   "CAST AI Enterprise Service Account resource.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		CustomizeDiff: resourceEnterpriseServiceAccountCustomizeDiff,

		Schema: map[string]*schema.Schema{
			FieldEnterpriseServiceAccountEnterpriseID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Enterprise organization ID.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldEnterpriseServiceAccountOrganizationID: {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				Description:      "Target organization ID where the service account is created. Defaults to enterprise_id (enterprise scope) when omitted.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldEnterpriseServiceAccountName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Service account name.",
			},
			FieldEnterpriseServiceAccountDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Service account description.",
			},
			FieldEnterpriseServiceAccountEmail: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Auto-generated service account email (read-only).",
			},
		},
	}
}

func resourceEnterpriseServiceAccountCustomizeDiff(_ context.Context, d *schema.ResourceDiff, _ any) error {
	if d.Get(FieldEnterpriseServiceAccountOrganizationID).(string) == "" {
		if enterpriseID := d.Get(FieldEnterpriseServiceAccountEnterpriseID).(string); enterpriseID != "" {
			return d.SetNew(FieldEnterpriseServiceAccountOrganizationID, enterpriseID)
		}
	}
	return nil
}

// getEnterpriseServiceAccountOrgID returns organization_id from state, falling back to enterprise_id.
// The fallback covers imports and older state entries written before organization_id became optional.
func getEnterpriseServiceAccountOrgID(d *schema.ResourceData) string {
	if orgID := d.Get(FieldEnterpriseServiceAccountOrganizationID).(string); orgID != "" {
		return orgID
	}
	return d.Get(FieldEnterpriseServiceAccountEnterpriseID).(string)
}

func resourceEnterpriseServiceAccountRead(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient

	tflog.Debug(ctx, "Reading enterprise service account", map[string]any{"id": data.Id()})

	enterpriseIDStr := data.Get(FieldEnterpriseServiceAccountEnterpriseID).(string)
	orgIDStr := getEnterpriseServiceAccountOrgID(data)

	var found *organization_management.ListEnterpriseServiceAccountsResponseServiceAccount
	var cursor *string
	for {
		resp, err := client.EnterpriseAPIListEnterpriseServiceAccountsWithResponse(
			ctx,
			enterpriseIDStr,
			&organization_management.EnterpriseAPIListEnterpriseServiceAccountsParams{
				OrganizationId: &[]string{orgIDStr},
				PageCursor:     cursor,
			},
		)
		if err := sdk.CheckOKResponse(resp, err); err != nil {
			return diag.FromErr(fmt.Errorf("list enterprise service accounts failed: %w", err))
		}
		if resp.JSON200 == nil {
			return diag.FromErr(fmt.Errorf("unexpected empty response from list enterprise service accounts"))
		}
		for _, item := range lo.FromPtr(resp.JSON200.Items) {
			if item.Id != nil && *item.Id == data.Id() {
				itemCopy := item
				found = &itemCopy
				break
			}
		}
		if found != nil || resp.JSON200.NextPageCursor == nil || *resp.JSON200.NextPageCursor == "" {
			break
		}
		cursor = resp.JSON200.NextPageCursor
	}

	if found == nil {
		tflog.Warn(ctx, "Enterprise service account not found, removing from state", map[string]any{
			"id":              data.Id(),
			"enterprise_id":   enterpriseIDStr,
			"organization_id": orgIDStr,
		})
		data.SetId("")
		return nil
	}

	if found.Name != nil {
		if err := data.Set(FieldEnterpriseServiceAccountName, *found.Name); err != nil {
			return diag.FromErr(err)
		}
	}
	if found.Description != nil {
		if err := data.Set(FieldEnterpriseServiceAccountDescription, *found.Description); err != nil {
			return diag.FromErr(err)
		}
	}
	if found.Email != nil {
		if err := data.Set(FieldEnterpriseServiceAccountEmail, *found.Email); err != nil {
			return diag.FromErr(err)
		}
	}

	tflog.Debug(ctx, "Finished reading enterprise service account", map[string]any{"id": data.Id()})

	return nil
}

func resourceEnterpriseServiceAccountCreate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient

	enterpriseID := data.Get(FieldEnterpriseServiceAccountEnterpriseID).(string)
	orgID := getEnterpriseServiceAccountOrgID(data)
	name := data.Get(FieldEnterpriseServiceAccountName).(string)
	description := data.Get(FieldEnterpriseServiceAccountDescription).(string)

	tflog.Debug(ctx, "Creating enterprise service account", map[string]any{"name": name})

	resp, err := client.EnterpriseAPIBatchCreateEnterpriseServiceAccountsWithResponse(
		ctx,
		enterpriseID,
		organization_management.BatchCreateEnterpriseServiceAccountsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchCreateEnterpriseServiceAccountsRequestServiceAccountRequest{
				{
					Name:           name,
					Description:    lo.ToPtr(description),
					OrganizationId: orgID,
				},
			},
		},
	)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("batch create enterprise service accounts failed: %w", err))
	}

	if resp.JSON200 == nil || resp.JSON200.Items == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from batch create enterprise service accounts"))
	}

	if len(*resp.JSON200.Items) != 1 {
		return diag.FromErr(fmt.Errorf("unexpected number of service accounts created: expected 1, got %d", len(*resp.JSON200.Items)))
	}

	created := (*resp.JSON200.Items)[0]

	if created.Id == nil {
		return diag.FromErr(fmt.Errorf("created enterprise service account has no ID"))
	}

	data.SetId(*created.Id)

	tflog.Debug(ctx, "Created enterprise service account", map[string]any{"id": *created.Id})

	return resourceEnterpriseServiceAccountRead(ctx, data, meta)
}

func resourceEnterpriseServiceAccountUpdate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient

	enterpriseID := data.Get(FieldEnterpriseServiceAccountEnterpriseID).(string)
	orgID := getEnterpriseServiceAccountOrgID(data)
	name := data.Get(FieldEnterpriseServiceAccountName).(string)
	description := data.Get(FieldEnterpriseServiceAccountDescription).(string)

	tflog.Debug(ctx, "Updating enterprise service account", map[string]any{"id": data.Id()})

	resp, err := client.EnterpriseAPIBatchUpdateEnterpriseServiceAccountsWithResponse(
		ctx,
		enterpriseID,
		organization_management.BatchUpdateEnterpriseServiceAccountsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchUpdateEnterpriseServiceAccountsRequestUpdateServiceAccountRequest{
				{
					ServiceAccountId: data.Id(),
					OrganizationId:   orgID,
					Name:             lo.ToPtr(name),
					Description:      lo.ToPtr(description),
				},
			},
		},
	)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("batch update enterprise service accounts failed: %w", err))
	}

	tflog.Debug(ctx, "Updated enterprise service account", map[string]any{"id": data.Id()})

	return resourceEnterpriseServiceAccountRead(ctx, data, meta)
}

func resourceEnterpriseServiceAccountDelete(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient

	enterpriseID := data.Get(FieldEnterpriseServiceAccountEnterpriseID).(string)
	orgID := getEnterpriseServiceAccountOrgID(data)

	tflog.Debug(ctx, "Deleting enterprise service account", map[string]any{"id": data.Id()})

	resp, err := client.EnterpriseAPIBatchDeleteEnterpriseServiceAccountsWithResponse(
		ctx,
		enterpriseID,
		organization_management.BatchDeleteEnterpriseServiceAccountsRequest{
			EnterpriseId: enterpriseID,
			Requests: []organization_management.BatchDeleteEnterpriseServiceAccountsRequestDeleteServiceAccountRequest{
				{
					Id:             data.Id(),
					OrganizationId: orgID,
				},
			},
		},
	)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("batch delete enterprise service accounts failed: %w", err))
	}

	data.SetId("")

	tflog.Debug(ctx, "Deleted enterprise service account", map[string]any{})

	return nil
}
