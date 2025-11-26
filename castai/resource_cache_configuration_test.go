package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	tfterraform "github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestCacheConfigurationResource_Create(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheConfiguration()

	cacheGroupID := "cache-group-123"
	configID := "config-123"
	databaseName := "test-db"
	mode := sdk.DboV1TTLMode("Auto")

	val := cty.ObjectVal(map[string]cty.Value{
		FieldCacheConfigurationCacheGroupID: cty.StringVal(cacheGroupID),
		FieldCacheConfigurationDatabaseName: cty.StringVal(databaseName),
		FieldCacheConfigurationMode:         cty.StringVal(string(mode)),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	emptyListResponse := sdk.DboV1ListCacheConfigurationsResponse{
		Items: &[]sdk.DboV1CacheConfiguration{},
	}
	emptyListBody, _ := json.Marshal(emptyListResponse)
	mockClient.EXPECT().
		DboAPIListCacheConfigurations(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(emptyListBody)),
		}, nil)

	createResponse := sdk.DboV1CacheConfiguration{
		Id:           &configID,
		DatabaseName: databaseName,
		Mode:         &mode,
	}

	createBody, _ := json.Marshal(createResponse)
	mockClient.EXPECT().
		DboAPICreateCacheConfiguration(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(createBody)),
		}, nil)

	listResponse := sdk.DboV1ListCacheConfigurationsResponse{
		Items: &[]sdk.DboV1CacheConfiguration{createResponse},
	}
	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		DboAPIListCacheConfigurations(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	diags := resourceCacheConfigurationCreate(ctx, data, provider)
	r.Nil(diags)
	r.NotEmpty(data.Id())
	r.Equal(configID, data.Id())
}

func TestCacheConfigurationResource_CreateIdempotent(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheConfiguration()

	cacheGroupID := "cache-group-123"
	existingConfigID := "existing-config-456"
	databaseName := "test-db"
	mode := sdk.DboV1TTLMode("Auto")

	val := cty.ObjectVal(map[string]cty.Value{
		FieldCacheConfigurationCacheGroupID: cty.StringVal(cacheGroupID),
		FieldCacheConfigurationDatabaseName: cty.StringVal(databaseName),
		FieldCacheConfigurationMode:         cty.StringVal(string(mode)),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	existingConfig := sdk.DboV1CacheConfiguration{
		Id:           &existingConfigID,
		DatabaseName: databaseName,
		Mode:         &mode,
	}
	listResponse := sdk.DboV1ListCacheConfigurationsResponse{
		Items: &[]sdk.DboV1CacheConfiguration{existingConfig},
	}
	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		DboAPIListCacheConfigurations(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	updateResponse := existingConfig
	updateBody, _ := json.Marshal(updateResponse)
	mockClient.EXPECT().
		DboAPIUpdateCacheConfiguration(ctx, cacheGroupID, existingConfigID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(updateBody)),
		}, nil)

	listReadResponse := sdk.DboV1ListCacheConfigurationsResponse{
		Items: &[]sdk.DboV1CacheConfiguration{updateResponse},
	}
	listReadBody, _ := json.Marshal(listReadResponse)
	mockClient.EXPECT().
		DboAPIListCacheConfigurations(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listReadBody)),
		}, nil)

	diags := resourceCacheConfigurationCreate(ctx, data, provider)
	r.Nil(diags)
	r.NotEmpty(data.Id())
	r.Equal(existingConfigID, data.Id())
}

