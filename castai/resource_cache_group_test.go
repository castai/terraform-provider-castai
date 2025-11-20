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

func TestCacheGroupResource_Create(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheGroup()

	cacheGroupID := "cache-group-123"
	protocolType := "MySQL"
	name := "test-cache-group"

	val := cty.ObjectVal(map[string]cty.Value{
		FieldCacheGroupProtocolType: cty.StringVal(protocolType),
		FieldCacheGroupName:         cty.StringVal(name),
		FieldCacheGroupDirectMode:   cty.BoolVal(true),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	// Mock create cache group response
	createResponse := sdk.DboV1CacheGroup{
		Id:           &cacheGroupID,
		ProtocolType: sdk.DboV1CacheGroupProtocolType(protocolType),
		Name:         &name,
	}

	createBody, _ := json.Marshal(createResponse)
	mockClient.EXPECT().
		DboAPICreateCacheGroup(ctx, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(createBody)),
		}, nil)

	// Mock read cache group response
	readResponse := sdk.DboV1CacheGroup{
		Id:           &cacheGroupID,
		ProtocolType: sdk.DboV1CacheGroupProtocolType(protocolType),
		Name:         &name,
	}

	readBody, _ := json.Marshal(readResponse)
	mockClient.EXPECT().
		DboAPIGetCacheGroup(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(readBody)),
		}, nil)

	// Create
	diags := resourceCacheGroupCreate(ctx, data, provider)
	r.Nil(diags)
	r.NotEmpty(data.Id())
	r.Equal(cacheGroupID, data.Id())
}

