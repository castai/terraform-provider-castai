package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldImpersonationServiceAccountId = "id"
)

func dataSourceImpersonationServiceAccount() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceImpersonationServiceAccountRead,
		Description: "Retrieve impersonation service account ID for AKS clusters",
		Schema: map[string]*schema.Schema{
			FieldImpersonationServiceAccountId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the impersonation service account",
			},
		},
	}
}

func dataSourceImpersonationServiceAccountRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	provider := "aks"
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
