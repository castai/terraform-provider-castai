package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/external_connections"
	mock_external_connections "github.com/castai/terraform-provider-castai/castai/sdk/external_connections/mock"
)

const (
	testExtConnID       = "conn-123"
	testExtConnOrgID    = "12345678-1234-1234-1234-123456789012"
	testExtConnScopeKey = "123456789012"
	testExtConnSuffix   = "c39cca43"
)

func extConnMockProvider(mockClient *mock_external_connections.MockClientInterface) *ProviderConfig {
	return &ProviderConfig{
		externalConnectionsClient: &external_connections.ClientWithResponses{ClientInterface: mockClient},
	}
}

func extConnTestTime() time.Time {
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}

func extConnUpsertJSON(t *testing.T) []byte {
	t.Helper()
	resp := external_connections.UpsertConnectionResponse{
		Connection: extConnTestConnection(),
	}
	body, err := json.Marshal(resp)
	require.NoError(t, err)
	return body
}

func extConnGetJSON(t *testing.T) []byte {
	t.Helper()
	body, err := json.Marshal(extConnTestConnection())
	require.NoError(t, err)
	return body
}

func extConnTestConnection() external_connections.Connection {
	return external_connections.Connection{
		Id:             testExtConnID,
		Cloud:          external_connections.ConnectionCloudAWS,
		Scope:          external_connections.ConnectionScopeAWSACCOUNT,
		ScopeKey:       testExtConnScopeKey,
		ResourceSuffix: testExtConnSuffix,
		OrganizationId: testExtConnOrgID,
		EnabledFeatures: []external_connections.EnabledFeature{
			{
				Feature:         external_connections.EnabledFeatureFeatureCLOUDCONNECT,
				RegistryVersion: "v1",
			},
		},
		CreateTime: extConnTestTime(),
		UpdateTime: extConnTestTime(),
	}
}

