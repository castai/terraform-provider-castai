package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestCacheGroupDataSourceRead(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	cacheGroupID := "cache-group-123"
	name := "test-cache-group"
	protocolType := "MySQL"
	directMode := true
	suffix := "primary"
	connectionString := "mysql://db.example.com:3306"

	readResponse := sdk.DboV1CacheGroup{
		Id:           &cacheGroupID,
		ProtocolType: sdk.DboV1CacheGroupProtocolType(protocolType),
		Name:         &name,
		DirectMode:   &directMode,
		Endpoints: &[]sdk.DboV1Endpoint{
			{
				Hostname:         "db.example.com",
				Port:             3306,
				Suffix:           &suffix,
				ConnectionString: &connectionString,
			},
		},
	}

	readBody, _ := json.Marshal(readResponse)
	mockClient.EXPECT().
		DboAPIGetCacheGroup(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(readBody)),
		}, nil)

	state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)

	resource := dataSourceCacheGroup()
	data := resource.Data(state)
	r.NoError(data.Set("id", cacheGroupID))

	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Equal(`ID = cache-group-123
direct_mode = true
endpoints.# = 1
endpoints.0.connection_string = mysql://db.example.com:3306
endpoints.0.hostname = db.example.com
endpoints.0.name = primary
endpoints.0.port = 3306
name = test-cache-group
protocol_type = MySQL
Tainted = false
`, data.State().String())
}

func TestCacheGroupDataSourceReadEmptyName(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	cacheGroupID := "cache-group-123"
	protocolType := "PostgreSQL"

	readResponse := sdk.DboV1CacheGroup{
		Id:           &cacheGroupID,
		ProtocolType: sdk.DboV1CacheGroupProtocolType(protocolType),
		Name:         nil,
	}

	readBody, _ := json.Marshal(readResponse)
	mockClient.EXPECT().
		DboAPIGetCacheGroup(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(readBody)),
		}, nil)

	state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)

	resource := dataSourceCacheGroup()
	data := resource.Data(state)
	r.NoError(data.Set("id", cacheGroupID))

	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Equal("ID = cache-group-123\nname = \nprotocol_type = PostgreSQL\nTainted = false\n", data.State().String())
}

func TestCacheGroupDataSourceReadNotFound(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	cacheGroupID := "non-existent-id"

	mockClient.EXPECT().
		DboAPIGetCacheGroup(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"message": "not found"}`))),
		}, nil)

	state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)

	resource := dataSourceCacheGroup()
	data := resource.Data(state)
	r.NoError(data.Set("id", cacheGroupID))

	result := resource.ReadContext(ctx, data, provider)
	r.True(result.HasError())
}
