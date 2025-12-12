package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdkterraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestAKSClusterResourceReadContext(t *testing.T) {
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
				  "name": "aks-cluster",
				  "organizationId": "2836f775-aaaa-eeee-bbbb-3d3c29512692",
				  "credentialsId": "9b8d0456-177b-4a3d-b162-e68030d656aa",
				  "createdAt": "2022-01-27T19:03:31.570829Z",
				  "status": "ready",
				  "agentSnapshotReceivedAt": "2022-03-21T10:33:56.192020Z",
				  "agentStatus": "online",
				  "providerType": "aks",
				  "aks": {
					"maxPodsPerNode": 100,
					"networkPlugin": "calico",
					"nodeResourceGroup": "ng",
					"region": "westeurope",
					"subscriptionId": "subID",
					"httpProxyConfig": {
						  "httpProxy": "http-proxy",
						  "httpsProxy": "https-proxy",
						  "noProxy": [
							"domain1", "domain2"
						  ]
						}
				  },
				  "clusterNameId": "aks-cluster-b6bfc074",
				  "private": true
				}`)))
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
			Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		aksResource := resourceAKSCluster()

		val := cty.ObjectVal(map[string]cty.Value{})
		state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
		state.ID = clusterID
		// If local credentials don't match remote, drift detection would trigger.
		// If local state has no credentials but remote has them, then the drift does exist so - there is separate test for that.
		state.Attributes[FieldClusterCredentialsId] = "9b8d0456-177b-4a3d-b162-e68030d656aa"

		data := aksResource.Data(state)
		result := aksResource.ReadContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal(`ID = b6bfc074-a267-400f-b8f1-db0850c369b1
credentials_id = 9b8d0456-177b-4a3d-b162-e68030d656aa
http_proxy_config.# = 1
http_proxy_config.0.http_proxy = http-proxy
http_proxy_config.0.https_proxy = https-proxy
http_proxy_config.0.no_proxy.# = 2
http_proxy_config.0.no_proxy.0 = domain1
http_proxy_config.0.no_proxy.1 = domain2
organization_id = 2836f775-aaaa-eeee-bbbb-3d3c29512692
region = westeurope
Tainted = false
`, data.State().String())
	})

	t.Run("when proxy config is reset remotely, shows drift", func(t *testing.T) {
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
				  "name": "aks-cluster",
				  "organizationId": "2836f775-aaaa-eeee-bbbb-3d3c29512692",
				  "credentialsId": "9b8d0456-177b-4a3d-b162-e68030d656aa",
				  "createdAt": "2022-01-27T19:03:31.570829Z",
				  "status": "ready",
				  "agentSnapshotReceivedAt": "2022-03-21T10:33:56.192020Z",
				  "agentStatus": "online",
				  "providerType": "aks",
				  "aks": {
					"maxPodsPerNode": 100,
					"networkPlugin": "calico",
					"nodeResourceGroup": "ng",
					"region": "westeurope",
					"subscriptionId": "subID"
				  },
				  "clusterNameId": "aks-cluster-b6bfc074",
				  "private": true
				}`)))
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
			Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		aksResource := resourceAKSCluster()

		val := cty.ObjectVal(map[string]cty.Value{
			FieldAKSHttpProxyConfig: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldAKSHttpProxyDestination: cty.StringVal("http-proxy"),
				}),
			}),
		})
		state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
		state.ID = clusterID
		// If local credentials don't match remote, drift detection would trigger.
		// If local state has no credentials but remote has them, then the drift does exist so - there is separate test for that.
		state.Attributes[FieldClusterCredentialsId] = "9b8d0456-177b-4a3d-b162-e68030d656aa"

		data := aksResource.Data(state)
		result := aksResource.ReadContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		// Note: even if the array for proxy is nil, terraform saves the length so we still have _some_ state about it below.
		r.Equal(`ID = b6bfc074-a267-400f-b8f1-db0850c369b1
credentials_id = 9b8d0456-177b-4a3d-b162-e68030d656aa
http_proxy_config.# = 0
organization_id = 2836f775-aaaa-eeee-bbbb-3d3c29512692
region = westeurope
Tainted = false
`, data.State().String())
	})

	t.Run("on credentials drift, changes client_id to trigger drift and re-apply", func(t *testing.T) {
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
				clientIDBeforeRead := "dummy-client-id"

				body := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"credentialsId": "%s"}`, tc.apiValue))))
				mockClient.EXPECT().
					ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

				aksResource := resourceAKSCluster()

				val := cty.ObjectVal(map[string]cty.Value{})
				state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
				state.ID = clusterID
				state.Attributes[FieldClusterCredentialsId] = tc.stateValue
				state.Attributes[FieldAKSClusterClientID] = clientIDBeforeRead

				data := aksResource.Data(state)
				result := aksResource.ReadContext(ctx, data, provider)
				r.Nil(result)
				r.False(result.HasError())

				clientIDAfterRead := data.Get(FieldAKSClusterClientID)

				r.NotEqual(clientIDBeforeRead, clientIDAfterRead)
				r.NotEmpty(clientIDAfterRead)
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
				clientIDBeforeRead := "dummy-client-id"

				body := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"credentialsId": "%s"}`, tc.apiValue))))
				mockClient.EXPECT().
					ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
					Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

				aksResource := resourceAKSCluster()

				val := cty.ObjectVal(map[string]cty.Value{})
				state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
				state.ID = clusterID
				state.Attributes[FieldClusterCredentialsId] = tc.stateValue
				state.Attributes[FieldAKSClusterClientID] = clientIDBeforeRead

				data := aksResource.Data(state)
				result := aksResource.ReadContext(ctx, data, provider)
				r.Nil(result)
				r.False(result.HasError())

				clientIDAfterRead := data.Get(FieldAKSClusterClientID)

				r.Equal(clientIDBeforeRead, clientIDAfterRead)
				r.NotEmpty(clientIDAfterRead)
			})
		}
	})
}

