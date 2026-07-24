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
)

func TestAccCloudAgnostic_ResourceSSOConnection(t *testing.T) {
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
			DoAndReturn(func(_ context.Context, body sdk.SSOAPICreateSSOConnectionJSONRequestBody) (*http.Response, error) {
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
			DoAndReturn(func(_ context.Context, body sdk.SSOAPICreateSSOConnectionJSONRequestBody) (*http.Response, error) {
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
		DoAndReturn(func(_ context.Context, body sdk.SSOAPICreateSSOConnectionJSONRequestBody) (*http.Response, error) {
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
			DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONRequestBody) (*http.Response, error) {
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
			DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONRequestBody) (*http.Response, error) {
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
		DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONRequestBody) (*http.Response, error) {
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

func TestSSOConnection_CreateOIDCConnector(t *testing.T) {
	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	mockClient.EXPECT().
		SSOAPICreateSSOConnection(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, body sdk.SSOAPICreateSSOConnectionJSONRequestBody) (*http.Response, error) {
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
		DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONRequestBody) (*http.Response, error) {
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
		DoAndReturn(func(_ context.Context, _ string, body sdk.SSOAPIUpdateSSOConnectionJSONRequestBody) (*http.Response, error) {
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
