package castai

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldDeleteNodesOnDisconnect = "delete_nodes_on_disconnect"
	FieldClusterSSHPublicKey     = "ssh_public_key"
	FieldClusterAgentToken       = "agent_token"
	FieldClusterCredentialsId    = "credentials_id"
)

func resourceCastaiPublicCloudClusterDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

		triggerDelete := func() *resource.RetryError {
			log.Printf("[INFO] Deleting cluster.")
			if err := sdk.CheckResponseNoContent(client.ExternalClusterAPIDeleteClusterWithResponse(ctx, clusterId)); err != nil {
				return resource.NonRetryableError(err)
			}
			return resource.RetryableError(fmt.Errorf("triggered cluster deletion"))
		}

		// If cluster doesn't have credentials we have to call delete cluster instead of disconnect because disconnect
		// will do nothing on cluster with empty credentials.
		if toString(clusterResponse.JSON200.CredentialsId) == "" {
			return triggerDelete()
		}

		if clusterStatus == sdk.ClusterStatusFailed {
			return triggerDelete()
		}

		if agentStatus == sdk.ClusterAgentStatusDisconnecting {
			return resource.RetryableError(fmt.Errorf("agent is disconnecting"))
		}

		if clusterStatus == sdk.ClusterStatusDeleting {
			return resource.RetryableError(fmt.Errorf("cluster is deleting"))
		}

		if toString(clusterResponse.JSON200.CredentialsId) != "" && agentStatus != sdk.ClusterAgentStatusDisconnected {
			log.Printf("[INFO] Disconnecting cluster.")
			response, err := client.ExternalClusterAPIDisconnectClusterWithResponse(ctx, clusterId, sdk.ExternalClusterAPIDisconnectClusterJSONRequestBody{
				DeleteProvisionedNodes:  getOptionalBool(data, FieldDeleteNodesOnDisconnect, false),
				KeepKubernetesResources: toBoolPtr(true),
			})
			if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
				return resource.NonRetryableError(err)
			}

			return resource.RetryableError(fmt.Errorf("triggered agent disconnection"))
		}

		if agentStatus == sdk.ClusterAgentStatusDisconnected && clusterStatus != sdk.ClusterStatusDeleted {
			return triggerDelete()
		}

		return resource.RetryableError(fmt.Errorf("retrying"))
	})

	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func fetchClusterData(ctx context.Context, client *sdk.ClientWithResponses, clusterID string) (*sdk.ExternalClusterAPIGetClusterResponse, error) {
	resp, err := client.ExternalClusterAPIGetClusterWithResponse(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Removing cluster %s from state because it no longer exists in CAST AI", clusterID)
		return nil, nil
	}

	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return nil, checkErr
	}

	if resp.JSON200 != nil && toString(resp.JSON200.Status) == sdk.ClusterStatusArchived {
		log.Printf("[WARN] Removing cluster %s from state because it is archived in CAST AI", clusterID)
		return nil, nil
	}

	return resp, nil
}
