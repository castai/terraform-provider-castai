package castai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdkterraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
	"github.com/castai/terraform-provider-castai/castai/sdk/organization_management"
	mockOrganizationManagement "github.com/castai/terraform-provider-castai/castai/sdk/organization_management/mock"
)

func TestAccCloudAgnostic_ResourceSSOConnection(t *testing.T) {
	rName := fmt.Sprintf("%v-sso-connection-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_sso_connection.test"

	// Non-secret identifiers of the "terraform-provider-acc-sso" Azure AD app;
	// its client secret is the only sensitive value and is stored in onyx.
	clientID := os.Getenv("SSO_CLIENT_ID")
	if clientID == "" {
		clientID = "324aec6c-bf7e-43a8-86e2-76ca5efaf891"
	}
	clientSecret := os.Getenv("SSO_CLIENT_SECRET")
	domain := os.Getenv("SSO_DOMAIN")
	if domain == "" {
		domain = "cast.ai"
	}
	enterpriseToken := os.Getenv("CASTAI_ENTERPRISE_API_TOKEN")

	var enterpriseClient *sdk.ClientWithResponses

	t.Cleanup(func() {
		if enterpriseClient == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := sweepSSOConnection(ctx, enterpriseClient); err != nil {
			t.Logf("warning: failed to sweep SSO connection after test: %v", err)
		}
	})

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if clientSecret == "" {
				t.Skipf("missing required env var: SSO_CLIENT_SECRET")
			}
			// The SSO API requires an enterprise organisation; CASTAI_API_TOKEN belongs to the standard acceptance test org.
			if enterpriseToken == "" {
				t.Skipf("missing required env var: CASTAI_ENTERPRISE_API_TOKEN")
			}
			// The settings endpoints (organization_id, block_invitations) are org-scoped and the
			// enterprise org is not the shared acceptance-test org.
			if os.Getenv("CASTAI_ENTERPRISE_ORGANIZATION_ID") == "" {
				t.Skipf("missing required env var: CASTAI_ENTERPRISE_ORGANIZATION_ID")
			}

			client, err := sdk.CreateClient(os.Getenv("CASTAI_API_URL"), enterpriseToken, "castai-terraform-provider-acc-test")
			if err != nil {
				t.Fatalf("failed to create enterprise API client: %v", err)
			}
			enterpriseClient = client

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := sweepSSOConnection(ctx, enterpriseClient); err != nil {
				t.Logf("warning: failed to sweep SSO connection before test: %v", err)
			}
		},
		// Fresh provider instances: the enterprise token in the config must not reconfigure the shared testAccProvider used by parallel tests.
		ProviderFactories: map[string]func() (*schema.Provider, error){
			ProviderName: func() (*schema.Provider, error) {
				return Provider("v1.0.0"), nil
			},
		},
		CheckDestroy: func(s *terraform.State) error {
			return testAccSSOConnectionConfigurationDestroy(enterpriseClient, s)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccCreateSSOConnectionConfig(rName, enterpriseToken, os.Getenv("CASTAI_ENTERPRISE_ORGANIZATION_ID"), clientID, clientSecret, domain),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "email_domain", "aad_connection@test.com"),
					resource.TestCheckResourceAttrSet(resourceName, "aad.0.client_id"),
					resource.TestCheckResourceAttrSet(resourceName, "aad.0.client_secret"),
					resource.TestCheckResourceAttrSet(resourceName, "aad.0.ad_domain"),
				),
			},
		},
	})
}

// sweepSSOConnection lists SSO connections and deletes the first one found.
// The acceptance test org enforces a quota of 1 SSO connection, so at most one
// can exist. Any connection present before the test runs is stale from a
// previous run and is removed to avoid exhausting the quota.
func sweepSSOConnection(ctx context.Context, client sdk.ClientWithResponsesInterface) error {
	resp, err := client.SSOAPIListSSOConnectionsWithResponse(ctx)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return err
	}
	if resp.JSON200 == nil || len(resp.JSON200.Connections) == 0 {
		return nil
	}

	conn := resp.JSON200.Connections[0]
	if conn.Id == nil {
		return fmt.Errorf("sso connection has nil id")
	}

	delResp, err := client.SSOAPIDeleteSSOConnectionWithResponse(ctx, *conn.Id)
	if err := sdk.CheckOKResponse(delResp, err); err != nil {
		return err
	}
	return nil
}

func TestSSOConnection_ReadContext(t *testing.T) {
	t.Run("read azure ad connector", func(t *testing.T) {
		t.Parallel()

		readBody := `{"connection":{"id":"fce35ba2-5c06-4078-8391-1ac8f7ba798b","name":"test_sso","createdAt":"2023-11-02T10:49:14.376757Z","updatedAt":"2023-11-02T10:49:14.450828Z","emailDomain":"test_email","additionalEmailDomains":[],"aad":{"adDomain":"test_connector","clientId":"test_client","clientSecret":"test_secret"}}}`

		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		connectionID := "fce35ba2-5c06-4078-8391-1ac8f7ba798b"

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(readBody))), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(connectionID),
			}), 0),
		)

		result := resource.ReadContext(context.Background(), data, &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		})

		r := require.New(t)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal("test_sso", data.Get(FieldSSOConnectionName))
		r.Equal("test_email", data.Get(FieldSSOConnectionEmailDomain))
		r.Empty(data.Get(FieldSSOConnectionAdditionalEmailDomains))
	})

	t.Run("read azure ad connector with additional email domains", func(t *testing.T) {
		t.Parallel()

		readBody := `{"connection":{"id":"fce35ba2-5c06-4078-8391-1ac8f7ba798b","name":"test_sso","createdAt":"2023-11-02T10:49:14.376757Z","updatedAt":"2023-11-02T10:49:14.450828Z","emailDomain":"test_email","additionalEmailDomains":["domain.com", "other.com"],"aad":{"adDomain":"test_connector","clientId":"test_client","clientSecret":"test_secret"}}}`

		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		connectionID := "fce35ba2-5c06-4078-8391-1ac8f7ba798b"

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(readBody))), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(connectionID),
			}), 0),
		)

		result := resource.ReadContext(context.Background(), data, &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		})

		r := require.New(t)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal("test_sso", data.Get(FieldSSOConnectionName))
		r.Equal("test_email", data.Get(FieldSSOConnectionEmailDomain))
		r.Equal([]interface{}{"domain.com", "other.com"}, data.Get(FieldSSOConnectionAdditionalEmailDomains))
	})
}

func TestSSOConnection_CreateADDConnector(t *testing.T) {
	t.Run("create azure ad connector", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		mockClient.EXPECT().
			SSOAPICreateSSOConnection(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, body sdk.SSOAPICreateSSOConnectionJSONRequestBody, _ ...sdk.RequestEditorFn) (*http.Response, error) {
				got, err := json.Marshal(body)
				r.NoError(err)

				expected := []byte(`{
  "aad": {
    "adDomain": "test_connector",
    "clientId": "test_client",
    "clientSecret": "test_secret"
  },
  "emailDomain": "test_email",
  "defaultRoleId": null,
  "name": "test_sso"
}
`)

				equal, err := JSONBytesEqual(got, expected)
				r.NoError(err)
				r.True(equal, fmt.Sprintf("got:      %v\n"+
					"expected: %v\n", string(got), string(expected)))

				return &http.Response{
					StatusCode: 200,
					Header:     map[string][]string{"Content-Type": {"json"}},
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"id": "b6bfc074-a267-400f-b8f1-db0850c369b1", "status": "STATUS_ACTIVE"}`))),
				}, nil
			})

		connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"
		readBody := io.NopCloser(bytes.NewReader([]byte(`{ "connection" :{
  "id": "b6bfc074-a267-400f-b8f1-db0850c369b1",
  "name": "test_sso",
  "createdAt": "2023-11-02T10:49:14.376757Z",
  "updatedAt": "2023-11-02T10:49:14.450828Z",
  "emailDomain": "test_email",
  "aad": {
    "adDomain": "test_connector",
    "clientId": "test_client",
    "clientSecret": "test_secret"
  }
}}`)))

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: readBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		resource := resourceSSOConnection()
		data := resource.Data(sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
			FieldSSOConnectionName:        cty.StringVal("test_sso"),
			FieldSSOConnectionEmailDomain: cty.StringVal("test_email"),
			FieldSSOConnectionAAD: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldSSOConnectionADDomain:       cty.StringVal("test_connector"),
					FieldSSOConnectionADClientID:     cty.StringVal("test_client"),
					FieldSSOConnectionADClientSecret: cty.StringVal("test_secret"),
				}),
			}),
		}), 0))

		result := resource.CreateContext(context.Background(), data, &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		})

		r.Nil(result)
		r.False(result.HasError())
		r.Equal("test_sso", data.Get(FieldSSOConnectionName))
		r.Equal("test_email", data.Get(FieldSSOConnectionEmailDomain))
		equalADConnector(t, r, data.Get(FieldSSOConnectionAAD), "test_connector", "test_client", "test_secret")
	})

	t.Run("create azure ad connector with additional email domains", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		mockClient.EXPECT().
			SSOAPICreateSSOConnection(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, body sdk.SSOAPICreateSSOConnectionJSONRequestBody, _ ...sdk.RequestEditorFn) (*http.Response, error) {
				got, err := json.Marshal(body)
				r.NoError(err)

				expected := []byte(`{
  "aad": {
    "adDomain": "test_connector",
    "clientId": "test_client",
    "clientSecret": "test_secret"
  },
  "emailDomain": "test_email",
  "defaultRoleId": null,
  "additionalEmailDomains": ["test_domain1.com", "test_domain2.com"],
  "name": "test_sso"
}
`)

				equal, err := JSONBytesEqual(got, expected)
				r.NoError(err)
				r.True(equal, fmt.Sprintf("got:      %v\n"+
					"expected: %v\n", string(got), string(expected)))

				return &http.Response{
					StatusCode: 200,
					Header:     map[string][]string{"Content-Type": {"json"}},
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"id": "b6bfc074-a267-400f-b8f1-db0850c369b1", "status": "STATUS_ACTIVE"}`))),
				}, nil
			})

		connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"
		readBody := io.NopCloser(bytes.NewReader([]byte(`{"connection":{
  "id": "b6bfc074-a267-400f-b8f1-db0850c369b1",
  "name": "test_sso",
  "createdAt": "2023-11-02T10:49:14.376757Z",
  "updatedAt": "2023-11-02T10:49:14.450828Z",
  "emailDomain": "test_email",
	"additionalEmailDomains": ["test_domain1.com", "test_domain2.com"],
  "aad": {
    "adDomain": "test_connector",
    "clientId": "test_client",
    "clientSecret": "test_secret"
  }
}}`)))

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: readBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		resource := resourceSSOConnection()
		data := resource.Data(sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
			FieldSSOConnectionName:                   cty.StringVal("test_sso"),
			FieldSSOConnectionEmailDomain:            cty.StringVal("test_email"),
			FieldSSOConnectionAdditionalEmailDomains: cty.ListVal([]cty.Value{cty.StringVal("test_domain1.com"), cty.StringVal("test_domain2.com")}),
			FieldSSOConnectionAAD: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					FieldSSOConnectionADDomain:       cty.StringVal("test_connector"),
					FieldSSOConnectionADClientID:     cty.StringVal("test_client"),
					FieldSSOConnectionADClientSecret: cty.StringVal("test_secret"),
				}),
			}),
		}), 0))

		result := resource.CreateContext(context.Background(), data, &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		})

		r.Nil(result)
		r.False(result.HasError())
		r.Equal("test_sso", data.Get(FieldSSOConnectionName))
		r.Equal("test_email", data.Get(FieldSSOConnectionEmailDomain))
		equalADConnector(t, r, data.Get(FieldSSOConnectionAAD), "test_connector", "test_client", "test_secret")
	})
}

