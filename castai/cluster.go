package castai

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
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
	FieldClusterOrganizationId   = "organization_id"
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

	// NEW: Check for orphaned nodes when delete_nodes_on_disconnect was requested.
	var diags diag.Diagnostics

	if deleteNodesRequested, ok := data.GetOk(FieldDeleteNodesOnDisconnect); ok && deleteNodesRequested.(bool) {
		diags = checkOrphanedNodes(ctx, client, clusterId)
	}

	data.SetId("")
	return diags
}

// checkOrphanedNodes queries the CAST AI API for any remaining nodes after cluster
// disconnect. If the cluster is already gone (404 or error), the check is skipped
// silently. Otherwise, any non-deleted CAST-managed nodes are reported as a warning
// so the user is aware that node cleanup may have failed (e.g. due to IAM
// credentials being revoked before the backend could delete the nodes).
func checkOrphanedNodes(ctx context.Context, client sdk.ClientWithResponsesInterface, clusterId string) diag.Diagnostics {
	resp, err := client.ExternalClusterAPIListNodesWithResponse(ctx, clusterId, &sdk.ExternalClusterAPIListNodesParams{
		ExcludeDeleting: toPtr(true),
	})
	if err != nil {
		log.Printf("[WARN] Failed to list nodes after cluster disconnect: %v", err)
		return nil
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil || resp.JSON200.Items == nil {
		// Cluster is likely already gone (e.g. 404), nothing to check.
		return nil
	}

	var orphaned []sdk.ExternalclusterV1Node
	for _, node := range *resp.JSON200.Items {
		// Skip nodes already in deleted state.
		if node.State.Phase != nil && *node.State.Phase == "deleted" {
			continue
		}
		// Only CAST-managed nodes are CAST AI's responsibility to clean up.
		// Nodes with no AddedBy marker or a non-"cast" value are customer-managed.
		if node.AddedBy == nil || *node.AddedBy != "cast" {
			continue
		}
		orphaned = append(orphaned, node)
	}

	if len(orphaned) == 0 {
		return nil
	}

	var instanceIDs []string
	for _, node := range orphaned {
		if node.InstanceId != nil && *node.InstanceId != "" {
			instanceIDs = append(instanceIDs, *node.InstanceId)
		} else {
			instanceIDs = append(instanceIDs, node.Id)
		}
	}

	log.Printf("[WARN] %d CAST-managed nodes remain after cluster disconnect: %v", len(orphaned), instanceIDs)

	return diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("%d CAST-managed node(s) remain after cluster disconnect", len(orphaned)),
			Detail: fmt.Sprintf(
				"Node cleanup may have failed, possibly because cloud credentials (IAM role) were revoked before the backend could delete the nodes. "+
					"The following instances may be orphaned in your cloud account: %s\n\n"+
					"To prevent this in the future, ensure your Terraform configuration destroys the IAM role AFTER the castai cluster resource "+
					"(use depends_on or reference the role ARN directly in assume_role_arn). Otherwise, IAM may be destroyed in parallel, causing "+
					"node cleanup to fail and leaving orphaned instances. "+
					"See: https://github.com/castai/terraform-provider-castai/blob/master/examples/eks/eks_cluster_existing/castai.tf",
				strings.Join(instanceIDs, ", "),
			),
		},
	}
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
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return "", fmt.Errorf("creating cluster token: %w", checkErr)
	}

	if resp == nil || resp.JSON200 == nil || resp.JSON200.Token == nil {
		return "", fmt.Errorf("response was empty when trying to create cluster token")
	}

	return *resp.JSON200.Token, nil
}
