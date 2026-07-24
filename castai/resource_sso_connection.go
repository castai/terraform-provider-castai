package castai

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"golang.org/x/crypto/bcrypt"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldSSOConnectionName                   = "name"
	FieldSSOConnectionEmailDomain            = "email_domain"
	FieldSSOConnectionAdditionalEmailDomains = "additional_email_domains"
	FieldSSOConnectionSynchronizeUserGroups  = "synchronize_user_groups"
	FieldSSOConnectionSyncAuthToken          = "sync_auth_token"

	FieldSSOConnectionAAD            = "aad"
	FieldSSOConnectionADDomain       = "ad_domain"
	FieldSSOConnectionADClientID     = "client_id"
	FieldSSOConnectionADClientSecret = "client_secret"

	FieldSSOConnectionOkta             = "okta"
	FieldSSOConnectionOktaDomain       = "okta_domain"
	FieldSSOConnectionOktaClientID     = "client_id"
	FieldSSOConnectionOktaClientSecret = "client_secret"

	FieldSSOConnectionOIDC             = "oidc"
	FieldSSOConnectionOIDCIssuerURL    = "issuer_url"
	FieldSSOConnectionOIDCClientID     = "client_id"
	FieldSSOConnectionOIDCClientSecret = "client_secret"
	FieldSSOConnectionOIDCType         = "type"
)

func resourceSSOConnection() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiSSOConnectionCreate,
		ReadContext:   resourceCastaiSSOConnectionRead,
		UpdateContext: resourceCastaiSSOConnectionUpdate,
		DeleteContext: resourceCastaiSSOConnectionDelete,
		CustomizeDiff: resourceCastaiSSOConnectionDiff,
		Description:   "SSO Connection resource allows creating SSO trust relationship with CAST AI.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(3 * time.Minute),
			Update: schema.DefaultTimeout(3 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldSSOConnectionName: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Connection name",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldSSOConnectionEmailDomain: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Email domain of the connection",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldSSOConnectionAdditionalEmailDomains: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Additional email domains that will be allowed to sign in via the connection",
				MinItems:    1,
				Elem: &schema.Schema{
					Required:         false,
					Type:             schema.TypeString,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				},
			},
			FieldSSOConnectionSynchronizeUserGroups: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "When enabled, user groups from the identity provider will be synchronized with CAST AI. A sync auth token is generated on activation and stored in sync_auth_token.",
			},
			FieldSSOConnectionSyncAuthToken: {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Auth token generated when synchronize_user_groups is enabled. Only populated on the transition from false to true.",
			},
			FieldSSOConnectionAAD: {
				Type:        schema.TypeList,
				MaxItems:    1,
				MinItems:    1,
				Optional:    true,
				Description: "Azure AD connector",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldSSOConnectionADDomain: {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "Azure AD domain",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
						FieldSSOConnectionADClientID: {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "Azure AD client ID",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
						FieldSSOConnectionADClientSecret: {
							Type:             schema.TypeString,
							Sensitive:        true,
							Required:         true,
							Description:      "Azure AD client secret",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
							DiffSuppressFunc: func(_, oldValue, newValue string, _ *schema.ResourceData) bool {
								decodedSecret, err := base64.StdEncoding.DecodeString(oldValue)
								if err != nil {
									return false
								}
								return bcrypt.CompareHashAndPassword(decodedSecret, []byte(newValue)) == nil
							},
						},
					},
				},
			},
			FieldSSOConnectionOkta: {
				Type:        schema.TypeList,
				MaxItems:    1,
				MinItems:    1,
				Optional:    true,
				Description: "Okta connector",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldSSOConnectionOktaDomain: {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "Okta domain",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
						FieldSSOConnectionOktaClientID: {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "Okta client ID",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
						FieldSSOConnectionOktaClientSecret: {
							Type:             schema.TypeString,
							Required:         true,
							Sensitive:        true,
							Description:      "Okta client secret",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
							DiffSuppressFunc: func(_, oldValue, newValue string, _ *schema.ResourceData) bool {
								decodedSecret, err := base64.StdEncoding.DecodeString(oldValue)
								if err != nil {
									return false
								}
								return bcrypt.CompareHashAndPassword(decodedSecret, []byte(newValue)) == nil
							},
						},
					},
				},
			},
			FieldSSOConnectionOIDC: {
				Type:        schema.TypeList,
				MaxItems:    1,
				MinItems:    1,
				Optional:    true,
				Description: "OIDC connector (e.g. Keycloak)",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldSSOConnectionOIDCIssuerURL: {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "Issuer URL of the OpenID Connect provider",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
						FieldSSOConnectionOIDCClientID: {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "OIDC client ID",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
						FieldSSOConnectionOIDCClientSecret: {
							Type:             schema.TypeString,
							Required:         true,
							Sensitive:        true,
							Description:      "OIDC client secret",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
							DiffSuppressFunc: func(_, oldValue, newValue string, _ *schema.ResourceData) bool {
								decodedSecret, err := base64.StdEncoding.DecodeString(oldValue)
								if err != nil {
									return false
								}
								return bcrypt.CompareHashAndPassword(decodedSecret, []byte(newValue)) == nil
							},
						},
						FieldSSOConnectionOIDCType: {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     string(sdk.CastaiSsoV1beta1OIDCTypeTYPEBACKCHANNEL),
							Description: "OIDC connection type (TYPE_BACK_CHANNEL or TYPE_FRONT_CHANNEL)",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
								string(sdk.CastaiSsoV1beta1OIDCTypeTYPEBACKCHANNEL),
								string(sdk.CastaiSsoV1beta1OIDCTypeTYPEFRONTCHANNEL),
							}, false)),
						},
					},
				},
			},
		},
	}
}

func resourceCastaiSSOConnectionCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	req := sdk.CastaiSsoV1beta1CreateSSOConnection{
		Name:        data.Get(FieldSSOConnectionName).(string),
		EmailDomain: data.Get(FieldSSOConnectionEmailDomain).(string),
	}

	if v, ok := data.Get(FieldSSOConnectionAdditionalEmailDomains).([]any); ok && len(v) > 0 {
		var domains []string
		for _, v := range v {
			domains = append(domains, v.(string))
		}
		req.AdditionalEmailDomains = toPtr(domains)
	}

	if v, ok := data.Get(FieldSSOConnectionAAD).([]any); ok && len(v) > 0 {
		req.Aad = toADConnector(v[0].(map[string]any))
	}

	if v, ok := data.Get(FieldSSOConnectionOkta).([]any); ok && len(v) > 0 {
		req.Okta = toOktaConnector(v[0].(map[string]any))
	}

	if v, ok := data.Get(FieldSSOConnectionOIDC).([]any); ok && len(v) > 0 {
		req.Oidc = toOIDCConnector(v[0].(map[string]any))
	}

	resp, err := client.SSOAPICreateSSOConnectionWithResponse(ctx, req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("creating sso connection: %v", err)
	}

	if err := checkSSOStatus(resp.JSON200); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(*resp.JSON200.Id)

	var syncDiags diag.Diagnostics
	if data.Get(FieldSSOConnectionSynchronizeUserGroups).(bool) {
		syncDiags = setSSOConnectionSync(ctx, client, data, true)
		if syncDiags.HasError() {
			return syncDiags
		}
	}

	return append(syncDiags, resourceCastaiSSOConnectionRead(ctx, data, meta)...)
}

func resourceCastaiSSOConnectionRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if data.Id() == "" {
		return nil
	}

	client := meta.(*ProviderConfig).api
	resp, err := client.SSOAPIGetSSOConnectionWithResponse(ctx, data.Id())
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("retrieving sso connection: %v", err)
	}

	connection := resp.JSON200

	if err := data.Set(FieldSSOConnectionName, connection.Connection.Name); err != nil {
		return diag.Errorf("setting connection name: %v", err)
	}
	if err := data.Set(FieldSSOConnectionEmailDomain, connection.Connection.EmailDomain); err != nil {
		return diag.Errorf("setting email domain: %v", err)
	}
	if err := data.Set(FieldSSOConnectionAdditionalEmailDomains, connection.Connection.AdditionalEmailDomains); err != nil {
		return diag.Errorf("setting additional email domains: %v", err)
	}

	isSynced := false
	if connection.Connection.IsSynced != nil {
		isSynced = *connection.Connection.IsSynced
	}
	if err := data.Set(FieldSSOConnectionSynchronizeUserGroups, isSynced); err != nil {
		return diag.Errorf("setting synchronize_user_groups: %v", err)
	}

	// sync_auth_token is a one-time value returned only by the SetSync endpoint — the GET
	// API never returns it. Explicitly write back the current state value so Terraform does
	// not drop it from state during a refresh.
	if err := data.Set(FieldSSOConnectionSyncAuthToken, data.Get(FieldSSOConnectionSyncAuthToken)); err != nil {
		return diag.Errorf("setting sync_auth_token: %v", err)
	}

	return nil
}

func resourceCastaiSSOConnectionUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !data.HasChanges(
		FieldSSOConnectionName,
		FieldSSOConnectionEmailDomain,
		FieldSSOConnectionAdditionalEmailDomains,
		FieldSSOConnectionAAD,
		FieldSSOConnectionOkta,
		FieldSSOConnectionOIDC,
		FieldSSOConnectionSynchronizeUserGroups,
	) {
		return nil
	}

	client := meta.(*ProviderConfig).api
	req := sdk.CastaiSsoV1beta1UpdateSSOConnection{}

	if v, ok := data.GetOk(FieldSSOConnectionName); ok {
		req.Name = toPtr(v.(string))
	}
	if v, ok := data.GetOk(FieldSSOConnectionEmailDomain); ok {
		req.EmailDomain = toPtr(v.(string))
	}

	if v, ok := data.Get(FieldSSOConnectionAdditionalEmailDomains).([]any); ok && len(v) > 0 {
		var domains []string
		for _, v := range v {
			domains = append(domains, v.(string))
		}
		req.AdditionalEmailDomains = toPtr(domains)
	}

	if v, ok := data.Get(FieldSSOConnectionAAD).([]any); ok && len(v) > 0 {
		req.Aad = toADConnector(v[0].(map[string]any))
	}

	if v, ok := data.Get(FieldSSOConnectionOkta).([]any); ok && len(v) > 0 {
		req.Okta = toOktaConnector(v[0].(map[string]any))
	}

	if v, ok := data.Get(FieldSSOConnectionOIDC).([]any); ok && len(v) > 0 {
		req.Oidc = toOIDCConnector(v[0].(map[string]any))
	}

	if data.HasChanges(
		FieldSSOConnectionName,
		FieldSSOConnectionEmailDomain,
		FieldSSOConnectionAdditionalEmailDomains,
		FieldSSOConnectionAAD,
		FieldSSOConnectionOkta,
		FieldSSOConnectionOIDC,
	) {
		resp, err := client.SSOAPIUpdateSSOConnectionWithResponse(ctx, data.Id(), req)
		if err := sdk.CheckOKResponse(resp, err); err != nil {
			return diag.Errorf("updating sso connection: %v", err)
		}

		if err := checkSSOStatus(resp.JSON200); err != nil {
			return diag.FromErr(err)
		}
	}

	var syncDiags diag.Diagnostics
	if data.HasChange(FieldSSOConnectionSynchronizeUserGroups) {
		oldVal, newVal := data.GetChange(FieldSSOConnectionSynchronizeUserGroups)
		oldSync := oldVal.(bool)
		newSync := newVal.(bool)

		// Only call SetSync on actual transitions.
		if oldSync != newSync {
			syncDiags = setSSOConnectionSync(ctx, client, data, newSync)
			if syncDiags.HasError() {
				return syncDiags
			}
		}
	}

	return append(syncDiags, resourceCastaiSSOConnectionRead(ctx, data, meta)...)
}

func resourceCastaiSSOConnectionDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	resp, err := client.SSOAPIDeleteSSOConnectionWithResponse(ctx, data.Id())
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("deleting sso connection: %v", err)
	}

	return nil
}

func checkSSOStatus(input *sdk.CastaiSsoV1beta1SSOConnection) error {
	if input == nil && input.Status == nil {
		return nil
	}

	if *input.Status == sdk.CastaiSsoV1beta1SSOConnectionStatusSTATUSACTIVE {
		return nil
	}

	if input.Error == nil {
		return fmt.Errorf("invalid SSO connection status: %s", *input.Status)
	}

	return fmt.Errorf("SSO connection status: %s failed with error: %s", *input.Status, *input.Error)
}