func TestAKSClusterResourceUpdateContext(t *testing.T) {
	clusterID := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	ctx := context.Background()

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
			clientID := "clientID"
			updateResponse := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"credentialsId": "%s"}`, credentialsIDAfterUpdate))))
			readResponse := io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"credentialsId": "%s"}`, credentialsIDAfterUpdate))))
			mockClient.EXPECT().
				ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
				Return(&http.Response{StatusCode: 200, Body: readResponse, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
			mockClient.EXPECT().
				ExternalClusterAPIUpdateCluster(gomock.Any(), clusterID, gomock.Any()).
				Return(&http.Response{StatusCode: 200, Body: updateResponse, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

			aksResource := resourceAKSCluster()

			diff := map[string]any{
				FieldAKSClusterClientID:   clientID,
				FieldClusterCredentialsId: "before-update-credentialsid",
			}
			data := schema.TestResourceDataRaw(t, aksResource.Schema, diff)
			data.SetId(clusterID)
			diagnostics := aksResource.UpdateContext(ctx, data, provider)

			r.Empty(diagnostics)

			r.Equal(credentialsIDAfterUpdate, data.Get(FieldClusterCredentialsId))
			r.Equal(clientID, data.Get(FieldAKSClusterClientID))
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

			aksResource := resourceAKSCluster()

			credentialsID := "credentialsID-before-updates"
			diff := map[string]any{
				FieldClusterCredentialsId: credentialsID,
			}
			data := schema.TestResourceDataRaw(t, aksResource.Schema, diff)
			data.SetId(clusterID)
			diagnostics := aksResource.UpdateContext(ctx, data, provider)

			r.NotEmpty(diagnostics)

			valueAfter := data.Get(FieldClusterCredentialsId)
			r.NotEqual(credentialsID, valueAfter)
			r.Contains(valueAfter, "drift")
		})
	})

	t.Run("Saves proxy settings correctly", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_sdk.NewMockClientInterface(mockctrl)
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		expectedHttpProxySettings := &sdk.ExternalClusterAPIUpdateClusterJSONRequestBody{
			Aks: &sdk.ExternalclusterV1UpdateAKSClusterParams{
				HttpProxyConfig: &sdk.ExternalclusterV1HttpProxyConfig{
					HttpProxy:  lo.ToPtr("http-proxy"),
					HttpsProxy: lo.ToPtr("https-proxy"),
					NoProxy:    lo.ToPtr([]string{"domain1", "domain2"}),
				},
			},
		}
		jsonHttpProxy, err := json.Marshal(expectedHttpProxySettings)
		r.NoError(err)

		readResponse := io.NopCloser(bytes.NewReader([]byte(`{"credentialsId": ""}`)))
		updateResponse := io.NopCloser(bytes.NewReader(jsonHttpProxy))
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
			Return(&http.Response{StatusCode: 200, Body: readResponse, Header: map[string][]string{"Content-Type": {"json"}}}, nil)
		mockClient.EXPECT().
			ExternalClusterAPIUpdateCluster(gomock.Any(), clusterID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, body sdk.ExternalClusterAPIUpdateClusterJSONRequestBody) (*http.Response, error) {
				r.Equal(expectedHttpProxySettings.Aks.HttpProxyConfig.HttpsProxy, body.Aks.HttpProxyConfig.HttpsProxy)
				r.Equal(expectedHttpProxySettings.Aks.HttpProxyConfig.HttpProxy, body.Aks.HttpProxyConfig.HttpProxy)
				r.ElementsMatch(*expectedHttpProxySettings.Aks.HttpProxyConfig.NoProxy, *body.Aks.HttpProxyConfig.NoProxy)
				return &http.Response{StatusCode: 200, Body: updateResponse, Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		aksResource := resourceAKSCluster()

		diff := map[string]any{
			FieldAKSHttpProxyConfig: []any{
				map[string]any{
					FieldAKSHttpProxyDestination:  "http-proxy",
					FieldAKSHttpsProxyDestination: "https-proxy",
					FieldAKSNoProxyDestinations:   []any{"domain1", "domain2"},
				},
			},
		}
		data := schema.TestResourceDataRaw(t, aksResource.Schema, diff)
		data.SetId(clusterID)
		diagnostics := aksResource.UpdateContext(ctx, data, provider)

		r.Empty(diagnostics)

		// Validate that the settings are populated in state as expected.
		stateProxyConfig := data.Get(FieldAKSHttpProxyConfig).([]any)
		r.NotNil(stateProxyConfig)
		r.Len(stateProxyConfig, 1)
		proxyConfigElem := stateProxyConfig[0].(map[string]any)
		r.Equal(proxyConfigElem[FieldAKSHttpProxyDestination], *expectedHttpProxySettings.Aks.HttpProxyConfig.HttpProxy)
		r.Equal(proxyConfigElem[FieldAKSHttpsProxyDestination], *expectedHttpProxySettings.Aks.HttpProxyConfig.HttpsProxy)
		r.ElementsMatch(proxyConfigElem[FieldAKSNoProxyDestinations], *expectedHttpProxySettings.Aks.HttpProxyConfig.NoProxy)
	})
}

