package castai

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

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

	FieldServiceAccountKeyID               = "id"
	FieldServiceAccountKeyOrganizationID   = "organization_id"
	FieldServiceAccountKeyServiceAccountID = "service_account_id"
	FieldServiceAccountKeyName             = "name"
	FieldServiceAccountKeyPrefix           = "prefix"
	FieldServiceAccountKeyLastUsedAt       = "last_used_at"
	FieldServiceAccountKeyExpiresAt        = "expires_at"
	FieldServiceAccountKeyActive           = "active"
)

func resourceServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServiceAccountCreate,
		ReadContext:   resourceServiceAccountRead,
		UpdateContext: resourceServiceAccountUpdate,
		DeleteContext: resourceServiceAccountDelete,

		Description: "Service account resource allows managing CAST AI service accounts.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(3 * time.Minute),
			Read:   schema.DefaultTimeout(3 * time.Minute),
			Update: schema.DefaultTimeout(3 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldServiceAccountOrganizationID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "ID of the organization.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldServiceAccountName: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Name of the service account.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldServiceAccountDescription: {
				Type:        schema.TypeString,
				Optional:    true,
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

	organizationID, err := getOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "reading service account", map[string]interface{}{
		"resource_id":     data.Id(),
		"organization_id": organizationID,
	})

	resp, err := client.ServiceAccountsAPIGetServiceAccountWithResponse(ctx, organizationID, data.Id())
	if err != nil {
		return diag.Errorf("getting service account: %v", err)
	}

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

	organizationID, err := getOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

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

func resourceServiceAccountUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	organizationID, err := getOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	serviceAccountID := data.Id()
	name := data.Get(FieldServiceAccountName).(string)
	description := data.Get(FieldServiceAccountDescription).(string)

	tflog.Info(ctx, "updating service account", map[string]interface{}{
		"resource_id":     serviceAccountID,
		"name":            name,
		"description":     description,
		"organization_id": organizationID,
	})

	resp, err := client.ServiceAccountsAPIUpdateServiceAccountWithResponse(
		ctx,
		organizationID,
		serviceAccountID,
		sdk.ServiceAccountsAPIUpdateServiceAccountRequest{
			ServiceAccount: sdk.CastaiServiceaccountsV1beta1UpdateServiceAccountRequestServiceAccount{
				Name:        name,
				Description: &description,
			},
		},
	)

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("updating service account: %v", err)
	}

	tflog.Info(ctx, "created service account", map[string]interface{}{
		"resource_id":     serviceAccountID,
		"organization_id": organizationID,
		"name":            name,
		"description":     description,
	})

	return resourceServiceAccountRead(ctx, data, meta)
}

func resourceServiceAccountDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	organizationID, err := getOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	serviceAccountID := data.Id()

	tflog.Info(ctx, "deleting service account", map[string]interface{}{
		"resource_id":     serviceAccountID,
		"organization_id": organizationID,
	})

	resp, err := client.ServiceAccountsAPIDeleteServiceAccountWithResponse(ctx, organizationID, serviceAccountID)
	if err := sdk.CheckResponseNoContent(resp, err); err != nil {
		return diag.Errorf("deleting service account: %v", err)
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

func getOrganizationID(ctx context.Context, data *schema.ResourceData, meta interface{}) (string, error) {
	var organizationID string
	var err error

	organizationID = data.Get(FieldServiceAccountOrganizationID).(string)
	if organizationID == "" {
		organizationID, err = getDefaultOrganizationId(ctx, meta)
		if err != nil {
			return "", fmt.Errorf("getting organization ID: %w", err)
		}
	}

	return organizationID, nil
}

func resourceServiceAccountKey() *schema.Resource {
	return &schema.Resource{
		Description:   "Service account key resource allows managing CAST AI service account keys.",
		CreateContext: resourceServiceAccountKeyCreate,
		ReadContext:   resourceServiceAccountKeyRead,
		UpdateContext: resourceServiceAccountKeyUpdate,
		DeleteContext: resourceServiceAccountKeyDelete,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(3 * time.Minute),
			Read:   schema.DefaultTimeout(3 * time.Minute),
			Update: schema.DefaultTimeout(3 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldServiceAccountKeyOrganizationID: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "ID of the organization.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldServiceAccountKeyServiceAccountID: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "ID of the service account.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldServiceAccountKeyName: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Name of the service account key.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldServiceAccountKeyPrefix: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prefix of the service account key.",
			},
			FieldServiceAccountKeyLastUsedAt: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Last time the service account key was used.",
			},
			FieldServiceAccountKeyExpiresAt: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Expiration date of the service account key.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsRFC3339Time),
			},
			FieldServiceAccountKeyActive: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Active status of the service account key.",
			},
		},
	}
}

func resourceServiceAccountKeyRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if data.Id() == "" {
		return diag.Errorf("service account key ID is not set")
	}

	organizationID, err := getOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	serviceAccountID := data.Get(FieldServiceAccountKeyServiceAccountID).(string)
	if serviceAccountID == "" {
		return diag.Errorf("service account ID is not set")
	}
	serviceAccountKeyID := data.Id()

	logKeys := map[string]interface{}{
		"resource_id":        serviceAccountKeyID,
		"organization_id":    organizationID,
		"service_account_id": serviceAccountID,
	}

	tflog.Info(ctx, "reading service account key", logKeys)

	resp, err := client.ServiceAccountsAPIGetServiceAccountKeyWithResponse(ctx, organizationID, serviceAccountID, serviceAccountKeyID)
	if err != nil {
		return diag.Errorf("reading service account key: %v", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "resource is not found, removing from state", logKeys)
		data.SetId("")
		return nil
	}

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("reading service account key: %v", err)
	}

	tflog.Info(ctx, "found service account key", logKeys)

	serviceAccountKey := resp.JSON200

	if err := data.Set(FieldServiceAccountKeyOrganizationID, organizationID); err != nil {
		return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyOrganizationID, err)
	}

	if err := data.Set(FieldServiceAccountKeyServiceAccountID, serviceAccountID); err != nil {
		return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyServiceAccountID, err)
	}

	if err := data.Set(FieldServiceAccountKeyName, serviceAccountKey.Key.Name); err != nil {
		return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyName, err)
	}

	if err := data.Set(FieldServiceAccountKeyPrefix, serviceAccountKey.Key.Prefix); err != nil {
		return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyPrefix, err)
	}

	if err := data.Set(FieldServiceAccountKeyLastUsedAt, serviceAccountKey.Key.LastUsedAt); err != nil {
		return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyLastUsedAt, err)
	}

	if serviceAccountKey.Key.ExpiresAt != nil {
		if err := data.Set(FieldServiceAccountKeyExpiresAt, serviceAccountKey.Key.ExpiresAt.String()); err != nil {
			return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyExpiresAt, err)
		}
	}

	if err := data.Set(FieldServiceAccountKeyActive, serviceAccountKey.Key.Active); err != nil {
		return diag.Errorf("setting field %s: %v", FieldServiceAccountKeyActive, err)
	}
	return nil
}

func resourceServiceAccountKeyCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	organizationID, err := getOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	serviceAccountID := data.Get(FieldServiceAccountKeyServiceAccountID).(string)
	name := data.Get(FieldServiceAccountKeyName).(string)
	expiresAt := data.Get(FieldServiceAccountKeyExpiresAt).(string)
	active := data.Get(FieldServiceAccountKeyActive).(bool)

	expiresAtParsed, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return diag.Errorf("parsing expires_at date: %v", err)
	}

	logKeys := map[string]interface{}{
		"name":               name,
		"organization_id":    organizationID,
		"service_account_id": serviceAccountID,
	}

	tflog.Info(ctx, "creating service account key", logKeys)

	resp, err := client.ServiceAccountsAPICreateServiceAccountKeyWithResponse(
		ctx,
		organizationID,
		serviceAccountID,
		sdk.ServiceAccountsAPICreateServiceAccountKeyRequest{
			Key: sdk.CastaiServiceaccountsV1beta1CreateServiceAccountKeyRequestKey{
				Active:    &active,
				ExpiresAt: &expiresAtParsed,
				Name:      name,
			},
		},
	)
	if err := sdk.CheckResponseCreated(resp, err); err != nil {
		return diag.Errorf("creating service account key: %v", err)
	}

	serviceAccountKeyID := *resp.JSON201.Id
	logKeys["resource_id"] = serviceAccountKeyID
	tflog.Info(ctx, "created service account key", logKeys)

	data.SetId(serviceAccountKeyID)
	return resourceServiceAccountKeyRead(ctx, data, meta)
}

func resourceServiceAccountKeyUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	organizationID, err := getOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	serviceAccountID := data.Get(FieldServiceAccountKeyServiceAccountID).(string)
	keyID := data.Id()
	active := data.Get(FieldServiceAccountKeyActive).(bool)

	logKeys := map[string]interface{}{
		"organization_id":    organizationID,
		"service_account_id": serviceAccountID,
		"resource_id":        keyID,
	}

	tflog.Info(ctx, "updating service account key", logKeys)

	resp, err := client.ServiceAccountsAPIUpdateServiceAccountKeyWithResponse(ctx, organizationID, serviceAccountID, keyID, &sdk.ServiceAccountsAPIUpdateServiceAccountKeyParams{
		KeyActive: active,
	})

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("updating service account key: %v", err)
	}

	tflog.Info(ctx, "updated service account key", logKeys)

	return resourceServiceAccountKeyRead(ctx, data, meta)
}

func resourceServiceAccountKeyDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	organizationID, err := getOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	serviceAccountID := data.Get(FieldServiceAccountKeyServiceAccountID).(string)
	keyID := data.Id()

	logKeys := map[string]interface{}{
		"organization_id":    organizationID,
		"service_account_id": serviceAccountID,
		"resource_id":        keyID,
	}

	tflog.Info(ctx, "deleting service account key", logKeys)

	resp, err := client.ServiceAccountsAPIDeleteServiceAccountKeyWithResponse(ctx, organizationID, serviceAccountID, keyID)
	if err := sdk.CheckResponseNoContent(resp, err); err != nil {
		return diag.Errorf("deleting service account key: %v", err)
	}

	tflog.Info(ctx, "deleted service account key", logKeys)

	return nil
}
