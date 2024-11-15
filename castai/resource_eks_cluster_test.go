package castai

import (
	"bytes"
	"context"
	"fmt"
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
	ctx := context.Background()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	t.Run("read should populate data correctly", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_sdk.NewMockClientInterface(mockctrl)

		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

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
    "dnsClusterIp": "10.100.100.1",
	"imdsV1": true
  },
  "sshPublicKey": "key-123",
  "clusterNameId": "eks-cluster-b6bfc074",
  "private": true
}`)))
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
			Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		resource := resourceEKSCluster()

		val := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(val, 0)
		state.ID = clusterID
		state.Attributes[FieldClusterCredentialsId] = "9b8d0456-177b-4a3d-b162-e68030d656aa"

		data := resource.Data(state)
		result := resource.ReadContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal(`ID = b6bfc074-a267-400f-b8f1-db0850c369b1
account_id = 487609000000
assume_role_arn = 
credentials_id = 9b8d0456-177b-4a3d-b162-e68030d656aa
name = eks-cluster
region = eu-central-1
Tainted = false
`, data.State().String())
	})

	t.Run("on credentials drift, changes role_arn to trigger drift and re-apply", func(t *testing.T) {
		testCase := []struct {
			name       string
			stateValue string
			apiValue   string
		}{
			{
				name:       "empty credentials in remote",
				stateValue: "credentials-id-local",
				apiValue:   "",
			},
			{
				name:       "different credentials in remote",
				stateValue: "credentials-id-local",
				apiValue:   "credentials-id-remote",
			},
			{
				name:       "empty credentials in local but exist in remote",
				stateValue: "",
				apiValue:   "credentials-id-remote",
			},
		}

		for _, tc := range testCase {
			t.Run(tc.name, func(t *testing.T) {
				r := require.New(t)
				mockctrl := gomock.NewController(t)
				mockClient := mock_sdk.NewMockClientInterface(mockctrl)
				provider := &ProviderConfig{
					api: &sdk.ClientWithResponses{
						ClientInterface: mockClient,
					},
				}
				roleARNBeforeRead := "dummy-rolearn"

				body := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"credentialsId": "%s"}`, tc.apiValue))))
				mockClient.EXPECT().
					ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

				resource := resourceEKSCluster()

				val := cty.ObjectVal(map[string]cty.Value{})
				state := terraform.NewInstanceStateShimmedFromValue(val, 0)
				state.ID = clusterID
				state.Attributes[FieldClusterCredentialsId] = tc.stateValue
				state.Attributes[FieldEKSClusterAssumeRoleArn] = roleARNBeforeRead

				data := resource.Data(state)
				result := resource.ReadContext(ctx, data, provider)
				r.Nil(result)
				r.False(result.HasError())

				roleARNAfter := data.Get(FieldEKSClusterAssumeRoleArn)

				r.NotEqual(roleARNBeforeRead, roleARNAfter)
				r.NotEmpty(roleARNAfter)
			})
		}
	})

	t.Run("when credentials match, no drift should be triggered", func(t *testing.T) {
		testCase := []struct {
			name       string
			stateValue string
			apiValue   string
		}{
			{
				name:       "empty credentials in both",
				stateValue: "",
				apiValue:   "",
			},
			{
				name:       "matching credentials",
				stateValue: "credentials-id",
				apiValue:   "credentials-id",
			},
		}

		for _, tc := range testCase {
			t.Run(tc.name, func(t *testing.T) {
				r := require.New(t)
				mockctrl := gomock.NewController(t)
				mockClient := mock_sdk.NewMockClientInterface(mockctrl)
				provider := &ProviderConfig{
					api: &sdk.ClientWithResponses{
						ClientInterface: mockClient,
					},
				}
				roleARNBefore := "dummy-roleARN"

				body := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"credentialsId": "%s"}`, tc.apiValue))))
				mockClient.EXPECT().
					ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

				resource := resourceEKSCluster()

				val := cty.ObjectVal(map[string]cty.Value{})
				state := terraform.NewInstanceStateShimmedFromValue(val, 0)
				state.ID = clusterID
				state.Attributes[FieldClusterCredentialsId] = tc.stateValue
				state.Attributes[FieldEKSClusterAssumeRoleArn] = roleARNBefore

				data := resource.Data(state)
				result := resource.ReadContext(ctx, data, provider)
				r.Nil(result)
				r.False(result.HasError())

				roleARNAfter := data.Get(FieldEKSClusterAssumeRoleArn)

				r.Equal(roleARNBefore, roleARNAfter)
				r.NotEmpty(roleARNAfter)
			})
		}
	})

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
    "dnsClusterIp": "10.100.100.1",
	"imdsV1": true
  },
  "sshPublicKey": "key-123",
  "clusterNameId": "eks-cluster-b6bfc074",
  "private": true
}`)))
	mockClient.EXPECT().
		ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceEKSCluster()

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
	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gk3d"
	ctx := context.Background()

	t.Run("resource update error generic propagated", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_sdk.NewMockClientInterface(mockctrl)

		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		mockClient.EXPECT().
			ExternalClusterAPIUpdateCluster(gomock.Any(), clusterID, gomock.Any(), gomock.Any()).
			Return(&http.Response{StatusCode: 400, Body: io.NopCloser(bytes.NewBufferString(`{"message":"Bad Request", "fieldViolations":[{"field":"credentials","description":"error"}]}`)), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		resource := resourceEKSCluster()

		raw := make(map[string]interface{})
		raw[FieldEKSClusterAssumeRoleArn] = "something"

		data := schema.TestResourceDataRaw(t, resource.Schema, raw)
		_ = data.Set(FieldEKSClusterAssumeRoleArn, "creds")
		data.SetId(clusterID)
		result := resource.UpdateContext(ctx, data, provider)
		r.NotNil(result)
		r.True(result.HasError())
		r.Equal("updating cluster configuration: expected status code 200, received: status=400 body={\"message\":\"Bad Request\", \"fieldViolations\":[{\"field\":\"credentials\",\"description\":\"error\"}]}", result[0].Summary)
	})

	t.Run("credentials_id special handling", func(t *testing.T) {
		t.Run("on successful update, should avoid drift on the read", func(t *testing.T) {
			r := require.New(t)
			mockctrl := gomock.NewController(t)
			mockClient := mock_sdk.NewMockClientInterface(mockctrl)
			provider := &ProviderConfig{
				api: &sdk.ClientWithResponses{
					ClientInterface: mockClient,
				},
			}

			credentialsIDAfterUpdate := "after-update-credentialsid"
			roleARN := "aws-role"
			updateResponse := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"credentialsId": "%s"}`, credentialsIDAfterUpdate))))
			readResponse := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"credentialsId": "%s"}`, credentialsIDAfterUpdate))))
			mockClient.EXPECT().
				ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
				Return(&http.Response{StatusCode: 200, Body: readResponse, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
			mockClient.EXPECT().
				ExternalClusterAPIUpdateCluster(gomock.Any(), clusterID, gomock.Any()).
				Return(&http.Response{StatusCode: 200, Body: updateResponse, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

			awsResource := resourceEKSCluster()

			diff := map[string]any{
				FieldEKSClusterAssumeRoleArn: roleARN,
				FieldClusterCredentialsId:    "before-update-credentialsid",
			}
			data := schema.TestResourceDataRaw(t, awsResource.Schema, diff)
			data.SetId(clusterID)
			diagnostics := awsResource.UpdateContext(ctx, data, provider)

			r.Empty(diagnostics)

			r.Equal(credentialsIDAfterUpdate, data.Get(FieldClusterCredentialsId))
			r.Equal(roleARN, data.Get(FieldEKSClusterAssumeRoleArn))
		})

		t.Run("on failed update, should overwrite credentialsID", func(t *testing.T) {
			r := require.New(t)
			mockctrl := gomock.NewController(t)
			mockClient := mock_sdk.NewMockClientInterface(mockctrl)
			provider := &ProviderConfig{
				api: &sdk.ClientWithResponses{
					ClientInterface: mockClient,
				},
			}

			mockClient.EXPECT().
				ExternalClusterAPIUpdateCluster(gomock.Any(), clusterID, gomock.Any()).
				Return(&http.Response{StatusCode: 400, Body: http.NoBody}, nil)

			awsResource := resourceEKSCluster()

			credentialsID := "credentialsID-before-updates"
			diff := map[string]any{
				FieldClusterCredentialsId: credentialsID,
			}
			data := schema.TestResourceDataRaw(t, awsResource.Schema, diff)
			data.SetId(clusterID)
			diagnostics := awsResource.UpdateContext(ctx, data, provider)

			r.NotEmpty(diagnostics)

			valueAfter := data.Get(FieldClusterCredentialsId)
			r.NotEqual(credentialsID, valueAfter)
			r.Contains(valueAfter, "drift")
		})
	})

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
	newClusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	body := bytes.NewBufferString(`{
  "id": "` + newClusterId + `",
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
    "dnsClusterIp": "10.100.100.1",
	"imdsV1": true
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

	resource := resourceEKSCluster()

	raw := make(map[string]interface{})
	raw[FieldEKSClusterAssumeRoleArn] = "something"

	data := schema.TestResourceDataRaw(t, resource.Schema, raw)
	_ = data.Set(FieldEKSClusterAssumeRoleArn, "creds")
	data.SetId(clusterId)
	result := resource.UpdateContext(ctx, data, provider)
	r.Nil(result)
}
