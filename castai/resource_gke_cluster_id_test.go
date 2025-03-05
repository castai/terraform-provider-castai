package castai

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestGKEClusterIdResourceReadContext(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c36gke"

	body := io.NopCloser(bytes.NewReader([]byte(`{
  "id": "b6bfc074-a267-400f-b8f1-db0850c36gk3",
  "name": "gke-cluster",
  "organizationId": "2836f775-aaaa-eeee-bbbb-3d3c29512GKE",
  "credentialsId": "9b8d0456-177b-4a3d-b162-e68030d65GKE",
  "createdAt": "2022-04-27T19:03:31.570829Z",
  "region": {
    "name": "eu-central-1",
    "displayName": "EU (Frankfurt)"
  },
  "status": "ready",
  "agentSnapshotReceivedAt": "2022-05-21T10:33:56.192020Z",
  "agentStatus": "online",
  "providerType": "gke",
  "gke": {
    "clusterName": "gke-cluster",
    "region": "eu-central-1",
	"location": "eu-central-1",
	"projectId": "project-id",
	"clientServiceAccount": "client-service-account",
	"castServiceAccount": "cast-service-account"
  },
  "clusterNameId": "gke-cluster-b6bfc074"
}`)))
	mockClient.EXPECT().
		ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceGKEClusterId()

	val := cty.ObjectVal(map[string]cty.Value{})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = clusterId

	data := resource.Data(state)
	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Equal(`ID = b6bfc074-a267-400f-b8f1-db0850c36gke
client_service_account = client-service-account
location = eu-central-1
name = gke-cluster
project_id = project-id
Tainted = false
`, data.State().String())
}

func TestGKEClusterIdResourceReadContextArchived(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c36gke"

	body := io.NopCloser(bytes.NewReader([]byte(`{
  "id": "b6bfc074-a267-400f-b8f1-db0850c36gk3",
  "name": "gke-cluster",
  "organizationId": "2836f775-aaaa-eeee-bbbb-3d3c29512GKE",
  "credentialsId": "9b8d0456-177b-4a3d-b162-e68030d65GKE",
  "createdAt": "2022-04-27T19:03:31.570829Z",
  "region": {
    "name": "eu-central-1",
    "displayName": "EU (Frankfurt)"
  },
  "status": "archived",
  "agentSnapshotReceivedAt": "2022-05-21T10:33:56.192020Z",
  "agentStatus": "online",
  "providerType": "gke",
  "gke": {
    "clusterName": "gke-cluster",
    "region": "eu-central-1",
	"location": "eu-central-1",
	"projectId": "project-id",
	"clientServiceAccount": "client-service-account",
	"castServiceAccount": "cast-service-account"
  },
  "sshPublicKey": "key-123",
  "clusterNameId": "gke-cluster-b6bfc074",
  "private": true
}`)))
	mockClient.EXPECT().
		ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceGKEClusterId()

	val := cty.ObjectVal(map[string]cty.Value{})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = clusterId

	data := resource.Data(state)
	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Equal(`<not created>`, data.State().String())
}
