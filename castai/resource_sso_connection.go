package castai

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"golang.org/x/crypto/bcrypt"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/organization_management"
)

const (
	FieldSSOConnectionName                   = "name"
	FieldSSOConnectionEmailDomain            = "email_domain"
	FieldSSOConnectionAdditionalEmailDomains = "additional_email_domains"
	FieldSSOConnectionSynchronizeUserGroups  = "synchronize_user_groups"
	FieldSSOConnectionSyncAuthToken          = "sync_auth_token"
	FieldSSOConnectionOrganizationID         = "organization_id"
	FieldSSOConnectionBlockInvitations       = "block_invitations"
	FieldSSOConnectionEmailDomainValidation  = "email_domain_validation"

	FieldSSOConnectionEmailDomainValidationDomain           = "domain"
	FieldSSOConnectionEmailDomainValidationStatus           = "status"
	FieldSSOConnectionEmailDomainValidationVerified         = "verified"
	FieldSSOConnectionEmailDomainValidationLastCheckOutcome = "last_check_outcome"
	FieldSSOConnectionEmailDomainValidationTxtRecordName    = "txt_record_name"
	FieldSSOConnectionEmailDomainValidationTxtRecordValue   = "txt_record_value"

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
			FieldSSOConnectionOrganizationID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "CAST AI organization ID that owns this SSO connection. The organization settings endpoint is used to manage block_invitations.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldSSOConnectionBlockInvitations: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "When true, invitations sent by castai_organization_members are refused by the API. Managed only when explicitly set in configuration: a set value is enforced, an unset value mirrors the organization setting without producing a diff.",
			},
			FieldSSOConnectionEmailDomainValidation: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Verification state of the connection's claimed email domains. The verification record is returned by the API exactly once when it is issued: pending claims without a record get one issued automatically (via the regenerate-tokens endpoint) and it is persisted in state; already issued records are never rotated by the provider. For each pending domain, publish txt_record_name/txt_record_value as a DNS TXT record; CAST AI re-checks DNS automatically until the claim is verified.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldSSOConnectionEmailDomainValidationDomain: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The claimed email domain.",
						},
						FieldSSOConnectionEmailDomainValidationStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Verification status of the claim: STATUS_PENDING, STATUS_VERIFIED, STATUS_EXPIRED, STATUS_DENIED, STATUS_ERROR or STATUS_UNKNOWN.",
						},
						FieldSSOConnectionEmailDomainValidationVerified: {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "True when the domain claim is verified.",
						},
						FieldSSOConnectionEmailDomainValidationLastCheckOutcome: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Outcome of the latest verification check: SSO_EMAIL_DOMAIN_CHECK_OUTCOME_VERIFIED, SSO_EMAIL_DOMAIN_CHECK_OUTCOME_TXT_RECORD_NOT_FOUND, SSO_EMAIL_DOMAIN_CHECK_OUTCOME_SECRET_MISMATCH, SSO_EMAIL_DOMAIN_CHECK_OUTCOME_LOOKUP_FAILED or SSO_EMAIL_DOMAIN_CHECK_OUTCOME_UNSPECIFIED.",
						},
						FieldSSOConnectionEmailDomainValidationTxtRecordName: {
							Type:        schema.TypeString,
							Computed:    true,
							Sensitive:   true,
							Description: "Name of the DNS TXT record to publish for email-domain verification.",
						},
						FieldSSOConnectionEmailDomainValidationTxtRecordValue: {
							Type:        schema.TypeString,
							Computed:    true,
							Sensitive:   true,
							Description: "Value of the DNS TXT record to publish for email-domain verification. Carries the verification secret.",
						},
					},
				},
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
							// The backend stores secrets as bcrypt hashes and returns them base64-encoded.
							// On reads, oldValue is base64(bcrypt(secret)); newValue is the plaintext from config.
							// bcrypt.CompareHashAndPassword is CPU-intensive by design; this cost is accepted
							// on every plan, consistent with the aad and okta connector implementations.
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

	// block_invitations is an org-wide setting managed through the organization-management
	// API. Only touch it when the user explicitly wrote the attribute in configuration:
	// d.Get cannot distinguish an unset Optional+Computed attribute from an explicit value.
	var settingsDiags diag.Diagnostics
	if blockInvitationsInConfig(data) {
		if err := updateSSOBlockInvitations(ctx, meta, data, data.Get(FieldSSOConnectionBlockInvitations).(bool)); err != nil {
			// A failure here must not taint the freshly created connection: report a warning
			// instead. The next plan shows the drift between the org setting and the config
			// and Update retries the settings call.
			settingsDiags = append(settingsDiags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "failed updating organization settings",
				Detail:   fmt.Sprintf("block_invitations may not be applied: %v", err),
			})
		}
	}

	returnDiags := append(settingsDiags, resourceCastaiSSOConnectionRead(ctx, data, meta)...)

	return append(syncDiags, returnDiags...)
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

	if err := data.Set(FieldSSOConnectionOrganizationID, data.Get(FieldSSOConnectionOrganizationID)); err != nil {
		return diag.Errorf("setting organization_id: %v", err)
	}

	// Surface the DNS verification state of the claimed email domains, and nudge the user
	// while any claim is still pending verification.
	validationDiags := setEmailDomainValidation(ctx, meta, data, connection.Connection.EmailDomains)
	if validationDiags.HasError() {
		return validationDiags
	}

	// block_invitations mirrors the org-wide settings endpoint. Without organization_id
	// (legacy state, unit tests) there is nothing to query — keep the prior value.
	orgID := data.Get(FieldSSOConnectionOrganizationID).(string)
	if orgID == "" {
		return validationDiags
	}

	orgClient := meta.(*ProviderConfig).organizationManagementClient
	settingsResp, err := orgClient.SettingsAPIGetSettingsWithResponse(ctx, orgID)
	if err := sdk.CheckOKResponse(settingsResp, err); err != nil {
		// A read failure must not break the refresh of an otherwise healthy connection;
		// surface a warning and keep the last known value instead.
		return append(validationDiags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "failed retrieving organization settings",
			Detail:   fmt.Sprintf("keeping the last known block_invitations value in state: %v", err),
		})
	}

	var blocked bool
	if settingsResp.JSON200 != nil && settingsResp.JSON200.BlockInvitations != nil {
		blocked = *settingsResp.JSON200.BlockInvitations
	}
	if err := data.Set(FieldSSOConnectionBlockInvitations, blocked); err != nil {
		return append(validationDiags, diag.Errorf("setting block_invitations: %v", err)...)
	}

	return validationDiags
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
		FieldSSOConnectionBlockInvitations,
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

	// block_invitations is managed through the organization-management settings API, kept
	// outside the SSO update above so changing only the flag never calls the SSO update API
	// and changing only SSO fields never calls the settings API.
	var settingsDiags diag.Diagnostics
	if blockInvitationsInConfig(data) && data.HasChange(FieldSSOConnectionBlockInvitations) {
		if err := updateSSOBlockInvitations(ctx, meta, data, data.Get(FieldSSOConnectionBlockInvitations).(bool)); err != nil {
			return append(syncDiags, diag.FromErr(err)...)
		}
	}

	returnDiags := append(settingsDiags, resourceCastaiSSOConnectionRead(ctx, data, meta)...)

	return append(syncDiags, returnDiags...)
}

func resourceCastaiSSOConnectionDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// With the SSO connection gone, invitations must flow again: reset the org-wide
	// block_invitations setting before the connection itself is deleted. Failing here
	// keeps the resource in state so the apply can be retried.
	if orgID := data.Get(FieldSSOConnectionOrganizationID).(string); orgID != "" {
		if err := updateSSOBlockInvitations(ctx, meta, data, false); err != nil {
			return diag.FromErr(err)
		}
	}

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
	if countSSOConnectors(rd.Get) != 1 {
		return errors.New("only 1 connector can be configured")
	}
	return nil
}

// countSSOConnectors counts how many connector blocks are non-empty.
// Accepts a getter func so the logic can be tested independently of schema.ResourceDiff.
func countSSOConnectors(get func(string) interface{}) int {
	n := 0
	for _, field := range []string{FieldSSOConnectionAAD, FieldSSOConnectionOkta, FieldSSOConnectionOIDC} {
		if v, ok := get(field).([]any); ok && len(v) > 0 {
			n++
		}
	}
	return n
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

// updateSSOBlockInvitations patches the org-wide settings endpoint with only the
// blockInvitations key so no other organization setting is touched.
func updateSSOBlockInvitations(ctx context.Context, meta interface{}, data *schema.ResourceData, block bool) error {
	orgID := data.Get(FieldSSOConnectionOrganizationID).(string)
	if orgID == "" {
		return fmt.Errorf("organization_id is required to manage block_invitations")
	}

	orgClient := meta.(*ProviderConfig).organizationManagementClient
	resp, err := orgClient.SettingsAPIUpdateSettingsWithResponse(ctx, orgID, organization_management.SettingsAPIUpdateSettingsJSONRequestBody{
		BlockInvitations: toPtr(block),
	})
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return fmt.Errorf("updating organization settings (block_invitations=%t): %w", block, err)
	}

	return nil
}

