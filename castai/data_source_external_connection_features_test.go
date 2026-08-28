package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/external_connections"
	mock_external_connections "github.com/castai/terraform-provider-castai/castai/sdk/external_connections/mock"
)

func TestDataSourceExternalConnectionFeatures(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	mockClient := mock_external_connections.NewMockClientInterface(gomock.NewController(t))
	provider := &ProviderConfig{
		externalConnectionsClient: &external_connections.ClientWithResponses{ClientInterface: mockClient},
	}

	version := "v3"
	listResp := external_connections.ListFeaturesResponse{
		Items: []external_connections.Feature{
			{
				Id:                  "cloud_connect",
				Name:                "Cloud Connect",
				Type:                external_connections.FeatureTypeCLOUDCONNECT,
				Category:            external_connections.FeatureCategoryORGANIZATION,
				Description:         "Connect cloud accounts to CAST AI",
				Owner:               "platform",
				CurrentVersion:      &version,
				BasePermissionCount: 5,
				SubFeatures: []external_connections.SubFeature{
					{
						Id:          "cloud_connect_cur_bucket",
						Name:        "CUR Bucket",
						Type:        external_connections.SubFeatureTypeCLOUDCONNECTCURBUCKET,
						Description: "Access CUR bucket for cost data",
						Required:    false,
					},
				},
			},
		},
	}
	body, err := json.Marshal(listResp)
	r.NoError(err)

	mockClient.EXPECT().
		ExternalConnectionsAPIListFeatures(gomock.Any(), gomock.Any()).
		Return(&http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(body)),
		}, nil)

	ds := dataSourceExternalConnectionFeatures()
	data := ds.TestResourceData()

	diags := dataSourceExternalConnectionFeaturesRead(context.Background(), data, provider)
	r.False(diags.HasError(), "read should not return errors: %v", diags)

	r.Equal("external_connection_features", data.Id())

	items := data.Get(FieldDSExtConnFeaturesItems).([]any)
	r.Len(items, 1)

	item := items[0].(map[string]any)
	r.Equal("cloud_connect", item[FieldDSExtConnFeatureID])
	r.Equal("Cloud Connect", item[FieldDSExtConnFeatureName])
	r.Equal("CLOUD_CONNECT", item[FieldDSExtConnFeatureType])
	r.Equal("ORGANIZATION", item[FieldDSExtConnFeatureCategory])
	r.Equal("Connect cloud accounts to CAST AI", item[FieldDSExtConnFeatureDescription])
	r.Equal("platform", item[FieldDSExtConnFeatureOwner])
	r.Equal("v3", item[FieldDSExtConnFeatureCurrentVersion])
	r.Equal(5, item[FieldDSExtConnFeatureBasePermCount])

	subFeatures := item[FieldDSExtConnFeatureSubFeatures].([]any)
	r.Len(subFeatures, 1)
	sf := subFeatures[0].(map[string]any)
	r.Equal("cloud_connect_cur_bucket", sf["id"])
	r.Equal("CUR Bucket", sf["name"])
	r.Equal("CLOUD_CONNECT_CUR_BUCKET", sf["type"])
	r.Equal("Access CUR bucket for cost data", sf["description"])
	r.False(sf["required"].(bool))
}
