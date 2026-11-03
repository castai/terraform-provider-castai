package castai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
castai_service_account_email = 
credentials_id = 9b8d0456-177b-4a3d-b162-e68030d65GKE
location = eu-central-1
name = gke-cluster
organization_id = 2836f775-aaaa-eeee-bbbb-3d3c29512GKE
project_id = project-id
target_service_account_email = 
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

func newProvider(mockClient *mock_sdk.MockClientInterface) *ProviderConfig {
	return &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}
}

func TestGKEClusterCredentialsSchema(t *testing.T) {
	base := map[string]interface{}{
		FieldGKEClusterName:      "cluster",
		FieldGKEClusterProjectId: "project",
		FieldGKEClusterLocation:  "europe-central2-a",
	}
	withFields := func(target, cast string) map[string]interface{} {
		raw := map[string]interface{}{}
		for k, v := range base {
			raw[k] = v
		}
		raw[FieldGKECredentialsTargetServiceAccountEmail] = target
		raw[FieldGKECredentialsCastaiServiceAccountEmail] = cast
		return raw
	}

	tests := []struct {
		name       string
		raw        map[string]interface{}
		wantErr    bool
		errMessage string
	}{
		{
			name: "valid impersonation",
			raw: withFields(
				"target-sa@project.iam.gserviceaccount.com",
				"cast-sa@cast-project.iam.gserviceaccount.com",
			),
		},
		{
			name: "legacy empty object valid",
			raw: func() map[string]interface{} {
				raw := map[string]interface{}{}
				for k, v := range base {
					raw[k] = v
				}
				raw[FieldGKEClusterCredentials] = "{}"
				return raw
			}(),
		},
		{
			name: "impersonation fields conflict with credentials_json",
			raw: func() map[string]interface{} {
				raw := withFields(
					"target-sa@project.iam.gserviceaccount.com",
					"cast-sa@cast-project.iam.gserviceaccount.com",
				)
				raw[FieldGKEClusterCredentials] = "{}"
				return raw
			}(),
			wantErr:    true,
			errMessage: "conflicts with",
		},
		{
			name: "target without castai service account",
			raw: func() map[string]interface{} {
				raw := map[string]interface{}{}
				for k, v := range base {
					raw[k] = v
				}
				raw[FieldGKECredentialsTargetServiceAccountEmail] = "target-sa@project.iam.gserviceaccount.com"
				return raw
			}(),
			wantErr:    true,
			errMessage: "must be specified",
		},
	}

	resource := resourceGKECluster()
	require.NoError(t, resource.InternalValidate(nil, true))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := terraform.NewResourceConfigRaw(tt.raw)
			diags := resource.Validate(cfg)
			if !tt.wantErr {
				require.False(t, diags.HasError(), "unexpected errors: %v", diags)
				return
			}
			require.True(t, diags.HasError())
			var found string
			for _, d := range diags {
				if d.Severity == diag.Error && strings.Contains(d.Summary+" "+d.Detail, tt.errMessage) {
					found = d.Summary + " " + d.Detail
					break
				}
			}
			require.NotEmpty(t, found, "expected error containing %q in %v", tt.errMessage, diags)
		})
	}
}

