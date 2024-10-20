package castai

import (
	"bytes"
	"context"
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

func TestRebalancingScheduleDataSourceRead(t *testing.T) {
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
"id": "id-1",
"name": "Test 1",
"schedule": {},
"trigger_conditions": {
"savings_percentage": 10
},
"launch_configuration": {
    "node_ttl_seconds": 300,
    "num_targeted_nodes": 10,
    "rebalancing_min_nodes": 1,
    "execution_conditions": {
      "achieved_savings_percentage": 10,
      "enabled":"true"
    }
  }
}`)))
	mockClient.EXPECT().
		ScheduledRebalancingAPIGetRebalancingSchedule(gomock.Any(), gomock.Any()).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)

	resource := dataSourceRebalancingSchedule()
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
