package castai

import (
	"bytes"
	"context"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRebalancingJobResourceReadContext(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	id := "abc4192d-a400-48ed-9b22-5fcc9e045258"
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	rebalancingScheduleId := "8155d717-ee9c-43b0-973f-55610beaf8c2"
	body := io.NopCloser(bytes.NewReader([]byte(`
		{
		  "id": "abc4192d-a400-48ed-9b22-5fcc9e045258",
		  "clusterId": "b6bfc074-a267-400f-b8f1-db0850c369b17",
		  "rebalancingScheduleId": "8155d717-ee9c-43b0-973f-55610beaf8c2",
		  "rebalancingPlanId": "87111656-1c69-4316-91a2-7bd5513641b3",
		  "enabled": true,
		  "lastTriggerAt": "2024-04-19T11:30:05.277333Z",
		  "nextTriggerAt": "2024-04-19T11:31:00Z",
		  "status": "JobStatusFailed"
		}
	`)))
	mockClient.EXPECT().
		ScheduledRebalancingAPIGetRebalancingJob(gomock.Any(), clusterId, id).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceRebalancingJob()
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:            cty.StringVal(clusterId),
		"rebalancing_schedule_id": cty.StringVal(rebalancingScheduleId),
		"id":                      cty.StringVal(id),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = id

	data := resource.Data(state)
	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.ElementsMatch(strings.Split(`ID = abc4192d-a400-48ed-9b22-5fcc9e045258
cluster_id = b6bfc074-a267-400f-b8f1-db0850c369b17
enabled = true
rebalancing_schedule_id = 8155d717-ee9c-43b0-973f-55610beaf8c2
Tainted = false
`, "\n"),
		strings.Split(data.State().String(), "\n"))
}

func TestRebalancingJobResourceReadContext_ByScheduledID(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	id := "abc4192d-a400-48ed-9b22-5fcc9e045258"
	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	rebalancingScheduleId := "8155d717-ee9c-43b0-973f-55610beaf8c2"
	body := io.NopCloser(bytes.NewReader([]byte(`
		{
		  "jobs": [
			{
			  "id": "abc4192d-a400-48ed-9b22-5fcc9e045258",
			  "clusterId": "b6bfc074-a267-400f-b8f1-db0850c369b1",
			  "rebalancingScheduleId": "8155d717-ee9c-43b0-973f-55610beaf8c2",
			  "rebalancingPlanId": "67f30f59-4eab-4f9c-b59e-8753ac5d6ed1",
			  "enabled": true,
			  "lastTriggerAt": "2024-04-19T11:24:07.501531Z",
			  "nextTriggerAt": "2024-04-19T11:25:00Z",
			  "status": "JobStatusFailed"
			}
		  ]
		}
	`)))
	mockClient.EXPECT().
		ScheduledRebalancingAPIGetRebalancingJob(gomock.Any(), clusterId, id).
		Return(&http.Response{StatusCode: 404, Body: io.NopCloser(bytes.NewReader([]byte{})), Header: map[string][]string{"Content-Type": {"json"}}}, nil)
	mockClient.EXPECT().
		ScheduledRebalancingAPIListRebalancingJobs(gomock.Any(), clusterId, &sdk.ScheduledRebalancingAPIListRebalancingJobsParams{RebalancingScheduleId: lo.ToPtr(rebalancingScheduleId)}).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	resource := resourceRebalancingJob()
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:            cty.StringVal(clusterId),
		"rebalancing_schedule_id": cty.StringVal(rebalancingScheduleId),
		"id":                      cty.StringVal(id),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	state.ID = id

	data := resource.Data(state)
	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.ElementsMatch(strings.Split(`ID = abc4192d-a400-48ed-9b22-5fcc9e045258
cluster_id = b6bfc074-a267-400f-b8f1-db0850c369b1
enabled = true
rebalancing_schedule_id = 8155d717-ee9c-43b0-973f-55610beaf8c2
Tainted = false
`, "\n"),
		strings.Split(data.State().String(), "\n"))
}
