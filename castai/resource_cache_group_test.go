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

	val := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal(cacheGroupID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = cacheGroupID
	data := resource.Data(state)

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

	val := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal(cacheGroupID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = cacheGroupID
	data := resource.Data(state)

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

	deleteResponse := sdk.DboV1DeleteCacheGroupResponse{}
	deleteBody, _ := json.Marshal(deleteResponse)
	mockClient.EXPECT().
		DboAPIDeleteCacheGroup(ctx, cacheGroupID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(deleteBody)),
		}, nil)

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

	mockClient.EXPECT().
		DboAPIGetCacheGroup(ctx, cacheGroupID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
		}, nil)

	diags := resourceCacheGroupRead(ctx, data, provider)
	r.Nil(diags)
	r.Empty(data.Id())
}

func TestCacheGroupResource_EndpointsValidation(t *testing.T) {
	r := require.New(t)

	resource := resourceCacheGroup()

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

func TestAccCloudAgnostic_ResourceCacheGroup(t *testing.T) {
	rName := fmt.Sprintf("%v-cache-group-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_cache_group.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCacheGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCreateCacheGroupConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "protocol_type", "PostgreSQL"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func testAccCreateCacheGroupConfig(name string) string {
	return fmt.Sprintf(`
resource "castai_cache_group" "test" {
  name          = %[1]q
  protocol_type = "PostgreSQL"
}`, name)
}

func testAccCacheGroupDestroy(s *tfterraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).api
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_cache_group" {
			continue
		}

		response, err := client.DboAPIGetCacheGroupWithResponse(ctx, rs.Primary.ID, &sdk.DboAPIGetCacheGroupParams{})
		if err != nil {
			return err
		}

		if response.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("cache group %s still exists", rs.Primary.ID)
	}

	return nil
}
