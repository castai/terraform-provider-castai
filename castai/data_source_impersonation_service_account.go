package castai

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldImpersonationServiceAccountProviderType = "provider_type"
	FieldImpersonationServiceAccountId           = "id"
	FieldImpersonationServiceAccountEmail        = "email"
)

func dataSourceImpersonationServiceAccount() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceImpersonationServiceAccountRead,
		Description: "Retrieve impersonation service account ID and email for AKS or GKE clusters",
		Schema: map[string]*schema.Schema{
			FieldImpersonationServiceAccountProviderType: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"gke", "aks"}, false)),
				Description:      "Type of the provider the impersonation service account is used for: \"gke\" or \"aks\"",
			},
			FieldImpersonationServiceAccountId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the impersonation service account",
			},
			FieldImpersonationServiceAccountEmail: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Email of the impersonation service account",
			},
		},
	}
}

func dataSourceImpersonationServiceAccountRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterType, ok := data.Get(FieldImpersonationServiceAccountProviderType).(string)
	if !ok || strings.TrimSpace(clusterType) == "" {
		return diag.FromErr(fmt.Errorf("%q must not be empty", FieldImpersonationServiceAccountProviderType))
	}
	resp, err := client.ExternalClusterAPIImpersonationServiceAccountWithResponse(ctx,
		sdk.ExternalclusterV1ImpersonationServiceAccountRequest{
			Provider: &clusterType,
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

	if resp.JSON200.Email != nil {
		if err := data.Set(FieldImpersonationServiceAccountEmail, *resp.JSON200.Email); err != nil {
			return diag.FromErr(fmt.Errorf("setting email: %w", err))
		}
	}

	return nil
}
