package castai

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestEKSClusterResourceReadContext(t *testing.T) {
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
    "subnets": ["sub1", "sub2"],
    "securityGroups": ["sg1"],
    "instanceProfileArn": "arn",
    "tags": {"aws":"tag"},
    "dnsClusterIp": "10.100.100.1"
  },
  "sshPublicKey": "key-123",
  "clusterNameId": "eks-cluster-b6bfc074",
  "private": true
}`)))
	mockClient.EXPECT().
		ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	mockClient.EXPECT().
		GetAgentInstallScript(gomock.Any(), gomock.Any()).
		Return(
			&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(`curl -H "Authorization: Token 123`)))},
			nil)

	resource := resourceCastaiEKSCluster()

	val := cty.ObjectVal(map[string]cty.Value{})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = clusterId

	data := resource.Data(state)
	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Equal(`ID = b6bfc074-a267-400f-b8f1-db0850c369b1
account_id = 487609000000
credentials_id = 9b8d0456-177b-4a3d-b162-e68030d656aa
dns_cluster_ip = 10.100.100.1
instance_profile_arn = arn
name = eks-cluster
region = eu-central-1
security_groups.# = 1
security_groups.0 = sg1
ssh_public_key = key-123
subnets.# = 2
subnets.0 = sub1
subnets.1 = sub2
tags.% = 1
tags.aws = tag
Tainted = false
`, data.State().String())
}

func TestEKSClusterResourceReadContextArchived(t *testing.T) {
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
  "status": "archived",
  "agentSnapshotReceivedAt": "2022-03-21T10:33:56.192020Z",
  "agentStatus": "disconnected",
  "providerType": "eks",
  "eks": {
    "clusterName": "eks-cluster",
    "region": "eu-central-1",
    "accountId": "487609000000",
    "subnets": ["sub1", "sub2"],
    "securityGroups": ["sg1"],
    "instanceProfileArn": "arn",
    "tags": {"aws":"tag"},
    "dnsClusterIp": "10.100.100.1"
  },
  "sshPublicKey": "key-123",
  "clusterNameId": "eks-cluster-b6bfc074",
  "private": true
}`)))
	mockClient.EXPECT().
		ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceCastaiEKSCluster()

	val := cty.ObjectVal(map[string]cty.Value{})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = clusterId

	data := resource.Data(state)
	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Equal(`<not created>`, data.State().String())
}

func TestEKSClusterResourceUpdateError(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c36gk3d"
	mockClient.EXPECT().
		ExternalClusterAPIUpdateCluster(gomock.Any(), clusterId, gomock.Any(), gomock.Any()).
		Return(&http.Response{StatusCode: 400, Body: io.NopCloser(bytes.NewBufferString(`{"message":"Bad Request", "fieldViolations":[{"field":"credentials","description":"error"}]}`)), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceCastaiEKSCluster()

	raw := make(map[string]interface{})
	raw[FieldEKSClusterAssumeRoleArn] = "something"

	data := schema.TestResourceDataRaw(t, resource.Schema, raw)
	data.Set(FieldEKSClusterAssumeRoleArn, "creds")
	data.SetId(clusterId)
	result := resource.UpdateContext(ctx, data, provider)
	r.NotNil(result)
	r.True(result.HasError())
	r.Equal("updating cluster configuration: expected status code 200, received: status=400 body={\"message\":\"Bad Request\", \"fieldViolations\":[{\"field\":\"credentials\",\"description\":\"error\"}]}", result[0].Summary)
}
func TestEKSClusterResourceUpdateRetry(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c36gk3d"
	body := bytes.NewBufferString(`{
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
    "subnets": ["sub1", "sub2"],
    "securityGroups": ["sg1"],
    "instanceProfileArn": "arn",
    "tags": {"aws":"tag"},
    "dnsClusterIp": "10.100.100.1"
  },
  "sshPublicKey": "key-123",
  "clusterNameId": "eks-cluster-b6bfc074",
  "private": true
}`)
	mockClient.EXPECT().
		ExternalClusterAPIUpdateCluster(gomock.Any(), clusterId, gomock.Any(), gomock.Any()).
		Return(&http.Response{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString(`{"message":"Internal Server Error"}`)), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	mockClient.EXPECT().
		ExternalClusterAPIUpdateCluster(gomock.Any(), clusterId, gomock.Any(), gomock.Any()).
		Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString("{}")), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	mockClient.EXPECT().
		ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
		Return(&http.Response{StatusCode: 200, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	mockClient.EXPECT().
		GetAgentInstallScript(gomock.Any(), gomock.Any()).
		Return(
			&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(`curl -H "Authorization: Token 123`)))},
			nil)

	resource := resourceCastaiEKSCluster()

	raw := make(map[string]interface{})
	raw[FieldEKSClusterAssumeRoleArn] = "something"

	data := schema.TestResourceDataRaw(t, resource.Schema, raw)
	data.Set(FieldEKSClusterAssumeRoleArn, "creds")
	data.SetId(clusterId)
	result := resource.UpdateContext(ctx, data, provider)
	r.Nil(result)
}