func TestGKEClusterResourceCreate(t *testing.T) {
	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gke"
	ctx := context.Background()

	targetSA := "target-sa@project.iam.gserviceaccount.com"
	castSA := "cast-sa@cast-project.iam.gserviceaccount.com"
	noneConfig := map[string]interface{}{
		FieldGKEClusterName:      "cluster",
		FieldGKEClusterProjectId: "project",
		FieldGKEClusterLocation:  "europe-central2-a",
	}
	impersonationConfig := map[string]interface{}{
		FieldGKEClusterName:                          "cluster",
		FieldGKEClusterProjectId:                     "project",
		FieldGKEClusterLocation:                      "europe-central2-a",
		FieldGKECredentialsTargetServiceAccountEmail: targetSA,
		FieldGKECredentialsCastaiServiceAccountEmail: castSA,
	}
	clusterBody := func(clientSA string) string {
		return fmt.Sprintf(`{"id":%q,"status":"ready","agentStatus":"online","organizationId":"org","credentialsId":"cid","gke":{"clusterName":"cluster","region":"europe-central2","location":"europe-central2-a","projectId":"project","targetServiceAccountEmail":%q}}`, clusterID, clientSA)
	}

	registerClient := func(mockClient *mock_sdk.MockClientInterface) {
		mockClient.EXPECT().
			ExternalClusterAPIRegisterCluster(gomock.Any(), gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"` + clusterID + `"}`))),
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)
		mockClient.EXPECT().
			ExternalClusterAPICreateClusterToken(gomock.Any(), clusterID).
			Return(&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"token":"token-123"}`))),
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)
	}
	getCluster := func(mockClient *mock_sdk.MockClientInterface, clientSA string) {
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
			Return(&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(clusterBody(clientSA)))),
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)
	}

	tests := []struct {
		name         string
		raw          map[string]interface{}
		updateStatus int
		wantTargetSA string
		wantCastSA   string
		apiClientSA  string
		wantErr      string
	}{
		{
			name:         "create impersonation registers and updates cluster with impersonation emails",
			raw:          impersonationConfig,
			updateStatus: 200,
			wantTargetSA: targetSA,
			wantCastSA:   castSA,
			apiClientSA:  targetSA,
		},
		{
			name:         "create impersonation propagates cluster update error",
			raw:          impersonationConfig,
			updateStatus: 400,
			wantTargetSA: targetSA,
			wantCastSA:   castSA,
			wantErr:      "updating cluster configuration",
		},
		{
			name: "create without credentials skips update",
			raw:  noneConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			mockctrl := gomock.NewController(t)
			mockClient := mock_sdk.NewMockClientInterface(mockctrl)
			provider := newProvider(mockClient)

			registerClient(mockClient)
			if tt.updateStatus != 0 {
				mockClient.EXPECT().
					ExternalClusterAPIUpdateCluster(gomock.Any(), clusterID, gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, _ string, body sdk.ExternalClusterAPIUpdateClusterJSONRequestBody, _ ...sdk.RequestEditorFn) {
						r.NotNil(body.Gke)
						r.Equal(tt.wantTargetSA, toString(body.Gke.TargetServiceAccountEmail))
						r.Equal(tt.wantCastSA, toString(body.Gke.CastServiceAccountEmail))
					}).
					Return(&http.Response{
						StatusCode: tt.updateStatus,
						Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"` + clusterID + `","credentialsId":"cid"}`))),
						Header:     map[string][]string{"Content-Type": {"json"}},
					}, nil)
			}
			if tt.wantErr == "" {
				getCluster(mockClient, tt.apiClientSA)
			}

			resource := resourceGKECluster()
			data := schema.TestResourceDataRaw(t, resource.Schema, tt.raw)
			diags := resource.CreateContext(ctx, data, provider)

			if tt.wantErr != "" {
				r.True(diags.HasError())
				r.Contains(diags[0].Summary, tt.wantErr)
				return
			}
			r.False(diags.HasError())
			r.Equal(clusterID, data.Id())
			r.Equal("cid", data.Get(FieldClusterCredentialsId))
			r.Equal("token-123", data.Get(FieldClusterToken))
			r.Equal(tt.wantTargetSA, data.Get(FieldGKECredentialsTargetServiceAccountEmail))
		})
	}
}

func TestGKEClusterResourceUpdateCredentials(t *testing.T) {
	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gk3d"
	ctx := context.Background()

	targetSA := "target-sa@project.iam.gserviceaccount.com"
	castSA := "cast-sa@cast-project.iam.gserviceaccount.com"
	credentialsJSON := `{"type":"service_account"}`

	tests := []struct {
		name                string
		raw                 map[string]interface{}
		updateCredentialsID string
		wantCredentials     string
		wantTargetSA        string
		wantCastSA          string
		apiClientSA         string
	}{
		{
			name: "update impersonation calls UpdateCluster with impersonation emails",
			raw: map[string]interface{}{
				FieldGKECredentialsTargetServiceAccountEmail: targetSA,
				FieldGKECredentialsCastaiServiceAccountEmail: castSA,
			},
			updateCredentialsID: "after-update-cid",
			wantTargetSA:        targetSA,
			wantCastSA:          castSA,
			apiClientSA:         targetSA,
		},
		{
			name: "update legacy sends credentials json only",
			raw: map[string]interface{}{
				FieldGKEClusterCredentials: credentialsJSON,
			},
			updateCredentialsID: "legacy-cid",
			wantCredentials:     credentialsJSON,
		},
		{
			name: "switching to static credentials omits sticky impersonation emails from prior state",
			raw: map[string]interface{}{
				FieldGKEClusterCredentials:                   credentialsJSON,
				FieldGKECredentialsTargetServiceAccountEmail: targetSA,
				FieldGKECredentialsCastaiServiceAccountEmail: castSA,
			},
			updateCredentialsID: "switched-cid",
			wantCredentials:     credentialsJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			mockctrl := gomock.NewController(t)
			mockClient := mock_sdk.NewMockClientInterface(mockctrl)
			provider := newProvider(mockClient)

			mockClient.EXPECT().
				ExternalClusterAPIUpdateCluster(gomock.Any(), clusterID, gomock.Any(), gomock.Any()).
				Do(func(_ context.Context, _ string, body sdk.ExternalClusterAPIUpdateClusterJSONRequestBody, _ ...sdk.RequestEditorFn) {
					r.Equal(tt.wantCredentials, toString(body.Credentials))
					if tt.wantCredentials != "" {
						r.Nil(body.Gke)
						return
					}
					r.NotNil(body.Gke)
					r.Equal(tt.wantTargetSA, toString(body.Gke.TargetServiceAccountEmail))
					r.Equal(tt.wantCastSA, toString(body.Gke.CastServiceAccountEmail))
				}).
				Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"` + clusterID + `","credentialsId":"` + tt.updateCredentialsID + `"}`))),
					Header:     map[string][]string{"Content-Type": {"json"}},
				}, nil)
			clusterBody := fmt.Sprintf(`{"id":%q,"status":"ready","agentStatus":"online","organizationId":"org","credentialsId":%q,"gke":{"clusterName":"cluster","region":"europe-central2","location":"europe-central2-a","projectId":"project","targetServiceAccountEmail":%q}}`, clusterID, tt.updateCredentialsID, tt.apiClientSA)
			mockClient.EXPECT().
				ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
				Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader([]byte(clusterBody))),
					Header:     map[string][]string{"Content-Type": {"json"}},
				}, nil)

			gkeResource := resourceGKECluster()
			data := schema.TestResourceDataRaw(t, gkeResource.Schema, tt.raw)
			data.SetId(clusterID)
			diags := gkeResource.UpdateContext(ctx, data, provider)

			r.Empty(diags)
			r.Equal(tt.updateCredentialsID, data.Get(FieldClusterCredentialsId))
		})
	}
}

