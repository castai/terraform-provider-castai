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
    "schedules": [
        {
            "id": "9302fdac-4922-4b09-afa6-0a12be99f112",
            "schedule": {
                "cron": "5 * * * * *"
            },
            "launchConfiguration": {
                "selector": {
                    "nodeSelectorTerms": []
                },
                "rebalancingOptions": {
                    "minNodes": 2,
                    "executionConditions": {
                        "enabled": true,
                        "achievedSavingsPercentage": 15
                    },
                    "keepDrainTimeoutNodes": false,
                    "evictGracefully": false,
                    "aggressiveMode": false,
                    "aggressiveModeConfig": {
                        "ignoreLocalPersistentVolumes": true,
                        "ignoreProblemJobPods": true,
                        "ignoreProblemRemovalDisabledPods": true,
                        "ignoreProblemPodsWithoutController": true
                    }
                },
                "numTargetedNodes": 20,
                "nodeTtlSeconds": 350,
                "targetNodeSelectionAlgorithm": "TargetNodeSelectionAlgorithmNormalizedPrice"
            },
            "triggerConditions": {
                "savingsPercentage": 15,
                "ignoreSavings": false
            },
            "nextTriggerAt": "2024-10-31T10:46:05Z",
            "name": "rebalancing schedule 1",
            "jobs": [],
            "lastTriggerAt": "2024-10-31T10:45:08.915021Z"
        },
        {
            "id": "d1954729-6fc0-4741-aeb0-c497e16f59f7",
            "schedule": {
                "cron": "5 * * * * *"
            },
            "launchConfiguration": {
                "selector": {
                    "nodeSelectorTerms": []
                },
                "rebalancingOptions": {
                    "minNodes": 2,
                    "executionConditions": {
                        "enabled": true,
                        "achievedSavingsPercentage": 15
                    },
                    "keepDrainTimeoutNodes": false,
                    "evictGracefully": false,
                    "aggressiveMode": false
                },
                "numTargetedNodes": 20,
                "nodeTtlSeconds": 350,
                "targetNodeSelectionAlgorithm": "TargetNodeSelectionAlgorithmNormalizedPrice"
            },
            "triggerConditions": {
                "savingsPercentage": 15,
                "ignoreSavings": false
            },
            "nextTriggerAt": "2024-10-31T10:46:05Z",
            "name": "rebalancing schedule 2",
            "jobs": [
                {
                    "id": "2ac90b71-8adc-468a-8680-ea4f99e4df27",
                    "clusterId": "d8bdd6d1-6b9a-4dbb-a276-5e44ff512322",
                    "rebalancingScheduleId": "d1954729-6fc0-4741-aeb0-c497e16f59f7",
                    "rebalancingPlanId": "",
                    "enabled": true,
                    "lastTriggerAt": "2024-10-31T10:38:06.594638Z",
                    "nextTriggerAt": "2024-10-31T10:46:05Z",
                    "status": "JobStatusSkipped"
                }
            ],
            "lastTriggerAt": "2024-10-31T10:45:08.922097Z"
        }
    ]
}`)))
	mockClient.EXPECT().
		ScheduledRebalancingAPIListRebalancingSchedules(gomock.Any()).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)

	resource := dataSourceRebalancingSchedule()
	data := resource.Data(state)

	r.NoError(data.Set("name", "rebalancing schedule 1"))

	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())

	expectedState := `ID = 9302fdac-4922-4b09-afa6-0a12be99f112
launch_configuration.# = 1
launch_configuration.0.aggressive_mode = false
launch_configuration.0.aggressive_mode_config.# = 1
launch_configuration.0.aggressive_mode_config.0.ignore_local_persistent_volumes = true
launch_configuration.0.aggressive_mode_config.0.ignore_problem_job_pods = true
launch_configuration.0.aggressive_mode_config.0.ignore_problem_pods_without_controller = true
launch_configuration.0.aggressive_mode_config.0.ignore_problem_removal_disabled_pods = true
launch_configuration.0.execution_conditions.# = 1
launch_configuration.0.execution_conditions.0.achieved_savings_percentage = 15
launch_configuration.0.execution_conditions.0.enabled = true
launch_configuration.0.keep_drain_timeout_nodes = false
launch_configuration.0.node_ttl_seconds = 350
launch_configuration.0.num_targeted_nodes = 20
launch_configuration.0.rebalancing_min_nodes = 2
launch_configuration.0.selector = 
launch_configuration.0.target_node_selection_algorithm = TargetNodeSelectionAlgorithmNormalizedPrice
name = rebalancing schedule 1
schedule.# = 1
schedule.0.cron = 5 * * * * *
trigger_conditions.# = 1
trigger_conditions.0.ignore_savings = false
trigger_conditions.0.savings_percentage = 15
Tainted = false
`
	r.Equal(expectedState, data.State().String())
}
