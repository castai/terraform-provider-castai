package castai

import (
	"context"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func strPtr(s string) *string { return &s }

func TestCheckOrphanedNodes_WithRemainingNodes(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	clusterID := "test-cluster-id"

	mockctrl := gomock.NewController(t)
	defer mockctrl.Finish()

	mockClient := mock_sdk.NewMockClientWithResponsesInterface(mockctrl)
	mockClient.EXPECT().
		ExternalClusterAPIListNodesWithResponse(gomock.Any(), clusterID, gomock.Any()).
		Return(&sdk.ExternalClusterAPIListNodesResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &sdk.ExternalclusterV1ListNodesResponse{
				Items: &[]sdk.ExternalclusterV1Node{
					{
						Id:         "node-1",
						InstanceId: strPtr("i-aaa"),
						AddedBy:    toPtr("cast"),
						State:      sdk.ExternalclusterV1NodeState{Phase: strPtr("ready")},
					},
					{
						Id:         "node-2",
						InstanceId: strPtr("i-bbb"),
						AddedBy:    toPtr("cast"),
						State:      sdk.ExternalclusterV1NodeState{Phase: strPtr("ready")},
					},
				},
			},
		}, nil)

	diags := checkOrphanedNodes(ctx, mockClient, clusterID)

	r.Len(diags, 1)
	r.Equal(diag.Warning, diags[0].Severity)
	r.Contains(diags[0].Summary, "2 CAST-managed node(s) remain after cluster disconnect")
	r.Contains(diags[0].Detail, "i-aaa")
	r.Contains(diags[0].Detail, "i-bbb")
	r.Contains(diags[0].Detail, "IAM")
}

func TestCheckOrphanedNodes_WithRemainingNodesNilInstanceId(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	clusterID := "test-cluster-id"

	mockctrl := gomock.NewController(t)
	defer mockctrl.Finish()

	mockClient := mock_sdk.NewMockClientWithResponsesInterface(mockctrl)
	mockClient.EXPECT().
		ExternalClusterAPIListNodesWithResponse(gomock.Any(), clusterID, gomock.Any()).
		Return(&sdk.ExternalClusterAPIListNodesResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &sdk.ExternalclusterV1ListNodesResponse{
				Items: &[]sdk.ExternalclusterV1Node{
					{
						Id:         "node-id-1",
						InstanceId: nil,
						AddedBy:    toPtr("cast"),
						State:      sdk.ExternalclusterV1NodeState{Phase: nil},
					},
				},
			},
		}, nil)

	diags := checkOrphanedNodes(ctx, mockClient, clusterID)

	r.Len(diags, 1)
	r.Equal(diag.Warning, diags[0].Severity)
	r.Contains(diags[0].Summary, "1 CAST-managed node(s) remain after cluster disconnect")
	// InstanceId is nil, so node Id should be used as fallback.
	r.Contains(diags[0].Detail, "node-id-1")
}

func TestCheckOrphanedNodes_NoRemainingNodes(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	clusterID := "test-cluster-id"

	mockctrl := gomock.NewController(t)
	defer mockctrl.Finish()

	mockClient := mock_sdk.NewMockClientWithResponsesInterface(mockctrl)
	emptyItems := []sdk.ExternalclusterV1Node{}
	mockClient.EXPECT().
		ExternalClusterAPIListNodesWithResponse(gomock.Any(), clusterID, gomock.Any()).
		Return(&sdk.ExternalClusterAPIListNodesResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &sdk.ExternalclusterV1ListNodesResponse{
				Items: &emptyItems,
			},
		}, nil)

	diags := checkOrphanedNodes(ctx, mockClient, clusterID)
	r.Nil(diags)
}

func TestCheckOrphanedNodes_OnlyDeletedNodes(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	clusterID := "test-cluster-id"

	mockctrl := gomock.NewController(t)
	defer mockctrl.Finish()

	mockClient := mock_sdk.NewMockClientWithResponsesInterface(mockctrl)
	mockClient.EXPECT().
		ExternalClusterAPIListNodesWithResponse(gomock.Any(), clusterID, gomock.Any()).
		Return(&sdk.ExternalClusterAPIListNodesResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &sdk.ExternalclusterV1ListNodesResponse{
				Items: &[]sdk.ExternalclusterV1Node{
					{
						Id:         "node-1",
						InstanceId: strPtr("i-aaa"),
						State:      sdk.ExternalclusterV1NodeState{Phase: strPtr("deleted")},
					},
				},
			},
		}, nil)

	diags := checkOrphanedNodes(ctx, mockClient, clusterID)
	r.Nil(diags)
}

