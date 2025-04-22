package castai

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

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
	FieldAKSHttpProxyConfig          = "http_proxy_config"
	FieldAKSHttpProxyDestination     = "http_proxy"
	FieldAKSHttpsProxyDestination    = "https_proxy"
	FieldAKSNoProxyDestinations      = "no_proxy"
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
			Delete: schema.DefaultTimeout(15 * time.Minute),
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
			FieldAKSHttpProxyConfig: {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "HTTP proxy configuration for CAST AI nodes and node components.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldAKSHttpProxyDestination: {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "Address to use for proxying HTTP requests.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
						FieldAKSHttpsProxyDestination: {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "Address to use for proxying HTTPS/TLS requests.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
						FieldAKSNoProxyDestinations: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "List of destinations that should not go through proxy.",
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
							},
						},
					},
				},
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

	if resp.JSON200.CredentialsId != nil && *resp.JSON200.CredentialsId != data.Get(FieldClusterCredentialsId) {
		log.Printf("[WARN] Drift in credentials from state (%q) and in API (%q), resetting client ID to force re-applying credentials from configuration",
			data.Get(FieldClusterCredentialsId), *resp.JSON200.CredentialsId)
		if err := data.Set(FieldAKSClusterClientID, "credentials-drift-detected-force-apply"); err != nil {
			return diag.FromErr(fmt.Errorf("setting client ID: %w", err))
		}
	}

	if err := data.Set(FieldClusterCredentialsId, *resp.JSON200.CredentialsId); err != nil {
		return diag.FromErr(fmt.Errorf("setting credentials: %w", err))
	}

	if aks := resp.JSON200.Aks; aks != nil {
		if err := data.Set(FieldAKSClusterRegion, toString(aks.Region)); err != nil {
			return diag.FromErr(fmt.Errorf("setting region: %w", err))
		}

		var proxyConfig []any
		if aks.HttpProxyConfig != nil {
			proxyConfig = []any{
				map[string]any{
					FieldAKSHttpProxyDestination:  lo.FromPtr(aks.HttpProxyConfig.HttpProxy),
					FieldAKSHttpsProxyDestination: lo.FromPtr(aks.HttpProxyConfig.HttpsProxy),
					FieldAKSNoProxyDestinations:   lo.FromPtr(aks.HttpProxyConfig.NoProxy),
				},
			}
		}

		if err := data.Set(FieldAKSHttpProxyConfig, proxyConfig); err != nil {
			return diag.FromErr(fmt.Errorf("setting http proxy config: %w", err))
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

func updateAKSClusterSettings(ctx context.Context, data *schema.ResourceData, client sdk.ClientWithResponsesInterface) error {
	if !data.HasChanges(
		FieldAKSClusterClientID,
		FieldAKSClusterClientSecret,
		FieldAKSClusterTenantID,
		FieldAKSClusterSubscriptionID,
		FieldClusterCredentialsId,
		FieldAKSHttpProxyConfig,
		FieldAKSHttpProxyDestination,
		FieldAKSHttpsProxyDestination,
		FieldAKSNoProxyDestinations,
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

	httpProxyConfigBlocks := data.Get(FieldAKSHttpProxyConfig).([]any)
	var reqHttpProxyConfig *sdk.ExternalclusterV1HttpProxyConfig
	if len(httpProxyConfigBlocks) > 0 {
		proxyConfig := httpProxyConfigBlocks[0].(map[string]any)
		noProxy := make([]string, 0)
		for _, r := range proxyConfig[FieldAKSNoProxyDestinations].([]any) {
			noProxy = append(noProxy, r.(string))
		}
		reqHttpProxyConfig = &sdk.ExternalclusterV1HttpProxyConfig{
			HttpProxy:  lo.ToPtr(proxyConfig[FieldAKSHttpProxyDestination].(string)),
			HttpsProxy: lo.ToPtr(proxyConfig[FieldAKSHttpsProxyDestination].(string)),
			NoProxy:    lo.ToPtr(noProxy),
		}
	}

	req.Credentials = &credentials
	req.Aks = &sdk.ExternalclusterV1UpdateAKSClusterParams{
		HttpProxyConfig: reqHttpProxyConfig,
	}

	return resourceCastaiClusterUpdate(ctx, client, data, &req)
}
