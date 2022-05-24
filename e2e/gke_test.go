package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func TestTerraformGKEOnboarding(t *testing.T) {
	varsFile, err := createVarsFile(map[string]interface{}{
		"cluster_name":     cfg.GKEClusterName,
		"cluster_location": cfg.GKEClusterLocation,
		"network_region":   cfg.GKENetworkRegion,
		"castai_api_token": cfg.Token,
		"project_id":       cfg.GKEProjectID,
		"gcp_credentials":  cfg.GKECredentials,
	}, "gke")

	if err != nil {
		panic(err)
	}
	defer os.Remove(varsFile)

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "./tests/gke_cluster_zonal",
		VarFiles:     []string{varsFile},
	})

	r := require.New(t)
	ctx := context.Background()
	//defer terraform.Destroy(t, terraformOptions)
	//terraform.InitAndApply(t, terraformOptions)
	terraform.Plan(t, terraformOptions)
	return
	clusterID := terraform.OutputRequired(t, terraformOptions, "castai_cluster_id")

	castAIClient, err := createClient(cfg.APIURL, cfg.Token)
	r.NoError(err)

	fmt.Println("Waiting for cluster to become ready")
	// Validate if cluster become ready in our console.
	r.Eventuallyf(func() bool {
		res, err := castAIClient.ExternalClusterAPIGetClusterWithResponse(ctx, clusterID)
		r.NoError(err)
		if res != nil && res.JSON200 != nil && res.JSON200.Status != nil && *res.JSON200.Status == "ready" {
			return true
		}
		return false
	}, time.Minute*5, time.Second*30, "cluster doesn't become ready after timeout")

	fmt.Println("Adding node")
	// Try to add node
	addNode, err := castAIClient.ExternalClusterAPIAddNodeWithResponse(ctx, clusterID, sdk.ExternalClusterAPIAddNodeJSONRequestBody{InstanceType: "e2-medium"})
	r.NoError(err)
	r.Equal(200, addNode.StatusCode(), fmt.Sprintf("Response from adding node should be 200, body: %s", string(addNode.Body)))

	fmt.Println("Waiting for node to be added")
	lastBodyForOp := ""
	r.Eventually(func() bool {
		opStatus, err := castAIClient.GetExternalClusterOperationWithResponse(ctx, addNode.JSON200.OperationId)
		r.NoError(err)
		lastBodyForOp = string(opStatus.Body)
		r.False(opStatus.JSON200 != nil && opStatus.JSON200.Error != nil, fmt.Sprintf("Error while waiting for operation end. body: %s", lastBodyForOp))
		return opStatus.JSON200 != nil && opStatus.JSON200.Done

	}, time.Minute*5, time.Second*15, fmt.Sprintf("waiting for add node operation timeout. body: %s, opID: %s", lastBodyForOp, addNode.JSON200.OperationId))

	node, err := castAIClient.ExternalClusterAPIGetNodeWithResponse(ctx, clusterID, addNode.JSON200.NodeId)
	r.NoError(err)
	r.NotNil(node.JSON200.State.Phase)
	r.Equal("ready", *node.JSON200.State.Phase)

}
