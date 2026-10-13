package castai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestGKEClusterCredentialsSchema(t *testing.T) {
	base := map[string]interface{}{
		FieldGKEClusterName:      "cluster",
		FieldGKEClusterProjectId: "project",
		FieldGKEClusterLocation:  "europe-central2-a",
	}
	withBlock := func(block map[string]interface{}) map[string]interface{} {
		raw := map[string]interface{}{}
		for k, v := range base {
			raw[k] = v
		}
		raw[FieldGKECredentials] = []interface{}{block}
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
			raw: withBlock(map[string]interface{}{
				FieldGKECredentialsTargetServiceAccount: "target-sa@project.iam.gserviceaccount.com",
				FieldGKECredentialsCastaiServiceAccount: "cast-sa@cast-project.iam.gserviceaccount.com",
			}),
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
			name: "block conflicts with credentials_json",
			raw: func() map[string]interface{} {
				raw := withBlock(map[string]interface{}{
					FieldGKECredentialsTargetServiceAccount: "target-sa@project.iam.gserviceaccount.com",
					FieldGKECredentialsCastaiServiceAccount: "cast-sa@cast-project.iam.gserviceaccount.com",
				})
				raw[FieldGKEClusterCredentials] = "{}"
				return raw
			}(),
			wantErr:    true,
			errMessage: "conflicts with",
		},
		{
			name: "partial pair rejected",
			raw: withBlock(map[string]interface{}{
				FieldGKECredentialsTargetServiceAccount: "target-sa@project.iam.gserviceaccount.com",
			}),
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

	newProvider := func(mockClient *mock_sdk.MockClientInterface) *ProviderConfig {
		return &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}
	}

	impersonationConfig := func() map[string]interface{} {
		return map[string]interface{}{
			FieldGKEClusterName:      "cluster",
			FieldGKEClusterProjectId: "project",
			FieldGKEClusterLocation:  "europe-central2-a",
			FieldGKECredentials: []interface{}{
				map[string]interface{}{
					FieldGKECredentialsTargetServiceAccount: targetSA,
					FieldGKECredentialsCastaiServiceAccount: castSA,
				},
			},
		}
	}
	noneConfig := func() map[string]interface{} {
		return map[string]interface{}{
			FieldGKEClusterName:      "cluster",
			FieldGKEClusterProjectId: "project",
			FieldGKEClusterLocation:  "europe-central2-a",
		}
	}

	clusterBody := `{"id":"` + clusterID + `","status":"ready","agentStatus":"online","organizationId":"org","credentialsId":"cid","gke":{"clusterName":"cluster","region":"europe-central2","location":"europe-central2-a","projectId":"project","clientServiceAccount":"` + targetSA + `"}}`

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
	getCluster := func(mockClient *mock_sdk.MockClientInterface) {
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
			Return(&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(clusterBody))),
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)
	}

	t.Run("create impersonation registers, creates SA and skips credentials update", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_sdk.NewMockClientInterface(mockctrl)
		provider := newProvider(mockClient)

		registerClient(mockClient)
		mockClient.EXPECT().
			ExternalClusterAPIGKECreateSAWithBody(gomock.Any(), clusterID, "application/json", gomock.Any()).
			Do(func(_ context.Context, _ string, _ string, body io.Reader, _ ...sdk.RequestEditorFn) {
				raw, err := io.ReadAll(body)
				r.NoError(err)
				r.JSONEq(fmt.Sprintf(`{"gke":{"gkeSaImpersonate":%q,"castServiceAccount":%q}}`, targetSA, castSA), string(raw))
			}).
			Return(&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"serviceAccount":"` + castSA + `"}`))),
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)
		getCluster(mockClient)
		// The explicit getCluster after GKECreateSA plus the trailing ReadContext.
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
			Return(&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(clusterBody))),
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)

		resource := resourceGKECluster()
		data := schema.TestResourceDataRaw(t, resource.Schema, impersonationConfig())
		diags := resource.CreateContext(ctx, data, provider)

		r.False(diags.HasError())
		r.Equal(clusterID, data.Id())
		r.Equal("cid", data.Get(FieldClusterCredentialsId))
		r.Equal("token-123", data.Get(FieldClusterToken))
		block := data.Get(FieldGKECredentials).([]interface{})[0].(map[string]interface{})
		r.Equal(targetSA, block[FieldGKECredentialsTargetServiceAccount])
		r.NotContains(block[FieldGKECredentialsTargetServiceAccount].(string), "drift")
	})

	t.Run("create impersonation propagates SA creation error", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_sdk.NewMockClientInterface(mockctrl)
		provider := newProvider(mockClient)

		registerClient(mockClient)
		mockClient.EXPECT().
			ExternalClusterAPIGKECreateSAWithBody(gomock.Any(), clusterID, "application/json", gomock.Any()).
			Return(&http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"message":"bad request"}`))),
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)

		resource := resourceGKECluster()
		data := schema.TestResourceDataRaw(t, resource.Schema, impersonationConfig())
		diags := resource.CreateContext(ctx, data, provider)

		r.True(diags.HasError())
		r.Contains(diags[0].Summary, "creating GKE impersonation service account")
	})

	t.Run("create without credentials skips update", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_sdk.NewMockClientInterface(mockctrl)
		provider := newProvider(mockClient)

		registerClient(mockClient)
		getCluster(mockClient)

		resource := resourceGKECluster()
		data := schema.TestResourceDataRaw(t, resource.Schema, noneConfig())
		diags := resource.CreateContext(ctx, data, provider)

		r.False(diags.HasError())
		r.Equal("cid", data.Get(FieldClusterCredentialsId))
	})
}