func resourceCastaiSSOConnectionDiff(_ context.Context, rd *schema.ResourceDiff, _ interface{}) error {
	connectors := 0
	if v, ok := rd.Get(FieldSSOConnectionAAD).([]any); ok && len(v) > 0 {
		connectors++
	}

	if v, ok := rd.Get(FieldSSOConnectionOkta).([]any); ok && len(v) > 0 {
		connectors++
	}

	if v, ok := rd.Get(FieldSSOConnectionOIDC).([]any); ok && len(v) > 0 {
		connectors++
	}

	if connectors != 1 {
		return errors.New("only 1 connector can be configured")
	}

	return nil
}

func toADConnector(obj map[string]any) *sdk.CastaiSsoV1beta1AzureAAD {
	if obj == nil {
		return nil
	}

	out := &sdk.CastaiSsoV1beta1AzureAAD{}
	if v, ok := obj[FieldSSOConnectionADDomain].(string); ok {
		out.AdDomain = v
	}
	if v, ok := obj[FieldSSOConnectionADClientID].(string); ok {
		out.ClientId = v
	}
	if v, ok := obj[FieldSSOConnectionADClientSecret].(string); ok {
		out.ClientSecret = toPtr(v)
	}

	return out
}

func toOktaConnector(obj map[string]any) *sdk.CastaiSsoV1beta1Okta {
	if obj == nil {
		return nil
	}

	out := &sdk.CastaiSsoV1beta1Okta{}
	if v, ok := obj[FieldSSOConnectionOktaDomain].(string); ok {
		out.OktaDomain = v
	}
	if v, ok := obj[FieldSSOConnectionOktaClientID].(string); ok {
		out.ClientId = v
	}
	if v, ok := obj[FieldSSOConnectionOktaClientSecret].(string); ok {
		out.ClientSecret = toPtr(v)
	}

	return out
}

func toOIDCConnector(obj map[string]any) *sdk.CastaiSsoV1beta1OIDC {
	if obj == nil {
		return nil
	}

	out := &sdk.CastaiSsoV1beta1OIDC{}
	if v, ok := obj[FieldSSOConnectionOIDCIssuerURL].(string); ok {
		out.IssuerUrl = v
	}
	if v, ok := obj[FieldSSOConnectionOIDCClientID].(string); ok {
		out.ClientId = v
	}
	if v, ok := obj[FieldSSOConnectionOIDCClientSecret].(string); ok {
		out.ClientSecret = toPtr(v)
	}
	if v, ok := obj[FieldSSOConnectionOIDCType].(string); ok {
		out.Type = sdk.CastaiSsoV1beta1OIDCType(v)
	}

	return out
}

// setSSOConnectionSync calls the SetSync API and, on a false→true transition, stores the
// returned auth token and emits a warning so the user knows to save it.
func setSSOConnectionSync(ctx context.Context, client sdk.ClientWithResponsesInterface, data *schema.ResourceData, sync bool) diag.Diagnostics {
	resp, err := client.SSOAPISetSyncForSSOConnectionWithResponse(ctx, data.Id(), sdk.SSOAPISetSyncForSSOConnectionJSONRequestBody{
		Sync: sync,
	})
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("setting sso connection sync: %v", err)
	}

	if sync {
		// Store the token if one was returned.
		if resp.JSON200 != nil && resp.JSON200.Token != nil && resp.JSON200.Token.Token != nil {
			if err := data.Set(FieldSSOConnectionSyncAuthToken, *resp.JSON200.Token.Token); err != nil {
				return diag.Errorf("setting sync_auth_token: %v", err)
			}
		}
		return diag.Diagnostics{
			{
				Severity: diag.Warning,
				Summary:  "SSO sync auth token generated",
				Detail:   "A sync auth token was generated and stored in sync_auth_token. Retrieve it from the Terraform state or outputs now — it is only returned once when synchronization is enabled.",
			},
		}
	}

	return nil
}
