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

	// Mock list response showing no existing configuration
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

	// Mock create cache configuration response
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

	// Mock list cache configurations response
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

	// Create
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

	// Mock list response showing configuration already exists
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

	// Mock update call (since configuration exists)
	updateResponse := existingConfig
	updateBody, _ := json.Marshal(updateResponse)
	mockClient.EXPECT().
		DboAPIUpdateCacheConfiguration(ctx, cacheGroupID, existingConfigID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(updateBody)),
		}, nil)

	// Mock read call after update
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

	// Create should detect existing configuration and update instead
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

	// Set up state with ID
	val := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal(configID),
		FieldCacheConfigurationCacheGroupID: cty.StringVal(cacheGroupID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = configID
	data := resource.Data(state)

	// Mock list cache configurations response
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

	// Read
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
		"id": cty.StringVal(configID),
		FieldCacheConfigurationCacheGroupID: cty.StringVal(cacheGroupID),
		FieldCacheConfigurationDatabaseName: cty.StringVal(databaseName),
		FieldCacheConfigurationMode:         cty.StringVal(string(mode)),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = configID
	data := resource.Data(state)

	// Mark as changed
	data.MarkNewResource()

	// Mock update response
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

	// Mock list response for read
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

	// Update
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
		"id": cty.StringVal(configID),
		FieldCacheConfigurationCacheGroupID: cty.StringVal(cacheGroupID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = configID
	data := resource.Data(state)

	// Mock delete response
	deleteResponse := sdk.DboV1DeleteCacheConfigurationResponse{}
	deleteBody, _ := json.Marshal(deleteResponse)
	mockClient.EXPECT().
		DboAPIDeleteCacheConfiguration(ctx, cacheGroupID, configID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(deleteBody)),
		}, nil)

	// Delete
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
		"id": cty.StringVal(configID),
		FieldCacheConfigurationCacheGroupID: cty.StringVal(cacheGroupID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = configID
	data := resource.Data(state)

	// Mock list response with no matching configuration
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

	// Read should remove from state
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


