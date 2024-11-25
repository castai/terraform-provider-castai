package castai

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldServiceAccountOrganizationID = "organization_id"
	FieldServiceAccountName           = "name"
	FieldServiceAccountID             = "service_account_id"
	FieldServiceAccountDescription    = "description"

	FieldServiceAccountKeyOrganizationID = "organization_id"
	FieldServiceAccountKeyName           = "name"
	FieldServiceAccountKeyDescription    = "description"
)

func resourceServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServiceAccountCreate,
		ReadContext:   resourceServiceAccountRead,
		UpdateContext: resourceServiceAccountUpdate,
		DeleteContext: resourceServiceAccountDelete,

		Description: "Service Account resource allows managing CAST AI service accounts.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(3 * time.Minute),
			Update: schema.DefaultTimeout(3 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldServiceAccountOrganizationID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the organization.",
			},
			FieldServiceAccountName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the service account.",
			},
			FieldServiceAccountDescription: {
				Type:        schema.TypeString,
				Required:    false,
				Description: "Description of the service account.",
			},
		},
	}
}

func resourceServiceAccountRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if data.Id() == "" {
		return nil
	}

	client := meta.(*ProviderConfig).api
	resp, err := client.ServiceAccountsAPIGetServiceAccountWithResponse(ctx, data.Id())
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("getting service account: %v", err)
	}

	serviceAccount := resp.JSON200

	if err := data.Set(FieldServiceAccountName, serviceAccount.ServiceAccount.Name); err != nil {
		return diag.Errorf("setting service account name: %v", err)
	}
	if err := data.Set(FieldServiceAccountDescription, serviceAccount.ServiceAccount.Description); err != nil {
		return diag.Errorf("setting service account description: %v", err)
	}

	return nil
}

func resourceServiceAccountCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	organizationID := data.Get(FieldServiceAccountOrganizationID).(string)
	name := data.Get(FieldServiceAccountName).(string)
	description := data.Get(FieldServiceAccountDescription).(string)

	resp, err := client.ServiceAccountsAPICreateServiceAccountWithResponse(ctx, organizationID, sdk.ServiceAccountsAPICreateServiceAccountRequest{
		ServiceAccount: sdk.CastaiServiceaccountsV1beta1CreateServiceAccountRequestServiceAccount{
			Description: &description,
			Name:        name,
		},
	})

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("creating service account: %v", err)
	}

	data.SetId(*resp.JSON201.Id)

	return nil
}

func resourceServiceAccountDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceServiceAccountUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceServiceAccountKey() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceServiceAccountKeyRead,
		CreateContext: resourceServiceAccountKeyCreate,
		UpdateContext: resourceServiceAccountKeyUpdate,
		DeleteContext: resourceServiceAccountKeyDelete,
		Description:   "Service Account Key resource allows managing CAST AI service account keys.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(3 * time.Minute),
			Update: schema.DefaultTimeout(3 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldServiceAccountKeyOrganizationID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the organization.",
			},
			FieldServiceAccountID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the service account.",
			},
			FieldServiceAccountKeyName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the service account key.",
			},
			FieldServiceAccountKeyDescription: {
				Type:        schema.TypeString,
				Required:    false,
				Description: "Description of the service account key.",
			},
		},
	}
}

func resourceServiceAccountKeyRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceServiceAccountKeyCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceServiceAccountKeyDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceServiceAccountKeyUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}