func TestGKEClusterResourceCredentialsDrift(t *testing.T) {
	ctx := context.Background()
	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gke"

	impersonationAttrs := map[string]string{
		FieldGKECredentialsTargetServiceAccountEmail: "cfg-sa@project.iam.gserviceaccount.com",
		FieldGKECredentialsCastaiServiceAccountEmail: "cast-sa@cast-project.iam.gserviceaccount.com",
		FieldClusterCredentialsId:                    "local-cid",
	}

	tests := []struct {
		name       string
		attributes map[string]string
		apiBody    string
		wantTarget string
		wantCast   string
	}{
		{
			name:       "impersonation credentials_id drift with unset API emails clears state emails",
			attributes: impersonationAttrs,
			apiBody:    `{"credentialsId":"remote-cid","gke":{"clusterName":"cluster","region":"europe-central2","location":"europe-central2-a","projectId":"project"}}`,
			wantTarget: "",
			wantCast:   "",
		},
		{
			name:       "legacy impersonation without config emails mirrors API emails into state",
			attributes: map[string]string{FieldClusterCredentialsId: "local-cid"},
			apiBody:    `{"credentialsId":"local-cid","gke":{"targetServiceAccountEmail":"api-sa@project.iam.gserviceaccount.com","castServiceAccountEmail":"api-cast@cast-project.iam.gserviceaccount.com"}}`,
			wantTarget: "api-sa@project.iam.gserviceaccount.com",
			wantCast:   "api-cast@cast-project.iam.gserviceaccount.com",
		},
		{
			name:       "impersonation emails mirrored from API override state",
			attributes: impersonationAttrs,
			apiBody:    `{"credentialsId":"remote-cid","gke":{"targetServiceAccountEmail":"api-sa@project.iam.gserviceaccount.com","castServiceAccountEmail":"api-cast@cast-project.iam.gserviceaccount.com"}}`,
			wantTarget: "api-sa@project.iam.gserviceaccount.com",
			wantCast:   "api-cast@cast-project.iam.gserviceaccount.com",
		},
		{
			name:       "impersonation target email mirrored from API",
			attributes: impersonationAttrs,
			apiBody:    `{"credentialsId":"local-cid","gke":{"targetServiceAccountEmail":"api-sa@project.iam.gserviceaccount.com"}}`,
			wantTarget: "api-sa@project.iam.gserviceaccount.com",
			wantCast:   "",
		},
		{
			name:       "impersonation matching API echo keeps state emails",
			attributes: impersonationAttrs,
			apiBody:    `{"credentialsId":"local-cid","gke":{"targetServiceAccountEmail":"cfg-sa@project.iam.gserviceaccount.com"}}`,
			wantTarget: "cfg-sa@project.iam.gserviceaccount.com",
			wantCast:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			mockctrl := gomock.NewController(t)
			mockClient := mock_sdk.NewMockClientInterface(mockctrl)
			provider := newProvider(mockClient)

			mockClient.EXPECT().
				ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
				Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader([]byte(tt.apiBody))),
					Header:     map[string][]string{"Content-Type": {"json"}},
				}, nil)

			resource := resourceGKECluster()
			val := cty.ObjectVal(map[string]cty.Value{})
			state := terraform.NewInstanceStateShimmedFromValue(val, 0)
			state.ID = clusterID
			state.Attributes = tt.attributes
			data := resource.Data(state)

			diags := resource.ReadContext(ctx, data, provider)

			r.False(diags.HasError())
			r.Equal(tt.wantTarget, data.Get(FieldGKECredentialsTargetServiceAccountEmail))
			r.Equal(tt.wantCast, data.Get(FieldGKECredentialsCastaiServiceAccountEmail))
		})
	}
}

