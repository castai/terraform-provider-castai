package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldImpersonationServiceAccountId            = "id"
	FieldImpersonationServiceAccountCloudProvider = "cloud_provider"
)

func dataSourceImpersonationServiceAccount() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceImpersonationServiceAccountRead,
		Description: "Retrieve the CAST AI per-organization impersonation service account used for cloud identity (e.g. for GKE impersonation onboarding).",
		Schema: map[string]*schema.Schema{
			FieldImpersonationServiceAccountId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the impersonation service account",
			},
			FieldImpersonationServiceAccountCloudProvider: {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "aks",
				Description: "Cloud provider to retrieve the impersonation service account for (e.g. aks, gcp).",
			},
		},
	}
}

func dataSourceImpersonationServiceAccountRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	provider := data.Get(FieldImpersonationServiceAccountCloudProvider).(string)
	resp, err := client.ExternalClusterAPIImpersonationServiceAccountWithResponse(ctx,
		sdk.ExternalclusterV1ImpersonationServiceAccountRequest{
			Provider: &provider,
		})
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving impersonation service account: %w", err))
	}

	if resp.JSON200 == nil {
		return diag.FromErr(fmt.Errorf("empty response received from impersonation service account API"))
	}

	if resp.JSON200.Id != nil {
		data.SetId(*resp.JSON200.Id)
		if err := data.Set(FieldImpersonationServiceAccountId, *resp.JSON200.Id); err != nil {
			return diag.FromErr(fmt.Errorf("setting id: %w", err))
		}
	}

	return nil
}
