package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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

func TestAccCloudAgnostic_DataSourceCacheGroup(t *testing.T) {
	rName := fmt.Sprintf("%v-cache-group-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_cache_group.test"
	dataSourceName := "data.castai_cache_group.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCacheGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceCacheGroupConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "protocol_type", resourceName, "protocol_type"),
					resource.TestCheckResourceAttr(dataSourceName, "name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "protocol_type", "PostgreSQL"),
				),
			},
		},
	})
}

func TestAccCloudAgnostic_DataSourceCacheGroupWithEndpoints(t *testing.T) {
	rName := fmt.Sprintf("%v-cache-group-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_cache_group.test"
	dataSourceName := "data.castai_cache_group.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCacheGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceCacheGroupWithEndpointsConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "name", resourceName, "name"),
					resource.TestCheckResourceAttrPair(dataSourceName, "protocol_type", resourceName, "protocol_type"),
					resource.TestCheckResourceAttrPair(dataSourceName, "direct_mode", resourceName, "direct_mode"),
					resource.TestCheckResourceAttr(dataSourceName, "name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "protocol_type", "PostgreSQL"),
					resource.TestCheckResourceAttr(dataSourceName, "direct_mode", "true"),
					// Verify endpoints
					resource.TestCheckResourceAttr(dataSourceName, "endpoints.#", "2"),
					resource.TestCheckResourceAttrPair(dataSourceName, "endpoints.0.hostname", resourceName, "endpoints.0.hostname"),
					resource.TestCheckResourceAttrPair(dataSourceName, "endpoints.0.port", resourceName, "endpoints.0.port"),
					resource.TestCheckResourceAttrPair(dataSourceName, "endpoints.0.name", resourceName, "endpoints.0.name"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoints.0.hostname", "primary.db.example.com"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoints.0.port", "5432"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoints.0.name", "primary"),
					resource.TestCheckResourceAttrPair(dataSourceName, "endpoints.1.hostname", resourceName, "endpoints.1.hostname"),
					resource.TestCheckResourceAttrPair(dataSourceName, "endpoints.1.port", resourceName, "endpoints.1.port"),
					resource.TestCheckResourceAttrPair(dataSourceName, "endpoints.1.name", resourceName, "endpoints.1.name"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoints.1.hostname", "replica.db.example.com"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoints.1.port", "5433"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoints.1.name", "replica"),
					// Note: connection_string is only available once DBO is deployed and running on the cluster
				),
			},
		},
	})
}

func testAccDataSourceCacheGroupConfig(name string) string {
	return fmt.Sprintf(`
resource "castai_cache_group" "test" {
  name          = %[1]q
  protocol_type = "PostgreSQL"
}

data "castai_cache_group" "test" {
  id = castai_cache_group.test.id
}`, name)
}

func testAccDataSourceCacheGroupWithEndpointsConfig(name string) string {
	return fmt.Sprintf(`
resource "castai_cache_group" "test" {
  name          = %[1]q
  protocol_type = "PostgreSQL"
  direct_mode   = true

  endpoints {
    hostname = "primary.db.example.com"
    port     = 5432
    name     = "primary"
  }

  endpoints {
    hostname = "replica.db.example.com"
    port     = 5433
    name     = "replica"
  }
}

data "castai_cache_group" "test" {
  id = castai_cache_group.test.id
}`, name)
}