// TestAccGKE_ClusterImpersonation onboards a pre-existing GKE cluster to CAST AI
// using service account impersonation instead of a service account key, then
// switches the cluster back to service account key credentials.
//
// Required environment variables:
//   - CASTAI_API_TOKEN, ACCEPTANCE_TEST_ORGANIZATION_ID (testAccPreCheck)
//   - GOOGLE_PROJECT_ID - GCP project containing the shared acceptance cluster
//   - GOOGLE_CLUSTER_NAME - name of the pre-existing GKE cluster
//   - GOOGLE_CLUSTER_LOCATION - zone or region of the cluster, e.g. us-central1-c
//   - CASTAI_GCP_CREDENTIALS - service account key JSON used for the static credentials step
func TestAccGKE_ClusterImpersonation(t *testing.T) {
	projectID := os.Getenv("GOOGLE_PROJECT_ID")
	clusterName := os.Getenv("GOOGLE_CLUSTER_NAME")
	location := os.Getenv("GOOGLE_CLUSTER_LOCATION")
	credentialsJSON := os.Getenv("CASTAI_GCP_CREDENTIALS")
	if projectID == "" || clusterName == "" || location == "" || credentialsJSON == "" {
		t.Skip("GOOGLE_PROJECT_ID, GOOGLE_CLUSTER_NAME, GOOGLE_CLUSTER_LOCATION and CASTAI_GCP_CREDENTIALS must be set for this acceptance test")
	}

	rName := fmt.Sprintf("%v-gke-imp-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_gke_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"google": {
				Source:            "hashicorp/google",
				VersionConstraint: "> 4.75.0",
			},
			"google-beta": {
				Source:            "hashicorp/google-beta",
				VersionConstraint: "> 4.75.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccGKEImpersonationConfig(rName, clusterName, location, projectID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_token"),
					resource.TestCheckResourceAttrSet(resourceName, "credentials_id"),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttr(resourceName, "name", clusterName),
					resource.TestCheckResourceAttr(resourceName, "project_id", projectID),
					resource.TestCheckResourceAttr(resourceName, "location", location),
					resource.TestCheckResourceAttrSet(resourceName, "target_service_account_email"),
					resource.TestCheckResourceAttrSet(resourceName, "castai_service_account_email"),
					resource.TestCheckResourceAttrPair(
						resourceName, "castai_service_account_email",
						"data.castai_impersonation_service_account.gke", "email",
					),
				),
			},
			{
				Config:   testAccGKEImpersonationConfig(rName, clusterName, location, projectID),
				PlanOnly: true,
			},
			{
				Config: testAccGKECredentialsConfig(rName, clusterName, location, projectID, credentialsJSON),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "credentials_id"),
					resource.TestCheckResourceAttr(resourceName, "target_service_account_email", ""),
					resource.TestCheckResourceAttr(resourceName, "castai_service_account_email", ""),
				),
			},
			{
				Config:   testAccGKECredentialsConfig(rName, clusterName, location, projectID, credentialsJSON),
				PlanOnly: true,
			},
		},
	})
}