func TestSSOConnection_CreateOktaConnector(t *testing.T) {
	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	mockClient.EXPECT().
		SSOAPICreateSSOConnection(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, body sdk.SSOAPICreateSSOConnectionJSONRequestBody, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			got, err := json.Marshal(body)
			r.NoError(err)

			expected := []byte(`{
  "okta": {
    "oktaDomain": "test_connector",
    "clientId": "test_client",
    "clientSecret": "test_secret"
  },
  "emailDomain": "test_email",
  "defaultRoleId": null,
  "name": "test_sso"
}`)

			equal, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(equal, fmt.Sprintf("got:      %v\n"+
				"expected: %v\n", string(got), string(expected)))

			return &http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"id": "b6bfc074-a267-400f-b8f1-db0850c369b1", "status": "STATUS_ACTIVE"}`))),
			}, nil
		})

	connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	readBody := io.NopCloser(bytes.NewReader([]byte(`{"connection":{
  "id": "b6bfc074-a267-400f-b8f1-db0850c369b1",
  "name": "test_sso",
  "createdAt": "2023-11-02T10:49:14.376757Z",
  "updatedAt": "2023-11-02T10:49:14.450828Z",
  "emailDomain": "test_email",
  "okta": {
    "oktaDomain": "test_connector",
    "clientId": "test_client",
    "clientSecret": "test_secret"
  }
}}`)))

	mockClient.EXPECT().
		SSOAPIGetSSOConnection(gomock.Any(), connectionID).
		Return(&http.Response{StatusCode: 200, Body: readBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceSSOConnection()
	data := resource.Data(sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		FieldSSOConnectionName:        cty.StringVal("test_sso"),
		FieldSSOConnectionEmailDomain: cty.StringVal("test_email"),
		FieldSSOConnectionOkta: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				FieldSSOConnectionOktaDomain:       cty.StringVal("test_connector"),
				FieldSSOConnectionOktaClientID:     cty.StringVal("test_client"),
				FieldSSOConnectionOktaClientSecret: cty.StringVal("test_secret"),
			}),
		}),
	}), 0))

	result := resource.CreateContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	})

	r.Nil(result)
	r.False(result.HasError())
	r.Equal("test_sso", data.Get(FieldSSOConnectionName))
	r.Equal("test_email", data.Get(FieldSSOConnectionEmailDomain))
	equalOktaConnector(t, r, data.Get(FieldSSOConnectionOkta), "test_connector", "test_client", "test_secret")
}

func TestSSOConnection_UpdateADDConnector(t *testing.T) {
	t.Run("update azure ad connector", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}
		connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

		raw := make(map[string]interface{})
		raw[FieldSSOConnectionName] = "updated_name"

		resource := resourceSSOConnection()
		data := schema.TestResourceDataRaw(t, resource.Schema, raw)
		data.SetId(connectionID)
		r.NoError(data.Set(FieldSSOConnectionAAD, []map[string]interface{}{
			{
				FieldSSOConnectionADDomain:       "updated_domain",
				FieldSSOConnectionADClientID:     "updated_client_id",
				FieldSSOConnectionADClientSecret: "updated_client_secret",
			},
		}))

		mockClient.EXPECT().SSOAPIUpdateSSOConnection(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONRequestBody, _ ...sdk.RequestEditorFn) (*http.Response, error) {
				got, err := json.Marshal(body)
				r.NoError(err)

				expected := []byte(`{
  "aad": {
    "adDomain": "updated_domain",
    "clientId": "updated_client_id",
    "clientSecret": "updated_client_secret"
  },
  "defaultRoleId": null,
  "name": "updated_name"
}`)

				eq, err := JSONBytesEqual(got, expected)
				r.NoError(err)
				r.True(eq, fmt.Sprintf("got:      %v\n"+
					"expected: %v\n", string(got), string(expected)))

				returnBody := []byte(`{
  "aad": {
    "adDomain": "updated_domain",
    "clientId": "updated_client_id",
    "clientSecret": "updated_client_secret"
  },
  "status": "STATUS_ACTIVE",
  "name": "updated_name"
}`)

				return &http.Response{
					StatusCode: 200,
					Header:     map[string][]string{"Content-Type": {"json"}},
					Body:       io.NopCloser(bytes.NewReader(returnBody)),
				}, nil
			}).Times(1)

		readBody := io.NopCloser(bytes.NewReader([]byte(`{"connection":{
  "id": "b6bfc074-a267-400f-b8f1-db0850c369b1",
  "name": "updated_name",
  "createdAt": "2023-11-02T10:49:14.376757Z",
  "updatedAt": "2023-11-02T10:49:14.450828Z",
  "emailDomain": "test_email",
  "aad": {
    "adDomain": "updated_domain",
    "clientId": "updated_client_id",
    "clientSecret": "updated_client_secret"
  }
}}`)))
		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: readBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		updateResult := resource.UpdateContext(ctx, data, provider)

		r.Nil(updateResult)
		r.False(updateResult.HasError())
		r.Equal("updated_name", data.Get(FieldSSOConnectionName))
		equalADConnector(t, r, data.Get(FieldSSOConnectionAAD), "updated_domain", "updated_client_id", "updated_client_secret")
	})

	t.Run("update azure ad connector with additional email domains", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}
		connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

		raw := make(map[string]interface{})
		raw[FieldSSOConnectionName] = "updated_name"

		resource := resourceSSOConnection()
		data := schema.TestResourceDataRaw(t, resource.Schema, raw)
		data.SetId(connectionID)
		r.NoError(data.Set(FieldSSOConnectionAAD, []map[string]interface{}{
			{
				FieldSSOConnectionADDomain:       "updated_domain",
				FieldSSOConnectionADClientID:     "updated_client_id",
				FieldSSOConnectionADClientSecret: "updated_client_secret",
			},
		}))
		r.NoError(data.Set(FieldSSOConnectionAdditionalEmailDomains, []interface{}{"updated_domain_one", "updated_domain_two"}))

		mockClient.EXPECT().SSOAPIUpdateSSOConnection(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONRequestBody, _ ...sdk.RequestEditorFn) (*http.Response, error) {
				got, err := json.Marshal(body)
				r.NoError(err)

				expected := []byte(`{
  "aad": {
    "adDomain": "updated_domain",
    "clientId": "updated_client_id",
    "clientSecret": "updated_client_secret"
  },
  "name": "updated_name",
  "defaultRoleId": null,
	"additionalEmailDomains": ["updated_domain_one", "updated_domain_two"]
}`)

				eq, err := JSONBytesEqual(got, expected)
				r.NoError(err)
				r.True(eq, fmt.Sprintf("got:      %v\n"+
					"expected: %v\n", string(got), string(expected)))

				returnBody := []byte(`{
  "aad": {
    "adDomain": "updated_domain",
    "clientId": "updated_client_id",
    "clientSecret": "updated_client_secret"
  },
  "status": "STATUS_ACTIVE",
  "name": "updated_name",
	"additionalEmailDomains": ["updated_domain_one", "updated_domain_two"]
}`)

				return &http.Response{
					StatusCode: 200,
					Header:     map[string][]string{"Content-Type": {"json"}},
					Body:       io.NopCloser(bytes.NewReader(returnBody)),
				}, nil
			}).Times(1)

		readBody := io.NopCloser(bytes.NewReader([]byte(`{"connection":{
  "id": "b6bfc074-a267-400f-b8f1-db0850c369b1",
  "name": "updated_name",
  "createdAt": "2023-11-02T10:49:14.376757Z",
  "updatedAt": "2023-11-02T10:49:14.450828Z",
  "emailDomain": "test_email",
	"additionalEmailDomains": ["updated_domain_one", "updated_domain_two"],
  "aad": {
    "adDomain": "updated_domain",
    "clientId": "updated_client_id",
    "clientSecret": "updated_client_secret"
  }
}}`)))
		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: readBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		updateResult := resource.UpdateContext(ctx, data, provider)

		r.Nil(updateResult)
		r.False(updateResult.HasError())
		r.Equal("updated_name", data.Get(FieldSSOConnectionName))
		r.Equal([]interface{}{"updated_domain_one", "updated_domain_two"}, data.Get(FieldSSOConnectionAdditionalEmailDomains))
		equalADConnector(t, r, data.Get(FieldSSOConnectionAAD), "updated_domain", "updated_client_id", "updated_client_secret")
	})
}

func TestSSOConnection_UpdateOktaConnector(t *testing.T) {
	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}
	connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	raw := make(map[string]interface{})
	raw[FieldSSOConnectionName] = "updated_name"

	resource := resourceSSOConnection()
	data := schema.TestResourceDataRaw(t, resource.Schema, raw)
	data.SetId(connectionID)
	r.NoError(data.Set(FieldSSOConnectionOkta, []map[string]interface{}{
		{
			FieldSSOConnectionOktaDomain:       "updated_domain",
			FieldSSOConnectionOktaClientID:     "updated_client_id",
			FieldSSOConnectionOktaClientSecret: "updated_client_secret",
		},
	}))

	mockClient.EXPECT().SSOAPIUpdateSSOConnection(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONRequestBody, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			got, err := json.Marshal(body)
			r.NoError(err)

			expected := []byte(`{
  "okta": {
    "oktaDomain": "updated_domain",
    "clientId": "updated_client_id",
    "clientSecret": "updated_client_secret"
  },
  "defaultRoleId": null,
  "name": "updated_name"
}`)

			eq, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(eq, fmt.Sprintf("got:      %v\n"+
				"expected: %v\n", string(got), string(expected)))

			returnBody := []byte(`{
  "okta": {
    "oktaDomain": "updated_domain",
    "clientId": "updated_client_id",
    "clientSecret": "updated_client_secret"
  },
  "status": "STATUS_ACTIVE",
  "name": "updated_name"
}`)

			return &http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader(returnBody)),
			}, nil
		}).Times(1)

	readBody := io.NopCloser(bytes.NewReader([]byte(`{"connection":{
  "id": "b6bfc074-a267-400f-b8f1-db0850c369b1",
  "name": "updated_name",
  "createdAt": "2023-11-02T10:49:14.376757Z",
  "updatedAt": "2023-11-02T10:49:14.450828Z",
  "emailDomain": "test_email",
  "okta": {
    "oktaDomain": "updated_domain",
    "clientId": "updated_client_id",
    "clientSecret": "updated_client_secret"
  }
}}`)))
	mockClient.EXPECT().
		SSOAPIGetSSOConnection(gomock.Any(), connectionID).
		Return(&http.Response{StatusCode: 200, Body: readBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	updateResult := resource.UpdateContext(ctx, data, provider)
	r.Nil(updateResult)
	r.False(updateResult.HasError())
	r.Equal("updated_name", data.Get(FieldSSOConnectionName))
	equalOktaConnector(t, r, data.Get(FieldSSOConnectionOkta), "updated_domain", "updated_client_id", "updated_client_secret")
}

func TestSSOConnection_DeleteContext(t *testing.T) {
	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	resource := resourceSSOConnection()
	data := resource.Data(
		sdkterraform.NewInstanceStateShimmedFromValue(
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(connectionID),
			}), 0,
		),
	)

	mockClient.EXPECT().
		SSOAPIDeleteSSOConnection(gomock.Any(), connectionID).
		Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(`{}`))), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	ctx := context.Background()
	result := resource.DeleteContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Empty(data.Get(FieldSSOConnectionOkta))
	r.Empty(data.Get(FieldSSOConnectionAAD))
	r.Empty(data.Get(FieldSSOConnectionName))
	r.Empty(data.Get(FieldSSOConnectionEmailDomain))
}

func TestSSOConnection_SynchronizeUserGroups(t *testing.T) {
	t.Run("create with synchronize_user_groups=true calls SetSync and stores token", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

		mockClient.EXPECT().
			SSOAPICreateSSOConnection(gomock.Any(), gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"id": "b6bfc074-a267-400f-b8f1-db0850c369b1", "status": "STATUS_ACTIVE"}`))),
			}, nil)

		mockClient.EXPECT().
			SSOAPISetSyncForSSOConnection(gomock.Any(), connectionID, sdk.SSOAPISetSyncForSSOConnectionJSONRequestBody{Sync: true}, gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"token": {"token": "test-sync-token", "name": "sync"}}`))),
			}, nil)

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body: io.NopCloser(bytes.NewReader([]byte(`{"connection":{
					"id": "b6bfc074-a267-400f-b8f1-db0850c369b1",
					"name": "test_sso",
					"emailDomain": "test_email",
					"isSynced": true,
					"aad": {"adDomain": "test_connector", "clientId": "test_client", "clientSecret": "test_secret"}
				}}`))),
			}, nil)

		res := resourceSSOConnection()
		data := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
			FieldSSOConnectionName:        "test_sso",
			FieldSSOConnectionEmailDomain: "test_email",
		})
		r.NoError(data.Set(FieldSSOConnectionSynchronizeUserGroups, true))
		r.NoError(data.Set(FieldSSOConnectionAAD, []map[string]interface{}{
			{
				FieldSSOConnectionADDomain:       "test_connector",
				FieldSSOConnectionADClientID:     "test_client",
				FieldSSOConnectionADClientSecret: "test_secret",
			},
		}))

		result := res.CreateContext(context.Background(), data, &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		})

		r.False(result.HasError())
		r.Equal("test-sync-token", data.Get(FieldSSOConnectionSyncAuthToken))
		r.True(data.Get(FieldSSOConnectionSynchronizeUserGroups).(bool))
		// Warning should be present.
		r.Len(result, 1)
		r.Equal(diag.Warning, result[0].Severity)
	})

	t.Run("create with synchronize_user_groups=false does not call SetSync", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

		mockClient.EXPECT().
			SSOAPICreateSSOConnection(gomock.Any(), gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"id": "b6bfc074-a267-400f-b8f1-db0850c369b1", "status": "STATUS_ACTIVE"}`))),
			}, nil)

		// No SSOAPISetSyncForSSOConnection call expected.

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body: io.NopCloser(bytes.NewReader([]byte(`{"connection":{
					"id": "b6bfc074-a267-400f-b8f1-db0850c369b1",
					"name": "test_sso",
					"emailDomain": "test_email",
					"isSynced": false,
					"aad": {"adDomain": "test_connector", "clientId": "test_client", "clientSecret": "test_secret"}
				}}`))),
			}, nil)

		res := resourceSSOConnection()
		data := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
			FieldSSOConnectionName:        "test_sso",
			FieldSSOConnectionEmailDomain: "test_email",
		})
		r.NoError(data.Set(FieldSSOConnectionAAD, []map[string]interface{}{
			{
				FieldSSOConnectionADDomain:       "test_connector",
				FieldSSOConnectionADClientID:     "test_client",
				FieldSSOConnectionADClientSecret: "test_secret",
			},
		}))

		result := res.CreateContext(context.Background(), data, &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		})

		r.False(result.HasError())
		r.Equal("", data.Get(FieldSSOConnectionSyncAuthToken))
		r.False(data.Get(FieldSSOConnectionSynchronizeUserGroups).(bool))
	})

	t.Run("setSSOConnectionSync enable stores token and returns warning", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

		mockClient.EXPECT().
			SSOAPISetSyncForSSOConnection(gomock.Any(), connectionID, sdk.SSOAPISetSyncForSSOConnectionJSONRequestBody{Sync: true}, gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"token": {"token": "new-sync-token", "name": "sync"}}`))),
			}, nil)

		res := resourceSSOConnection()
		data := res.Data(sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal(connectionID),
		}), 0))

		diags := setSSOConnectionSync(context.Background(), &sdk.ClientWithResponses{ClientInterface: mockClient}, data, true)

		r.False(diags.HasError())
		r.Len(diags, 1)
		r.Equal(diag.Warning, diags[0].Severity)
		r.Equal("new-sync-token", data.Get(FieldSSOConnectionSyncAuthToken))
	})

	t.Run("setSSOConnectionSync disable preserves token for reuse and returns no warning", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

		mockClient.EXPECT().
			SSOAPISetSyncForSSOConnection(gomock.Any(), connectionID, sdk.SSOAPISetSyncForSSOConnectionJSONRequestBody{Sync: false}, gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"emptyResponse": {}}`))),
			}, nil)

		res := resourceSSOConnection()
		data := res.Data(sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal(connectionID),
		}), 0))
		r.NoError(data.Set(FieldSSOConnectionSyncAuthToken, "old-sync-token"))

		diags := setSSOConnectionSync(context.Background(), &sdk.ClientWithResponses{ClientInterface: mockClient}, data, false)

		r.False(diags.HasError())
		r.Len(diags, 0)
		// Token is preserved so it can be reused if sync is re-enabled.
		r.Equal("old-sync-token", data.Get(FieldSSOConnectionSyncAuthToken))
	})
}

