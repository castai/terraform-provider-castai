package castai

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldServiceAccountOrganizationID = "organization_id"
	FieldServiceAccountName           = "name"
	FieldServiceAccountID             = "service_account_id"
	FieldServiceAccountDescription    = "description"
	FieldServiceAccountEmail          = "email"

	FieldServiceAccountAuthor      = "author"
	FieldServiceAccountAuthorID    = "id"
	FieldServiceAccountAuthorEmail = "email"
	FieldServiceAccountAuthorKind  = "kind"
)

func resourceServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServiceAccountCreate,
		ReadContext:   resourceServiceAccountRead,
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
				ForceNew:    true,
				Description: "Name of the service account.",
			},
			FieldServiceAccountDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Description of the service account.",
			},
			FieldServiceAccountEmail: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Email of the service account.",
			},
			FieldServiceAccountAuthor: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldServiceAccountAuthorID:    {Type: schema.TypeString, Computed: true},
						FieldServiceAccountAuthorEmail: {Type: schema.TypeString, Computed: true},
						FieldServiceAccountAuthorKind:  {Type: schema.TypeString, Computed: true},
					},
				},
				Computed:    true,
				Description: "Author of the service account.",
			},
		},
	}
}

func resourceServiceAccountRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if data.Id() == "" {
		return diag.Errorf("service account ID is not set")
	}

	organizationID := data.Get(FieldServiceAccountOrganizationID).(string)

	tflog.Info(ctx, "reading service account", map[string]interface{}{
		"resource_id":     data.Id(),
		"organization_id": organizationID,
	})

	resp, err := client.ServiceAccountsAPIGetServiceAccountWithResponse(ctx, organizationID, data.Id())
	if resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "resource is not found, removing from state", map[string]interface{}{
			"resource_id":     data.Id(),
			"organization_id": organizationID,
		})
		data.SetId("") // Mark resource as deleted
		return nil
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("getting service account: %v", err)
	}

	tflog.Info(ctx, "found service account", map[string]interface{}{
		"resource_id":     data.Id(),
		"organization_id": organizationID,
	})
	serviceAccount := resp.JSON200

	if err := data.Set(FieldServiceAccountName, serviceAccount.ServiceAccount.Name); err != nil {
		return diag.Errorf("setting service account name: %v", err)
	}

	if err := data.Set(FieldServiceAccountEmail, serviceAccount.ServiceAccount.Email); err != nil {
		return diag.Errorf("setting service account email: %v", err)
	}

	if err := data.Set(FieldServiceAccountDescription, serviceAccount.ServiceAccount.Description); err != nil {
		return diag.Errorf("setting service account description: %v", err)
	}

	authorData := flattenServiceAccountAuthor(serviceAccount.ServiceAccount.Author)
	if err := data.Set(FieldServiceAccountAuthor, authorData); err != nil {
		return diag.Errorf("setting service account author: %v", err)
	}
	return nil
}

func resourceServiceAccountCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	organizationID := data.Get(FieldServiceAccountOrganizationID).(string)
	name := data.Get(FieldServiceAccountName).(string)
	description := data.Get(FieldServiceAccountDescription).(string)

	tflog.Info(ctx, "creating service account", map[string]interface{}{
		"name":            name,
		"description":     description,
		"organization_id": organizationID,
	})

	resp, err := client.ServiceAccountsAPICreateServiceAccountWithResponse(ctx, organizationID, sdk.CastaiServiceaccountsV1beta1CreateServiceAccountRequestServiceAccount{
		Name:        name,
		Description: &description,
	},
	)

	if err := sdk.CheckResponseCreated(resp, err); err != nil {
		return diag.Errorf("creating service account: %v", err)
	}

	tflog.Info(ctx, "created service account", map[string]interface{}{
		"resource_id":     *resp.JSON201.Id,
		"organization_id": organizationID,
	})
	data.SetId(*resp.JSON201.Id)

	return resourceServiceAccountRead(ctx, data, meta)
}

func resourceServiceAccountDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	organizationID := data.Get(FieldServiceAccountOrganizationID).(string)
	serviceAccountID := data.Id()

	tflog.Info(ctx, "deleting service account", map[string]interface{}{
		"resource_id":     serviceAccountID,
		"organization_id": organizationID,
	})

	resp, err := client.ServiceAccountsAPIDeleteServiceAccount(ctx, organizationID, serviceAccountID)
	if err != nil {
		return diag.Errorf("deleting service account: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		return diag.Errorf("deleteting service account: expected status: [204], received status: [%d]", resp.StatusCode)
	}

	tflog.Info(ctx, "deleted service account", map[string]interface{}{
		"resource_id":     serviceAccountID,
		"organization_id": organizationID,
	})

	return nil
}

func flattenServiceAccountAuthor(author *sdk.CastaiServiceaccountsV1beta1ServiceAccountAuthor) []map[string]interface{} {
	if author == nil {
		return []map[string]interface{}{}
	}

	return []map[string]interface{}{
		{
			FieldServiceAccountAuthorID:    stringValue(author.Id),
			FieldServiceAccountAuthorEmail: stringValue(author.Email),
			FieldServiceAccountAuthorKind:  stringValue(author.Kind),
		},
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