func testAccGKEImpersonationConfig(rName, clusterName, location, projectID string) string {
	return ConfigCompose(testAccGKEImpersonationPrereqConfig(rName, clusterName, projectID), fmt.Sprintf(`
resource "castai_gke_cluster" "test" {
  project_id                 = %[1]q
  location                   = %[2]q
  name                       = %[3]q
  delete_nodes_on_disconnect = false

  target_service_account_email  = google_service_account.target.email
  castai_service_account_email = data.castai_impersonation_service_account.gke.email

  depends_on = [
    google_project_iam_member.castai_role_binding,
    google_service_account_iam_member.token_creator_binding,
  ]
}
`, projectID, location, clusterName))
}

func testAccGKECredentialsConfig(rName, clusterName, location, projectID, credentialsJSON string) string {
	return ConfigCompose(testAccGKEImpersonationPrereqConfig(rName, clusterName, projectID), fmt.Sprintf(`
resource "castai_gke_cluster" "test" {
  project_id                 = %[1]q
  location                   = %[2]q
  name                       = %[3]q
  delete_nodes_on_disconnect = false

  credentials_json = %q
}
`, projectID, location, clusterName, credentialsJSON))
}

func testAccGKEImpersonationPrereqConfig(rName, clusterName, projectID string) string {
	return fmt.Sprintf(`
data "castai_impersonation_service_account" "gke" {
  provider_type = "gke"
}

data "castai_gke_user_policies" "gke" {}

resource "google_service_account" "target" {
  account_id   = "%[1]s"
  display_name = "Terraform acceptance tests GKE impersonation target for %[2]s"
  project      = %[3]q
}

resource "google_project_iam_custom_role" "castai_role" {
  role_id     = "castai.tfAccGkeImp.${substr(sha1("%[1]s"), 0, 8)}"
  title       = "Terraform acceptance tests GKE impersonation role for %[2]s"
  description = "Role to manage GKE cluster via CAST AI"
  permissions = toset(data.castai_gke_user_policies.gke.policy)
  project     = %[3]q
  stage       = "GA"
}

resource "google_project_iam_member" "castai_role_binding" {
  project = %[3]q
  role    = google_project_iam_custom_role.castai_role.id
  member  = "serviceAccount:${google_service_account.target.email}"
}

resource "google_service_account_iam_member" "token_creator_binding" {
  service_account_id = google_service_account.target.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:${data.castai_impersonation_service_account.gke.email}"
}
`, rName, clusterName, projectID)
}
