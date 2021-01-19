package castai

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	CredentialsFieldName                      = "name"
	CredentialsFieldCloud                     = "cloud"
	CredentialsFieldGcp                       = "gcp"
	CredentialsFieldGcpServiceAccountJson     = "service_account_json"
	CredentialsFieldAws                       = "aws"
	CredentialsFieldAwsAccessKeyId            = "access_key_id"
	CredentialsFieldAwsSecretAccessKey        = "secret_access_key"
	CredentialsFieldAzure                     = "azure"
	CredentialsFieldAzureServicePrincipalJson = "service_principal_json"
)

func resourceCastaiClusterCredentials() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiCredentialsCreate,
		ReadContext:   resourceCastaiCloudCredentialsRead,
		UpdateContext: nil,
		DeleteContext: resourceCastaiCredentialsDelete,

		Schema: map[string]*schema.Schema{
			CredentialsFieldName: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			CredentialsFieldCloud: {
				Type:     schema.TypeString,
				Computed: true,
			},
			CredentialsFieldGcp: {
				Type:         schema.TypeList,
				Optional:     true,
				Sensitive:    true,
				ForceNew:     true,
				MaxItems:     1,
				ExactlyOneOf: []string{CredentialsFieldGcp, CredentialsFieldAws, CredentialsFieldAzure},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						CredentialsFieldGcpServiceAccountJson: {
							Type:             schema.TypeString,
							Required:         true,
							Sensitive:        true,
							ForceNew:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.All(validation.StringIsNotWhiteSpace, validation.StringIsJSON)),
						},
					},
				},
			},
			CredentialsFieldAws: {
				Type:      schema.TypeList,
				Optional:  true,
				Sensitive: true,
				ForceNew:  true,
				MaxItems:  1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						CredentialsFieldAwsAccessKeyId: {
							Type:             schema.TypeString,
							Required:         true,
							Sensitive:        true,
							ForceNew:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
						CredentialsFieldAwsSecretAccessKey: {
							Type:             schema.TypeString,
							Required:         true,
							Sensitive:        true,
							ForceNew:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
					},
				},
			},
			CredentialsFieldAzure: {
				Type:      schema.TypeList,
				Optional:  true,
				Sensitive: true,
				ForceNew:  true,
				MaxItems:  1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						CredentialsFieldAzureServicePrincipalJson: {
							Type:             schema.TypeString,
							Required:         true,
							Sensitive:        true,
							ForceNew:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.All(validation.StringIsNotWhiteSpace, validation.StringIsJSON)),
						},
					},
				},
			},
		},
	}
}

func resourceCastaiCredentialsCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	var cloud string
	var credentials string

	if _, ok := data.GetOk(CredentialsFieldGcp); ok {
		cloud = CredentialsFieldGcp
		credentials = data.Get(CredentialsFieldGcp + ".0." + CredentialsFieldGcpServiceAccountJson).(string)
	} else if _, ok := data.GetOk(CredentialsFieldAws); ok {
		credentialsJson, err := json.Marshal(struct {
			AccessKeyId     string `json:"accessKeyId"`
			SecretAccessKey string `json:"secretAccessKey"`
		}{
			AccessKeyId:     data.Get(CredentialsFieldAws + ".0." + CredentialsFieldAwsAccessKeyId).(string),
			SecretAccessKey: data.Get(CredentialsFieldAws + ".0." + CredentialsFieldAwsSecretAccessKey).(string),
		})
		if err != nil {
			return diag.Errorf("building aws credentials json, value=%v: %v", credentials, err)
		}

		cloud = CredentialsFieldAws
		credentials = string(credentialsJson)
	} else if _, ok := data.GetOk(CredentialsFieldAzure); ok {
		cloud = CredentialsFieldAzure
		credentials = data.Get(CredentialsFieldAzure + ".0." + CredentialsFieldAzureServicePrincipalJson).(string)
	} else {
		return diag.Errorf("none of supported cloud credentials were specified.")
	}

	response, err := client.CreateCloudCredentialsWithResponse(ctx, sdk.CreateCloudCredentialsJSONRequestBody{
		Name:        data.Get(CredentialsFieldName).(string),
		Cloud:       cloud,
		Credentials: credentials,
	})
	if checkErr := sdk.CheckCreateResponse(response, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	data.SetId(response.JSON201.Id)
	return nil
}

func resourceCastaiCloudCredentialsRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	response, err := client.GetCloudCredentialsWithResponse(ctx, sdk.CredentialsId(data.Id()))
	if err != nil {
		return diag.FromErr(err)
	} else if response.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Removing credentials %s from state because it no longer exists in CAST.AI", data.Id())
		data.SetId("")
		return nil
	}

	data.Set(CredentialsFieldName, response.JSON200.Name)
	data.Set(CredentialsFieldCloud, response.JSON200.Cloud)

	return nil
}

func resourceCastaiCredentialsDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if err := sdk.CheckDeleteResponse(client.DeleteCloudCredentialsWithResponse(ctx, sdk.CredentialsId(data.Id()))); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
