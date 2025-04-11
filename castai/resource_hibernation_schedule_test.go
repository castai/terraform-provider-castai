package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler"
	mock_cluster_autoscaler "github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler/mock"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestHibernationSchedule_CreateContext(t *testing.T) {
	t.Parallel()

	t.Run("should create schedule and populate the state", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
		clusterAutoscalerClient := mock_cluster_autoscaler.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
			clusterAutoscalerClient: &cluster_autoscaler.ClientWithResponses{
				ClientInterface: clusterAutoscalerClient,
			},
		}

		scheduleID := uuid.NewString()
		name := "schedule"
		organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
		schedule := &cluster_autoscaler.HibernationSchedule{
			Id:   &scheduleID,
			Name: name,
		}

		clusterAutoscalerClient.EXPECT().
			HibernationSchedulesAPICreateHibernationSchedule(gomock.Any(), organizationID, gomock.Any()).
			DoAndReturn(func(_ context.Context, organizationId string, req cluster_autoscaler.HibernationSchedulesAPICreateHibernationScheduleJSONRequestBody) (*http.Response, error) {
				r.Equal(name, req.Name)
				r.Equal(true, req.Enabled)
				r.Equal("* * * * *", req.PauseConfig.Schedule.CronExpression)
				r.Equal(true, req.PauseConfig.Enabled)
				r.Equal("* * * * *", req.ResumeConfig.Schedule.CronExpression)
				r.Equal(true, req.ResumeConfig.Enabled)

				nodeConfig := req.ResumeConfig.JobConfig.NodeConfig
				r.Equal("e2-standard-4", nodeConfig.InstanceType)
				r.Equal("NVIDIA_TESLA_T4", *nodeConfig.GpuConfig.Type)
				r.Equal(int32(1), *nodeConfig.GpuConfig.Count)

				r.Len(*nodeConfig.KubernetesLabels, 2)
				r.Equal("value1", (*nodeConfig.KubernetesLabels)["key1"])
				r.Equal("value2", (*nodeConfig.KubernetesLabels)["key2"])

				r.Len(*nodeConfig.KubernetesTaints, 1)
				taint := (*nodeConfig.KubernetesTaints)[0]
				r.Equal("key1", taint.Key)
				r.Equal("value1", taint.Value)
				r.Equal("NoSchedule", taint.Effect)

				r.Equal("some group", *nodeConfig.NodeAffinity.DedicatedGroup)
				r.Len(*nodeConfig.NodeAffinity.Affinity, 2)

				firstAffinity := (*nodeConfig.NodeAffinity.Affinity)[0]
				r.Equal("key1", firstAffinity.Key)
				r.Equal([]string{"value1"}, firstAffinity.Values)
				r.Equal(cluster_autoscaler.IN, firstAffinity.Operator)

				secondAffinity := (*nodeConfig.NodeAffinity.Affinity)[1]
				r.Equal("key2", secondAffinity.Key)
				r.Equal([]string{"value2"}, secondAffinity.Values)
				r.Equal(cluster_autoscaler.IN, secondAffinity.Operator)

				r.Equal(true, *nodeConfig.SpotConfig.Spot)
				r.Equal("0.5", *nodeConfig.SpotConfig.PriceHourly)

				r.NotNil(nodeConfig.Volume)
				r.Equal(int32(10), *nodeConfig.Volume.SizeGib)
				r.NotNil(nodeConfig.Volume.RaidConfig)
				r.Equal(int32(128), *nodeConfig.Volume.RaidConfig.ChunkSizeKb)

				body := bytes.NewBuffer([]byte(""))
				err := json.NewEncoder(body).Encode(schedule)
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		clusterAutoscalerClient.EXPECT().
			HibernationSchedulesAPIGetHibernationSchedule(gomock.Any(), organizationID, scheduleID).
			DoAndReturn(func(_ context.Context, organizationID, scheduleID string) (*http.Response, error) {
				body := bytes.NewBuffer([]byte(""))
				err := json.NewEncoder(body).Encode(schedule)
				r.NoError(err)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body), Header: map[string][]string{"Content-Type": {"json"}}}, nil
			})

		resource := resourceHibernationSchedule()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			"name":            cty.StringVal(name),
			"organization_id": cty.StringVal(organizationID),
			"enabled":         cty.BoolVal(true),
			"pause_config": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"enabled": cty.BoolVal(true),
					"schedule": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"cron_expression": cty.StringVal("* * * * *"),
						}),
					}),
				}),
			}),
			"resume_config": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"enabled": cty.BoolVal(true),
					"schedule": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"cron_expression": cty.StringVal("* * * * *"),
						}),
					}),
					"job_config": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"node_config": cty.ListVal([]cty.Value{
								cty.ObjectVal(map[string]cty.Value{
									"instance_type": cty.StringVal("e2-standard-4"),
									"gpu_config": cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											"type":  cty.StringVal("NVIDIA_TESLA_T4"),
											"count": cty.NumberIntVal(1),
										}),
									}),
									"kubernetes_labels": cty.ObjectVal(map[string]cty.Value{
										"key1": cty.StringVal("value1"),
										"key2": cty.StringVal("value2"),
									}),
									"kubernetes_taints": cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											"key":    cty.StringVal("key1"),
											"value":  cty.StringVal("value1"),
											"effect": cty.StringVal("NoSchedule"),
										}),
									}),
									"node_affinity": cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											"dedicated_group": cty.StringVal("some group"),
											"affinity": cty.ListVal([]cty.Value{
												cty.ObjectVal(map[string]cty.Value{
													"key":      cty.StringVal("key1"),
													"operator": cty.StringVal("IN"),
													"values": cty.ListVal([]cty.Value{
														cty.StringVal("value1"),
													}),
												}),
												cty.ObjectVal(map[string]cty.Value{
													"key":      cty.StringVal("key2"),
													"operator": cty.StringVal("IN"),
													"values": cty.ListVal([]cty.Value{
														cty.StringVal("value2"),
													}),
												}),
											}),
										}),
									}),
									"spot_config": cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											"spot":         cty.BoolVal(true),
											"price_hourly": cty.StringVal("0.5"),
										}),
									}),
									"volume": cty.ListVal([]cty.Value{
										cty.ObjectVal(map[string]cty.Value{
											"size_gib": cty.NumberIntVal(10),
											"raid_config": cty.ListVal([]cty.Value{
												cty.ObjectVal(map[string]cty.Value{
													"chunk_size_kb": cty.NumberIntVal(128),
												}),
											}),
										}),
									}),
								}),
							}),
						}),
					}),
				}),
			}),
			"cluster_assignments": cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"assignment": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"cluster_id": cty.StringVal(uuid.NewString()),
						}),
						cty.ObjectVal(map[string]cty.Value{
							"cluster_id": cty.StringVal(uuid.NewString()),
						}),
					}),
				}),
			}),
		})

		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Equal(scheduleID, data.Id())
		r.Equal(name, data.Get(FieldHibernationScheduleName))
	})
}