// setEmailDomainValidation stores the verification state of the claimed email domains in
// state and emits one warning per domain that is not verified yet.
//
// The verification secret is never stored in plaintext server-side and the connection GET
// never returns it: the record to publish is returned exactly once, when it is issued via
// the regenerate-tokens endpoint (the same flow the console uses). Therefore:
//
//   - records already stored in state are preserved (write-back, like sync_auth_token);
//   - pending claims without any record (first refresh after create, new domains, lost
//     state) get one issued via the regenerate endpoint — for a claim with no record yet
//     this is the first issuance, so nothing the user already published is rotated;
//   - if issuance fails, the claim stays without a record and the warning says so.
//
// Pending-domain warnings include the full record (name and value) so the user can act on
// the message alone; the value is only obtainable until verified and is shown by the
// console as well.
func setEmailDomainValidation(ctx context.Context, meta interface{}, data *schema.ResourceData, domains *[]sdk.CastaiSsoV1beta1SSOEmailDomainValidation) diag.Diagnostics {
	parsed := parseEmailDomainValidation(domains)

	// Preserve one-time records already stored in state.
	previous := previousEmailDomainRecords(data)
	for _, entry := range parsed {
		name := entry[FieldSSOConnectionEmailDomainValidationTxtRecordName].(string)
		if name != "" {
			continue
		}
		domain := entry[FieldSSOConnectionEmailDomainValidationDomain].(string)
		if rec, ok := previous[domain]; ok {
			entry[FieldSSOConnectionEmailDomainValidationTxtRecordName] = rec[FieldSSOConnectionEmailDomainValidationTxtRecordName]
			entry[FieldSSOConnectionEmailDomainValidationTxtRecordValue] = rec[FieldSSOConnectionEmailDomainValidationTxtRecordValue]
		}
	}

	// Issue records for pending claims that still have none.
	var needRecord []string
	for _, entry := range parsed {
		if entry[FieldSSOConnectionEmailDomainValidationVerified].(bool) {
			continue
		}
		if entry[FieldSSOConnectionEmailDomainValidationTxtRecordName].(string) == "" {
			needRecord = append(needRecord, entry[FieldSSOConnectionEmailDomainValidationDomain].(string))
		}
	}

	var diags diag.Diagnostics
	if len(needRecord) > 0 {
		issued, err := issueEmailDomainRecords(ctx, meta, data, needRecord)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "failed issuing SSO email-domain verification records",
				Detail:   fmt.Sprintf("domains %v: %v", needRecord, err),
			})
		} else {
			mergeIssuedRecords(parsed, issued)
		}
	}

	if err := data.Set(FieldSSOConnectionEmailDomainValidation, parsed); err != nil {
		return diag.Errorf("setting email_domain_validation: %v", err)
	}

	for _, entry := range parsed {
		if entry[FieldSSOConnectionEmailDomainValidationVerified].(bool) {
			continue
		}
		domain := entry[FieldSSOConnectionEmailDomainValidationDomain].(string)
		status := entry[FieldSSOConnectionEmailDomainValidationStatus].(string)
		outcome := entry[FieldSSOConnectionEmailDomainValidationLastCheckOutcome].(string)
		name := entry[FieldSSOConnectionEmailDomainValidationTxtRecordName].(string)
		if name == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "SSO email domain is not verified yet",
				Detail: fmt.Sprintf("Email domain %q has verification status %s (last check outcome: %s). The DNS verification record could not be issued automatically: the record is returned exactly once when issued, and no record is available for this claim. Publish the record shown in the CAST AI console, or remove and recreate the connection to issue a fresh record; CAST AI re-checks DNS automatically once it is published.",
					domain, status, outcome),
			})
			continue
		}
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "SSO email domain is not verified yet",
			Detail: fmt.Sprintf("Publish the DNS TXT record %q with the value %q. CAST AI re-checks DNS automatically until the claim is verified. (Email domain %q, status: %s, last check outcome: %s.)",
				name,
				entry[FieldSSOConnectionEmailDomainValidationTxtRecordValue],
				domain, status, outcome),
		})
	}

	return diags
}

// previousEmailDomainRecords indexes the email_domain_validation records currently stored in
// state by domain, so one-time verification records survive refreshes.
func previousEmailDomainRecords(data *schema.ResourceData) map[string]map[string]any {
	previous := map[string]map[string]any{}
	for _, v := range data.Get(FieldSSOConnectionEmailDomainValidation).([]any) {
		entry, ok := v.(map[string]any)
		if !ok {
			continue
		}
		domain, _ := entry[FieldSSOConnectionEmailDomainValidationDomain].(string)
		name, _ := entry[FieldSSOConnectionEmailDomainValidationTxtRecordName].(string)
		if domain == "" || name == "" {
			continue
		}
		previous[domain] = entry
	}
	return previous
}

