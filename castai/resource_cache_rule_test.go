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

func TestCacheRuleResource_Create(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheRule()

	cacheGroupID := "cache-group-123"
	cacheConfigID := "config-123"
	ruleID := "rule-123"
	mode := "Auto"

	val := cty.ObjectVal(map[string]cty.Value{
		FieldCacheRuleCacheGroupID:         cty.StringVal(cacheGroupID),
		FieldCacheRuleCacheConfigurationID: cty.StringVal(cacheConfigID),
		FieldCacheRuleMode:                 cty.StringVal(mode),
		FieldCacheRuleTable:                cty.StringVal("users"),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	createResponse := sdk.DboV1TTLConfiguration{
		Id:   &ruleID,
		Mode: sdk.DboV1TTLMode(mode),
	}
	createBody, _ := json.Marshal(createResponse)
	mockClient.EXPECT().
		DboAPICreateCacheTTL(ctx, cacheGroupID, cacheConfigID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(createBody)),
		}, nil)

	table := "users"
	listResponse := sdk.DboV1ListCacheTTLsResponse{
		Items: &[]sdk.DboV1TTLConfiguration{
			{
				Id:    &ruleID,
				Mode:  sdk.DboV1TTLMode(mode),
				Table: &table,
			},
		},
	}
	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		DboAPIListCacheTTLs(ctx, cacheGroupID, cacheConfigID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	diags := resourceCacheRuleCreate(ctx, data, provider)
	r.Nil(diags)
	r.NotEmpty(data.Id())
	r.Equal(ruleID, data.Id())
}

func TestCacheRuleResource_CreateWithManualTTL(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheRule()

	cacheGroupID := "cache-group-123"
	cacheConfigID := "config-123"
	ruleID := "rule-123"
	mode := "Manual"
	manualTTL := int64(3600)

	val := cty.ObjectVal(map[string]cty.Value{
		FieldCacheRuleCacheGroupID:         cty.StringVal(cacheGroupID),
		FieldCacheRuleCacheConfigurationID: cty.StringVal(cacheConfigID),
		FieldCacheRuleMode:                 cty.StringVal(mode),
		FieldCacheRuleManualTTL:            cty.NumberIntVal(int64(manualTTL)),
		FieldCacheRuleTable:                cty.StringVal("users"),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	createResponse := sdk.DboV1TTLConfiguration{
		Id:        &ruleID,
		Mode:      sdk.DboV1TTLMode(mode),
		ManualTtl: &manualTTL,
	}
	createBody, _ := json.Marshal(createResponse)
	mockClient.EXPECT().
		DboAPICreateCacheTTL(ctx, cacheGroupID, cacheConfigID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(createBody)),
		}, nil)

	listResponse := sdk.DboV1ListCacheTTLsResponse{
		Items: &[]sdk.DboV1TTLConfiguration{
			{
				Id:        &ruleID,
				Mode:      sdk.DboV1TTLMode(mode),
				ManualTtl: &manualTTL,
			},
		},
	}
	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		DboAPIListCacheTTLs(ctx, cacheGroupID, cacheConfigID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	diags := resourceCacheRuleCreate(ctx, data, provider)
	r.Nil(diags)
	r.Equal(int(manualTTL), data.Get(FieldCacheRuleManualTTL))
}

func TestCacheRuleResource_Read(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheRule()

	cacheGroupID := "cache-group-123"
	cacheConfigID := "config-123"
	ruleID := "rule-123"

	val := cty.ObjectVal(map[string]cty.Value{
		"id":                               cty.StringVal(ruleID),
		FieldCacheRuleCacheGroupID:         cty.StringVal(cacheGroupID),
		FieldCacheRuleCacheConfigurationID: cty.StringVal(cacheConfigID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = ruleID
	data := resource.Data(state)

	mode := "DontCache"
	table := "sessions"
	listResponse := sdk.DboV1ListCacheTTLsResponse{
		Items: &[]sdk.DboV1TTLConfiguration{
			{
				Id:    &ruleID,
				Mode:  sdk.DboV1TTLMode(mode),
				Table: &table,
			},
		},
	}
	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		DboAPIListCacheTTLs(ctx, cacheGroupID, cacheConfigID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	diags := resourceCacheRuleRead(ctx, data, provider)
	r.Nil(diags)
	r.Equal(mode, data.Get(FieldCacheRuleMode))
	r.Equal(table, data.Get(FieldCacheRuleTable))
}

func TestCacheRuleResource_Update(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheRule()

	cacheGroupID := "cache-group-123"
	cacheConfigID := "config-123"
	ruleID := "rule-123"
	updatedMode := "Manual"
	manualTTL := int64(7200)

	val := cty.ObjectVal(map[string]cty.Value{
		"id":                               cty.StringVal(ruleID),
		FieldCacheRuleCacheGroupID:         cty.StringVal(cacheGroupID),
		FieldCacheRuleCacheConfigurationID: cty.StringVal(cacheConfigID),
		FieldCacheRuleMode:                 cty.StringVal(updatedMode),
		FieldCacheRuleManualTTL:            cty.NumberIntVal(manualTTL),
		FieldCacheRuleTable:                cty.StringVal("users"),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = ruleID
	data := resource.Data(state)

	updateResponse := sdk.DboV1TTLConfiguration{
		Id:        &ruleID,
		Mode:      sdk.DboV1TTLMode(updatedMode),
		ManualTtl: &manualTTL,
	}
	updateBody, _ := json.Marshal(updateResponse)
	mockClient.EXPECT().
		DboAPIUpdateCacheTTL(ctx, cacheGroupID, cacheConfigID, ruleID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(updateBody)),
		}, nil)

	listResponse := sdk.DboV1ListCacheTTLsResponse{
		Items: &[]sdk.DboV1TTLConfiguration{
			{
				Id:        &ruleID,
				Mode:      sdk.DboV1TTLMode(updatedMode),
				ManualTtl: &manualTTL,
			},
		},
	}
	listBody, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		DboAPIListCacheTTLs(ctx, cacheGroupID, cacheConfigID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(listBody)),
		}, nil)

	diags := resourceCacheRuleUpdate(ctx, data, provider)
	r.Nil(diags)
}

func TestCacheRuleResource_Delete(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheRule()

	cacheGroupID := "cache-group-123"
	cacheConfigID := "config-123"
	ruleID := "rule-123"

	val := cty.ObjectVal(map[string]cty.Value{
		"id":                               cty.StringVal(ruleID),
		FieldCacheRuleCacheGroupID:         cty.StringVal(cacheGroupID),
		FieldCacheRuleCacheConfigurationID: cty.StringVal(cacheConfigID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = ruleID
	data := resource.Data(state)

	deleteResponse := map[string]interface{}{}
	deleteBody, _ := json.Marshal(deleteResponse)
	mockClient.EXPECT().
		DboAPIDeleteCacheTTL(ctx, cacheGroupID, cacheConfigID, ruleID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(deleteBody)),
		}, nil)

	diags := resourceCacheRuleDelete(ctx, data, provider)
	r.Nil(diags)
}

func TestCacheRuleResource_NotFound(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheRule()

	cacheGroupID := "cache-group-123"
	cacheConfigID := "config-123"
	ruleID := "rule-123"

	val := cty.ObjectVal(map[string]cty.Value{
		"id":                               cty.StringVal(ruleID),
		FieldCacheRuleCacheGroupID:         cty.StringVal(cacheGroupID),
		FieldCacheRuleCacheConfigurationID: cty.StringVal(cacheConfigID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = ruleID
	data := resource.Data(state)

	mockClient.EXPECT().
		DboAPIListCacheTTLs(ctx, cacheGroupID, cacheConfigID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
		}, nil)

	diags := resourceCacheRuleRead(ctx, data, provider)
	r.Nil(diags)
	r.Empty(data.Id())
}

func TestAccCloudAgnostic_ResourceCacheRule(t *testing.T) {
	rName := fmt.Sprintf("%v-cache-rule-%v", ResourcePrefix, acctest.RandString(8))
	dbName := fmt.Sprintf("testdb_%v", acctest.RandString(8))
	tableName := fmt.Sprintf("test_table_%v", acctest.RandString(8))
	resourceName := "castai_cache_rule.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCacheRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCreateCacheRuleConfig(rName, dbName, tableName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "mode", "Auto"),
					resource.TestCheckResourceAttr(resourceName, "table", tableName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "cache_group_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cache_configuration_id"),
				),
			},
		},
	})
}

func testAccCreateCacheRuleConfig(groupName, dbName, tableName string) string {
	return fmt.Sprintf(`
resource "castai_cache_group" "test" {
  name          = %[1]q
  protocol_type = "PostgreSQL"
}

resource "castai_cache_configuration" "test" {
  cache_group_id = castai_cache_group.test.id
  database_name  = %[2]q
  mode           = "Auto"
}

resource "castai_cache_rule" "test" {
  cache_group_id         = castai_cache_group.test.id
  cache_configuration_id = castai_cache_configuration.test.id
  mode                   = "Auto"
  table                  = %[3]q
}`, groupName, dbName, tableName)
}

func testAccCacheRuleDestroy(s *tfterraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).api
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_cache_rule" {
			continue
		}

		cacheGroupID := rs.Primary.Attributes["cache_group_id"]
		cacheConfigID := rs.Primary.Attributes["cache_configuration_id"]
		ruleID := rs.Primary.ID

		response, err := client.DboAPIListCacheTTLsWithResponse(ctx, cacheGroupID, cacheConfigID, &sdk.DboAPIListCacheTTLsParams{})
		if err != nil {
			return err
		}

		if response.StatusCode() == http.StatusNotFound {
			return nil
		}

		if response.JSON200 != nil && response.JSON200.Items != nil {
			for _, rule := range *response.JSON200.Items {
				if rule.Id != nil && *rule.Id == ruleID {
					return fmt.Errorf("cache rule %s still exists", ruleID)
				}
			}
		}
	}

	return nil
}
