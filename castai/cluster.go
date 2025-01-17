package castai

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
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

		triggerDisconnect := func() *retry.RetryError {
			response, err := client.ExternalClusterAPIDisconnectClusterWithResponse(ctx, clusterId, sdk.ExternalClusterAPIDisconnectClusterJSONRequestBody{
				DeleteProvisionedNodes:  getOptionalBool(data, FieldDeleteNodesOnDisconnect, false),
				KeepKubernetesResources: toPtr(true),
			})
			if checkErr := sdk.CheckOKResponse(response, err); checkErr != nil {
				return retry.NonRetryableError(err)
			}

			return retry.RetryableError(fmt.Errorf("triggered agent disconnection cluster status %s agent status %s", clusterStatus, agentStatus))
		}

		triggerDelete := func() *retry.RetryError {
			log.Printf("[INFO] Deleting cluster.")
			res, err := client.ExternalClusterAPIDeleteClusterWithResponse(ctx, clusterId)
			if res.StatusCode() == 400 {
				return triggerDisconnect()
			}

			if checkErr := sdk.CheckResponseNoContent(res, err); checkErr != nil {
				return retry.NonRetryableError(fmt.Errorf("error when deleting cluster status %s agent status %s error: %w", clusterStatus, agentStatus, err))
			}
			return retry.RetryableError(fmt.Errorf("triggered cluster deletion"))
		}

		if agentStatus == sdk.ClusterAgentStatusDisconnected || clusterStatus == sdk.ClusterStatusDeleted {
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
			return retry.RetryableError(fmt.Errorf("cluster is deleting cluster status %s agent status %s", clusterStatus, agentStatus))
		}

		if toString(clusterResponse.JSON200.CredentialsId) != "" && agentStatus != sdk.ClusterAgentStatusDisconnected {
			log.Printf("[INFO] Disconnecting cluster.")
			return triggerDisconnect()
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

func fetchClusterData(ctx context.Context, client sdk.ClientWithResponsesInterface, clusterID string) (*sdk.ExternalClusterAPIGetClusterResponse, error) {
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

// resourceCastaiClusterUpdate performs the update call to Cast API for a given cluster.
// Handles backoffs and data drift for fields that are not provider-specific.
// Caller is responsible to populate data and request parameters with all data.
func resourceCastaiClusterUpdate(
	ctx context.Context,
	client sdk.ClientWithResponsesInterface,
	data *schema.ResourceData,
	request *sdk.ExternalClusterAPIUpdateClusterJSONRequestBody,
) error {
	b := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)

	var lastErr error
	var credentialsID string
	if err := backoff.RetryNotify(func() error {
		response, err := client.ExternalClusterAPIUpdateClusterWithResponse(ctx, data.Id(), *request)
		if err != nil {
			return fmt.Errorf("error when calling update cluster API: %w", err)
		}

		err = sdk.StatusOk(response)

		if err != nil {
			// In case of malformed user request return error to user right away.
			// Credentials error is omitted as permissions propagate eventually and sometimes aren't visible immediately.
			if response.StatusCode() == 400 && !sdk.IsCredentialsError(response) {
				return backoff.Permanent(err)
			}

			if response.StatusCode() == 400 && sdk.IsCredentialsError(response) {
				log.Printf("[WARN] Received credentials error from backend, will retry in case the issue is caused by IAM eventual consistency.")
			}
			return fmt.Errorf("error in update cluster response: %w", err)
		}

		if response.JSON200.CredentialsId != nil {
			credentialsID = *response.JSON200.CredentialsId
		}
		return nil
	}, b, func(err error, _ time.Duration) {
		// Only store non-context errors so we can surface the last "real" error to the user at the end
		if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			lastErr = err
		}
		log.Printf("[WARN] Encountered error while updating cluster settings, will retry: %v", err)
	}); err != nil {
		// Reset CredentialsID in state in case of failed updates.
		// This is because TF will save the raw credentials in state even on failed updates.
		// Since the raw values are not exposed via API, TF cannot see drift and will not try to re-apply them next time, leaving the caller stuck.
		// Resetting this value here will trigger our credentialsID drift detection on Read() and force re-apply to fix the drift.
		// Note: cannot use empty string; if first update failed then credentials will also be empty on remote => no drift on Read.
		// Src: https://developer.hashicorp.com/terraform/plugin/framework/diagnostics#returning-errors-and-warnings
		if err := data.Set(FieldClusterCredentialsId, "drift-protection-failed-update"); err != nil {
			log.Printf("[ERROR] Failed to reset cluster credentials ID after failed update: %v", err)
		}

		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return fmt.Errorf("updating cluster configuration failed due to context: %w; last observed error was: %v", err, lastErr)
		}
		return fmt.Errorf("updating cluster configuration: %w", err)
	}

	// In case the update succeeded, we must update the state with the *generated* credentials_id before re-reading.
	// This is because on update, the credentials_id always changes => read drift detection would see that and trigger infinite drift
	err := data.Set(FieldClusterCredentialsId, credentialsID)
	if err != nil {
		return fmt.Errorf("failed to update credentials ID after successful update: %w", err)
	}

	return nil
}

func createClusterToken(ctx context.Context, client sdk.ClientWithResponsesInterface, clusterID string) (string, error) {
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