func TestSSOConnection_ReadContext_IsSynced(t *testing.T) {
	t.Run("read sets synchronize_user_groups from isSynced=true", func(t *testing.T) {
		t.Parallel()

		readBody := `{"connection":{"id":"fce35ba2-5c06-4078-8391-1ac8f7ba798b","name":"test_sso","emailDomain":"test_email","isSynced":true,"aad":{"adDomain":"d","clientId":"c","clientSecret":"s"}}}`
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
		connectionID := "fce35ba2-5c06-4078-8391-1ac8f7ba798b"

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(readBody))), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		res := resourceSSOConnection()
		data := res.Data(sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal(connectionID),
		}), 0))

		result := res.ReadContext(context.Background(), data, &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		})

		r := require.New(t)
		r.False(result.HasError())
		r.True(data.Get(FieldSSOConnectionSynchronizeUserGroups).(bool))
	})

	t.Run("read sets synchronize_user_groups from isSynced=false", func(t *testing.T) {
		t.Parallel()

		readBody := `{"connection":{"id":"fce35ba2-5c06-4078-8391-1ac8f7ba798b","name":"test_sso","emailDomain":"test_email","isSynced":false,"aad":{"adDomain":"d","clientId":"c","clientSecret":"s"}}}`
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
		connectionID := "fce35ba2-5c06-4078-8391-1ac8f7ba798b"

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(readBody))), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		res := resourceSSOConnection()
		data := res.Data(sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal(connectionID),
		}), 0))

		result := res.ReadContext(context.Background(), data, &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		})

		r := require.New(t)
		r.False(result.HasError())
		r.False(data.Get(FieldSSOConnectionSynchronizeUserGroups).(bool))
	})

	t.Run("read preserves existing sync_auth_token from state", func(t *testing.T) {
		t.Parallel()

		readBody := `{"connection":{"id":"fce35ba2-5c06-4078-8391-1ac8f7ba798b","name":"test_sso","emailDomain":"test_email","isSynced":true,"aad":{"adDomain":"d","clientId":"c","clientSecret":"s"}}}`
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
		connectionID := "fce35ba2-5c06-4078-8391-1ac8f7ba798b"

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(readBody))), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		res := resourceSSOConnection()
		// Simulate state where sync_auth_token was previously stored.
		data := res.Data(sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
			"id":                            cty.StringVal(connectionID),
			FieldSSOConnectionSyncAuthToken: cty.StringVal("previously-stored-token"),
		}), 0))

		result := res.ReadContext(context.Background(), data, &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		})

		r := require.New(t)
		r.False(result.HasError())
		r.Equal("previously-stored-token", data.Get(FieldSSOConnectionSyncAuthToken),
			"sync_auth_token must be preserved across reads since the API never returns it")
	})
}