func TestAccAKS_ResourceAKSCluster(t *testing.T) {
	rName := fmt.Sprintf("%v-node-cfg-aks-%v", ResourcePrefix, acctest.RandString(8))
	const (
		clusterResourceName  = "castai_aks_cluster.test"
		clusterName          = "terraform-tests-december-2025"
		resourceGroupName    = "terraform-tests-december-2025"
		nodeConfResourceName = "castai_node_configuration.test"
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		// Destroy of the cluster is not working properly. Cluster wasn't full onboarded and it's getting destroyed.
		// https://castai.atlassian.net/browse/CORE-2868 should solve the issue
		//CheckDestroy:      testAccCheckAKSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAKSWithClientSecretConfig(clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(clusterResourceName, "name", clusterName),
					resource.TestCheckResourceAttrSet(clusterResourceName, "credentials_id"),
					resource.TestCheckResourceAttr(clusterResourceName, "region", "westeurope"),
					resource.TestCheckResourceAttrSet(clusterResourceName, "cluster_token"),
				),
			},
			{
				Config: testAccAKSWithFederationIDConfig(clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(clusterResourceName, "name", clusterName),
					resource.TestCheckResourceAttrSet(clusterResourceName, "credentials_id"),
					resource.TestCheckResourceAttr(clusterResourceName, "region", "westeurope"),
					resource.TestCheckResourceAttrSet(clusterResourceName, "cluster_token"),
				),
			},
			{
				Config: testAccAKSNodeConfigurationConfig(rName, clusterName, resourceGroupName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(nodeConfResourceName, "name", rName),
					resource.TestCheckResourceAttr(nodeConfResourceName, "disk_cpu_ratio", "35"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "min_disk_size", "122"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.max_pods_per_node", "31"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.aks_image_family", "ubuntu"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "eks.#", "0"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "kops.#", "0"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "gke.#", "0"),
				),
			},
			{
				Config: testAccAKSNodeConfigurationUpdated(rName, clusterName, resourceGroupName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(nodeConfResourceName, "name", rName),
					resource.TestCheckResourceAttr(nodeConfResourceName, "disk_cpu_ratio", "0"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "min_disk_size", "121"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.max_pods_per_node", "32"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.aks_image_family", "azure-linux"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.ephemeral_os_disk.0.placement", "cacheDisk"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.ephemeral_os_disk.0.cache", "ReadOnly"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.loadbalancers.0.name", "test-lb"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.loadbalancers.0.ip_based_backend_pools.0.name", "test"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.network_security_group", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/networkSecurityGroups/test-nsg"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.application_security_groups.0", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/applicationSecurityGroups/test-asg"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.public_ip.0.public_ip_prefix", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/publicIPAddresses/test-ip"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.public_ip.0.tags.FirstPartyUsage", "something"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "aks.0.public_ip.0.idle_timeout_in_minutes", "10"),
					resource.TestCheckResourceAttrSet(nodeConfResourceName, "aks.0.pod_subnet_id"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "eks.#", "0"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "kops.#", "0"),
					resource.TestCheckResourceAttr(nodeConfResourceName, "gke.#", "0"),
				),
			},
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"azurerm": {
				Source:            "hashicorp/azurerm",
				VersionConstraint: "~> 3.7.0",
			},
			"azuread": {
				Source:            "hashicorp/azuread",
				VersionConstraint: "~> 2.22.0",
			},
		},
	})
}

