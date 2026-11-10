package castai

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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
organization_id = 2836f775-aaaa-eeee-bbbb-3d3c29512GKE
project_id = project-id
Tainted = false
`, data.State().String())
}

func TestGKEClusterIdResourceDeprecated(t *testing.T) {
	r := require.New(t)
	r.NotEmpty(resourceGKEClusterId().DeprecationMessage)
}

func TestGKEClusterIdResourceDeleteContextSkipsWhenCredentialsAssigned(t *testing.T) {
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
  "id": "b6bfc074-a267-400f-b8f1-db0850c36gke",
  "name": "gke-cluster",
  "organizationId": "2836f775-aaaa-eeee-bbbb-3d3c29512GKE",
  "credentialsId": "9b8d0456-177b-4a3d-b162-e68030d65GKE",
  "status": "ready",
  "agentStatus": "online",
  "providerType": "gke",
  "gke": {
    "clusterName": "gke-cluster",
    "region": "eu-central-1",
    "location": "eu-central-1",
    "projectId": "project-id"
  }
}`)))
	mockClient.EXPECT().
		ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceGKEClusterId()

	val := cty.ObjectVal(map[string]cty.Value{})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = clusterId

	data := resource.Data(state)
	result := resource.DeleteContext(ctx, data, provider)
	r.False(result.HasError())
	r.Len(result, 1)
	r.Equal(diag.Warning, result[0].Severity)
	r.Equal(clusterId, data.Id())
}

func TestGKEClusterIdResourceDeleteContextDeletesWithoutCredentials(t *testing.T) {
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

	readyJSON := `{
  "id": "b6bfc074-a267-400f-b8f1-db0850c36gke",
  "name": "gke-cluster",
  "organizationId": "2836f775-aaaa-eeee-bbbb-3d3c29512GKE",
  "status": "ready",
  "agentStatus": "online",
  "providerType": "gke",
  "gke": {
    "clusterName": "gke-cluster",
    "region": "eu-central-1",
    "location": "eu-central-1",
    "projectId": "project-id"
  }
}`
	archivedJSON := `{
  "id": "b6bfc074-a267-400f-b8f1-db0850c36gke",
  "name": "gke-cluster",
  "organizationId": "2836f775-aaaa-eeee-bbbb-3d3c29512GKE",
  "status": "archived",
  "agentStatus": "disconnected",
  "providerType": "gke"
}`
	jsonHeader := map[string][]string{"Content-Type": {"json"}}
	clusterResponse := func(body string) func(context.Context, string, ...sdk.RequestEditorFn) (*http.Response, error) {
		return func(context.Context, string, ...sdk.RequestEditorFn) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(body))), Header: jsonHeader}, nil
		}
	}

	gomock.InOrder(
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
			DoAndReturn(clusterResponse(readyJSON)),
		mockClient.EXPECT().
			ExternalClusterAPIDeleteCluster(gomock.Any(), clusterId).
			Return(&http.Response{StatusCode: 204, Body: http.NoBody}, nil),
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
			DoAndReturn(clusterResponse(archivedJSON)),
	)

	resource := resourceGKEClusterId()

	val := cty.ObjectVal(map[string]cty.Value{})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = clusterId

	data := resource.Data(state)
	result := resource.DeleteContext(ctx, data, provider)
	r.False(result.HasError())
	r.Empty(data.Id())
}

func TestGKEClusterIdResourceDeleteContextClusterGone(t *testing.T) {
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

	body := io.NopCloser(bytes.NewReader([]byte(`{"message": "cluster not found"}`)))
	mockClient.EXPECT().
		ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
		Return(&http.Response{StatusCode: 404, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceGKEClusterId()

	val := cty.ObjectVal(map[string]cty.Value{})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = clusterId

	data := resource.Data(state)
	result := resource.DeleteContext(ctx, data, provider)
	r.False(result.HasError())
}

func TestGKEClusterIdResourceDeleteContextFetchError(t *testing.T) {
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

	body := io.NopCloser(bytes.NewReader([]byte(`{"message": "internal server error"}`)))
	mockClient.EXPECT().
		ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
		Return(&http.Response{StatusCode: 500, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceGKEClusterId()

	val := cty.ObjectVal(map[string]cty.Value{})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = clusterId

	data := resource.Data(state)
	result := resource.DeleteContext(ctx, data, provider)
	r.True(result.HasError())
	r.Len(result, 1)
	r.Contains(result[0].Summary, "expected status code 200")
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