func testAccCreateSSOConnectionConfig(rName, apiToken, organizationID, clientID, clientSecret, adDomain string) string {
	return fmt.Sprintf(`
provider "castai" {
  api_token = %[2]q
}

resource "castai_sso_connection" "test" {
  name            = %[1]q
  organization_id = %[6]q
  email_domain = "aad_connection@test.com"
  aad {
    client_id     = %[3]q
    client_secret = %[4]q
    ad_domain     = %[5]q
  }
}`, rName, apiToken, clientID, clientSecret, adDomain, organizationID)
}

func testAccSSOConnectionConfigurationDestroy(client sdk.ClientWithResponsesInterface, s *terraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_sso_connection" {
			continue
		}

		response, err := client.SSOAPIGetSSOConnectionWithResponse(ctx, rs.Primary.ID)
		if err != nil {
			return err
		}

		if response.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("sso connection %s still exists", rs.Primary.ID)
	}

	return nil
}

func TestSSOConnection_CreateOIDCConnector(t *testing.T) {
	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	mockClient.EXPECT().
		SSOAPICreateSSOConnection(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, body sdk.SSOAPICreateSSOConnectionJSONRequestBody, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			got, err := json.Marshal(body)
			r.NoError(err)

			expected := []byte(`{
  "oidc": {
    "issuerUrl": "https://keycloak.example.com/realms/master",
    "clientId": "test_client",
    "clientSecret": "test_secret",
    "type": "TYPE_BACK_CHANNEL"
  },
  "emailDomain": "test_email",
  "defaultRoleId": null,
  "name": "test_sso"
}`)

			equal, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(equal, fmt.Sprintf("got:      %v\n"+
				"expected: %v\n", string(got), string(expected)))

			return &http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"id": "b6bfc074-a267-400f-b8f1-db0850c369b1", "status": "STATUS_ACTIVE"}`))),
			}, nil
		})

	connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	readBody := io.NopCloser(bytes.NewReader([]byte(`{"connection":{
  "id": "b6bfc074-a267-400f-b8f1-db0850c369b1",
  "name": "test_sso",
  "createdAt": "2023-11-02T10:49:14.376757Z",
  "updatedAt": "2023-11-02T10:49:14.450828Z",
  "emailDomain": "test_email",
  "oidc": {
    "issuerUrl": "https://keycloak.example.com/realms/master",
    "clientId": "test_client",
    "clientSecret": "test_secret",
    "type": "TYPE_BACK_CHANNEL"
  }
}}`)))

	mockClient.EXPECT().
		SSOAPIGetSSOConnection(gomock.Any(), connectionID).
		Return(&http.Response{StatusCode: 200, Body: readBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceSSOConnection()
	data := resource.Data(sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		FieldSSOConnectionName:        cty.StringVal("test_sso"),
		FieldSSOConnectionEmailDomain: cty.StringVal("test_email"),
		FieldSSOConnectionOIDC: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				FieldSSOConnectionOIDCIssuerURL:    cty.StringVal("https://keycloak.example.com/realms/master"),
				FieldSSOConnectionOIDCClientID:     cty.StringVal("test_client"),
				FieldSSOConnectionOIDCClientSecret: cty.StringVal("test_secret"),
				FieldSSOConnectionOIDCType:         cty.StringVal("TYPE_BACK_CHANNEL"),
			}),
		}),
	}), 0))

	result := resource.CreateContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	})

	r.Nil(result)
	r.False(result.HasError())
	r.Equal("test_sso", data.Get(FieldSSOConnectionName))
	r.Equal("test_email", data.Get(FieldSSOConnectionEmailDomain))
	equalOIDCConnector(t, r, data.Get(FieldSSOConnectionOIDC), "https://keycloak.example.com/realms/master", "test_client", "test_secret", "TYPE_BACK_CHANNEL")
}

func TestSSOConnection_UpdateOIDCConnector(t *testing.T) {
	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}
	connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	raw := make(map[string]interface{})
	raw[FieldSSOConnectionName] = "updated_name"

	resource := resourceSSOConnection()
	data := schema.TestResourceDataRaw(t, resource.Schema, raw)
	data.SetId(connectionID)
	r.NoError(data.Set(FieldSSOConnectionOIDC, []map[string]interface{}{
		{
			FieldSSOConnectionOIDCIssuerURL:    "https://keycloak.example.com/realms/updated",
			FieldSSOConnectionOIDCClientID:     "updated_client_id",
			FieldSSOConnectionOIDCClientSecret: "updated_client_secret",
			FieldSSOConnectionOIDCType:         "TYPE_FRONT_CHANNEL",
		},
	}))

	mockClient.EXPECT().SSOAPIUpdateSSOConnection(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONRequestBody, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			got, err := json.Marshal(body)
			r.NoError(err)

			expected := []byte(`{
  "oidc": {
    "issuerUrl": "https://keycloak.example.com/realms/updated",
    "clientId": "updated_client_id",
    "clientSecret": "updated_client_secret",
    "type": "TYPE_FRONT_CHANNEL"
  },
  "defaultRoleId": null,
  "name": "updated_name"
}`)

			eq, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(eq, fmt.Sprintf("got:      %v\n"+
				"expected: %v\n", string(got), string(expected)))

			returnBody := []byte(`{
  "oidc": {
    "issuerUrl": "https://keycloak.example.com/realms/updated",
    "clientId": "updated_client_id",
    "clientSecret": "updated_client_secret",
    "type": "TYPE_FRONT_CHANNEL"
  },
  "status": "STATUS_ACTIVE",
  "name": "updated_name"
}`)

			return &http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader(returnBody)),
			}, nil
		}).Times(1)

	readBody := io.NopCloser(bytes.NewReader([]byte(`{"connection":{
  "id": "b6bfc074-a267-400f-b8f1-db0850c369b1",
  "name": "updated_name",
  "createdAt": "2023-11-02T10:49:14.376757Z",
  "updatedAt": "2023-11-02T10:49:14.450828Z",
  "emailDomain": "test_email",
  "oidc": {
    "issuerUrl": "https://keycloak.example.com/realms/updated",
    "clientId": "updated_client_id",
    "clientSecret": "updated_client_secret",
    "type": "TYPE_FRONT_CHANNEL"
  }
}}`)))
	mockClient.EXPECT().
		SSOAPIGetSSOConnection(gomock.Any(), connectionID).
		Return(&http.Response{StatusCode: 200, Body: readBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	updateResult := resource.UpdateContext(ctx, data, provider)
	r.Nil(updateResult)
	r.False(updateResult.HasError())
	r.Equal("updated_name", data.Get(FieldSSOConnectionName))
	equalOIDCConnector(t, r, data.Get(FieldSSOConnectionOIDC), "https://keycloak.example.com/realms/updated", "updated_client_id", "updated_client_secret", "TYPE_FRONT_CHANNEL")
}

func TestSSOConnection_OIDCSecretDiffSuppress(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	// Get the DiffSuppressFunc from the schema definition.
	res := resourceSSOConnection()
	suppress := res.Schema[FieldSSOConnectionOIDC].Elem.(*schema.Resource).Schema[FieldSSOConnectionOIDCClientSecret].DiffSuppressFunc

	// Simulate a read response: backend returns base64(bcrypt(secret)).
	plaintext := "super-secret"
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.MinCost)
	r.NoError(err)
	encoded := base64.StdEncoding.EncodeToString(hash)

	// Same plaintext as stored hash → suppress diff.
	r.True(suppress(FieldSSOConnectionOIDCClientSecret, encoded, plaintext, nil))
	// Different plaintext → do not suppress.
	r.False(suppress(FieldSSOConnectionOIDCClientSecret, encoded, "wrong-secret", nil))
	// Invalid base64 (e.g. first apply before any read) → do not suppress.
	r.False(suppress(FieldSSOConnectionOIDCClientSecret, "not-valid-base64!!", plaintext, nil))
}

func TestSSOConnection_ConnectorMutualExclusivity(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	makeGetter := func(fields map[string][]any) func(string) interface{} {
		return func(key string) interface{} {
			if v, ok := fields[key]; ok {
				return v
			}
			return []any{}
		}
	}

	// Exactly one connector → valid.
	r.Equal(1, countSSOConnectors(makeGetter(map[string][]any{
		FieldSSOConnectionOIDC: {map[string]any{}},
	})))
	r.Equal(1, countSSOConnectors(makeGetter(map[string][]any{
		FieldSSOConnectionAAD: {map[string]any{}},
	})))

	// Two connectors → invalid.
	r.Equal(2, countSSOConnectors(makeGetter(map[string][]any{
		FieldSSOConnectionOIDC: {map[string]any{}},
		FieldSSOConnectionAAD:  {map[string]any{}},
	})))

	// Zero connectors → invalid.
	r.Equal(0, countSSOConnectors(makeGetter(map[string][]any{})))
}

func TestSSOConnection_UpdateOIDCToAADConnector(t *testing.T) {
	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}
	connectionID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	raw := make(map[string]interface{})
	raw[FieldSSOConnectionName] = "updated_name"

	resource := resourceSSOConnection()
	data := schema.TestResourceDataRaw(t, resource.Schema, raw)
	data.SetId(connectionID)
	// Switch: remove OIDC, add AAD.
	r.NoError(data.Set(FieldSSOConnectionAAD, []map[string]interface{}{
		{
			FieldSSOConnectionADDomain:       "new-ad-domain",
			FieldSSOConnectionADClientID:     "new-client-id",
			FieldSSOConnectionADClientSecret: "new-client-secret",
		},
	}))

	mockClient.EXPECT().SSOAPIUpdateSSOConnection(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONRequestBody, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			got, err := json.Marshal(body)
			r.NoError(err)

			expected := []byte(`{
  "aad": {
    "adDomain": "new-ad-domain",
    "clientId": "new-client-id",
    "clientSecret": "new-client-secret"
  },
  "defaultRoleId": null,
  "name": "updated_name"
}`)
			eq, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(eq, fmt.Sprintf("got:      %v\nexpected: %v\n", string(got), string(expected)))

			return &http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"status":"STATUS_ACTIVE","name":"updated_name"}`))),
			}, nil
		}).Times(1)

	readBody := io.NopCloser(bytes.NewReader([]byte(`{"connection":{
  "id": "b6bfc074-a267-400f-b8f1-db0850c369b1",
  "name": "updated_name",
  "emailDomain": "test_email",
  "aad": {
    "adDomain": "new-ad-domain",
    "clientId": "new-client-id",
    "clientSecret": "new-client-secret"
  }
}}`)))
	mockClient.EXPECT().
		SSOAPIGetSSOConnection(gomock.Any(), connectionID).
		Return(&http.Response{StatusCode: 200, Body: readBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	result := resource.UpdateContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Empty(data.Get(FieldSSOConnectionOIDC))
	equalADConnector(t, r, data.Get(FieldSSOConnectionAAD), "new-ad-domain", "new-client-id", "new-client-secret")
}

func equalOIDCConnector(t *testing.T, r *require.Assertions, in interface{}, expectedIssuerURL, expectedClientID, expectedClientSecret, expectedType string) {
	t.Helper()
	r.NotNil(in)

	array, ok := in.([]interface{})
	r.True(ok)
	r.Len(array, 1)
	values, ok := array[0].(map[string]interface{})
	r.True(ok)

	issuerURL, ok := values["issuer_url"]
	r.True(ok)
	r.Equal(expectedIssuerURL, issuerURL)
	clientID, ok := values["client_id"]
	r.True(ok)
	r.Equal(expectedClientID, clientID)
	clientSecret, ok := values["client_secret"]
	r.True(ok)
	r.Equal(expectedClientSecret, clientSecret)
	oidcType, ok := values["type"]
	r.True(ok)
	r.Equal(expectedType, oidcType)
}

func equalOktaConnector(t *testing.T, r *require.Assertions, in interface{}, expectedDomain, expectedClientID, expectedClientSecret string) {
	t.Helper()
	r.NotNil(in)

	array, ok := in.([]interface{})
	r.True(ok)
	r.Len(array, 1)
	values, ok := array[0].(map[string]interface{})
	r.True(ok)

	domain, ok := values["okta_domain"]
	r.True(ok)
	r.Equal(expectedDomain, domain)
	clientID, ok := values["client_id"]
	r.True(ok)
	r.Equal(expectedClientID, clientID)
	clientSecret, ok := values["client_secret"]
	r.True(ok)
	r.Equal(expectedClientSecret, clientSecret)
}

func equalADConnector(t *testing.T, r *require.Assertions, in interface{}, expectedDomain, expectedClientID, expectedClientSecret string) {
	t.Helper()
	r.NotNil(in)

	array, ok := in.([]interface{})
	r.True(ok)
	r.Len(array, 1)
	values, ok := array[0].(map[string]interface{})
	r.True(ok)

	domain, ok := values["ad_domain"]
	r.True(ok)
	r.Equal(expectedDomain, domain)
	clientID, ok := values["client_id"]
	r.True(ok)
	r.Equal(expectedClientID, clientID)
	clientSecret, ok := values["client_secret"]
	r.True(ok)
	r.Equal(expectedClientSecret, clientSecret)
}

// newSSOTestProvider builds a ProviderConfig with mocked main-API and organization-management
// clients, returning both mocks for expectation setup.
func newSSOTestProvider(t *testing.T) (*ProviderConfig, *mock_sdk.MockClientInterface, *mockOrganizationManagement.MockClientWithResponsesInterface) {
	t.Helper()

	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
	mockOrgClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))
	return &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
		organizationManagementClient: mockOrgClient,
	}, mockClient, mockOrgClient
}