func testAccAKSWithClientSecretConfig(clusterName string) string {
	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	tenantID := os.Getenv("ARM_TENANT_ID")
	clientID := os.Getenv("ARM_CLIENT_ID")
	clientSecret := os.Getenv("ARM_CLIENT_SECRET")
	return fmt.Sprintf(`
resource "castai_aks_cluster" "test" {
  name            = %[1]q

  region          = "westeurope"
  subscription_id = %[2]q
  tenant_id       = %[3]q
  client_id       = %[4]q
  client_secret   = %[5]q
  node_resource_group = "%[1]s-ng"

}

`, clusterName, subscriptionID, tenantID, clientID, clientSecret)
}

func testAccAKSWithFederationIDConfig(clusterName string) string {
	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	federationID := os.Getenv("AZURE_TF_ACCEPTANCE_TEST_FEDERATION_ID")
	tenantID := os.Getenv("AZURE_TF_ACCEPTANCE_TEST_FEDERATION_TENANT_ID")
	clientID := os.Getenv("AZURE_TF_ACCEPTANCE_TEST_FEDERATION_CLIENT_ID")

	return fmt.Sprintf(`
resource "castai_aks_cluster" "test" {
  name = %[3]q

  region              = "westeurope"
  subscription_id     = %[1]q
  tenant_id           = %[4]q
  client_id           = %[5]q
  federation_id       = %[2]q
  node_resource_group = "%[3]s-ng"
}
`, subscriptionID, federationID, clusterName, tenantID, clientID)
}