func TestCacheGroupResource_Read(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheGroup()

	cacheGroupID := "cache-group-123"
	protocolType := "PostgreSQL"
	name := "test-cache-group"

	// Set up state with ID
	val := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal(cacheGroupID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = cacheGroupID
	data := resource.Data(state)

	// Mock read cache group response
	readResponse := sdk.DboV1CacheGroup{
		Id:           &cacheGroupID,
		ProtocolType: sdk.DboV1CacheGroupProtocolType(protocolType),
		Name:         &name,
		Endpoints: &[]sdk.DboV1Endpoint{
			{
				Hostname: "db.example.com",
				Port:     3306,
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

	// Read
	diags := resourceCacheGroupRead(ctx, data, provider)
	r.Nil(diags)
	r.Equal(protocolType, data.Get(FieldCacheGroupProtocolType))
	r.Equal(name, data.Get(FieldCacheGroupName))
}

func TestCacheGroupResource_Update(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheGroup()

	cacheGroupID := "cache-group-123"
	protocolType := "MySQL"
	updatedName := "updated-cache-group"

	// Set up state with ID
	val := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal(cacheGroupID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = cacheGroupID
	data := resource.Data(state)

	// Mock read response (update calls read at the end)
	readResponse := sdk.DboV1CacheGroup{
		Id:           &cacheGroupID,
		ProtocolType: sdk.DboV1CacheGroupProtocolType(protocolType),
		Name:         &updatedName,
	}
	readBody, _ := json.Marshal(readResponse)
	mockClient.EXPECT().
		DboAPIGetCacheGroup(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(readBody)),
		}, nil)

	// Update (no changes detected, will just call read)
	diags := resourceCacheGroupUpdate(ctx, data, provider)
	r.Nil(diags)
	r.Equal(updatedName, data.Get(FieldCacheGroupName))
}

func TestCacheGroupResource_Delete(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheGroup()

	cacheGroupID := "cache-group-123"

	val := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal(cacheGroupID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = cacheGroupID
	data := resource.Data(state)

	// Mock delete response
	deleteResponse := sdk.DboV1DeleteCacheGroupResponse{}
	deleteBody, _ := json.Marshal(deleteResponse)
	mockClient.EXPECT().
		DboAPIDeleteCacheGroup(ctx, cacheGroupID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(deleteBody)),
		}, nil)

	// Delete
	diags := resourceCacheGroupDelete(ctx, data, provider)
	r.Nil(diags)
}

func TestCacheGroupResource_NotFound(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceCacheGroup()

	cacheGroupID := "cache-group-123"

	val := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal(cacheGroupID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = cacheGroupID
	data := resource.Data(state)

	// Mock 404 response
	mockClient.EXPECT().
		DboAPIGetCacheGroup(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
		}, nil)

	// Read should remove from state
	diags := resourceCacheGroupRead(ctx, data, provider)
	r.Nil(diags)
	r.Empty(data.Id())
}

func TestCacheGroupResource_EndpointsValidation(t *testing.T) {
	r := require.New(t)

	resource := resourceCacheGroup()

	// Verify MinItems validation is set
	endpointsSchema := resource.Schema[FieldCacheGroupEndpoints]
	r.Equal(1, endpointsSchema.MinItems)
}

func TestExpandEndpoints(t *testing.T) {
	suffix1 := "primary"
	suffix2 := "replica"

	tests := map[string]struct {
		input    []interface{}
		expected *[]sdk.DboV1Endpoint
	}{
		"nil input": {
			input:    nil,
			expected: nil,
		},
		"empty input": {
			input:    []interface{}{},
			expected: nil,
		},
		"single endpoint with name": {
			input: []interface{}{
				map[string]any{
					FieldCacheGroupEndpointHostname: "db.example.com",
					FieldCacheGroupEndpointPort:     3306,
					FieldCacheGroupEndpointName:     "primary",
				},
			},
			expected: &[]sdk.DboV1Endpoint{
				{
					Hostname: "db.example.com",
					Port:     3306,
					Suffix:   &suffix1,
				},
			},
		},
		"multiple endpoints": {
			input: []interface{}{
				map[string]any{
					FieldCacheGroupEndpointHostname: "db1.example.com",
					FieldCacheGroupEndpointPort:     5432,
					FieldCacheGroupEndpointName:     "primary",
				},
				map[string]any{
					FieldCacheGroupEndpointHostname: "db2.example.com",
					FieldCacheGroupEndpointPort:     5433,
					FieldCacheGroupEndpointName:     "replica",
				},
			},
			expected: &[]sdk.DboV1Endpoint{
				{
					Hostname: "db1.example.com",
					Port:     5432,
					Suffix:   &suffix1,
				},
				{
					Hostname: "db2.example.com",
					Port:     5433,
					Suffix:   &suffix2,
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			result := expandEndpoints(tc.input)

			if tc.expected == nil {
				r.Nil(result)
			} else {
				r.NotNil(result)
				r.Equal(*tc.expected, *result)
			}
		})
	}
}

func TestFlattenEndpoints(t *testing.T) {
	suffix1 := "primary"
	suffix2 := "replica"

	tests := map[string]struct {
		input    []sdk.DboV1Endpoint
		expected []interface{}
	}{
		"nil input": {
			input:    nil,
			expected: nil,
		},
		"empty input": {
			input:    []sdk.DboV1Endpoint{},
			expected: nil,
		},
		"single endpoint with suffix": {
			input: []sdk.DboV1Endpoint{
				{
					Hostname: "db.example.com",
					Port:     3306,
					Suffix:   &suffix1,
				},
			},
			expected: []interface{}{
				map[string]any{
					FieldCacheGroupEndpointHostname: "db.example.com",
					FieldCacheGroupEndpointPort:     int32(3306),
					FieldCacheGroupEndpointName:     "primary",
				},
			},
		},
		"single endpoint without suffix": {
			input: []sdk.DboV1Endpoint{
				{
					Hostname: "db.example.com",
					Port:     3306,
					Suffix:   nil,
				},
			},
			expected: []interface{}{
				map[string]any{
					FieldCacheGroupEndpointHostname: "db.example.com",
					FieldCacheGroupEndpointPort:     int32(3306),
				},
			},
		},
		"multiple endpoints with mixed suffixes": {
			input: []sdk.DboV1Endpoint{
				{
					Hostname: "db1.example.com",
					Port:     5432,
					Suffix:   &suffix1,
				},
				{
					Hostname: "db2.example.com",
					Port:     5433,
					Suffix:   &suffix2,
				},
			},
			expected: []interface{}{
				map[string]any{
					FieldCacheGroupEndpointHostname: "db1.example.com",
					FieldCacheGroupEndpointPort:     int32(5432),
					FieldCacheGroupEndpointName:     "primary",
				},
				map[string]any{
					FieldCacheGroupEndpointHostname: "db2.example.com",
					FieldCacheGroupEndpointPort:     int32(5433),
					FieldCacheGroupEndpointName:     "replica",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			result := flattenEndpoints(tc.input)

			if tc.expected == nil {
				r.Nil(result)
			} else {
				r.NotNil(result)
				r.Equal(tc.expected, result)
			}
		})
	}
}
