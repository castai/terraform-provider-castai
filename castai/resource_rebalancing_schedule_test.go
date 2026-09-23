package castai

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestRebalancingSchedule_stateToSchedule_EvictGracefullyAndDrainOptions(t *testing.T) {
	r := require.New(t)
	resource := resourceRebalancingSchedule()

	state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		"name": cty.StringVal("test-schedule"),
		"schedule": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"cron": cty.StringVal("5 4 * * *"),
		})}),
		"trigger_conditions": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"savings_percentage": cty.NumberFloatVal(15),
			"ignore_savings":     cty.BoolVal(false),
		})}),
		"launch_configuration": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"evict_gracefully":        cty.BoolVal(true),
			"max_simultaneous_drains": cty.NumberIntVal(5),
			"aggressive_mode_config": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"ignore_local_persistent_volumes":        cty.BoolVal(true),
				"ignore_problem_job_pods":                cty.BoolVal(true),
				"ignore_problem_removal_disabled_pods":   cty.BoolVal(false),
				"ignore_problem_pods_without_controller": cty.BoolVal(false),
				"ignore_problem_prevented_drain_pods":    cty.BoolVal(true),
			})}),
		})}),
	}), 0)
	// Raw config sets the optional fields explicitly.
	state.RawConfig = cty.ObjectVal(map[string]cty.Value{
		"launch_configuration": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"evict_gracefully":        cty.BoolVal(true),
			"max_simultaneous_drains": cty.NumberIntVal(5),
			"aggressive_mode_config": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"ignore_problem_prevented_drain_pods": cty.BoolVal(true),
			})}),
		})}),
	})

	schedule, err := stateToSchedule(resource.Data(state))
	r.NoError(err)

	opts := schedule.LaunchConfiguration.RebalancingOptions
	r.NotNil(opts)
	r.NotNil(opts.EvictGracefully)
	r.True(*opts.EvictGracefully)
	r.NotNil(opts.MaxSimultaneousDrains)
	r.Equal(int32(5), *opts.MaxSimultaneousDrains)
	r.NotNil(opts.AggressiveModeConfig)
	r.NotNil(opts.AggressiveModeConfig.IgnoreProblemPreventedDrainPods)
	r.True(*opts.AggressiveModeConfig.IgnoreProblemPreventedDrainPods)

	// When the optional field is omitted (null in config), it must stay nil
	// in the request body instead of being sent as an explicit false.
	unsetState := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		"name": cty.StringVal("test-schedule-unset"),
		"schedule": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"cron": cty.StringVal("5 4 * * *"),
		})}),
		"trigger_conditions": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"savings_percentage": cty.NumberFloatVal(15),
			"ignore_savings":     cty.BoolVal(false),
		})}),
		"launch_configuration": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"aggressive_mode_config": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"ignore_local_persistent_volumes":        cty.BoolVal(true),
				"ignore_problem_job_pods":                cty.BoolVal(true),
				"ignore_problem_removal_disabled_pods":   cty.BoolVal(false),
				"ignore_problem_pods_without_controller": cty.BoolVal(false),
				"ignore_problem_prevented_drain_pods":    cty.NullVal(cty.Bool),
			})}),
		})}),
	}), 0)
	// Raw config omits the field: it is null, as Terraform sends it when the
	// user does not set it.
	unsetState.RawConfig = cty.ObjectVal(map[string]cty.Value{
		"launch_configuration": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"evict_gracefully":        cty.NullVal(cty.Bool),
			"max_simultaneous_drains": cty.NullVal(cty.Number),
			"aggressive_mode_config": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"ignore_problem_prevented_drain_pods": cty.NullVal(cty.Bool),
			})}),
		})}),
	})

	unsetSchedule, err := stateToSchedule(resource.Data(unsetState))
	r.NoError(err)
	unsetOpts := unsetSchedule.LaunchConfiguration.RebalancingOptions
	r.NotNil(unsetOpts)
	r.NotNil(unsetOpts.AggressiveModeConfig)
	r.Nil(unsetOpts.AggressiveModeConfig.IgnoreProblemPreventedDrainPods)
	// Unset optional fields must stay nil so they are omitted from the
	// request body instead of being sent as explicit false/zero.
	r.Nil(unsetOpts.EvictGracefully)
	r.Nil(unsetOpts.MaxSimultaneousDrains)
}

