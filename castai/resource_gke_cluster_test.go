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

func TestGKEClusterResourceReadContext(t *testing.T) {
	ctx := context.Background()
	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gke"

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
	"projectId": "project-id"
  },
  "clusterNameId": "gke-cluster-b6bfc074"
}`)))
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
			Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		resource := resourceGKECluster()

		val := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(val, 0)
		state.ID = clusterID
		state.Attributes[FieldClusterCredentialsId] = "9b8d0456-177b-4a3d-b162-e68030d65GKE" // Avoid drift detection

		data := resource.Data(state)
		result := resource.ReadContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal(`ID = b6bfc074-a267-400f-b8f1-db0850c36gke
credentials_id = 9b8d0456-177b-4a3d-b162-e68030d65GKE
location = eu-central-1
name = gke-cluster
organization_id = 2836f775-aaaa-eeee-bbbb-3d3c29512GKE
project_id = project-id
Tainted = false
`, data.State().String())
	})

	t.Run("on credentials drift, changes credentials_json to trigger drift and re-apply", func(t *testing.T) {
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
				credentialsBeforeRead := "dummy-credentials"

				body := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"credentialsId": "%s"}`, tc.apiValue))))
				mockClient.EXPECT().
					ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

				gkeResource := resourceGKECluster()

				val := cty.ObjectVal(map[string]cty.Value{})
				state := terraform.NewInstanceStateShimmedFromValue(val, 0)
				state.ID = clusterID
				state.Attributes[FieldClusterCredentialsId] = tc.stateValue
				state.Attributes[FieldGKEClusterCredentials] = credentialsBeforeRead

				data := gkeResource.Data(state)
				result := gkeResource.ReadContext(ctx, data, provider)
				r.Nil(result)
				r.False(result.HasError())

				credentialsAfterRead := data.Get(FieldGKEClusterCredentials)

				r.NotEqual(credentialsBeforeRead, credentialsAfterRead)
				r.NotEmpty(credentialsAfterRead)
				r.Contains(credentialsAfterRead, "drift")
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

				credentialsBeforeRead := "dummy-credentials"

				body := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"credentialsId": "%s"}`, tc.apiValue))))
				mockClient.EXPECT().
					ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

				gkeResource := resourceGKECluster()

				val := cty.ObjectVal(map[string]cty.Value{})
				state := terraform.NewInstanceStateShimmedFromValue(val, 0)
				state.ID = clusterID
				state.Attributes[FieldClusterCredentialsId] = tc.stateValue
				state.Attributes[FieldGKEClusterCredentials] = credentialsBeforeRead

				data := gkeResource.Data(state)
				result := gkeResource.ReadContext(ctx, data, provider)
				r.Nil(result)
				r.False(result.HasError())

				credentialsAfterRead := data.Get(FieldGKEClusterCredentials)

				r.Equal(credentialsBeforeRead, credentialsAfterRead)
				r.NotEmpty(credentialsAfterRead)
			})
		}
	})
}

func TestGKEClusterResourceReadContextArchived(t *testing.T) {
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
	"projectId": "project-id"
  },
  "sshPublicKey": "key-123",
  "clusterNameId": "gke-cluster-b6bfc074",
  "private": true
}`)))
	mockClient.EXPECT().
		ExternalClusterAPIGetCluster(gomock.Any(), clusterId).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceGKECluster()

	val := cty.ObjectVal(map[string]cty.Value{})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = clusterId

	data := resource.Data(state)
	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Equal(`<not created>`, data.State().String())
}

func TestGKEClusterResourceUpdate(t *testing.T) {
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

		resource := resourceGKECluster()

		raw := make(map[string]interface{})
		raw[FieldGKEClusterCredentials] = "something"

		data := schema.TestResourceDataRaw(t, resource.Schema, raw)
		_ = data.Set(FieldGKEClusterCredentials, "creds")
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
			googleCredentials := "google-creds"
			updateResponse := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"credentialsId": "%s"}`, credentialsIDAfterUpdate))))
			readResponse := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"credentialsId": "%s"}`, credentialsIDAfterUpdate))))
			mockClient.EXPECT().
				ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
				Return(&http.Response{StatusCode: 200, Body: readResponse, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
			mockClient.EXPECT().
				ExternalClusterAPIUpdateCluster(gomock.Any(), clusterID, gomock.Any()).
				Return(&http.Response{StatusCode: 200, Body: updateResponse, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

			gkeResource := resourceGKECluster()

			diff := map[string]any{
				FieldGKEClusterCredentials: googleCredentials,
				FieldClusterCredentialsId:  "before-update-credentialsid",
			}
			data := schema.TestResourceDataRaw(t, gkeResource.Schema, diff)
			data.SetId(clusterID)
			diagnostics := gkeResource.UpdateContext(ctx, data, provider)

			r.Empty(diagnostics)

			r.Equal(credentialsIDAfterUpdate, data.Get(FieldClusterCredentialsId))
			r.Equal(googleCredentials, data.Get(FieldGKEClusterCredentials))
		})

		t.Run("on failed update, should overwrite credentialsID to force drift on next read", func(t *testing.T) {
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

			gkeResource := resourceGKECluster()

			credentialsID := "credentialsID-before-updates"
			diff := map[string]any{
				FieldClusterCredentialsId: credentialsID,
			}
			data := schema.TestResourceDataRaw(t, gkeResource.Schema, diff)
			data.SetId(clusterID)
			diagnostics := gkeResource.UpdateContext(ctx, data, provider)

			r.NotEmpty(diagnostics)

			valueAfter := data.Get(FieldClusterCredentialsId)
			r.NotEqual(credentialsID, valueAfter)
			r.Contains(valueAfter, "drift")
		})
	})
}