func okOrgSettingsGetResponse(blockInvitations *bool) *organization_management.SettingsAPIGetSettingsResponse {
	return &organization_management.SettingsAPIGetSettingsResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200:      &organization_management.GetSettingsResponse{BlockInvitations: blockInvitations},
	}
}

func okOrgSettingsUpdateResponse(blockInvitations *bool) *organization_management.SettingsAPIUpdateSettingsResponse {
	return &organization_management.SettingsAPIUpdateSettingsResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200:      &organization_management.UpdateSettingsResponse{BlockInvitations: blockInvitations},
	}
}

// okOrgEmailDomainIssueResponse builds a successful regenerate-tokens response carrying a
// TXT record for each given domain.
func okOrgEmailDomainIssueResponse(domains ...string) *organization_management.SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensResponse {
	issued := make([]organization_management.SSOEmailDomainValidation, 0, len(domains))
	for _, domain := range domains {
		issued = append(issued, organization_management.SSOEmailDomainValidation{
			Domain:    toPtr(domain),
			Status:    toPtr(organization_management.SSOEmailDomainValidationStatusSSOEMAILDOMAINSTATUSPENDING),
			TxtRecord: &organization_management.SSOEmailDomainTxtRecord{Name: toPtr("_castai-verification." + domain), Value: toPtr("issued-secret-" + domain)},
		})
	}
	return &organization_management.SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
		JSON200:      &organization_management.RegenerateSSOEmailDomainsTokensResponse{EmailDomains: &issued},
	}
}

// okSSOGetConnectionResponse returns a 200 response for SSOAPIGetSSOConnection with a
// minimal connection payload.
func okSSOGetConnectionResponse(connectionID string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Header:     map[string][]string{"Content-Type": {"json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"connection":{"id":%q,"name":"test_sso","emailDomain":"test_email"}}`, connectionID)))),
	}
}

func TestSSOConnection_SchemaOrganizationAndBlockInvitations(t *testing.T) {
	r := require.New(t)
	res := resourceSSOConnection()

	// organization_id must be Required, ForceNew and UUID-validated. Being Required is what
	// makes a plan fail when block_invitations is set without organization_id (Terraform core
	// rejects a missing Required argument).
	orgSchema := res.Schema[FieldSSOConnectionOrganizationID]
	r.Equal(schema.TypeString, orgSchema.Type)
	r.True(orgSchema.Required)
	r.True(orgSchema.ForceNew)
	r.NotNil(orgSchema.ValidateDiagFunc)

	blockSchema := res.Schema[FieldSSOConnectionBlockInvitations]
	r.Equal(schema.TypeBool, blockSchema.Type)
	r.True(blockSchema.Optional)
	r.True(blockSchema.Computed)
	r.False(blockSchema.Required)
}

func TestSSOConnection_BlockInvitationsInConfig(t *testing.T) {
	tests := []struct {
		name      string
		rawConfig cty.Value
		expected  bool
	}{
		{
			name:      "nil raw config (unit-test shim, import) counts as unset",
			rawConfig: cty.NilVal,
			expected:  false,
		},
		{
			name: "attribute missing from raw config counts as unset",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				FieldSSOConnectionName: cty.StringVal("conn"),
			}),
			expected: false,
		},
		{
			name: "null attribute counts as unset",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				FieldSSOConnectionBlockInvitations: cty.NullVal(cty.Bool),
			}),
			expected: false,
		},
		{
			name: "explicit false counts as set",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				FieldSSOConnectionBlockInvitations: cty.False,
			}),
			expected: true,
		},
		{
			name: "explicit true counts as set",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				FieldSSOConnectionBlockInvitations: cty.True,
			}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)

			resource := resourceSSOConnection()
			state := sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(uuid.NewString()),
			}), 0)
			state.RawConfig = tt.rawConfig
			data := resource.Data(state)

			r.Equal(tt.expected, blockInvitationsInConfig(data))
		})
	}
}