func extConnJSONResponse(code int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: code,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func extConnSetBaseFields(t *testing.T, data interface {
	Set(key string, value interface{}) error
	Get(key string) interface{}
}) {
	require.NoError(t, data.Set(FieldExternalConnectionCloud, "AWS"))
	require.NoError(t, data.Set(FieldExternalConnectionScope, "AWS_ACCOUNT"))
	require.NoError(t, data.Set(FieldExternalConnectionScopeKey, testExtConnScopeKey))
	require.NoError(t, data.Set(FieldExternalConnectionResourceSuffix, testExtConnSuffix))
	require.NoError(t, data.Set(FieldExternalConnectionOrganizationID, testExtConnOrgID))
	require.NoError(t, data.Set(FieldExternalConnectionEnabledFeatures, []map[string]any{
		{"feature": "CLOUD_CONNECT"},
	}))
}

func TestExternalConnection_Create(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	mockClient := mock_external_connections.NewMockClientInterface(gomock.NewController(t))
	provider := extConnMockProvider(mockClient)

	// Create calls Upsert, then Read calls Get
	mockClient.EXPECT().
		ExternalConnectionsAPIUpsertConnection(gomock.Any(), testExtConnOrgID, gomock.Any()).
		Return(extConnJSONResponse(200, extConnUpsertJSON(t)), nil)
	mockClient.EXPECT().
		ExternalConnectionsAPIGetConnection(gomock.Any(), testExtConnOrgID, testExtConnID).
		Return(extConnJSONResponse(200, extConnGetJSON(t)), nil)

	res := resourceExternalConnection()
	data := res.TestResourceData()
	extConnSetBaseFields(t, data)

	diags := resourceExternalConnectionCreate(context.Background(), data, provider)
	r.False(diags.HasError(), "create should not return errors: %v", diags)

	r.Equal(testExtConnID, data.Id())
	r.Equal("AWS", data.Get(FieldExternalConnectionCloud))
	r.Equal("AWS_ACCOUNT", data.Get(FieldExternalConnectionScope))
	r.Equal(testExtConnScopeKey, data.Get(FieldExternalConnectionScopeKey))
	r.Equal(testExtConnSuffix, data.Get(FieldExternalConnectionResourceSuffix))
	r.Equal(testExtConnOrgID, data.Get(FieldExternalConnectionOrganizationID))
	r.Equal("2024-01-01T00:00:00Z", data.Get(FieldExternalConnectionCreateTime))
	r.Equal("2024-01-01T00:00:00Z", data.Get(FieldExternalConnectionUpdateTime))
}

func TestExternalConnection_Read(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	mockClient := mock_external_connections.NewMockClientInterface(gomock.NewController(t))
	provider := extConnMockProvider(mockClient)

	mockClient.EXPECT().
		ExternalConnectionsAPIGetConnection(gomock.Any(), testExtConnOrgID, testExtConnID).
		Return(extConnJSONResponse(200, extConnGetJSON(t)), nil)

	res := resourceExternalConnection()
	data := res.TestResourceData()
	data.SetId(testExtConnID)
	require.NoError(t, data.Set(FieldExternalConnectionOrganizationID, testExtConnOrgID))

	diags := resourceExternalConnectionRead(context.Background(), data, provider)
	r.False(diags.HasError(), "read should not return errors: %v", diags)

	r.Equal(testExtConnID, data.Id())
	r.Equal("AWS", data.Get(FieldExternalConnectionCloud))
	r.Equal("AWS_ACCOUNT", data.Get(FieldExternalConnectionScope))
	r.Equal(testExtConnScopeKey, data.Get(FieldExternalConnectionScopeKey))
	r.Equal(testExtConnSuffix, data.Get(FieldExternalConnectionResourceSuffix))
	r.Equal(testExtConnOrgID, data.Get(FieldExternalConnectionOrganizationID))
	r.Equal("2024-01-01T00:00:00Z", data.Get(FieldExternalConnectionCreateTime))
}

func TestExternalConnection_Update(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	mockClient := mock_external_connections.NewMockClientInterface(gomock.NewController(t))
	provider := extConnMockProvider(mockClient)

	// Update calls Upsert, then Read calls Get
	mockClient.EXPECT().
		ExternalConnectionsAPIUpsertConnection(gomock.Any(), testExtConnOrgID, gomock.Any()).
		Return(extConnJSONResponse(200, extConnUpsertJSON(t)), nil)
	mockClient.EXPECT().
		ExternalConnectionsAPIGetConnection(gomock.Any(), testExtConnOrgID, testExtConnID).
		Return(extConnJSONResponse(200, extConnGetJSON(t)), nil)

	res := resourceExternalConnection()
	data := res.TestResourceData()
	data.SetId(testExtConnID)
	extConnSetBaseFields(t, data)

	diags := resourceExternalConnectionUpdate(context.Background(), data, provider)
	r.False(diags.HasError(), "update should not return errors: %v", diags)

	r.Equal(testExtConnID, data.Id())
	r.Equal("AWS", data.Get(FieldExternalConnectionCloud))
	r.Equal("AWS_ACCOUNT", data.Get(FieldExternalConnectionScope))
	r.Equal("2024-01-01T00:00:00Z", data.Get(FieldExternalConnectionUpdateTime))
}

func TestExternalConnection_Delete(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	mockClient := mock_external_connections.NewMockClientInterface(gomock.NewController(t))
	provider := extConnMockProvider(mockClient)

	// Delete returns 200 with no Content-Type and empty body — the parse function
	// skips JSON parsing when Content-Type is absent.
	mockClient.EXPECT().
		ExternalConnectionsAPIDeleteConnection(gomock.Any(), testExtConnOrgID, testExtConnID).
		Return(&http.Response{
			StatusCode: 200,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader([]byte{})),
		}, nil)

	res := resourceExternalConnection()
	data := res.TestResourceData()
	data.SetId(testExtConnID)
	require.NoError(t, data.Set(FieldExternalConnectionOrganizationID, testExtConnOrgID))

	diags := resourceExternalConnectionDelete(context.Background(), data, provider)
	r.False(diags.HasError(), "delete should not return errors: %v", diags)
}

func TestExternalConnection_CreateWithAWSMetadata(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	mockClient := mock_external_connections.NewMockClientInterface(gomock.NewController(t))
	provider := extConnMockProvider(mockClient)

	var capturedReq external_connections.UpsertConnectionRequest
	mockClient.EXPECT().
		ExternalConnectionsAPIUpsertConnection(gomock.Any(), testExtConnOrgID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, req external_connections.UpsertConnectionRequest, _ ...external_connections.RequestEditorFn) (*http.Response, error) {
			capturedReq = req
			return extConnJSONResponse(200, extConnUpsertJSON(t)), nil
		})
	mockClient.EXPECT().
		ExternalConnectionsAPIGetConnection(gomock.Any(), testExtConnOrgID, testExtConnID).
		Return(extConnJSONResponse(200, extConnGetJSON(t)), nil)

	res := resourceExternalConnection()
	data := res.TestResourceData()
	extConnSetBaseFields(t, data)
	r.NoError(data.Set(FieldExternalConnectionMetadata, []map[string]any{
		{
			"aws": []map[string]any{
				{"role_arn": "arn:aws:iam::123456789012:role/castai-role"},
			},
		},
	}))

	diags := resourceExternalConnectionCreate(context.Background(), data, provider)
	r.False(diags.HasError(), "create should not return errors: %v", diags)

	r.NotNil(capturedReq.Metadata)
	r.NotNil(capturedReq.Metadata.Aws)
	r.NotNil(capturedReq.Metadata.Aws.RoleArn)
	r.Equal("arn:aws:iam::123456789012:role/castai-role", *capturedReq.Metadata.Aws.RoleArn)
}