func TestAccResourceHibernationSchedule_basic(t *testing.T) {
	resourceName := fmt.Sprintf("%v-hibernation-schedule-%v", ResourcePrefix, acctest.RandString(8))
	renamedResourceName := fmt.Sprintf("%s %s", resourceName, "renamed")
	organizationID := os.Getenv("ACCEPTANCE_TEST_ORGANIZATION_ID")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				// Test creation
				Config: makeInitialHibernationScheduleConfig(resourceName, organizationID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "name", resourceName),
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "enabled", "false"),
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "pause_config.0.enabled", "true"),
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "pause_config.0.schedule.0.cron_expression", "0 0 * * *"),
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "resume_config.0.enabled", "true"),
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "resume_config.0.schedule.0.cron_expression", "0 0 * * *"),
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "resume_config.0.job_config.0.node_config.0.instance_type", "e2-standard-4"),
				),
			},
			{
				// Test update
				Config: makeUpdateHibernationScheduleConfig(renamedResourceName, organizationID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "name", renamedResourceName),
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "enabled", "true"),
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "pause_config.0.enabled", "false"),
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "pause_config.0.schedule.0.cron_expression", "1 0 * * *"),
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "resume_config.0.enabled", "false"),
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "resume_config.0.schedule.0.cron_expression", "1 0 * * *"),
					resource.TestCheckResourceAttr("castai_hibernation_schedule.test_hibernation_schedule", "resume_config.0.job_config.0.node_config.0.instance_type", "e2-standard-8"),
				),
			},
			{
				// Test import
				ImportState:       true,
				ResourceName:      "castai_hibernation_schedule.test_hibernation_schedule",
				ImportStateId:     fmt.Sprintf("%s/%s", organizationID, renamedResourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func makeInitialHibernationScheduleConfig(rName string, organizationID string) string {
	template := `
resource "castai_hibernation_schedule" "test_hibernation_schedule" {
  name    = %q
  enabled = false
  organization_id = %q

  pause_config {
    enabled = true

    schedule {
      cron_expression = "0 0 * * *"
    }
  }

  resume_config {
    enabled = true

    schedule {
      cron_expression = "0 0 * * *"
    }

    job_config {
      node_config {
        instance_type = "e2-standard-4"
      }
    }
  }

  cluster_assignments {}
}
`
	return fmt.Sprintf(template, rName, organizationID)
}

func makeUpdateHibernationScheduleConfig(rName string, organizationID string) string {
	template := `
resource "castai_hibernation_schedule" "test_hibernation_schedule" {
  name    = %q
  enabled = true
  organization_id = %q

  pause_config {
    enabled = false

    schedule {
      cron_expression = "1 0 * * *"
    }
  }

  resume_config {
    enabled = false

    schedule {
      cron_expression = "1 0 * * *"
    }

    job_config {
      node_config {
        instance_type = "e2-standard-8"
      }
    }
  }

  cluster_assignments {}
}
`
	return fmt.Sprintf(template, rName, organizationID)
}
