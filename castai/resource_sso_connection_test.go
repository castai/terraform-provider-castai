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
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestAccResourceSSOConnection(t *testing.T) {
	rName := fmt.Sprintf("%v-sso-connection-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_sso_connection.test"

	clientID := os.Getenv("SSO_CLIENT_ID")
	clientSecret := os.Getenv("SSO_CLIENT_SECRET")
	domain := os.Getenv("SSO_DOMAIN")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccSSOConnectionConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCreateSSOConnectionConfig(rName, clientID, clientSecret, domain),
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

func TestSSOConnection_ReadContext(t *testing.T) {
	t.Run("read azure ad connector", func(t *testing.T) {
		t.Parallel()

		readBody := `{"id":"fce35ba2-5c06-4078-8391-1ac8f7ba798b","name":"test_sso","createdAt":"2023-11-02T10:49:14.376757Z","updatedAt":"2023-11-02T10:49:14.450828Z","emailDomain":"test_email","additionalEmailDomains":[],"aad":{"adDomain":"test_connector","clientId":"test_client","clientSecret":"test_secret"}}`

		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		connectionID := "fce35ba2-5c06-4078-8391-1ac8f7ba798b"

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(readBody))), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		resource := resourceSSOConnection()
		data := resource.Data(
			terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(connectionID),
			}), 0))

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

		readBody := `{"id":"fce35ba2-5c06-4078-8391-1ac8f7ba798b","name":"test_sso","createdAt":"2023-11-02T10:49:14.376757Z","updatedAt":"2023-11-02T10:49:14.450828Z","emailDomain":"test_email","additionalEmailDomains":["domain.com", "other.com"],"aad":{"adDomain":"test_connector","clientId":"test_client","clientSecret":"test_secret"}}`

		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		connectionID := "fce35ba2-5c06-4078-8391-1ac8f7ba798b"

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte(readBody))), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		resource := resourceSSOConnection()
		data := resource.Data(
			terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(connectionID),
			}), 0))

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
			DoAndReturn(func(_ context.Context, body sdk.SSOAPICreateSSOConnectionJSONBody) (*http.Response, error) {
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
		readBody := io.NopCloser(bytes.NewReader([]byte(`{
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
}`)))

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: readBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		resource := resourceSSOConnection()
		data := resource.Data(terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
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
			DoAndReturn(func(_ context.Context, body sdk.SSOAPICreateSSOConnectionJSONBody) (*http.Response, error) {
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
		readBody := io.NopCloser(bytes.NewReader([]byte(`{
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
}`)))

		mockClient.EXPECT().
			SSOAPIGetSSOConnection(gomock.Any(), connectionID).
			Return(&http.Response{StatusCode: 200, Body: readBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		resource := resourceSSOConnection()
		data := resource.Data(terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
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
		DoAndReturn(func(_ context.Context, body sdk.SSOAPICreateSSOConnectionJSONBody) (*http.Response, error) {
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
	readBody := io.NopCloser(bytes.NewReader([]byte(`{
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
}`)))

	mockClient.EXPECT().
		SSOAPIGetSSOConnection(gomock.Any(), connectionID).
		Return(&http.Response{StatusCode: 200, Body: readBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceSSOConnection()
	data := resource.Data(terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
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
			DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONBody) (*http.Response, error) {
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

		readBody := io.NopCloser(bytes.NewReader([]byte(`{
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
}`)))
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
			DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONBody) (*http.Response, error) {
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

		readBody := io.NopCloser(bytes.NewReader([]byte(`{
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
}`)))
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
		DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONBody) (*http.Response, error) {
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

	readBody := io.NopCloser(bytes.NewReader([]byte(`{
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
}`)))
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
		terraform.NewInstanceStateShimmedFromValue(
			cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal(connectionID),
			}), 0),
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

func testAccCreateSSOConnectionConfig(rName, clientID, clientSecret, adDomain string) string {
	return fmt.Sprintf(`
resource "castai_sso_connection" "test" {
  name            = %[1]q
  email_domain = "aad_connection@test.com"
  aad {
    client_id     = %[2]q
    client_secret = %[3]q
    ad_domain     = %[4]q
  }
}`, rName, clientID, clientSecret, adDomain)
}

func testAccSSOConnectionConfigurationDestroy(s *terraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).api
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