// issueEmailDomainRecords calls the regenerate-tokens endpoint for the given pending domains
// and returns the updated validation entries containing the records to publish. The record
// is returned exactly once per issuance.
func issueEmailDomainRecords(ctx context.Context, meta interface{}, data *schema.ResourceData, domains []string) ([]organization_management.SSOEmailDomainValidation, error) {
	orgID := data.Get(FieldSSOConnectionOrganizationID).(string)
	if orgID == "" {
		return nil, fmt.Errorf("organization_id is required to issue email-domain verification records")
	}

	orgClient := meta.(*ProviderConfig).organizationManagementClient
	resp, err := orgClient.SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensWithResponse(ctx, orgID, organization_management.SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensJSONRequestBody{
		OrganizationId: orgID,
		Domains:        domains,
	})
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return nil, fmt.Errorf("issuing verification records: %w", err)
	}
	if resp.JSON200 == nil || resp.JSON200.EmailDomains == nil {
		return nil, fmt.Errorf("issuing verification records: empty response")
	}

	return *resp.JSON200.EmailDomains, nil
}

// mergeIssuedRecords copies the newly issued verification records into the parsed entries.
func mergeIssuedRecords(parsed []map[string]any, issued []organization_management.SSOEmailDomainValidation) {
	for _, issuedDomain := range issued {
		if issuedDomain.TxtRecord == nil {
			continue
		}
		for _, entry := range parsed {
			if entry[FieldSSOConnectionEmailDomainValidationDomain] != derefString(issuedDomain.Domain) {
				continue
			}
			entry[FieldSSOConnectionEmailDomainValidationTxtRecordName] = derefString(issuedDomain.TxtRecord.Name)
			entry[FieldSSOConnectionEmailDomainValidationTxtRecordValue] = derefString(issuedDomain.TxtRecord.Value)
			entry[FieldSSOConnectionEmailDomainValidationStatus] = derefString(issuedDomain.Status)
			entry[FieldSSOConnectionEmailDomainValidationVerified] = issuedDomain.Status != nil && *issuedDomain.Status == organization_management.SSOEmailDomainValidationStatusSSOEMAILDOMAINSTATUSVERIFIED
			entry[FieldSSOConnectionEmailDomainValidationLastCheckOutcome] = derefString(issuedDomain.LastCheckOutcome)
		}
	}
}

// parseEmailDomainValidation flattens the API validation entries into schema-compatible maps,
// sorted by domain so the computed list has a stable order across refreshes.
func parseEmailDomainValidation(domains *[]sdk.CastaiSsoV1beta1SSOEmailDomainValidation) []map[string]any {
	parsed := []map[string]any{}
	if domains == nil {
		return parsed
	}

	for _, d := range *domains {
		entry := map[string]any{
			FieldSSOConnectionEmailDomainValidationDomain:           derefString(d.Domain),
			FieldSSOConnectionEmailDomainValidationStatus:           derefString(d.Status),
			FieldSSOConnectionEmailDomainValidationVerified:         d.Status != nil && *d.Status == sdk.CastaiSsoV1beta1SSOEmailDomainValidationStatusSTATUSVERIFIED,
			FieldSSOConnectionEmailDomainValidationLastCheckOutcome: derefString(d.LastCheckOutcome),
			FieldSSOConnectionEmailDomainValidationTxtRecordName:    "",
			FieldSSOConnectionEmailDomainValidationTxtRecordValue:   "",
		}
		if d.TxtRecord != nil {
			entry[FieldSSOConnectionEmailDomainValidationTxtRecordName] = derefString(d.TxtRecord.Name)
			entry[FieldSSOConnectionEmailDomainValidationTxtRecordValue] = derefString(d.TxtRecord.Value)
		}
		parsed = append(parsed, entry)
	}

	slices.SortFunc(parsed, func(a, b map[string]any) int {
		return strings.Compare(
			a[FieldSSOConnectionEmailDomainValidationDomain].(string),
			b[FieldSSOConnectionEmailDomainValidationDomain].(string),
		)
	})

	return parsed
}

// stringValue dereferences an optional string-ish value into a plain string.
func derefString[T ~string](v *T) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

// blockInvitationsInConfig reports whether block_invitations is explicitly written in the
// configuration. d.Get cannot distinguish an unset Optional+Computed attribute from an
// explicit value, so the raw config is inspected instead. A missing raw config (unit tests
// built without one, import paths) counts as unset.
func blockInvitationsInConfig(data *schema.ResourceData) bool {
	raw := data.GetRawConfig()
	if raw.IsNull() || !raw.Type().IsObjectType() || !raw.Type().HasAttribute(FieldSSOConnectionBlockInvitations) {
		return false
	}
	v := raw.GetAttr(FieldSSOConnectionBlockInvitations)
	return !v.IsNull() && v.IsKnown()
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
