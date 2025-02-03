package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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
		state := terraform.NewInstanceStateShimmedFromValue(val, 0)
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
		state := terraform.NewInstanceStateShimmedFromValue(val, 0)
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
				state := terraform.NewInstanceStateShimmedFromValue(val, 0)
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
				state := terraform.NewInstanceStateShimmedFromValue(val, 0)
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

func TestAccResourceAKSCluster(t *testing.T) {
	rName := fmt.Sprintf("%v-aks-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_aks_cluster.test"
	clusterName := "core-tf-acc"
	resourceGroupName := "core-tf-acc"
	nodeResourceGroupName := "core-tf-acc-ng"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		// Destroy of the cluster is not working properly. Cluster wasn't full onboarded and it's getting destroyed.
		// https://castai.atlassian.net/browse/CORE-2868 should solve the issue
		//CheckDestroy:      testAccCheckAKSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAKSClusterConfig(rName, clusterName, resourceGroupName, nodeResourceGroupName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", clusterName),
					resource.TestCheckResourceAttrSet(resourceName, "credentials_id"),
					resource.TestCheckResourceAttr(resourceName, "region", "westeurope"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_token"),
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

func testAccAKSClusterConfig(rName string, clusterName string, resourceGroupName, nodeResourceGroup string) string {
	return ConfigCompose(testAccAzureConfig(rName, resourceGroupName, nodeResourceGroup), fmt.Sprintf(`
resource "castai_aks_cluster" "test" {
  name            = %[1]q

  region          = "westeurope"
  subscription_id = data.azurerm_subscription.current.subscription_id 
  tenant_id       = data.azurerm_subscription.current.tenant_id
  client_id       = azuread_application.castai.application_id
  client_secret   = azuread_application_password.castai.value
  node_resource_group        = %[2]q

}

`, clusterName, nodeResourceGroup))
}

func testAccAzureConfig(rName, rgName, ngName string) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

data "azurerm_subscription" "current" {}

data "azurerm_subnet" "internal" {
  name                 =  "internal"
  virtual_network_name = "%[2]s-network"
  resource_group_name  = %[2]q 
}

provider "azuread" {}

// Azure RM
resource "azurerm_role_definition" "castai" {
  name            = %[1]q
  description = "Role used by CAST AI"

  scope = "/subscriptions/${data.azurerm_subscription.current.subscription_id}/resourceGroups/%[2]s"

  permissions {
    actions = [
      "Microsoft.Compute/*/read",
      "Microsoft.Compute/virtualMachines/*",
      "Microsoft.Compute/virtualMachineScaleSets/*",
      "Microsoft.Compute/disks/write",
      "Microsoft.Compute/disks/delete",
      "Microsoft.Compute/disks/beginGetAccess/action",
      "Microsoft.Compute/galleries/write",
      "Microsoft.Compute/galleries/delete",
      "Microsoft.Compute/galleries/images/write",
      "Microsoft.Compute/galleries/images/delete",
      "Microsoft.Compute/galleries/images/versions/write",
      "Microsoft.Compute/galleries/images/versions/delete",
      "Microsoft.Compute/snapshots/write",
      "Microsoft.Compute/snapshots/delete",
      "Microsoft.Network/*/read",
      "Microsoft.Network/networkInterfaces/write",
      "Microsoft.Network/networkInterfaces/delete",
      "Microsoft.Network/networkInterfaces/join/action",
      "Microsoft.Network/networkSecurityGroups/join/action",
      "Microsoft.Network/publicIPAddresses/write",
      "Microsoft.Network/publicIPAddresses/delete",
      "Microsoft.Network/publicIPAddresses/join/action",
      "Microsoft.Network/virtualNetworks/subnets/join/action",
      "Microsoft.Network/virtualNetworks/subnets/write",
      "Microsoft.Network/applicationGateways/backendhealth/action",
      "Microsoft.Network/applicationGateways/backendAddressPools/join/action",
      "Microsoft.Network/applicationSecurityGroups/joinIpConfiguration/action",
      "Microsoft.Network/loadBalancers/backendAddressPools/write",
      "Microsoft.Network/loadBalancers/backendAddressPools/join/action",
      "Microsoft.ContainerService/*/read",
      "Microsoft.ContainerService/managedClusters/start/action",
      "Microsoft.ContainerService/managedClusters/stop/action",
      "Microsoft.ContainerService/managedClusters/runCommand/action",
      "Microsoft.ContainerService/managedClusters/agentPools/*",
      "Microsoft.Resources/*/read",
      "Microsoft.Resources/tags/write",
      "Microsoft.Authorization/locks/read",
      "Microsoft.Authorization/roleAssignments/read",
      "Microsoft.Authorization/roleDefinitions/read",
      "Microsoft.ManagedIdentity/userAssignedIdentities/assign/action"
    ]
    not_actions = []
  }

  assignable_scopes = [
    "/subscriptions/${data.azurerm_subscription.current.subscription_id}/resourceGroups/%[2]s",
    "/subscriptions/${data.azurerm_subscription.current.subscription_id}/resourceGroups/%[3]s"
  ]
}


resource "azurerm_role_assignment" "castai_resource_group" {
  principal_id       = azuread_service_principal.castai.id
  role_definition_id = azurerm_role_definition.castai.role_definition_resource_id

  scope = "/subscriptions/${data.azurerm_subscription.current.subscription_id}/resourceGroups/%[2]s"
}

resource "azurerm_role_assignment" "castai_node_resource_group" {
  principal_id       = azuread_service_principal.castai.id
  role_definition_id = azurerm_role_definition.castai.role_definition_resource_id

  scope = "/subscriptions/${data.azurerm_subscription.current.subscription_id}/resourceGroups/%[3]s"
}

// Azure AD

data "azuread_client_config" "current" {}

resource "azuread_application" "castai" {
  display_name = %[1]q
}

resource "azuread_application_password" "castai" {
  application_object_id = azuread_application.castai.object_id
}

resource "azuread_service_principal" "castai" {
  application_id               = azuread_application.castai.application_id
  app_role_assignment_required = false
  owners                       = [data.azuread_client_config.current.object_id]
}

`, rName, rgName, ngName)
}
