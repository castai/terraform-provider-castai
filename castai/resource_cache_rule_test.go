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
