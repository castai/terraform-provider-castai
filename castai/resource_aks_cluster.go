package castai

import (
	"context"
	"errors"
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

func resourceAKSCluster() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceCastaiAKSClusterRead,
		CreateContext: resourceCastaiAKSClusterCreate,
		UpdateContext: resourceCastaiAKSClusterUpdate,
		DeleteContext: resourceCastaiClusterDelete,
		CustomizeDiff: clusterTokenDiff,
		Description:   "AKS cluster resource allows connecting an existing AKS cluster to CAST AI.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(6 * time.Minute),
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
			FieldClusterCredentialsId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "CAST AI internal credentials ID",
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

	if err := data.Set(FieldClusterCredentialsId, *resp.JSON200.CredentialsId); err != nil {
		return diag.FromErr(fmt.Errorf("setting credentials: %w", err))
	}

	if aks := resp.JSON200.Aks; aks != nil {
		if err := data.Set(FieldAKSClusterRegion, toString(aks.Region)); err != nil {
			return diag.FromErr(fmt.Errorf("setting region: %w", err))
		}
	}

	return nil
}

func resourceCastaiAKSClusterCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	req := sdk.ExternalClusterAPIRegisterClusterJSONRequestBody{
		Name: data.Get(FieldAKSClusterName).(string),
	}

	req.Aks = &sdk.ExternalclusterV1AKSClusterParams{
		Region:            toPtr(data.Get(FieldAKSClusterRegion).(string)),
		SubscriptionId:    toPtr(data.Get(FieldAKSClusterSubscriptionID).(string)),
		NodeResourceGroup: toPtr(data.Get(FieldAKSClusterNodeResourceGroup).(string)),
	}

	log.Printf("[INFO] Registering new external AKS cluster: %#v", req)

	resp, err := client.ExternalClusterAPIRegisterClusterWithResponse(ctx, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	clusterID := *resp.JSON200.Id
	tkn, err := createClusterToken(ctx, client, clusterID)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := data.Set(FieldClusterToken, tkn); err != nil {
		return diag.FromErr(fmt.Errorf("setting cluster token: %w", err))
	}
	data.SetId(clusterID)

	if err := updateAKSClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Cluster with id %q has been registered, don't forget to install castai-agent helm chart", data.Id())

	return resourceCastaiAKSClusterRead(ctx, data, meta)
}

func resourceCastaiAKSClusterUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if err := updateAKSClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}

	return resourceCastaiAKSClusterRead(ctx, data, meta)
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

	req := sdk.ExternalClusterAPIUpdateClusterJSONRequestBody{}

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
	var lastErr error
	if err = backoff.Retry(func() error {
		response, err := client.ExternalClusterAPIUpdateClusterWithResponse(ctx, data.Id(), req)
		lastErr = err
		return sdk.CheckOKResponse(response, err)
	}, b); err != nil {
		if errors.Is(err, context.DeadlineExceeded) && lastErr != nil {
			return fmt.Errorf("updating cluster configuration: %w: %v", err, lastErr)
		}
		return fmt.Errorf("updating cluster configuration: %w", err)
	}

	return nil
}