func TestSSOConnection_ReadContext_EmailDomainValidation(t *testing.T) {
	t.Run("read stores pending domain validation and warns with the record to publish", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, _ := newSSOTestProvider(t)
		connectionID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"connection":{"id":%q,"name":"test_sso","emailDomain":"pending.com","emailDomains":[{"domain":"pending.com","status":"STATUS_PENDING","lastCheckOutcome":"SSO_EMAIL_DOMAIN_CHECK_OUTCOME_TXT_RECORD_NOT_FOUND","txtRecord":{"name":"_castai-verification.pending.com","value":"verification-secret-123"}}]}}`, connectionID))))}, nil)
		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(connectionID),
			}), 0),
		)

		result := resource.ReadContext(context.Background(), data, provider)
		r.False(result.HasError())

		validation := data.Get(FieldSSOConnectionEmailDomainValidation).([]interface{})
		r.Len(validation, 1)
		entry := validation[0].(map[string]interface{})
		r.Equal("pending.com", entry[FieldSSOConnectionEmailDomainValidationDomain])
		r.Equal("STATUS_PENDING", entry[FieldSSOConnectionEmailDomainValidationStatus])
		r.Equal(false, entry[FieldSSOConnectionEmailDomainValidationVerified])
		r.Equal("SSO_EMAIL_DOMAIN_CHECK_OUTCOME_TXT_RECORD_NOT_FOUND", entry[FieldSSOConnectionEmailDomainValidationLastCheckOutcome])
		r.Equal("_castai-verification.pending.com", entry[FieldSSOConnectionEmailDomainValidationTxtRecordName])
		r.Equal("verification-secret-123", entry[FieldSSOConnectionEmailDomainValidationTxtRecordValue])

		r.Len(result, 1)
		r.Equal(diag.Warning, result[0].Severity)
		// The nudge carries the full record so it can be acted on directly.
		r.Contains(result[0].Detail, "_castai-verification.pending.com")
		r.Contains(result[0].Detail, "verification-secret-123")
	})

	t.Run("read stores verified domain without warning", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, _ := newSSOTestProvider(t)
		connectionID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"connection":{"id":%q,"name":"test_sso","emailDomain":"verified.com","emailDomains":[{"domain":"verified.com","status":"STATUS_VERIFIED","lastCheckOutcome":"SSO_EMAIL_DOMAIN_CHECK_OUTCOME_VERIFIED","verifiedAt":"2026-11-26T10:00:00Z","txtRecord":{"name":"_castai-verification.verified.com","value":"verification-secret-123"}}]}}`, connectionID))))}, nil)
		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(connectionID),
			}), 0),
		)

		result := resource.ReadContext(context.Background(), data, provider)
		r.Empty(result)

		validation := data.Get(FieldSSOConnectionEmailDomainValidation).([]interface{})
		r.Len(validation, 1)
		entry := validation[0].(map[string]interface{})
		r.Equal("STATUS_VERIFIED", entry[FieldSSOConnectionEmailDomainValidationStatus])
		r.Equal(true, entry[FieldSSOConnectionEmailDomainValidationVerified])
	})

	t.Run("read stores multiple domains sorted by domain and warns only for pending ones", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, _ := newSSOTestProvider(t)
		connectionID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"connection":{"id":%q,"name":"test_sso","emailDomain":"zeta.com","emailDomains":[{"domain":"zeta.com","status":"STATUS_VERIFIED"},{"domain":"alpha.com","status":"STATUS_PENDING","txtRecord":{"name":"_castai-verification.alpha.com","value":"alpha-secret"}}]}}`, connectionID))))}, nil)
		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(connectionID),
			}), 0),
		)

		result := resource.ReadContext(context.Background(), data, provider)
		r.False(result.HasError())

		validation := data.Get(FieldSSOConnectionEmailDomainValidation).([]interface{})
		r.Len(validation, 2)
		r.Equal("alpha.com", validation[0].(map[string]interface{})[FieldSSOConnectionEmailDomainValidationDomain])
		r.Equal("zeta.com", validation[1].(map[string]interface{})[FieldSSOConnectionEmailDomainValidationDomain])
		r.Equal(false, validation[0].(map[string]interface{})[FieldSSOConnectionEmailDomainValidationVerified])
		r.Equal(true, validation[1].(map[string]interface{})[FieldSSOConnectionEmailDomainValidationVerified])

		// Only the pending domain produces a warning.
		r.Len(result, 1)
		r.Contains(result[0].Detail, "alpha.com")
		r.NotContains(result[0].Detail, "zeta.com")
	})
}

func TestSSOConnection_ReadContext_EmailDomainRecordIssuance(t *testing.T) {
	t.Run("read does not issue records and warns when a pending claim has none", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()
		orgID := uuid.NewString()

		// The connection GET returns the claim status but never the secret.
		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"connection":{"id":%q,"name":"test_sso","emailDomain":"pending.com","emailDomains":[{"domain":"pending.com","status":"STATUS_PENDING"}]}}`, connectionID)))),
			}, nil)

		mockOrgClient.EXPECT().
			SettingsAPIGetSettingsWithResponse(gomock.Any(), orgID).
			Return(okOrgSettingsGetResponse(nil), nil)

		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id":                             cty.StringVal(connectionID),
				FieldSSOConnectionOrganizationID: cty.StringVal(orgID),
			}), 0),
		)

		// No SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensWithResponse expectation:
		// Read is side-effect free — issuance only happens in Create/Update, so a plan
		// refresh can never rotate a verification secret.

		result := resource.ReadContext(context.Background(), data, provider)
		r.False(result.HasError())

		validation := data.Get(FieldSSOConnectionEmailDomainValidation).([]interface{})
		r.Len(validation, 1)
		entry := validation[0].(map[string]interface{})
		r.Empty(entry[FieldSSOConnectionEmailDomainValidationTxtRecordName])
		r.Empty(entry[FieldSSOConnectionEmailDomainValidationTxtRecordValue])

		found := false
		for _, d := range result {
			if d.Summary == "SSO email domain is not verified yet" {
				found = true
				r.Contains(d.Detail, "could not be issued automatically")
			}
		}
		r.True(found)
	})

	t.Run("read preserves an already issued record from state without regenerating", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()
		orgID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"connection":{"id":%q,"name":"test_sso","emailDomain":"pending.com","emailDomains":[{"domain":"pending.com","status":"STATUS_PENDING"}]}}`, connectionID)))),
			}, nil)

		// No regenerate expectation: the record already lives in state and must not be rotated.
		// Read also mirrors block_invitations because organization_id is set.
		mockOrgClient.EXPECT().
			SettingsAPIGetSettingsWithResponse(gomock.Any(), orgID).
			Return(okOrgSettingsGetResponse(nil), nil)

		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id":                             cty.StringVal(connectionID),
				FieldSSOConnectionOrganizationID: cty.StringVal(orgID),
				FieldSSOConnectionEmailDomainValidation: cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					FieldSSOConnectionEmailDomainValidationDomain:           cty.StringVal("pending.com"),
					FieldSSOConnectionEmailDomainValidationStatus:           cty.StringVal("STATUS_PENDING"),
					FieldSSOConnectionEmailDomainValidationVerified:         cty.False,
					FieldSSOConnectionEmailDomainValidationLastCheckOutcome: cty.StringVal("SSO_EMAIL_DOMAIN_CHECK_OUTCOME_UNSPECIFIED"),
					FieldSSOConnectionEmailDomainValidationTxtRecordName:    cty.StringVal("_castai-verification.pending.com"),
					FieldSSOConnectionEmailDomainValidationTxtRecordValue:   cty.StringVal("state-secret"),
				})}),
			}), 0),
		)

		result := resource.ReadContext(context.Background(), data, provider)
		r.False(result.HasError())

		validation := data.Get(FieldSSOConnectionEmailDomainValidation).([]interface{})
		r.Len(validation, 1)
		entry := validation[0].(map[string]interface{})
		r.Equal("state-secret", entry[FieldSSOConnectionEmailDomainValidationTxtRecordValue])

		found := false
		for _, d := range result {
			if d.Summary == "SSO email domain is not verified yet" {
				found = true
				r.Contains(d.Detail, "_castai-verification.pending.com")
			}
		}
		r.True(found)
	})
}

func TestSSOConnection_CreateContext_EmailDomainRecords(t *testing.T) {
	t.Run("create issues verification records for the claimed domains", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()
		orgID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPICreateSSOConnection(gomock.Any(), gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"id":%q,"status":"STATUS_ACTIVE"}`, connectionID)))),
			}, nil)

		mockOrgClient.EXPECT().
			SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensWithResponse(gomock.Any(), orgID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, body organization_management.SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensJSONRequestBody) (*organization_management.SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensResponse, error) {
				r.Equal(orgID, body.OrganizationId)
				r.ElementsMatch([]string{"pending.com", "other.com"}, body.Domains)
				// Statuses deliberately use the org-management enum — they must not leak into state.
				return &organization_management.SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensResponse{
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200: &organization_management.RegenerateSSOEmailDomainsTokensResponse{
						EmailDomains: &[]organization_management.SSOEmailDomainValidation{
							{
								Domain:    toPtr("pending.com"),
								Status:    toPtr(organization_management.SSOEmailDomainValidationStatusSSOEMAILDOMAINSTATUSPENDING),
								TxtRecord: &organization_management.SSOEmailDomainTxtRecord{Name: toPtr("_castai-verification.pending.com"), Value: toPtr("issued-secret-1")},
							},
							{
								Domain:    toPtr("other.com"),
								Status:    toPtr(organization_management.SSOEmailDomainValidationStatusSSOEMAILDOMAINSTATUSPENDING),
								TxtRecord: &organization_management.SSOEmailDomainTxtRecord{Name: toPtr("_castai-verification.other.com"), Value: toPtr("issued-secret-2")},
							},
						},
					},
				}, nil
			})

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"connection":{"id":%q,"name":"test_sso","emailDomain":"pending.com","emailDomains":[{"domain":"other.com","status":"STATUS_PENDING"},{"domain":"pending.com","status":"STATUS_PENDING"}]}}`, connectionID)))),
			}, nil)

		mockOrgClient.EXPECT().
			SettingsAPIGetSettingsWithResponse(gomock.Any(), orgID).
			Return(okOrgSettingsGetResponse(nil), nil)

		resource := resourceSSOConnection()
		val := cty.ObjectVal(map[string]cty.Value{
			FieldSSOConnectionName:                   cty.StringVal("test_sso"),
			FieldSSOConnectionEmailDomain:            cty.StringVal("pending.com"),
			FieldSSOConnectionAdditionalEmailDomains: cty.ListVal([]cty.Value{cty.StringVal("other.com")}),
			FieldSSOConnectionOrganizationID:         cty.StringVal(orgID),
		})
		state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
		state.RawConfig = val
		data := resource.Data(state)

		result := resource.CreateContext(context.Background(), data, provider)
		r.False(result.HasError())

		validation := data.Get(FieldSSOConnectionEmailDomainValidation).([]interface{})
		r.Len(validation, 2)
		byDomain := map[string]map[string]interface{}{}
		for _, v := range validation {
			e := v.(map[string]interface{})
			byDomain[e[FieldSSOConnectionEmailDomainValidationDomain].(string)] = e
		}
		r.Equal("issued-secret-1", byDomain["pending.com"][FieldSSOConnectionEmailDomainValidationTxtRecordValue])
		r.Equal("issued-secret-2", byDomain["other.com"][FieldSSOConnectionEmailDomainValidationTxtRecordValue])
		// Statuses come from the connection GET (SSO enum), not from the issuance response.
		r.Equal("STATUS_PENDING", byDomain["pending.com"][FieldSSOConnectionEmailDomainValidationStatus])
		r.Equal("STATUS_PENDING", byDomain["other.com"][FieldSSOConnectionEmailDomainValidationStatus])
	})
}

func TestSSOConnection_UpdateContext_EmailDomainRecords(t *testing.T) {
	t.Run("update issues verification records for claimed domains without one", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()
		orgID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPIUpdateSSOConnection(gomock.Any(), connectionID, gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"name":"test_sso","status":"STATUS_ACTIVE"}`))),
			}, nil)

		mockOrgClient.EXPECT().
			SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensWithResponse(gomock.Any(), orgID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, body organization_management.SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensJSONRequestBody) (*organization_management.SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensResponse, error) {
				r.Equal([]string{"new.com"}, body.Domains)
				return &organization_management.SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensResponse{
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					JSON200: &organization_management.RegenerateSSOEmailDomainsTokensResponse{
						EmailDomains: &[]organization_management.SSOEmailDomainValidation{{
							Domain:    toPtr("new.com"),
							Status:    toPtr(organization_management.SSOEmailDomainValidationStatusSSOEMAILDOMAINSTATUSPENDING),
							TxtRecord: &organization_management.SSOEmailDomainTxtRecord{Name: toPtr("_castai-verification.new.com"), Value: toPtr("issued-secret")},
						}},
					},
				}, nil
			})

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"connection":{"id":%q,"name":"test_sso","emailDomain":"new.com","emailDomains":[{"domain":"new.com","status":"STATUS_PENDING"}]}}`, connectionID)))),
			}, nil)

		mockOrgClient.EXPECT().
			SettingsAPIGetSettingsWithResponse(gomock.Any(), orgID).
			Return(okOrgSettingsGetResponse(nil), nil)

		resource := resourceSSOConnection()
		data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
			FieldSSOConnectionEmailDomain:    "new.com",
			FieldSSOConnectionOrganizationID: orgID,
		})
		data.SetId(connectionID)

		result := resource.UpdateContext(context.Background(), data, provider)
		r.False(result.HasError())

		validation := data.Get(FieldSSOConnectionEmailDomainValidation).([]interface{})
		r.Len(validation, 1)
		entry := validation[0].(map[string]interface{})
		r.Equal("issued-secret", entry[FieldSSOConnectionEmailDomainValidationTxtRecordValue])
		r.Equal("STATUS_PENDING", entry[FieldSSOConnectionEmailDomainValidationStatus])
	})
}

func TestSSOConnection_ReadContext_BlockInvitations(t *testing.T) {
	t.Run("read keeps prior block_invitations when the settings response omits the field", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()
		orgID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(okSSOGetConnectionResponse(connectionID), nil)

		// 200 with an empty payload: BlockInvitations is nil and must not clobber state.
		mockOrgClient.EXPECT().
			SettingsAPIGetSettingsWithResponse(gomock.Any(), orgID).
			Return(&organization_management.SettingsAPIGetSettingsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &organization_management.GetSettingsResponse{},
			}, nil)

		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id":                               cty.StringVal(connectionID),
				FieldSSOConnectionOrganizationID:   cty.StringVal(orgID),
				FieldSSOConnectionBlockInvitations: cty.True,
			}), 0),
		)

		result := resource.ReadContext(context.Background(), data, provider)
		r.False(result.HasError())
		r.True(data.Get(FieldSSOConnectionBlockInvitations).(bool))
	})

	t.Run("read mirrors block_invitations from organization settings", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()
		orgID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(okSSOGetConnectionResponse(connectionID), nil)

		mockOrgClient.EXPECT().
			SettingsAPIGetSettingsWithResponse(gomock.Any(), orgID).
			Return(okOrgSettingsGetResponse(toPtr(true)), nil)

		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id":                               cty.StringVal(connectionID),
				FieldSSOConnectionOrganizationID:   cty.StringVal(orgID),
				FieldSSOConnectionBlockInvitations: cty.False,
			}), 0),
		)

		result := resource.ReadContext(context.Background(), data, provider)
		r.False(result.HasError())
		r.True(data.Get(FieldSSOConnectionBlockInvitations).(bool))
		r.Equal(orgID, data.Get(FieldSSOConnectionOrganizationID))
	})

	t.Run("read without organization_id keeps prior block_invitations and skips settings API", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(okSSOGetConnectionResponse(connectionID), nil)

		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id":                               cty.StringVal(connectionID),
				FieldSSOConnectionBlockInvitations: cty.True,
			}), 0),
		)

		result := resource.ReadContext(context.Background(), data, provider)
		r.False(result.HasError())
		// No SettingsAPIGetSettingsWithResponse expectation: an unexpected call would fail the mock.
		r.True(data.Get(FieldSSOConnectionBlockInvitations).(bool))
		_ = mockOrgClient
	})

	t.Run("read keeps prior block_invitations and warns when settings call fails", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()
		orgID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(okSSOGetConnectionResponse(connectionID), nil)

		mockOrgClient.EXPECT().
			SettingsAPIGetSettingsWithResponse(gomock.Any(), orgID).
			Return(&organization_management.SettingsAPIGetSettingsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusForbidden},
				Body:         []byte(`{"message":"forbidden"}`),
			}, nil)

		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id":                               cty.StringVal(connectionID),
				FieldSSOConnectionOrganizationID:   cty.StringVal(orgID),
				FieldSSOConnectionBlockInvitations: cty.True,
			}), 0),
		)

		result := resource.ReadContext(context.Background(), data, provider)
		r.False(result.HasError())
		r.Len(result, 1)
		r.Equal(diag.Warning, result[0].Severity)
		// Prior value is kept in state.
		r.True(data.Get(FieldSSOConnectionBlockInvitations).(bool))
	})
}

func TestSSOConnection_CreateContext_BlockInvitations(t *testing.T) {
	t.Run("explicit block_invitations=false is sent to organization settings", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()
		orgID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPICreateSSOConnection(gomock.Any(), gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"id":%q,"status":"STATUS_ACTIVE"}`, connectionID)))),
			}, nil)
		mockOrgClient.EXPECT().
			SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensWithResponse(gomock.Any(), orgID, gomock.Any()).
			Return(okOrgEmailDomainIssueResponse("test_email"), nil)

		mockOrgClient.EXPECT().
			SettingsAPIUpdateSettingsWithResponse(gomock.Any(), orgID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, body organization_management.SettingsAPIUpdateSettingsJSONRequestBody) (*organization_management.SettingsAPIUpdateSettingsResponse, error) {
				got, err := json.Marshal(body)
				r.NoError(err)
				expected := []byte(`{"blockInvitations": false}`)
				eq, err := JSONBytesEqual(got, expected)
				r.NoError(err)
				r.True(eq, fmt.Sprintf("got: %s, expected: %s", got, expected))
				return okOrgSettingsUpdateResponse(toPtr(false)), nil
			})

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(okSSOGetConnectionResponse(connectionID), nil)

		mockOrgClient.EXPECT().
			SettingsAPIGetSettingsWithResponse(gomock.Any(), orgID).
			Return(okOrgSettingsGetResponse(toPtr(false)), nil)

		resource := resourceSSOConnection()
		val := cty.ObjectVal(map[string]cty.Value{
			FieldSSOConnectionName:             cty.StringVal("test_sso"),
			FieldSSOConnectionEmailDomain:      cty.StringVal("test_email"),
			FieldSSOConnectionOrganizationID:   cty.StringVal(orgID),
			FieldSSOConnectionBlockInvitations: cty.False,
		})
		state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
		state.RawConfig = val
		data := resource.Data(state)

		result := resource.CreateContext(context.Background(), data, provider)
		r.False(result.HasError())
		r.Equal(connectionID, data.Id())
		r.False(data.Get(FieldSSOConnectionBlockInvitations).(bool))
	})

	t.Run("unset block_invitations does not call the settings update endpoint", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()
		orgID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPICreateSSOConnection(gomock.Any(), gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"id":%q,"status":"STATUS_ACTIVE"}`, connectionID)))),
			}, nil)
		mockOrgClient.EXPECT().
			SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensWithResponse(gomock.Any(), orgID, gomock.Any()).
			Return(okOrgEmailDomainIssueResponse("test_email"), nil)

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(okSSOGetConnectionResponse(connectionID), nil)

		// Read mirrors the org setting whenever organization_id is known; the PATCH endpoint
		// has no expectation, so an unexpected call fails the mock.
		mockOrgClient.EXPECT().
			SettingsAPIGetSettingsWithResponse(gomock.Any(), orgID).
			Return(okOrgSettingsGetResponse(toPtr(true)), nil)

		resource := resourceSSOConnection()
		val := cty.ObjectVal(map[string]cty.Value{
			FieldSSOConnectionName:           cty.StringVal("test_sso"),
			FieldSSOConnectionEmailDomain:    cty.StringVal("test_email"),
			FieldSSOConnectionOrganizationID: cty.StringVal(orgID),
		})
		state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
		state.RawConfig = val
		data := resource.Data(state)

		result := resource.CreateContext(context.Background(), data, provider)
		r.False(result.HasError())
		r.True(data.Get(FieldSSOConnectionBlockInvitations).(bool))
	})

	t.Run("settings update failure on create returns a warning and keeps the connection", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()
		orgID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPICreateSSOConnection(gomock.Any(), gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"id":%q,"status":"STATUS_ACTIVE"}`, connectionID)))),
			}, nil)
		mockOrgClient.EXPECT().
			SSOEmailDomainAPIRegenerateSSOEmailDomainsTokensWithResponse(gomock.Any(), orgID, gomock.Any()).
			Return(okOrgEmailDomainIssueResponse("test_email"), nil)

		mockOrgClient.EXPECT().
			SettingsAPIUpdateSettingsWithResponse(gomock.Any(), orgID, gomock.Any()).
			Return(&organization_management.SettingsAPIUpdateSettingsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError},
				Body:         []byte(`{"message":"boom"}`),
			}, nil)

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(okSSOGetConnectionResponse(connectionID), nil)

		mockOrgClient.EXPECT().
			SettingsAPIGetSettingsWithResponse(gomock.Any(), orgID).
			Return(okOrgSettingsGetResponse(toPtr(true)), nil)

		resource := resourceSSOConnection()
		val := cty.ObjectVal(map[string]cty.Value{
			FieldSSOConnectionName:             cty.StringVal("test_sso"),
			FieldSSOConnectionEmailDomain:      cty.StringVal("test_email"),
			FieldSSOConnectionOrganizationID:   cty.StringVal(orgID),
			FieldSSOConnectionBlockInvitations: cty.False,
		})
		state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
		state.RawConfig = val
		data := resource.Data(state)

		result := resource.CreateContext(context.Background(), data, provider)
		// Warning only: the connection stays in state (not tainted) and the next plan shows drift.
		r.False(result.HasError())
		r.Len(result, 1)
		r.Equal(diag.Warning, result[0].Severity)
		r.Equal(connectionID, data.Id())
	})
}

