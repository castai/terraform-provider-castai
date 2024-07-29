package castai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func resourceRebalancingSchedule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRebalancingScheduleCreate,
		ReadContext:   resourceRebalancingScheduleRead,
		DeleteContext: resourceRebalancingScheduleDelete,
		UpdateContext: resourceRebalancingScheduleUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: rebalancingScheduleStateImporter,
		},
		Description: "CAST AI rebalancing schedule resource to manage rebalancing schedules.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "Name of the schedule.",
			},
			"schedule": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cron": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
							Description: "Cron expression defining when the schedule should trigger.\n\n" +
								"  The `cron` expression can optionally include the `CRON_TZ` variable at the beginning to specify the timezone in which the schedule should be interpreted.\n\n" +
								"  Example:\n" +
								"  ```plaintext\n" +
								"  CRON_TZ=America/New_York 0 12 * * ?\n" +
								"  ```\n" +
								"  In the example above, the `CRON_TZ` variable is set to \"America/New_York\" indicating that the cron expression should be interpreted in the Eastern Time (ET) timezone.\n\n" +
								"  To retrieve a list of available timezone values, you can use the following API endpoint:\n\n" +
								"  GET https://api.cast.ai/v1/time-zones\n\n" +
								"  When using the `CRON_TZ` variable, ensure that the specified timezone is valid and supported by checking the list of available timezones from the API endpoint." +
								"  If the `CRON_TZ` variable is not specified, the cron expression will be interpreted in the UTC timezone.",
						},
					},
				},
			},
			"trigger_conditions": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"savings_percentage": {
							Type:             schema.TypeFloat,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.FloatAtLeast(0.0)),
							Description:      "Defines the minimum percentage of savings expected.",
						},
						"ignore_savings": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "If true, the savings percentage will be ignored and the rebalancing will be triggered regardless of the savings percentage.",
						},
					},
				},
			},
			"launch_configuration": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"node_ttl_seconds": {
							Type:             schema.TypeInt,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
							Description:      "Specifies amount of time since node creation before the node is allowed to be considered for automated rebalancing.",
						},
						"num_targeted_nodes": {
							Type:             schema.TypeInt,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
							Description:      "Maximum number of nodes that will be selected for rebalancing.",
						},
						"rebalancing_min_nodes": {
							Type:             schema.TypeInt,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
							Description:      "Minimum number of nodes that should be kept in the cluster after rebalancing.",
						},
						"keep_drain_timeout_nodes": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Defines whether the nodes that failed to get drained until a predefined timeout, will be kept with a rebalancing.cast.ai/status=drain-failed annotation instead of forcefully drained.",
						},
						"aggressive_mode": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "When enabled rebalancing will also consider problematic pods (pods without controller, job pods, pods with removal-disabled annotation) as not-problematic.",
						},
						"execution_conditions": {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:        schema.TypeBool,
										Required:    true,
										Description: "Enables or disables the execution conditions.",
									},
									"achieved_savings_percentage": {
										Type:             schema.TypeInt,
										Optional:         true,
										Default:          0,
										ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 100)),
										Description: "The percentage of the predicted savings that must be achieved in order to fully execute the plan." +
											"If the savings are not achieved after creating the new nodes, the plan will fail and delete the created nodes.",
									},
								},
							},
						},
						"selector": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "Node selector in JSON format.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsJSON),
						},
						"target_node_selection_algorithm": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Defines the algorithm used to select the target nodes for rebalancing.",
							Default:     "TargetNodeSelectionAlgorithmNormalizedPrice",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
								"TargetNodeSelectionAlgorithmNormalizedPrice",
								"TargetNodeSelectionAlgorithmUtilizedPrice",
								"TargetNodeSelectionAlgorithmUtilization",
							}, false)),
						},
					},
				},
			},
		},
	}
}

func resourceRebalancingScheduleCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	schedule, err := stateToSchedule(d)
	if err != nil {
		return diag.FromErr(err)
	}

	req := sdk.ScheduledRebalancingAPICreateRebalancingScheduleJSONRequestBody{
		Name:                schedule.Name,
		Schedule:            schedule.Schedule,
		LaunchConfiguration: schedule.LaunchConfiguration,
		TriggerConditions:   schedule.TriggerConditions,
	}

	resp, err := client.ScheduledRebalancingAPICreateRebalancingScheduleWithResponse(ctx, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	d.SetId(*resp.JSON200.Id)

	return resourceRebalancingScheduleRead(ctx, d, meta)
}

func resourceRebalancingScheduleRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	schedule, err := getRebalancingScheduleById(ctx, meta.(*ProviderConfig).api, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if !d.IsNewResource() && schedule == nil {
		tflog.Warn(ctx, "Rebalancing schedule not found, removing from state", map[string]any{"id": d.Id()})
		d.SetId("")
		return nil
	}

	if err := scheduleToState(schedule, d); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceRebalancingScheduleUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	schedule, err := stateToSchedule(d)
	if err != nil {
		return diag.FromErr(err)
	}

	req := sdk.ScheduledRebalancingAPIUpdateRebalancingScheduleJSONRequestBody{
		Name:                lo.ToPtr(schedule.Name),
		Schedule:            &schedule.Schedule,
		LaunchConfiguration: &schedule.LaunchConfiguration,
		TriggerConditions:   &schedule.TriggerConditions,
	}

	resp, err := client.ScheduledRebalancingAPIUpdateRebalancingScheduleWithResponse(ctx, &sdk.ScheduledRebalancingAPIUpdateRebalancingScheduleParams{
		Id: lo.ToPtr(d.Id()),
	}, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}
	return resourceRebalancingScheduleRead(ctx, d, meta)
}

func resourceRebalancingScheduleDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	resp, err := client.ScheduledRebalancingAPIDeleteRebalancingScheduleWithResponse(ctx, d.Id())
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}
	return nil
}

func rebalancingScheduleStateImporter(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	client := meta.(*ProviderConfig).api

	// if importing by UUID, nothing to do; if importing by name, fetch schedule ID and set that as resource ID
	if _, err := uuid.Parse(d.Id()); err != nil {
		tflog.Info(ctx, "provided schedule ID is not a UUID, will import by name")
		schedule, err := getRebalancingScheduleByName(ctx, client, d.Id())
		if err != nil {
			return nil, err
		}
		d.SetId(lo.FromPtr(schedule.Id))
	}

	return []*schema.ResourceData{d}, nil
}

func stateToSchedule(d *schema.ResourceData) (*sdk.ScheduledrebalancingV1RebalancingSchedule, error) {
	scheduleData := toSection(d, "schedule")

	result := sdk.ScheduledrebalancingV1RebalancingSchedule{
		Id:   lo.ToPtr(d.Id()),
		Name: d.Get("name").(string),
		Schedule: sdk.ScheduledrebalancingV1Schedule{
			Cron: scheduleData["cron"].(string),
		},
	}
	if triggerConditions := toSection(d, "trigger_conditions"); triggerConditions != nil {
		result.TriggerConditions = sdk.ScheduledrebalancingV1TriggerConditions{
			SavingsPercentage: readOptionalNumber[float64, float32](triggerConditions, "savings_percentage"),
			IgnoreSavings:     readOptionalValue[bool](triggerConditions, "ignore_savings"),
		}
	}

	if launchConfigurationData := toSection(d, "launch_configuration"); launchConfigurationData != nil {
		selector, err := readOptionalJson[sdk.ScheduledrebalancingV1NodeSelector](launchConfigurationData, "selector")
		if err != nil {
			return nil, fmt.Errorf("parsing selector: %w", err)
		}

		keepDrainTimeoutNodes := readOptionalValue[bool](launchConfigurationData, "keep_drain_timeout_nodes")

		var executionConditions *sdk.ScheduledrebalancingV1ExecutionConditions
		executionConditionsData := launchConfigurationData["execution_conditions"].([]any)
		if len(executionConditionsData) != 0 {
			executionConditions = &sdk.ScheduledrebalancingV1ExecutionConditions{
				Enabled:                   lo.ToPtr(executionConditionsData[0].(map[string]any)["enabled"].(bool)),
				AchievedSavingsPercentage: lo.ToPtr(int32(executionConditionsData[0].(map[string]any)["achieved_savings_percentage"].(int))),
			}
		}

		aggresiveMode := readOptionalValue[bool](launchConfigurationData, "aggressive_mode")

		targetAlgorithm := sdk.ScheduledrebalancingV1TargetNodeSelectionAlgorithm(*readOptionalValue[string](launchConfigurationData, "target_node_selection_algorithm"))
		result.LaunchConfiguration = sdk.ScheduledrebalancingV1LaunchConfiguration{
			NodeTtlSeconds:   readOptionalNumber[int, int32](launchConfigurationData, "node_ttl_seconds"),
			NumTargetedNodes: readOptionalNumber[int, int32](launchConfigurationData, "num_targeted_nodes"),
			RebalancingOptions: &sdk.ScheduledrebalancingV1RebalancingOptions{
				MinNodes:              readOptionalNumber[int, int32](launchConfigurationData, "rebalancing_min_nodes"),
				KeepDrainTimeoutNodes: keepDrainTimeoutNodes,
				ExecutionConditions:   executionConditions,
				AggressiveMode:        aggresiveMode,
			},
			Selector:                     selector,
			TargetNodeSelectionAlgorithm: &targetAlgorithm,
		}
	}

	return &result, nil
}

