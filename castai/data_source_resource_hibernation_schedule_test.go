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
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler"
	mock_cluster_autoscaler "github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler/mock"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestHibernationScheduleDataSourceRead(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientWithResponsesInterface(ctrl)
	clusterAutoscalerClient := mock_cluster_autoscaler.NewMockClientInterface(ctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: mockClient,
		clusterAutoscalerClient: &cluster_autoscaler.ClientWithResponses{
			ClientInterface: clusterAutoscalerClient,
		},
	}

	organizationID := "0d0111f9-e5a4-4acc-85b2-4c3a8318dfc2"
	body := io.NopCloser(bytes.NewReader([]byte(`{
    "items": [
        {
            "id": "75c04e4e-f95c-4f24-a814-b8e753a5194d",
            "organizationId": "0d0111f9-e5a4-4acc-85b2-4c3a8318dfc2",
            "enabled": false,
            "name": "schedule",
            "pauseConfig": {
                "enabled": true,
                "schedule": {
                    "cronExpression": "1 0 * * *"
                }
            },
            "resumeConfig": {
                "enabled": true,
                "schedule": {
                    "cronExpression": "1 0 * * *"
                },
                "jobConfig": {
                    "nodeConfig": {
                        "instanceType": "e2-standard-4",
                        "kubernetesLabels": {},
                        "kubernetesTaints": []
                    }
                }
            },
            "clusterAssignments": {
                "items": [
                    {
                        "clusterId": "38a49ce8-e900-4a10-be89-48fb2efb1025"
                    }
                ]
            },
            "createTime": "2025-04-10T12:52:07.732194Z",
            "updateTime": "2025-04-10T12:52:07.732194Z"
        }
    ],
    "nextPageCursor": "",
    "totalCount": 1
}`)))

	mockClient.EXPECT().UsersAPIListOrganizationsWithResponse(gomock.Any()).Return(&sdk.UsersAPIListOrganizationsResponse{
		JSON200: &sdk.CastaiUsersV1beta1ListOrganizationsResponse{
			Organizations: []sdk.CastaiUsersV1beta1UserOrganization{
				{Id: lo.ToPtr(organizationID)},
			},
		},
		HTTPResponse: &http.Response{StatusCode: http.StatusOK},
	}, nil).Times(1)

	clusterAutoscalerClient.EXPECT().
		HibernationSchedulesAPIListHibernationSchedules(gomock.Any(), organizationID, gomock.Any()).
		Return(&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)

	resource := dataSourceHibernationSchedule()
	data := resource.Data(state)

	r.NoError(data.Set("name", "schedule"))

	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())

	expectedState := `ID = 75c04e4e-f95c-4f24-a814-b8e753a5194d
cluster_assignments.# = 1
cluster_assignments.0.assignment.# = 1
cluster_assignments.0.assignment.0.cluster_id = 38a49ce8-e900-4a10-be89-48fb2efb1025
enabled = false
name = schedule
organization_id = 0d0111f9-e5a4-4acc-85b2-4c3a8318dfc2
pause_config.# = 1
pause_config.0.enabled = true
pause_config.0.schedule.# = 1
pause_config.0.schedule.0.cron_expression = 1 0 * * *
resume_config.# = 1
resume_config.0.enabled = true
resume_config.0.job_config.# = 1
resume_config.0.job_config.0.node_config.# = 1
resume_config.0.job_config.0.node_config.0.config_id = 
resume_config.0.job_config.0.node_config.0.config_name = 
resume_config.0.job_config.0.node_config.0.gpu_config.# = 0
resume_config.0.job_config.0.node_config.0.instance_type = e2-standard-4
resume_config.0.job_config.0.node_config.0.kubernetes_labels.% = 0
resume_config.0.job_config.0.node_config.0.kubernetes_taints.# = 0
resume_config.0.job_config.0.node_config.0.node_affinity.# = 0
resume_config.0.job_config.0.node_config.0.spot_config.# = 0
resume_config.0.job_config.0.node_config.0.subnet_id = 
resume_config.0.job_config.0.node_config.0.volume.# = 0
resume_config.0.job_config.0.node_config.0.zone = 
resume_config.0.schedule.# = 1
resume_config.0.schedule.0.cron_expression = 1 0 * * *
Tainted = false
`
	r.Equal(expectedState, data.State().String())
}
