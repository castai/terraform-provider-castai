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
	FieldGKEClusterName                    = "name"
	FieldGKEClusterProjectId               = "project_id"
	FieldGKEClusterLocation                = "location"
	FieldGKEClusterToken                   = "cluster_token"
	FieldGKEClusterDeleteNodesOnDisconnect = "delete_nodes_on_disconnect"
	FieldGKEClusterSSHPublicKey            = "ssh_public_key"
	FieldGKEClusterCredentialsId           = "credentials_id"
	FieldGKEClusterCredentials             = "credentials_json"
)

func resourceCastaiGKECluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiGKEClusterCreate,
		ReadContext:   resourceCastaiGKEClusterRead,
		UpdateContext: resourceCastaiGKEClusterUpdate,
		DeleteContext: resourceCastaiGKEClusterDelete,
		Description:   "GKE cluster resource allows connecting an existing GEK cluster to CAST AI.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(6 * time.Minute), // Cluster action timeout is 5 minutes.
		},

		Schema: map[string]*schema.Schema{
			FieldGKEClusterName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "GKE cluster name",
			},
			FieldGKEClusterCredentialsId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "CAST AI credentials id for cluster",
			},
			FieldGKEClusterProjectId: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "GCP project id",
			},
			FieldGKEClusterLocation: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "GCP cluster zone in case of zonal or region in case of regional cluster",
			},
			FieldGKEClusterToken: {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "CAST.AI agent cluster token",
			},
			FieldGKEClusterCredentials: {
				Type:             schema.TypeString,
				Sensitive:        true,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "GCP credentials.json from ServiceAccount with credentials for CAST AI",
			},
			FieldGKEClusterDeleteNodesOnDisconnect: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Should CAST AI remove nodes managed by CAST.AI on disconnect",
			},
			FieldGKEClusterSSHPublicKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "SSHPublicKey for nodes",
			},
		},
	}
}

func resourceCastaiGKEClusterCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	req := sdk.ExternalClusterAPIRegisterClusterJSONRequestBody{
		Name: data.Get(FieldGKEClusterName).(string),
	}

	location := data.Get(FieldGKEClusterLocation).(string)
	region := location
	// Check if location is zone or location.
	if strings.Count(location, "-") > 1 {
		// region "europe-central2"
		// zone "europe-central2-a"
		regionParts := strings.Split(location, "-")
		regionParts = regionParts[:2]
		region = strings.Join(regionParts, "-")
	}

	req.Gke = &sdk.ExternalclusterV1GKEClusterParams{
		ProjectId:   toStringPtr(data.Get(FieldGKEClusterProjectId).(string)),
		Region:      &region,
		Location:    &location,
		ClusterName: toStringPtr(data.Get(FieldGKEClusterName).(string)),
	}

	log.Printf("[INFO] Registering new external cluster: %#v", req)

	resp, err := client.ExternalClusterAPIRegisterClusterWithResponse(ctx, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	clusterID := *resp.JSON200.Id
	tkn, err := createClusterToken(ctx, client, clusterID)
	if err != nil {
		return diag.FromErr(err)
	}
	data.Set(FieldGKEClusterToken, tkn)
	data.SetId(clusterID)

	if err := updateGKEClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Cluster with id %q has been registered, don't forget to install castai-agent helm chart", data.Id())

	return resourceCastaiGKEClusterRead(ctx, data, meta)
}

func resourceCastaiGKEClusterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		log.Printf("[WARN] Removing cluster %s from state because it no longer exists in CAST AI", data.Id())
		data.SetId("")
		return nil
	}

	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	data.Set(FieldGKEClusterCredentialsId, toString(resp.JSON200.CredentialsId))
	if resp.JSON200.SshPublicKey != nil {
		data.Set(FieldGKEClusterSSHPublicKey, toString(resp.JSON200.SshPublicKey))
	}
	if GKE := resp.JSON200.Gke; GKE != nil {
		data.Set(FieldGKEClusterProjectId, toString(GKE.ProjectId))
		data.Set(FieldGKEClusterLocation, toString(GKE.Location))
		data.Set(FieldGKEClusterName, toString(GKE.ClusterName))
	}
	clusterID := *resp.JSON200.Id

	if _, ok := data.GetOk(FieldGKEClusterToken); !ok {
		tkn, err := createClusterToken(ctx, client, clusterID)
		if err != nil {
			return diag.FromErr(err)
		}
		data.Set(FieldGKEClusterToken, tkn)
	}

	return nil
}

func resourceCastaiGKEClusterUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if err := updateGKEClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}

	return resourceCastaiGKEClusterRead(ctx, data, meta)
}

func resourceCastaiGKEClusterDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
			response, err := client.ExternalClusterAPIDisconnectClusterWithResponse(ctx, clusterId, sdk.ExternalClusterAPIDisconnectClusterJSONRequestBody{
				DeleteProvisionedNodes: getOptionalBool(data, FieldGKEClusterDeleteNodesOnDisconnect, false),
			})
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
func updateGKEClusterSettings(ctx context.Context, data *schema.ResourceData, client *sdk.ClientWithResponses) error {
	if !data.HasChanges(
		FieldGKEClusterSSHPublicKey,
		FieldGKEClusterCredentials,
	) {
		log.Printf("[INFO] Nothing to update in cluster setttings.")
		return nil
	}

	log.Printf("[INFO] Updating cluster settings.")

	req := sdk.ExternalClusterAPIUpdateClusterJSONRequestBody{}

	credentialsJSON, ok := data.GetOk(FieldGKEClusterCredentials)
	if ok {
		req.Credentials = toStringPtr(credentialsJSON.(string))
	}

	if s, ok := data.GetOk(FieldGKEClusterSSHPublicKey); ok {
		req.SshPublicKey = toStringPtr(s.(string))
	}

	response, err := client.ExternalClusterAPIUpdateClusterWithResponse(ctx, data.Id(), req)
	if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
		return fmt.Errorf("updating cluster settings: %w", checkErr)
	}

	return nil
}