func TestCheckOrphanedNodes_ListNodesError(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	clusterID := "test-cluster-id"

	mockctrl := gomock.NewController(t)
	defer mockctrl.Finish()

	mockClient := mock_sdk.NewMockClientWithResponsesInterface(mockctrl)
	mockClient.EXPECT().
		ExternalClusterAPIListNodesWithResponse(gomock.Any(), clusterID, gomock.Any()).
		Return(nil, errListNodes("network error"))

	diags := checkOrphanedNodes(ctx, mockClient, clusterID)
	r.Nil(diags)
}

func TestCheckOrphanedNodes_ListNodes404(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	clusterID := "test-cluster-id"

	mockctrl := gomock.NewController(t)
	defer mockctrl.Finish()

	mockClient := mock_sdk.NewMockClientWithResponsesInterface(mockctrl)
	mockClient.EXPECT().
		ExternalClusterAPIListNodesWithResponse(gomock.Any(), clusterID, gomock.Any()).
		Return(&sdk.ExternalClusterAPIListNodesResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
		}, nil)

	diags := checkOrphanedNodes(ctx, mockClient, clusterID)
	r.Nil(diags)
}

func TestCheckOrphanedNodes_NilItems(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	clusterID := "test-cluster-id"

	mockctrl := gomock.NewController(t)
	defer mockctrl.Finish()

	mockClient := mock_sdk.NewMockClientWithResponsesInterface(mockctrl)
	mockClient.EXPECT().
		ExternalClusterAPIListNodesWithResponse(gomock.Any(), clusterID, gomock.Any()).
		Return(&sdk.ExternalClusterAPIListNodesResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &sdk.ExternalclusterV1ListNodesResponse{Items: nil},
		}, nil)

	diags := checkOrphanedNodes(ctx, mockClient, clusterID)
	r.Nil(diags)
}

func TestCheckOrphanedNodes_OnlyNonCastManagedNodes(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	clusterID := "test-cluster-id"

	mockctrl := gomock.NewController(t)
	defer mockctrl.Finish()

	mockClient := mock_sdk.NewMockClientWithResponsesInterface(mockctrl)
	mockClient.EXPECT().
		ExternalClusterAPIListNodesWithResponse(gomock.Any(), clusterID, gomock.Any()).
		Return(&sdk.ExternalClusterAPIListNodesResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &sdk.ExternalclusterV1ListNodesResponse{
				Items: &[]sdk.ExternalclusterV1Node{
					{
						Id:         "node-1",
						InstanceId: strPtr("i-aaa"),
						AddedBy:    strPtr("user"),
						State:      sdk.ExternalclusterV1NodeState{Phase: strPtr("ready")},
					},
					{
						Id:         "node-2",
						InstanceId: strPtr("i-bbb"),
						AddedBy:    nil,
						State:      sdk.ExternalclusterV1NodeState{Phase: strPtr("ready")},
					},
				},
			},
		}, nil)

	diags := checkOrphanedNodes(ctx, mockClient, clusterID)
	// Non-cast-managed nodes should NOT be counted as orphaned.
	r.Nil(diags)
}

func TestResourceCastaiClusterDelete_DeleteNodesOnDisconnectFalse_NoListNodesCall(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	clusterID := "test-cluster-id"

	mockctrl := gomock.NewController(t)
	defer mockctrl.Finish()

	mockClient := mock_sdk.NewMockClientWithResponsesInterface(mockctrl)

	// Only GetCluster is expected — it returns archived so the retry loop exits.
	// No ExternalClusterAPIListNodesWithResponse expectation is set, proving
	// checkOrphanedNodes is not called when delete_nodes_on_disconnect is false.
	archivedStatus := sdk.ClusterStatusArchived
	onlineAgent := sdk.ClusterAgentStatusDisconnected
	mockClient.EXPECT().
		ExternalClusterAPIGetClusterWithResponse(gomock.Any(), clusterID).
		Return(&sdk.ExternalClusterAPIGetClusterResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			JSON200: &sdk.ExternalclusterV1Cluster{
				Status:      &archivedStatus,
				AgentStatus: &onlineAgent,
			},
		}, nil)

	provider := &ProviderConfig{api: mockClient}

	resource := resourceEKSCluster()
	raw := map[string]interface{}{
		FieldDeleteNodesOnDisconnect: false,
	}
	data := schema.TestResourceDataRaw(t, resource.Schema, raw)
	data.SetId(clusterID)

	diags := resourceCastaiClusterDelete(ctx, data, provider)
	r.False(diags.HasError())
	r.Empty(diags)
	r.Equal("", data.Id())
}

// errListNodes is a simple error type used for testing the network error path.
type errListNodes string

func (e errListNodes) Error() string { return string(e) }