func TestCacheConfigurationResource_Read(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheConfiguration()

	cacheGroupID := "cache-group-123"
	configID := "config-123"
	databaseName := "test-db"
	mode := sdk.DboV1TTLMode("Manual")

	val := cty.ObjectVal(map[string]cty.Value{
		"id":                                cty.StringVal(configID),
		FieldCacheConfigurationCacheGroupID: cty.StringVal(cacheGroupID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = configID
	data := resource.Data(state)

	readResponse := sdk.DboV1CacheConfiguration{
		Id:           &configID,
		DatabaseName: databaseName,
		Mode:         &mode,
	}
	listResponse := sdk.DboV1ListCacheConfigurationsResponse{
		Items: &[]sdk.DboV1CacheConfiguration{readResponse},
	}

	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		DboAPIListCacheConfigurations(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	diags := resourceCacheConfigurationRead(ctx, data, provider)
	r.Nil(diags)
	r.Equal(cacheGroupID, data.Get(FieldCacheConfigurationCacheGroupID))
	r.Equal(databaseName, data.Get(FieldCacheConfigurationDatabaseName))
	r.Equal(string(mode), data.Get(FieldCacheConfigurationMode))
}

func TestCacheConfigurationResource_Update(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheConfiguration()

	cacheGroupID := "cache-group-123"
	configID := "config-123"
	databaseName := "test-db"
	mode := sdk.DboV1TTLMode("DontCache")

	val := cty.ObjectVal(map[string]cty.Value{
		"id":                                cty.StringVal(configID),
		FieldCacheConfigurationCacheGroupID: cty.StringVal(cacheGroupID),
		FieldCacheConfigurationDatabaseName: cty.StringVal(databaseName),
		FieldCacheConfigurationMode:         cty.StringVal(string(mode)),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = configID
	data := resource.Data(state)

	data.MarkNewResource()

	updateResponse := sdk.DboV1CacheConfiguration{
		Id:           &configID,
		DatabaseName: databaseName,
		Mode:         &mode,
	}
	updateBody, _ := json.Marshal(updateResponse)
	mockClient.EXPECT().
		DboAPIUpdateCacheConfiguration(ctx, cacheGroupID, configID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(updateBody)),
		}, nil)

	readResponse := updateResponse
	listReadResponse := sdk.DboV1ListCacheConfigurationsResponse{
		Items: &[]sdk.DboV1CacheConfiguration{readResponse},
	}
	listReadBody, _ := json.Marshal(listReadResponse)
	mockClient.EXPECT().
		DboAPIListCacheConfigurations(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listReadBody)),
		}, nil)

	diags := resourceCacheConfigurationUpdate(ctx, data, provider)
	r.Nil(diags)
}

func TestCacheConfigurationResource_Delete(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheConfiguration()

	cacheGroupID := "cache-group-123"
	configID := "config-123"

	val := cty.ObjectVal(map[string]cty.Value{
		"id":                                cty.StringVal(configID),
		FieldCacheConfigurationCacheGroupID: cty.StringVal(cacheGroupID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = configID
	data := resource.Data(state)

	deleteResponse := sdk.DboV1DeleteCacheConfigurationResponse{}
	deleteBody, _ := json.Marshal(deleteResponse)
	mockClient.EXPECT().
		DboAPIDeleteCacheConfiguration(ctx, cacheGroupID, configID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(deleteBody)),
		}, nil)

	diags := resourceCacheConfigurationDelete(ctx, data, provider)
	r.Nil(diags)
}

func TestCacheConfigurationResource_NotFound(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheConfiguration()

	cacheGroupID := "cache-group-123"
	configID := "config-123"

	val := cty.ObjectVal(map[string]cty.Value{
		"id":                                cty.StringVal(configID),
		FieldCacheConfigurationCacheGroupID: cty.StringVal(cacheGroupID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = configID
	data := resource.Data(state)

	listResponse := sdk.DboV1ListCacheConfigurationsResponse{
		Items: &[]sdk.DboV1CacheConfiguration{},
	}
	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		DboAPIListCacheConfigurations(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	diags := resourceCacheConfigurationRead(ctx, data, provider)
	r.Nil(diags)
	r.Empty(data.Id())
}

func TestCacheConfigurationResource_ModeValidation(t *testing.T) {
	resource := resourceCacheConfiguration()
	modeSchema := resource.Schema[FieldCacheConfigurationMode]
	require.NotNil(t, modeSchema.ValidateDiagFunc)

	tests := map[string]struct {
		value       string
		expectError bool
	}{
		"valid: Auto": {
			value:       "Auto",
			expectError: false,
		},
		"valid: DontCache": {
			value:       "DontCache",
			expectError: false,
		},
		"valid: Manual": {
			value:       "Manual",
			expectError: false,
		},
		"invalid: lowercase auto": {
			value:       "auto",
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			diags := modeSchema.ValidateDiagFunc(tt.value, cty.Path{})
			if tt.expectError {
				require.NotEmpty(t, diags, "Expected validation error for value: %s", tt.value)
			} else {
				require.Empty(t, diags, "Expected no validation error for value: %s", tt.value)
			}
		})
	}
}

func TestAccCloudAgnostic_ResourceCacheConfiguration(t *testing.T) {
	rName := fmt.Sprintf("%v-cache-config-%v", ResourcePrefix, acctest.RandString(8))
	dbName := fmt.Sprintf("testdb_%v", acctest.RandString(8))
	resourceName := "castai_cache_configuration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCacheConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCreateCacheConfigurationConfig(rName, dbName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "database_name", dbName),
					resource.TestCheckResourceAttr(resourceName, "mode", "Auto"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "cache_group_id"),
				),
			},
		},
	})
}

func testAccCreateCacheConfigurationConfig(groupName, dbName string) string {
	return fmt.Sprintf(`
resource "castai_cache_group" "test" {
  name          = %[1]q
  protocol_type = "PostgreSQL"
}

resource "castai_cache_configuration" "test" {
  cache_group_id = castai_cache_group.test.id
  database_name  = %[2]q
  mode           = "Auto"
}`, groupName, dbName)
}

func testAccCacheConfigurationDestroy(s *tfterraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).api
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_cache_configuration" {
			continue
		}

		cacheGroupID := rs.Primary.Attributes["cache_group_id"]
		configID := rs.Primary.ID

		response, err := client.DboAPIListCacheConfigurationsWithResponse(ctx, cacheGroupID, &sdk.DboAPIListCacheConfigurationsParams{})
		if err != nil {
			return err
		}

		if response.StatusCode() == http.StatusNotFound {
			return nil
		}

		if response.JSON200 != nil && response.JSON200.Items != nil {
			for _, cfg := range *response.JSON200.Items {
				if cfg.Id != nil && *cfg.Id == configID {
					return fmt.Errorf("cache configuration %s still exists", configID)
				}
			}
		}
	}

	return nil
}
