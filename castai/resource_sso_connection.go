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

	FieldSSOConnectionAAD            = "aad"
	FieldSSOConnectionADDomain       = "ad_domain"
	FieldSSOConnectionADClientID     = "client_id"
	FieldSSOConnectionADClientSecret = "client_secret"

	FieldSSOConnectionOkta             = "okta"
	FieldSSOConnectionOktaDomain       = "okta_domain"
	FieldSSOConnectionOktaClientID     = "client_id"
	FieldSSOConnectionOktaClientSecret = "client_secret"
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

	resp, err := client.SSOAPICreateSSOConnectionWithResponse(ctx, req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("creating sso connection: %v", err)
	}

	if err := checkSSOStatus(resp.JSON200); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(*resp.JSON200.Id)

	return resourceCastaiSSOConnectionRead(ctx, data, meta)
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

	if err := data.Set(FieldSSOConnectionName, connection.Name); err != nil {
		return diag.Errorf("setting connection name: %v", err)
	}
	if err := data.Set(FieldSSOConnectionEmailDomain, connection.EmailDomain); err != nil {
		return diag.Errorf("setting email domain: %v", err)
	}
	if err := data.Set(FieldSSOConnectionAdditionalEmailDomains, connection.AdditionalEmailDomains); err != nil {
		return diag.Errorf("setting additional email domains: %v", err)
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

	resp, err := client.SSOAPIUpdateSSOConnectionWithResponse(ctx, data.Id(), req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("updating sso connection: %v", err)
	}

	if err := checkSSOStatus(resp.JSON200); err != nil {
		return diag.FromErr(err)
	}

	return resourceCastaiSSOConnectionRead(ctx, data, meta)
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

	if *input.Status == sdk.STATUSACTIVE {
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
