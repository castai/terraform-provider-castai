package castai

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestOrganizationDataSourceRead(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	body := io.NopCloser(bytes.NewReader([]byte(`{
  "organizations": [
    {
      "id": "id-1",
      "name": "Test 1",
      "createdAt": "2023-04-18T16:03:18.800099Z",
      "role": "owner"
    },
    {
      "id": "id-2",
      "name": "Test 2",
      "createdAt": "2023-09-04T05:45:51.000552Z",
      "role": "owner"
    }
  ]
}`)))

	mockClient.EXPECT().
		ListOrganizations(gomock.Any()).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)

	resource := dataSourceOrganization()
	data := resource.Data(state)
	r.NoError(data.Set("name", "Test 1"))

	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Equal(`ID = id-1
name = Test 1
Tainted = false
`, data.State().String())
}

func TestOrganizationDataSourceReadError(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	body := io.NopCloser(bytes.NewReader([]byte(`{
  "organizations": [
    {
      "id": "id-1",
      "name": "Test 1",
      "createdAt": "2023-04-18T16:03:18.800099Z",
      "role": "owner"
    },
    {
      "id": "id-2",
      "name": "Test 2",
      "createdAt": "2023-09-04T05:45:51.000552Z",
      "role": "owner"
    }
  ]
}`)))

	mockClient.EXPECT().
		ListOrganizations(gomock.Any()).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)

	resource := dataSourceOrganization()
	data := resource.Data(state)
	r.NoError(data.Set("name", "non-existent"))

	result := resource.ReadContext(ctx, data, provider)
	r.True(result.HasError())
	r.Equal("organization non-existent not found", result[0].Summary)
	r.Equal(diag.Error, result[0].Severity)
}
