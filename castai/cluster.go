package castai

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldDeleteNodesOnDisconnect = "delete_nodes_on_disconnect"
	FieldClusterCredentialsId    = "credentials_id"
	FieldClusterID               = "cluster_id"
	FieldClusterToken            = "cluster_token"
)

func resourceCastaiClusterDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	clusterId := data.Id()

	log.Printf("[INFO] Checking current status of the cluster.")

	err := retry.RetryContext(ctx, data.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		clusterResponse, err := client.ExternalClusterAPIGetClusterWithResponse(ctx, clusterId)
		if checkErr := sdk.CheckOKResponse(clusterResponse, err); checkErr != nil {
			return retry.NonRetryableError(err)
		}

		clusterStatus := *clusterResponse.JSON200.Status
		agentStatus := *clusterResponse.JSON200.AgentStatus
		log.Printf("[INFO] Current cluster status=%s, agent_status=%s", clusterStatus, agentStatus)

		if clusterStatus == sdk.ClusterStatusArchived {
			log.Printf("[INFO] Cluster is already deleted, removing from state.")
			data.SetId("")
			return nil
		}

		triggerDelete := func() *retry.RetryError {
			log.Printf("[INFO] Deleting cluster.")
			if err := sdk.CheckResponseNoContent(client.ExternalClusterAPIDeleteClusterWithResponse(ctx, clusterId)); err != nil {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(fmt.Errorf("triggered cluster deletion"))
		}

		if agentStatus == sdk.ClusterAgentStatusDisconnected || agentStatus == "connecting" {
			return triggerDelete()
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
			return retry.RetryableError(fmt.Errorf("agent is disconnecting cluster status %s agent status %s", clusterStatus, agentStatus))
		}

		if clusterStatus == sdk.ClusterStatusDeleting {
			return retry.RetryableError(fmt.Errorf("cluster is deleting cluster status %s agent status", clusterStatus, agentStatus))
		}

		if toString(clusterResponse.JSON200.CredentialsId) != "" && agentStatus != sdk.ClusterAgentStatusDisconnected {
			log.Printf("[INFO] Disconnecting cluster.")
			response, err := client.ExternalClusterAPIDisconnectClusterWithResponse(ctx, clusterId, sdk.ExternalClusterAPIDisconnectClusterJSONRequestBody{
				DeleteProvisionedNodes:  getOptionalBool(data, FieldDeleteNodesOnDisconnect, false),
				KeepKubernetesResources: toPtr(true),
			})
			if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
				return retry.NonRetryableError(err)
			}

			return retry.RetryableError(fmt.Errorf("triggered agent disconnection cluster status %s agent status %s", clusterStatus, agentStatus))
		}

		if agentStatus == sdk.ClusterAgentStatusDisconnected && clusterStatus != sdk.ClusterStatusDeleted {
			return triggerDelete()
		}

		return retry.RetryableError(fmt.Errorf("retrying cluster status %s agent status %s", clusterStatus, agentStatus))
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

func createClusterToken(ctx context.Context, client *sdk.ClientWithResponses, clusterID string) (string, error) {
	resp, err := client.ExternalClusterAPICreateClusterTokenWithResponse(ctx, clusterID)
	if err != nil {
		return "", fmt.Errorf("creating cluster token: %w", err)
	}

	return *resp.JSON200.Token, nil
}

func clusterTokenDiff(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	if diff.Id() == "" {
		return nil
	}
	if diff.Get(FieldClusterToken).(string) != "" {
		return nil
	}

	// During migration to the latest version, cluster resource might have empty token as it was introduced later on.
	// If that's the case - we are forcing re-creation by providing random new value and setting "ForceNew" flag.
	log.Print("[INFO] token not set, forcing re-create")
	if err := diff.SetNew(FieldClusterToken, uuid.NewString()); err != nil {
		return fmt.Errorf("setting cluster token: %w", err)
	}

	return diff.ForceNew(FieldClusterToken)
}