func TestSSOConnection_UpdateContext_BlockInvitations(t *testing.T) {
	// Note: schema.TestResourceDataRaw produces a real diff (so HasChange works) but never
	// populates RawConfig, so blockInvitationsInConfig is false here and the settings PATCH
	// branch cannot be exercised in this unit-test form. The subtest therefore pins the
	// negative contract: a flag-only change must never reach the SSO update API. The
	// positive path (managed flag → settings PATCH) is covered by Create tests and the
	// acceptance test.
	t.Run("changing only block_invitations does not call the sso connection update API", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()
		orgID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(okSSOGetConnectionResponse(connectionID), nil)

		mockOrgClient.EXPECT().
			SettingsAPIGetSettingsWithResponse(gomock.Any(), orgID).
			Return(okOrgSettingsGetResponse(toPtr(true)), nil)

		resource := resourceSSOConnection()
		data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
			FieldSSOConnectionOrganizationID:   orgID,
			FieldSSOConnectionBlockInvitations: true,
		})
		data.SetId(connectionID)

		result := resource.UpdateContext(context.Background(), data, provider)
		r.False(result.HasError())
		// No SSOAPIUpdateSSOConnection nor SettingsAPIUpdateSettingsWithResponse expectation:
		// an unexpected call would fail the mock.
		r.Equal("test_sso", data.Get(FieldSSOConnectionName))
	})

	t.Run("updating sso fields does not call the settings update endpoint", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, mockClient, mockOrgClient := newSSOTestProvider(t)
		connectionID := uuid.NewString()
		orgID := uuid.NewString()

		mockClient.EXPECT().
			SSOAPIUpdateSSOConnection(gomock.Any(), connectionID, gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"name":"updated_name","status":"STATUS_ACTIVE"}`))),
			}, nil)

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{
				StatusCode: 200,
				Header:     map[string][]string{"Content-Type": {"json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"connection":{"id":%q,"name":"updated_name","emailDomain":"test_email"}}`, connectionID)))),
			}, nil)

		mockOrgClient.EXPECT().
			SettingsAPIGetSettingsWithResponse(gomock.Any(), orgID).
			Return(okOrgSettingsGetResponse(toPtr(true)), nil)

		resource := resourceSSOConnection()
		data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
			FieldSSOConnectionName:           "updated_name",
			FieldSSOConnectionOrganizationID: orgID,
		})
		data.SetId(connectionID)

		result := resource.UpdateContext(context.Background(), data, provider)
		r.False(result.HasError())
		// No SettingsAPIUpdateSettingsWithResponse expectation: an unexpected call would fail the mock.
		r.Equal("updated_name", data.Get(FieldSSOConnectionName))
	})
}

