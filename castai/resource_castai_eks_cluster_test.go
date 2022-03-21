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

func TestEKSClusterResource(t *testing.T) {

	t.Run("test get existing cluster", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_sdk.NewMockClientInterface(mockctrl)

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"

		body := io.NopCloser(bytes.NewReader([]byte(`{
  "id": "b6bfc074-a267-400f-b8f1-db0850c369b1",
  "name": "eks-cluster",
  "organizationId": "2836f775-aaaa-eeee-bbbb-3d3c29512692",
  "credentialsId": "9b8d0456-177b-4a3d-b162-e68030d656aa",
  "createdAt": "2022-01-27T19:03:31.570829Z",
  "region": {
    "name": "eu-central-1",
    "displayName": "EU (Frankfurt)"
  },
  "status": "ready",
  "agentSnapshotReceivedAt": "2022-03-21T10:33:56.192020Z",
  "agentStatus": "online",
  "providerType": "eks",
  "eks": {
    "clusterName": "eks-cluster",
    "region": "eu-central-1",
    "accountId": "487609000000",
    "subnets": [],
    "securityGroups": [],
    "instanceProfileArn": "",
    "tags": {},
    "dnsClusterIp": ""
  },
  "subnets": [
    {
      "id": "subnet-0bbb192080507aa35",
      "name": "",
      "zoneName": "eu-central-1a"
    },
    {
      "id": "subnet-01a88dbdefa2d3838",
      "name": "",
      "zoneName": "eu-central-1b"
    },
    {
      "id": "subnet-07027d0f432135aac",
      "name": "",
      "zoneName": "eu-central-1c"
    }
  ],
  "zones": [
    {
      "id": "euc1-az2",
      "name": "eu-central-1a"
    },
    {
      "id": "euc1-az3",
      "name": "eu-central-1b"
    },
    {
      "id": "euc1-az1",
      "name": "eu-central-1c"
    }
  ],
  "clusterNameId": "eks-cluster-b6bfc074",
  "private": true,
  "allRegionZones": [
    {
      "id": "euc1-az2",
      "name": "eu-central-1a"
    },
    {
      "id": "euc1-az3",
      "name": "eu-central-1b"
    },
    {
      "id": "euc1-az1",
      "name": "eu-central-1c"
    }
  ]
}`)))
		response := &http.Response{StatusCode: 200, Body: body}
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
			Return(response, nil)

		resource := resourceCastaiEKSCluster()

		val := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(val, 0)
		state.ID = clusterId
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.Nil(result)
	})
}
