package castai

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldAKSClusterName              = "name"
	FieldAKSClusterRegion            = "region"
	FieldAKSClusterSubscriptionID    = "subscription_id"
	FieldAKSClusterNodeResourceGroup = "node_resource_group"
	FieldAKSClusterClientID          = "client_id"
	FieldAKSClusterClientSecret      = "client_secret"
	FieldAKSClusterTenantID          = "tenant_id"
)

func resourceCastaiAKSCluster() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceCastaiAKSClusterRead,
		CreateContext: resourceCastaiAKSClusterCreate,
		UpdateContext: resourceCastaiAKSClusterUpdate,
		DeleteContext: resourceCastaiPublicCloudClusterDelete,
		Description:   "AKS cluster resource allows connecting an existing EKS cluster to CAST AI.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(2 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldAKSClusterName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "AKS cluster name.",
			},
			FieldAKSClusterRegion: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "AKS cluster region.",
			},
			FieldAKSClusterSubscriptionID: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "ID of the Azure subscription.",
			},
			FieldAKSClusterNodeResourceGroup: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "Azure resource group in which nodes are and will be created.",
			},
			FieldAKSClusterTenantID: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "Azure AD tenant ID from the used subscription.",
			},
			FieldAKSClusterClientID: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "Azure AD application ID that is created and used by CAST AI.",
			},
			FieldAKSClusterClientSecret: {
				Type:             schema.TypeString,
				Required:         true,
				Sensitive:        true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "Azure AD application password that will be used by CAST AI.",
			},
			FieldClusterToken: {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "CAST AI cluster token.",
			},
			FieldDeleteNodesOnDisconnect: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Should CAST AI remove nodes managed by CAST.AI on disconnect.",
			},
		},
	}
}

func resourceCastaiAKSClusterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if data.Id() == "" {
		log.Printf("[INFO] id is null not fetching anything.")
		return nil
	}

	log.Printf("[INFO] Getting cluster information.")

	resp, err := fetchClusterData(ctx, client, data.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if resp == nil {
		data.SetId("")
		return nil
	}

	data.Set(FieldClusterCredentialsId, *resp.JSON200.CredentialsId)

	if aks := resp.JSON200.Aks; aks != nil {
		data.Set(FieldAKSClusterRegion, toString(aks.Region))
	}
	clusterID := *resp.JSON200.Id

	if _, ok := data.GetOk(FieldClusterToken); !ok {
		tkn, err := createClusterToken(ctx, client, clusterID)
		if err != nil {
			return diag.FromErr(err)
		}
		data.Set(FieldClusterToken, tkn)
	}

	return nil
}

func resourceCastaiAKSClusterCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	req := sdk.ExternalClusterAPIRegisterClusterJSONRequestBody{
		Name: data.Get(FieldAKSClusterName).(string),
	}

	req.Aks = &sdk.ExternalclusterV1AKSClusterParams{
		Region:            toStringPtr(data.Get(FieldAKSClusterRegion).(string)),
		SubscriptionId:    toStringPtr(data.Get(FieldAKSClusterSubscriptionID).(string)),
		NodeResourceGroup: toStringPtr(data.Get(FieldAKSClusterNodeResourceGroup).(string)),
	}

	log.Printf("[INFO] Registering new external AKS cluster: %#v", req)

	resp, err := client.ExternalClusterAPIRegisterClusterWithResponse(ctx, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	clusterID := *resp.JSON200.Id
	data.SetId(clusterID)

	if err := updateAKSClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}

	return resourceCastaiAKSClusterRead(ctx, data, meta)
}

func resourceCastaiAKSClusterUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if err := updateAKSClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}

	return resourceCastaiEKSClusterRead(ctx, data, meta)
}

func updateAKSClusterSettings(ctx context.Context, data *schema.ResourceData, client *sdk.ClientWithResponses) error {
	if !data.HasChanges(
		FieldAKSClusterClientID,
		FieldAKSClusterClientSecret,
		FieldAKSClusterTenantID,
		FieldAKSClusterSubscriptionID,
	) {
		log.Printf("[INFO] Nothing to update in cluster setttings.")
		return nil
	}

	log.Printf("[INFO] Updating cluster settings.")

	req := sdk.ExternalClusterAPIUpdateClusterJSONRequestBody{
		Aks: &sdk.ExternalclusterV1UpdateAKSClusterParams{},
	}

	clientID := data.Get(FieldAKSClusterClientID).(string)
	tenantID := data.Get(FieldAKSClusterTenantID).(string)
	clientSecret := data.Get(FieldAKSClusterClientSecret).(string)
	subscriptionID := data.Get(FieldAKSClusterSubscriptionID).(string)

	credentials, err := sdk.ToCloudCredentialsAzure(clientID, clientSecret, tenantID, subscriptionID)
	if err != nil {
		return err
	}

	req.Credentials = &credentials

	// Retries are required for newly created IAM resources to initialise on Azure side.
	b := backoff.WithContext(backoff.WithMaxRetries(backoff.NewConstantBackOff(10*time.Second), 30), ctx)
	if err = backoff.Retry(func() error {
		response, err := client.ExternalClusterAPIUpdateClusterWithResponse(ctx, data.Id(), req)
		return sdk.CheckOKResponse(response, err)
	}, b); err != nil {
		return fmt.Errorf("submitting AKS credentials to CAST AI: %w", err)
	}

	return nil
}