func scheduleToState(schedule *sdk.ScheduledrebalancingV1RebalancingSchedule, d *schema.ResourceData) error {
	d.SetId(*schedule.Id)
	if err := d.Set("name", schedule.Name); err != nil {
		return err
	}
	if err := d.Set("schedule", []map[string]any{
		{
			"cron": schedule.Schedule.Cron,
		},
	}); err != nil {
		return err
	}

	launchConfig := map[string]any{
		"node_ttl_seconds":   schedule.LaunchConfiguration.NodeTtlSeconds,
		"num_targeted_nodes": schedule.LaunchConfiguration.NumTargetedNodes,
	}

	if schedule.LaunchConfiguration.RebalancingOptions != nil {
		launchConfig["rebalancing_min_nodes"] = schedule.LaunchConfiguration.RebalancingOptions.MinNodes
		launchConfig["keep_drain_timeout_nodes"] = schedule.LaunchConfiguration.RebalancingOptions.KeepDrainTimeoutNodes
		launchConfig["aggressive_mode"] = schedule.LaunchConfiguration.RebalancingOptions.AggressiveMode
		launchConfig["target_node_selection_algorithm"] = schedule.LaunchConfiguration.TargetNodeSelectionAlgorithm

		executionConditions := schedule.LaunchConfiguration.RebalancingOptions.ExecutionConditions
		if executionConditions != nil {
			launchConfig["execution_conditions"] = []map[string]any{
				{
					"enabled":                     executionConditions.Enabled,
					"achieved_savings_percentage": executionConditions.AchievedSavingsPercentage,
				},
			}
		}
	}

	selector := schedule.LaunchConfiguration.Selector
	if selector != nil && selector.NodeSelectorTerms != nil && len(*selector.NodeSelectorTerms) > 0 {
		nullifySelectorEmptyLists(selector)
		selectorJSON, err := json.Marshal(selector)
		if err != nil {
			return fmt.Errorf("serializing selector: %w", err)
		}
		launchConfig["selector"] = string(selectorJSON)
	}

	if err := d.Set("launch_configuration", []map[string]any{launchConfig}); err != nil {
		return err
	}

	triggerConditions := map[string]any{
		"savings_percentage": toFloat64PtrTruncated(schedule.TriggerConditions.SavingsPercentage),
		"ignore_savings":     schedule.TriggerConditions.IgnoreSavings,
	}
	if err := d.Set("trigger_conditions", []map[string]any{triggerConditions}); err != nil {
		return err
	}

	return nil
}

// toFloat64PtrTruncated returns float truncated to 5 numbers of precision
// truncation is needed to avoid state mismatches during all the conversions of float32<->float64
func toFloat64PtrTruncated(v *float32) *float64 {
	if v == nil {
		return nil
	}
	const floatPrecisionTruncate float64 = 100000.0
	return lo.ToPtr(math.Round(float64(*v)*floatPrecisionTruncate) / floatPrecisionTruncate)
}

// nullifySelectorRequirements converts empty lists to null values; even though semantically
// both are the same for business logic, terraform complains about mismatches in state after re-reading the resource
func nullifySelectorEmptyLists(selector *sdk.ScheduledrebalancingV1NodeSelector) {
	selector.NodeSelectorTerms = toNilList(selector.NodeSelectorTerms)
	if selector.NodeSelectorTerms != nil {
		for i := range *selector.NodeSelectorTerms {
			t := &(*selector.NodeSelectorTerms)[i]
			t.MatchExpressions = toNilList(t.MatchExpressions)
			t.MatchFields = toNilList(t.MatchFields)

			nullifySelectorRequirements(t.MatchExpressions)
			nullifySelectorRequirements(t.MatchFields)
		}
	}
}

func nullifySelectorRequirements(requirements *[]sdk.ScheduledrebalancingV1NodeSelectorRequirement) {
	if requirements == nil {
		return
	}
	for i := range *requirements {
		r := &(*requirements)[i]
		r.Values = toNilList(r.Values)
	}
}

func getRebalancingScheduleByName(ctx context.Context, client *sdk.ClientWithResponses, name string) (*sdk.ScheduledrebalancingV1RebalancingSchedule, error) {
	resp, err := client.ScheduledRebalancingAPIListRebalancingSchedulesWithResponse(ctx)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return nil, checkErr
	}

	for _, schedule := range *resp.JSON200.Schedules {
		if schedule.Name == name {
			return &schedule, nil
		}
	}

	return nil, fmt.Errorf("rebalancing schedule %q was not found", name)
}

func getRebalancingScheduleById(ctx context.Context, client *sdk.ClientWithResponses, id string) (*sdk.ScheduledrebalancingV1RebalancingSchedule, error) {
	resp, err := client.ScheduledRebalancingAPIGetRebalancingScheduleWithResponse(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
