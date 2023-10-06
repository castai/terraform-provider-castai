package castai

import (
	"context"
	"fmt"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	FieldOrganizationName = "name"
)

func dataSourceOrganization() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOrganizationRead,
		Description: "Retrieve organization ID",
		Schema: map[string]*schema.Schema{
			FieldOrganizationName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
		},
	}
}

func dataSourceOrganizationRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	resp, err := client.UsersAPIListOrganizationsWithResponse(ctx, &sdk.UsersAPIListOrganizationsParams{})
	if err := sdk.CheckOKResponse(resp.HTTPResponse, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving organizations: %w", err))
	}

	organizationName := data.Get(FieldOrganizationName).(string)

	var organizationID string
	for _, organization := range *resp.JSON200.Organizations {
		if organizationName == organization.Name {
			organizationID = *organization.Id
			break
		}
	}

	if organizationID == "" {
		return diag.FromErr(fmt.Errorf("organization %s not found", organizationName))
	}

	data.SetId(organizationID)

	return nil
}
