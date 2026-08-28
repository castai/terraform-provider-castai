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

func TestDataSourceExternalConnections(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	mockClient := mock_external_connections.NewMockClientInterface(gomock.NewController(t))
	provider := &ProviderConfig{
		externalConnectionsClient: &external_connections.ClientWithResponses{ClientInterface: mockClient},
	}

	testTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	listResp := external_connections.ListConnectionsResponse{
		Items: []external_connections.Connection{
			{
				Id:             "conn-1",
				Cloud:          external_connections.ConnectionCloudAWS,
				Scope:          external_connections.ConnectionScopeAWSACCOUNT,
				ScopeKey:       "123456789012",
				ResourceSuffix: "c39cca43",
				OrganizationId: testExtConnOrgID,
				EnabledFeatures: []external_connections.EnabledFeature{
					{Feature: external_connections.EnabledFeatureFeatureCLOUDCONNECT, RegistryVersion: "v1"},
				},
				CreateTime: testTime,
				UpdateTime: testTime,
			},
			{
				Id:             "conn-2",
				Cloud:          external_connections.ConnectionCloudGCP,
				Scope:          external_connections.ConnectionScopeGCPPROJECT,
				ScopeKey:       "my-project-123",
				ResourceSuffix: "a1b2c3d4",
				OrganizationId: testExtConnOrgID,
				EnabledFeatures: []external_connections.EnabledFeature{
					{Feature: external_connections.EnabledFeatureFeatureCOSTMONITORING, RegistryVersion: "v2"},
				},
				CreateTime: testTime,
				UpdateTime: testTime,
			},
		},
	}
	body, err := json.Marshal(listResp)
	r.NoError(err)

	mockClient.EXPECT().
		ExternalConnectionsAPIListConnections(gomock.Any(), testExtConnOrgID, gomock.Any()).
		Return(&http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(body)),
		}, nil)

	ds := dataSourceExternalConnections()
	data := ds.TestResourceData()
	require.NoError(t, data.Set(FieldDSExternalConnectionsOrganizationID, testExtConnOrgID))

	diags := dataSourceExternalConnectionsRead(context.Background(), data, provider)
	r.False(diags.HasError(), "read should not return errors: %v", diags)

	r.Equal(testExtConnOrgID, data.Id())

	items := data.Get(FieldDSExternalConnectionsItems).([]any)
	r.Len(items, 2)

	// First item
	item0 := items[0].(map[string]any)
	r.Equal("conn-1", item0["id"])
	r.Equal("AWS", item0["cloud"])
	r.Equal("AWS_ACCOUNT", item0["scope"])
	r.Equal("123456789012", item0["scope_key"])
	r.Equal("c39cca43", item0["resource_suffix"])

	// Second item
	item1 := items[1].(map[string]any)
	r.Equal("conn-2", item1["id"])
	r.Equal("GCP", item1["cloud"])
	r.Equal("GCP_PROJECT", item1["scope"])
	r.Equal("my-project-123", item1["scope_key"])
	r.Equal("a1b2c3d4", item1["resource_suffix"])
}