func TestSSOConnection_UpdateSSOBlockInvitations(t *testing.T) {
	t.Run("sends only the blockInvitations key", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, _, mockOrgClient := newSSOTestProvider(t)
		orgID := uuid.NewString()

		mockOrgClient.EXPECT().
			SettingsAPIUpdateSettingsWithResponse(gomock.Any(), orgID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, body organization_management.SettingsAPIUpdateSettingsJSONRequestBody) (*organization_management.SettingsAPIUpdateSettingsResponse, error) {
				got, err := json.Marshal(body)
				r.NoError(err)
				expected := []byte(`{"blockInvitations": false}`)
				eq, err := JSONBytesEqual(got, expected)
				r.NoError(err)
				r.True(eq, fmt.Sprintf("got: %s, expected: %s", got, expected))
				return okOrgSettingsUpdateResponse(toPtr(false)), nil
			})

		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id":                             cty.StringVal(uuid.NewString()),
				FieldSSOConnectionOrganizationID: cty.StringVal(orgID),
			}), 0),
		)

		err := updateSSOBlockInvitations(context.Background(), provider, data, false)
		r.NoError(err)
	})

	t.Run("returns error when the settings call fails", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, _, mockOrgClient := newSSOTestProvider(t)
		orgID := uuid.NewString()

		mockOrgClient.EXPECT().
			SettingsAPIUpdateSettingsWithResponse(gomock.Any(), orgID, gomock.Any()).
			Return(&organization_management.SettingsAPIUpdateSettingsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError},
				Body:         []byte(`{"message":"boom"}`),
			}, nil)

		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id":                             cty.StringVal(uuid.NewString()),
				FieldSSOConnectionOrganizationID: cty.StringVal(orgID),
			}), 0),
		)

		err := updateSSOBlockInvitations(context.Background(), provider, data, true)
		r.Error(err)
		r.Contains(err.Error(), "updating organization settings")
	})

	t.Run("returns error without organization_id", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		provider, _, _ := newSSOTestProvider(t)

		resource := resourceSSOConnection()
		data := resource.Data(
			sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(uuid.NewString()),
			}), 0),
		)

		err := updateSSOBlockInvitations(context.Background(), provider, data, true)
		r.Error(err)
	})
}

func TestSSOConnection_DeleteContext_WithOrganization(t *testing.T) {
	r := require.New(t)
	provider, mockClient, mockOrgClient := newSSOTestProvider(t)
	connectionID := uuid.NewString()
	orgID := uuid.NewString()

	// The org-wide flag must be reset before the connection is deleted: if the connection
	// is removed first, a failed reset could not be retried through Terraform anymore.
	settingsCall := mockOrgClient.EXPECT().
		SettingsAPIUpdateSettingsWithResponse(gomock.Any(), orgID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, body organization_management.SettingsAPIUpdateSettingsJSONRequestBody) (*organization_management.SettingsAPIUpdateSettingsResponse, error) {
			got, err := json.Marshal(body)
			r.NoError(err)
			expected := []byte(`{"blockInvitations": false}`)
			eq, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(eq, fmt.Sprintf("got: %s, expected: %s", got, expected))
			return okOrgSettingsUpdateResponse(toPtr(false)), nil
		})

	mockClient.EXPECT().
		SSOAPIDeleteSSOConnection(gomock.Any(), connectionID).
		After(settingsCall).
		Return(&http.Response{
			StatusCode: 200,
			Header:     map[string][]string{"Content-Type": {"json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		}, nil)

	resource := resourceSSOConnection()
	val := cty.ObjectVal(map[string]cty.Value{
		"id":                               cty.StringVal(connectionID),
		FieldSSOConnectionOrganizationID:   cty.StringVal(orgID),
		FieldSSOConnectionBlockInvitations: cty.True,
	})
	state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
	state.RawConfig = val
	data := resource.Data(state)

	result := resource.DeleteContext(context.Background(), data, provider)
	r.False(result.HasError())
}

// Deleting a connection that does not manage block_invitations (attribute absent from the
// configuration) must not touch the org-wide setting: it may have been set by the console
// or by another SSO connection in the same organization.
func TestSSOConnection_DeleteContext_UnmanagedBlockInvitations(t *testing.T) {
	r := require.New(t)
	provider, mockClient, mockOrgClient := newSSOTestProvider(t)
	connectionID := uuid.NewString()
	orgID := uuid.NewString()

	mockClient.EXPECT().
		SSOAPIDeleteSSOConnection(gomock.Any(), connectionID).
		Return(&http.Response{
			StatusCode: 200,
			Header:     map[string][]string{"Content-Type": {"json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		}, nil)

	resource := resourceSSOConnection()
	val := cty.ObjectVal(map[string]cty.Value{
		"id":                             cty.StringVal(connectionID),
		FieldSSOConnectionName:           cty.StringVal("test_sso"),
		FieldSSOConnectionEmailDomain:    cty.StringVal("test_email"),
		FieldSSOConnectionOrganizationID: cty.StringVal(orgID),
		// No block_invitations attribute: the flag is not managed by this resource.
	})
	state := sdkterraform.NewInstanceStateShimmedFromValue(val, 0)
	state.RawConfig = val
	data := resource.Data(state)

	// No SettingsAPIUpdateSettingsWithResponse expectation: an unexpected call would fail the mock.
	_ = mockOrgClient

	result := resource.DeleteContext(context.Background(), data, provider)
	r.False(result.HasError())
}