func TestAccCloudAgnostic_ResourceRebalancingSchedule_basic(t *testing.T) {
	rName := fmt.Sprintf("%v-rebalancing-schedule-%v", ResourcePrefix, acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: makeInitialRebalancingScheduleConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "name", rName),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "schedule.0.cron", "5 4 * * *"),
				),
			},
			{
				// test edits
				Config: makeUpdatedRebalancingScheduleConfig(rName + " renamed"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "name", rName+" renamed"),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "schedule.0.cron", "1 4 * * *"),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "launch_configuration.0.aggressive_mode", "true"),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "launch_configuration.0.aggressive_mode_config.0.ignore_local_persistent_volumes", "true"),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "launch_configuration.0.aggressive_mode_config.0.ignore_problem_job_pods", "true"),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "launch_configuration.0.aggressive_mode_config.0.ignore_problem_removal_disabled_pods", "true"),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "launch_configuration.0.aggressive_mode_config.0.ignore_problem_pods_without_controller", "true"),
				),
			},
			{
				Config: makeUpdatedMinNodes(rName + " min_nodes_zero"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "name", rName+" min_nodes_zero"),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "launch_configuration.0.rebalancing_min_nodes", "0"),
				),
			},
			{
				Config: makeConfigWithDrainFailureConfig(rName + " drain_failure"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "name", rName+" drain_failure"),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "launch_configuration.0.evict_gracefully", "true"),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "launch_configuration.0.drain_failure_config.0.disable_uncordon", "false"),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "launch_configuration.0.drain_failure_config.0.uncordon_after_seconds", "7200"),
				),
			},
			// We keep the ImportState test cases at the end so they will test any newly added fields.
			// This way it will also verify that after importing the state the output of `tf plan` is empty.
			{
				// Import state by ID
				ImportState:       true,
				ResourceName:      "castai_rebalancing_schedule.test",
				ImportStateVerify: true,
			},
			{
				// Import state by name
				ImportState:  true,
				ResourceName: "castai_rebalancing_schedule.test",
				// Make sure to use the resource (rebalancing schedule) name from the latest prior step that changes the resource,
				// otherwise this step will not to see the changes made after the State ID you provided,
				// and this step will fail.
				ImportStateId:     rName + " drain_failure",
				ImportStateVerify: true,
			},
		},
	})
}

func makeInitialRebalancingScheduleConfig(rName string) string {
	template := `
resource "castai_rebalancing_schedule" "test" {
	name = %q
	schedule {
		cron = "5 4 * * *"
	}
	trigger_conditions {
		savings_percentage = 15.25
	}
	launch_configuration {
		execution_conditions {
			enabled = false
			achieved_savings_percentage = 0
		}
	}
}
`
	return fmt.Sprintf(template, rName)
}

func makeUpdatedRebalancingScheduleConfig(rName string) string {
	template := `
resource "castai_rebalancing_schedule" "test" {
	name = %q
	schedule {
		cron = "1 4 * * *"
	}
	trigger_conditions {
		savings_percentage = 1.23456
	}
	launch_configuration {
		node_ttl_seconds = 10
		num_targeted_nodes = 3
		rebalancing_min_nodes = 2
		evict_gracefully = true
		aggressive_mode = true
		aggressive_mode_config {
      		ignore_local_persistent_volumes = true
      		ignore_problem_job_pods = true
      		ignore_problem_removal_disabled_pods = true
      		ignore_problem_pods_without_controller = true
      		ignore_problem_prevented_drain_pods = true
    	}
		selector = jsonencode({
			nodeSelectorTerms = [{
				matchExpressions = [
					{
						key =  "thing"
						operator = "In"
						values = ["a", "b", "c"]
					}
				]
			}]
		})
		execution_conditions {
			enabled = true
			achieved_savings_percentage = 10
		}
	}
}
`
	return fmt.Sprintf(template, rName)
}

func makeConfigWithDrainFailureConfig(rName string) string {
	template := `
resource "castai_rebalancing_schedule" "test" {
	name = %q
	schedule {
		cron = "1 4 * * *"
	}
	trigger_conditions {
		savings_percentage = 10
	}
	launch_configuration {
		evict_gracefully = true
		drain_failure_config {
			disable_uncordon       = false
			uncordon_after_seconds = 7200
		}
		execution_conditions {
			enabled = true
			achieved_savings_percentage = 0
		}
	}
}
`
	return fmt.Sprintf(template, rName)
}

func makeUpdatedMinNodes(rName string) string {
	template := `
resource "castai_rebalancing_schedule" "test" {
	name = %q
	schedule {
		cron = "1 4 * * *"
	}
	trigger_conditions {
		savings_percentage = 1.23456
	}
	launch_configuration {
		rebalancing_min_nodes = 0
		execution_conditions {
			enabled = true
			achieved_savings_percentage = 10
		}
	}
}
`
	return fmt.Sprintf(template, rName)
}
