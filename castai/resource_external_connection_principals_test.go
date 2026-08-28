package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/external_connections"
	mock_external_connections "github.com/castai/terraform-provider-castai/castai/sdk/external_connections/mock"
)

func extPrincipalsMockProvider(mockClient *mock_external_connections.MockClientInterface) *ProviderConfig {
	return &ProviderConfig{
		externalConnectionsClient: &external_connections.ClientWithResponses{ClientInterface: mockClient},
	}
}

func extPrincipalsTestResponse() external_connections.ProvisionCastPrincipalsResponse {
	connID := "conn-123"
	saEmail := "cast-c39cca43@castai-identity.iam.gserviceaccount.com"
	saID := "1234567890"
	return external_connections.ProvisionCastPrincipalsResponse{
		ResourceSuffix: "c39cca43",
		ConnectionId:   &connID,
		ProvisionedResources: []external_connections.ProvisionedResource{
			{
				Feature: external_connections.ProvisionedResourceFeatureCLOUDCONNECT,
				GcpServiceAccount: &external_connections.GCPServiceAccount{
					Email: saEmail,
					Id:    saID,
				},
			},
		},
		Permissions: []external_connections.Permission{
			{
				Name:         "iam.serviceAccounts.actAs",
				AccessType:   external_connections.WRITE,
				ResourceType: external_connections.GCPIAMBINDING,
				Scope:        external_connections.PermissionScopeGCPPROJECT,
				FeatureId:    "cloud_connect",
				Justification: "Allow Cast to impersonate the customer service account.",
			},
		},
	}
}

func extPrincipalsJSONResponse(t *testing.T) []byte {
	t.Helper()
	body, err := json.Marshal(extPrincipalsTestResponse())
	require.NoError(t, err)
	return body
}

func extPrincipalsJSONHTTPResponse(code int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: code,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func setExtPrincipalsBaseFields(t *testing.T, data interface {
	Set(key string, value interface{}) error
}) {
	require.NoError(t, data.Set(FieldExtConnPrincipalsCloudProvider, "GCP"))
	require.NoError(t, data.Set(FieldExtConnPrincipalsConnectionScope, "GCP_PROJECT"))
	require.NoError(t, data.Set(FieldExtConnPrincipalsOrganizationID, testExtConnOrgID))
	require.NoError(t, data.Set(FieldExtConnPrincipalsScopeKey, "my-project-123"))
	require.NoError(t, data.Set(FieldExtConnPrincipalsFeatures, []map[string]any{
		{"feature": "CLOUD_CONNECT"},
	}))
}

func TestExternalConnectionPrincipals_Create(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	mockClient := mock_external_connections.NewMockClientInterface(gomock.NewController(t))
	provider := extPrincipalsMockProvider(mockClient)

	mockClient.EXPECT().
		ExternalConnectionsAPIProvisionCastPrincipals(gomock.Any(), testExtConnOrgID, gomock.Any()).
		Return(extPrincipalsJSONHTTPResponse(200, extPrincipalsJSONResponse(t)), nil)

	res := resourceExternalConnectionPrincipals()
	data := res.TestResourceData()
	setExtPrincipalsBaseFields(t, data)

	diags := resourceExternalConnectionPrincipalsCreate(context.Background(), data, provider)
	r.False(diags.HasError(), "create should not return errors: %v", diags)

	// ID is derived deterministically from orgID-cloud-scopeKey
	r.Equal(testExtConnOrgID+"-GCP-my-project-123", data.Id())
	r.Equal("c39cca43", data.Get(FieldExtConnPrincipalsResourceSuffix))
	r.Equal("conn-123", data.Get(FieldExtConnPrincipalsConnectionID))

	// Check provisioned_resources
	pr := data.Get(FieldExtConnPrincipalsProvisionedRes).([]any)
	r.Len(pr, 1)
	prMap := pr[0].(map[string]any)
	r.Equal("CLOUD_CONNECT", prMap[FieldExtConnPrincipalsPRFeature])
	r.Equal("cast-c39cca43@castai-identity.iam.gserviceaccount.com", prMap[FieldExtConnPrincipalsPRGcpSAEmail])
	r.Equal("1234567890", prMap[FieldExtConnPrincipalsPRGcpSAID])

	// Check permissions
	perms := data.Get(FieldExtConnPrincipalsPermissions).([]any)
	r.Len(perms, 1)
	permMap := perms[0].(map[string]any)
	r.Equal("iam.serviceAccounts.actAs", permMap[FieldExtConnPrincipalsPermName])
	r.Equal("WRITE", permMap[FieldExtConnPrincipalsPermAccessType])
	r.Equal("GCP_IAM_BINDING", permMap[FieldExtConnPrincipalsPermResourceType])
	r.Equal("cloud_connect", permMap[FieldExtConnPrincipalsPermFeatureID])
}

func TestExternalConnectionPrincipals_Delete(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	res := resourceExternalConnectionPrincipals()
	data := res.TestResourceData()
	data.SetId(testExtConnOrgID + "-GCP-my-project-123")
	setExtPrincipalsBaseFields(t, data)

	// Delete is a no-op — no mock expectations set since no API call is made.
	provider := extPrincipalsMockProvider(
		mock_external_connections.NewMockClientInterface(gomock.NewController(t)),
	)

	diags := resourceExternalConnectionPrincipalsDelete(context.Background(), data, provider)

	// Should return no error but a Warning diagnostic
	r.False(diags.HasError(), "delete should not return errors")
	r.Len(diags, 1)
	r.Equal(diag.Warning, diags[0].Severity)
	r.Contains(diags[0].Summary, "not deleted")
}