func TestGKEClusterResourceUpdateCredentialsBlock(t *testing.T) {
	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gk3d"
	ctx := context.Background()

	targetSA := "target-sa@project.iam.gserviceaccount.com"
	castSA := "cast-sa@cast-project.iam.gserviceaccount.com"

	newProvider := func(mockClient *mock_sdk.MockClientInterface) *ProviderConfig {
		return &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}
	}

	t.Run("update impersonation re-calls GKECreateSA and refreshes credentials id", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_sdk.NewMockClientInterface(mockctrl)
		provider := newProvider(mockClient)

		mockClient.EXPECT().
			ExternalClusterAPIGKECreateSAWithBody(gomock.Any(), clusterID, "application/json", gomock.Any()).
			Do(func(_ context.Context, _ string, _ string, body io.Reader, _ ...sdk.RequestEditorFn) {
				raw, err := io.ReadAll(body)
				r.NoError(err)
				r.JSONEq(fmt.Sprintf(`{"gke":{"gkeSaImpersonate":%q,"castServiceAccount":%q}}`, targetSA, castSA), string(raw))
			}).
			Return(&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"serviceAccount":"` + castSA + `"}`))),
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)
		clusterBody := `{"id":"` + clusterID + `","status":"ready","agentStatus":"online","organizationId":"org","credentialsId":"after-update-cid","gke":{"clusterName":"cluster","region":"europe-central2","location":"europe-central2-a","projectId":"project","clientServiceAccount":"` + targetSA + `"}}`
		// The explicit getCluster after GKECreateSA plus the trailing ReadContext.
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
			Return(&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(clusterBody))),
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)
		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
			Return(&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(clusterBody))),
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)

		gkeResource := resourceGKECluster()
		raw := map[string]interface{}{
			FieldGKECredentials: []interface{}{
				map[string]interface{}{
					FieldGKECredentialsTargetServiceAccount: targetSA,
					FieldGKECredentialsCastaiServiceAccount: castSA,
				},
			},
		}
		data := schema.TestResourceDataRaw(t, gkeResource.Schema, raw)
		data.SetId(clusterID)
		diags := gkeResource.UpdateContext(ctx, data, provider)

		r.Empty(diags)
		r.Equal("after-update-cid", data.Get(FieldClusterCredentialsId))
	})
}

func TestGKEClusterResourceCredentialsDrift(t *testing.T) {
	ctx := context.Background()
	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gke"

	newProvider := func(mockClient *mock_sdk.MockClientInterface) *ProviderConfig {
		return &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}
	}

	newData := func(t *testing.T, resource *schema.Resource, attributes map[string]string) *schema.ResourceData {
		t.Helper()
		val := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(val, 0)
		state.ID = clusterID
		state.Attributes = attributes
		return resource.Data(state)
	}

	run := func(t *testing.T, attributes map[string]string, apiBody string, wantTarget string) {
		t.Helper()
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_sdk.NewMockClientInterface(mockctrl)
		provider := newProvider(mockClient)

		mockClient.EXPECT().
			ExternalClusterAPIGetCluster(gomock.Any(), clusterID).
			Return(&http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(apiBody))),
				Header:     map[string][]string{"Content-Type": {"json"}},
			}, nil)

		resource := resourceGKECluster()
		data := newData(t, resource, attributes)
		diags := resource.ReadContext(ctx, data, provider)

		r.False(diags.HasError())
		block := data.Get(FieldGKECredentials).([]interface{})[0].(map[string]interface{})
		r.Equal(wantTarget, block[FieldGKECredentialsTargetServiceAccount].(string))
	}

	impersonationAttrs := map[string]string{
		"credentials.#":                        "1",
		"credentials.0.target_service_account": "cfg-sa@project.iam.gserviceaccount.com",
		"credentials.0.castai_service_account": "cast-sa@cast-project.iam.gserviceaccount.com",
		FieldClusterCredentialsId:              "local-cid",
	}

	t.Run("impersonation credentials_id drift triggers no fake and keeps target", func(t *testing.T) {
		run(t, impersonationAttrs, `{"credentialsId":"remote-cid"}`, "cfg-sa@project.iam.gserviceaccount.com")
	})

	t.Run("impersonation drift surfaces target_service_account from API", func(t *testing.T) {
		run(t, impersonationAttrs, `{"credentialsId":"local-cid","gke":{"clientServiceAccount":"api-sa@project.iam.gserviceaccount.com"}}`, "api-sa@project.iam.gserviceaccount.com")
	})

	t.Run("impersonation no drift keeps target_service_account", func(t *testing.T) {
		run(t, impersonationAttrs, `{"credentialsId":"local-cid","gke":{"clientServiceAccount":"cfg-sa@project.iam.gserviceaccount.com"}}`, "cfg-sa@project.iam.gserviceaccount.com")
	})
}