func TestExternalConnection_CreateWithAzureMetadata(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	mockClient := mock_external_connections.NewMockClientInterface(gomock.NewController(t))
	provider := extConnMockProvider(mockClient)

	var capturedReq external_connections.UpsertConnectionRequest
	mockClient.EXPECT().
		ExternalConnectionsAPIUpsertConnection(gomock.Any(), testExtConnOrgID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, req external_connections.UpsertConnectionRequest, _ ...external_connections.RequestEditorFn) (*http.Response, error) {
			capturedReq = req
			return extConnJSONResponse(200, extConnUpsertJSON(t)), nil
		})
	mockClient.EXPECT().
		ExternalConnectionsAPIGetConnection(gomock.Any(), testExtConnOrgID, testExtConnID).
		Return(extConnJSONResponse(200, extConnGetJSON(t)), nil)

	res := resourceExternalConnection()
	data := res.TestResourceData()
	r.NoError(data.Set(FieldExternalConnectionCloud, "AZURE"))
	r.NoError(data.Set(FieldExternalConnectionScope, "AZURE_SUBSCRIPTION"))
	r.NoError(data.Set(FieldExternalConnectionScopeKey, "sub-123"))
	r.NoError(data.Set(FieldExternalConnectionResourceSuffix, testExtConnSuffix))
	r.NoError(data.Set(FieldExternalConnectionOrganizationID, testExtConnOrgID))
	r.NoError(data.Set(FieldExternalConnectionEnabledFeatures, []map[string]any{
		{"feature": "CLOUD_CONNECT"},
	}))
	r.NoError(data.Set(FieldExternalConnectionMetadata, []map[string]any{
		{
			"azure": []map[string]any{
				{
					"subscription_id": "sub-123",
					"apps": []map[string]any{
						{"feature": "cloud_connect", "client_id": "client-abc", "tenant_id": "tenant-xyz"},
					},
				},
			},
		},
	}))

	diags := resourceExternalConnectionCreate(context.Background(), data, provider)
	r.False(diags.HasError(), "create should not return errors: %v", diags)

	r.NotNil(capturedReq.Metadata)
	r.NotNil(capturedReq.Metadata.Azure)
	r.NotNil(capturedReq.Metadata.Azure.SubscriptionId)
	r.Equal("sub-123", *capturedReq.Metadata.Azure.SubscriptionId)
	r.NotNil(capturedReq.Metadata.Azure.Apps)
	apps := *capturedReq.Metadata.Azure.Apps
	app, ok := apps["cloud_connect"]
	r.True(ok, "expected cloud_connect app")
	r.Equal("client-abc", app.ClientId)
	r.Equal("tenant-xyz", app.TenantId)
}

func TestExternalConnection_CreateWithGCPMetadata(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	mockClient := mock_external_connections.NewMockClientInterface(gomock.NewController(t))
	provider := extConnMockProvider(mockClient)

	var capturedReq external_connections.UpsertConnectionRequest
	mockClient.EXPECT().
		ExternalConnectionsAPIUpsertConnection(gomock.Any(), testExtConnOrgID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, req external_connections.UpsertConnectionRequest, _ ...external_connections.RequestEditorFn) (*http.Response, error) {
			capturedReq = req
			return extConnJSONResponse(200, extConnUpsertJSON(t)), nil
		})
	mockClient.EXPECT().
		ExternalConnectionsAPIGetConnection(gomock.Any(), testExtConnOrgID, testExtConnID).
		Return(extConnJSONResponse(200, extConnGetJSON(t)), nil)

	res := resourceExternalConnection()
	data := res.TestResourceData()
	r.NoError(data.Set(FieldExternalConnectionCloud, "GCP"))
	r.NoError(data.Set(FieldExternalConnectionScope, "GCP_PROJECT"))
	r.NoError(data.Set(FieldExternalConnectionScopeKey, "my-project-123"))
	r.NoError(data.Set(FieldExternalConnectionResourceSuffix, testExtConnSuffix))
	r.NoError(data.Set(FieldExternalConnectionOrganizationID, testExtConnOrgID))
	r.NoError(data.Set(FieldExternalConnectionEnabledFeatures, []map[string]any{
		{"feature": "CLOUD_CONNECT"},
	}))
	r.NoError(data.Set(FieldExternalConnectionMetadata, []map[string]any{
		{
			"gcp": []map[string]any{
				{
					"service_account_emails": map[string]any{
						"cloud_connect": "sa-c39cca43@my-project.iam.gserviceaccount.com",
					},
				},
			},
		},
	}))

	diags := resourceExternalConnectionCreate(context.Background(), data, provider)
	r.False(diags.HasError(), "create should not return errors: %v", diags)

	r.NotNil(capturedReq.Metadata)
	r.NotNil(capturedReq.Metadata.Gcp)
	r.NotNil(capturedReq.Metadata.Gcp.ServiceAccountEmails)
	sa := *capturedReq.Metadata.Gcp.ServiceAccountEmails
	r.Equal("sa-c39cca43@my-project.iam.gserviceaccount.com", sa["cloud_connect"])
}
