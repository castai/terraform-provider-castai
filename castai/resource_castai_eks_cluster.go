package castai

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldEKSClusterName               = "name"
	FieldEKSClusterAccountId          = "account_id"
	FieldEKSClusterRegion             = "region"
	FieldEKSClusterAccessKeyId        = "access_key_id"
	FieldEKSClusterSecretAccessKey    = "secret_access_key"
	FieldEKSClusterInstanceProfileArn = "instance_profile_arn"
	FieldEKSClusterAgentToken         = "agent_token"
	FieldEKSClusterToken              = "cluster_token"
	FieldEKSClusterCredentialsId      = "credentials_id"
)

func resourceCastaiEKSCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiEKSClusterCreate,
		ReadContext:   resourceCastaiEKSClusterRead,
		UpdateContext: resourceCastaiEKSClusterUpdate,
		DeleteContext: resourceCastaiEKSClusterDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(2 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldEKSClusterName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldEKSClusterAccountId: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldEKSClusterRegion: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldEKSClusterAccessKeyId: {
				Type:             schema.TypeString,
				Sensitive:        true,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldEKSClusterSecretAccessKey: {
				Type:             schema.TypeString,
				Sensitive:        true,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldEKSClusterInstanceProfileArn: {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldEKSClusterCredentialsId: {
				Type:     schema.TypeString,
				Computed: true,
			},
			FieldEKSClusterAgentToken: {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "agent_token is deprecated, use cluster_token instead",
				Sensitive:  true,
			},
			FieldEKSClusterToken: {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, i interface{}) error {
			_, accessKeyIdProvided := diff.GetOk(FieldEKSClusterAccessKeyId)
			_, secretAccessKeyProvided := diff.GetOk(FieldEKSClusterSecretAccessKey)

			if accessKeyIdProvided != secretAccessKeyProvided {
				return fmt.Errorf("when used `%s` and `%s` must be both specified", FieldEKSClusterAccessKeyId, FieldEKSClusterSecretAccessKey)
			}

			return nil
		},
	}
}

func resourceCastaiEKSClusterCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	req := sdk.ExternalClusterAPIRegisterClusterJSONRequestBody{
		Name: data.Get(FieldEKSClusterName).(string),
	}

	req.Eks = &sdk.ExternalclusterV1EKSClusterParams{
		AccountId:   toStringPtr(data.Get(FieldEKSClusterAccountId).(string)),
		Region:      toStringPtr(data.Get(FieldEKSClusterRegion).(string)),
		ClusterName: toStringPtr(data.Get(FieldEKSClusterName).(string)),
	}

	log.Printf("[INFO] Registering new external cluster: %#v", req)

	response, err := client.ExternalClusterAPIRegisterClusterWithResponse(ctx, req)
	if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	data.SetId(*response.JSON200.Id)

	log.Printf("[INFO] Cluster with id %q has been registered, don't forget to install castai-agent helm chart", data.Id())

	if err := updateClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}

	return resourceCastaiEKSClusterRead(ctx, data, meta)
}

func resourceCastaiEKSClusterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if data.Id() == "" {
		log.Printf("[INFO] id is null not fetching anything.")
		return nil
	}

	log.Printf("[INFO] Getting cluster information.")

	resp, err := client.ExternalClusterAPIGetClusterWithResponse(ctx, data.Id())
	if err != nil {
		return diag.FromErr(err)
	} else if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Removing cluster %s from state because it no longer exists in CAST.AI", data.Id())
		data.SetId("")
		return nil
	}

	data.Set(FieldEKSClusterCredentialsId, *resp.JSON200.CredentialsId)

	if resp.JSON200.Eks != nil {
		data.Set(FieldEKSClusterAccountId, *resp.JSON200.Eks.AccountId)
		data.Set(FieldEKSClusterRegion, *resp.JSON200.Eks.Region)
		data.Set(FieldEKSClusterName, *resp.JSON200.Eks.ClusterName)
		data.Set(FieldEKSClusterInstanceProfileArn, *resp.JSON200.Eks.InstanceProfileArn)
	}

	if _, ok := data.GetOk(FieldEKSClusterAgentToken); !ok {
		tkn, err := retrieveAgentToken(ctx, client)
		if err != nil {
			return diag.FromErr(err)
		}
		data.Set(FieldEKSClusterAgentToken, tkn)
	}

	// Create token only if missing.
	if _, ok := data.GetOk(FieldEKSClusterToken); !ok {
		tkn, err := createClusterToken(ctx, client, data.Id())
		if err != nil {
			return diag.FromErr(err)
		}
		data.Set(FieldEKSClusterToken, tkn)
	}

	return nil
}

func resourceCastaiEKSClusterUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if err := updateClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}

	return resourceCastaiEKSClusterRead(ctx, data, meta)
}

func resourceCastaiEKSClusterDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterId := data.Id()

	log.Printf("[INFO] Checking current status of the cluster.")

	err := resource.RetryContext(ctx, data.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		clusterResponse, err := client.ExternalClusterAPIGetClusterWithResponse(ctx, clusterId)
		if checkErr := sdk.CheckOKResponse(clusterResponse, err); checkErr != nil {
			return resource.NonRetryableError(err)
		}

		clusterStatus := *clusterResponse.JSON200.Status
		agentStatus := *clusterResponse.JSON200.AgentStatus
		log.Printf("[INFO] Current cluster status=%s, agent_status=%s", clusterStatus, agentStatus)

		if clusterStatus == sdk.ClusterStatusDeleted || clusterStatus == sdk.ClusterStatusArchived {
			log.Printf("[INFO] Cluster is already deleted, removing from state.")
			data.SetId("")
			return nil
		}

		if agentStatus == sdk.ClusterAgentStatusDisconnecting {
			return resource.RetryableError(fmt.Errorf("agent is disconnecting"))
		}

		if clusterStatus == sdk.ClusterStatusDeleting {
			return resource.RetryableError(fmt.Errorf("cluster is deleting"))
		}

		if clusterResponse.JSON200.CredentialsId != nil && agentStatus != sdk.ClusterAgentStatusDisconnected {
			log.Printf("[INFO] Disconnecting cluster.")

			response, err := client.ExternalClusterAPIDisconnectClusterWithResponse(ctx, clusterId, sdk.ExternalClusterAPIDisconnectClusterJSONRequestBody{})
			if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
				return resource.NonRetryableError(err)
			}

			return resource.RetryableError(fmt.Errorf("triggered agent disconnection"))
		}

		if agentStatus == sdk.ClusterAgentStatusDisconnected && clusterStatus != sdk.ClusterStatusDeleted {
			log.Printf("[INFO] Deleting cluster.")

			if err := sdk.CheckResponseNoContent(client.ExternalClusterAPIDeleteClusterWithResponse(ctx, clusterId)); err != nil {
				return resource.NonRetryableError(err)
			}

			return resource.RetryableError(fmt.Errorf("triggered cluster deletion"))

		}

		return resource.RetryableError(fmt.Errorf("retrying"))
	})

	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func updateClusterSettings(ctx context.Context, data *schema.ResourceData, client *sdk.ClientWithResponses) error {
	if !data.HasChanges(FieldEKSClusterAccessKeyId, FieldEKSClusterSecretAccessKey, FieldEKSClusterInstanceProfileArn) {
		log.Printf("[INFO] Nothing to update in cluster setttings.")
		return nil
	}

	log.Printf("[INFO] Updating cluster settings.")

	req := sdk.ExternalClusterAPIUpdateClusterJSONRequestBody{
		Eks: &sdk.ExternalclusterV1UpdateEKSClusterParams{},
	}

	accessKeyId, accessKeyIdProvided := data.GetOk(FieldEKSClusterAccessKeyId)
	secretAccessKey, secretAccessKeyProvided := data.GetOk(FieldEKSClusterSecretAccessKey)
	if accessKeyIdProvided && secretAccessKeyProvided {
		credentials, err := sdk.ToCloudCredentialsAWS(accessKeyId.(string), secretAccessKey.(string))
		if err != nil {
			return fmt.Errorf("marshaling credentials for cluster access: %w", err)
		}

		req.Credentials = &credentials
	}

	if arn, ok := data.GetOk(FieldEKSClusterInstanceProfileArn); ok {
		req.Eks.InstanceProfileArn = toStringPtr(arn.(string))
	}

	response, err := client.ExternalClusterAPIUpdateClusterWithResponse(ctx, data.Id(), req)
	if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
		return fmt.Errorf("updating cluster settings: %w", checkErr)
	}

	return nil
}

// Deprecated. Remove with agent_token.
func retrieveAgentToken(ctx context.Context, client *sdk.ClientWithResponses) (string, error) {
	response, err := client.GetAgentInstallScriptWithResponse(ctx, &sdk.GetAgentInstallScriptParams{})
	if err != nil {
		return "", fmt.Errorf("retrieving agent install script: %w", err)
	}

	// at the moment, agent registration token only appears in `curl agent manifests` snippet and is extracted from there.
	return strings.Split(strings.TrimPrefix(string(response.Body), `curl -H "Authorization: Token `), `"`)[0], nil
}

func createClusterToken(ctx context.Context, client *sdk.ClientWithResponses, clusterID string) (string, error) {
	resp, err := client.ExternalClusterAPICreateClusterTokenWithResponse(ctx, clusterID)
	if err != nil {
		return "", fmt.Errorf("creating cluster token: %w", err)
	}

	return *resp.JSON200.Token, nil
}
