package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

		Schema: map[string]*schema.Schema{
			FieldEnterpriseServiceAccountEnterpriseID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Enterprise organization ID.",
			},
			FieldEnterpriseServiceAccountOrganizationID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Target organization ID where the service account is created.",
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

func resourceEnterpriseServiceAccountRead(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient

	tflog.Debug(ctx, "Reading enterprise service account", map[string]any{"id": data.Id()})

	enterpriseID, ok := data.GetOk(FieldEnterpriseServiceAccountEnterpriseID)
	if !ok {
		return diag.FromErr(fmt.Errorf("enterprise ID is not set"))
	}

	orgID, ok := data.GetOk(FieldEnterpriseServiceAccountOrganizationID)
	if !ok {
		return diag.FromErr(fmt.Errorf("organization ID is not set"))
	}

	orgIDStr := orgID.(string)

	resp, err := client.EnterpriseAPIListEnterpriseServiceAccountsWithResponse(
		ctx,
		enterpriseID.(string),
		&organization_management.EnterpriseAPIListEnterpriseServiceAccountsParams{
			OrganizationId: &[]string{orgIDStr},
		},
	)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("list enterprise service accounts failed: %w", err))
	}

	if resp.JSON200 == nil || resp.JSON200.Items == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from list enterprise service accounts"))
	}

	var found *organization_management.ListEnterpriseServiceAccountsResponseServiceAccount
	for _, item := range *resp.JSON200.Items {
		if item.Id != nil && *item.Id == data.Id() {
			found = &item
			break
		}
	}

	if found == nil {
		tflog.Warn(ctx, "Enterprise service account not found, removing from state", map[string]any{
			"id":              data.Id(),
			"enterprise_id":   enterpriseID.(string),
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
	orgID := data.Get(FieldEnterpriseServiceAccountOrganizationID).(string)
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

	if created.Email != nil {
		if err := data.Set(FieldEnterpriseServiceAccountEmail, *created.Email); err != nil {
			return diag.FromErr(err)
		}
	}

	data.SetId(*created.Id)

	tflog.Debug(ctx, "Created enterprise service account", map[string]any{"id": *created.Id})

	return nil
}

func resourceEnterpriseServiceAccountUpdate(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
	return diag.Errorf("updating enterprise service accounts is not yet supported (pending CID-236); to change name or description, delete and recreate the resource")
}

func resourceEnterpriseServiceAccountDelete(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient

	enterpriseID := data.Get(FieldEnterpriseServiceAccountEnterpriseID).(string)
	orgID := data.Get(FieldEnterpriseServiceAccountOrganizationID).(string)

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
