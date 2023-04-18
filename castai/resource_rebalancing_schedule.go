package castai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"
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
		Description: "CAST AI rebalancing schedule resource to manage rebalancing schedules",

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
				Description:      "Name of the schedule",
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
							Description:      "Cron expression defining when the schedule should trigger",
						},
					},
				},
			},
			"trigger_conditions": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"savings_percentage": {
							Type:             schema.TypeFloat,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.FloatAtLeast(0.0)),
							Description:      "Defines minimum number of savings expected",
						},
					},
				},
			},
			"launch_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"node_ttl_seconds": {
							Type:             schema.TypeInt,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
							Description:      "Specifies amount of time since node creation before the node is allowed to be considered for automated rebalancing",
						},
						"num_targeted_nodes": {
							Type:             schema.TypeInt,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
							Description:      "Maximum number of nodes that will be selected for rebalancing",
						},
						"rebalancing_min_nodes": {
							Type:             schema.TypeInt,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
							Description:      "Minimum number of nodes that should be kept in the cluster after rebalancing",
						},
						"selector": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "Node selector in JSON format.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsJSON),
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
		}
	}

	if launchConfigurationData := toSection(d, "launch_configuration"); launchConfigurationData != nil {
		selector, err := readOptionalJson[sdk.ScheduledrebalancingV1NodeSelector](launchConfigurationData, "selector")
		if err != nil {
			return nil, fmt.Errorf("parsing selector: %w", err)
		}
		result.LaunchConfiguration = sdk.ScheduledrebalancingV1LaunchConfiguration{
			NodeTtlSeconds:   readOptionalNumber[int, int32](launchConfigurationData, "node_ttl_seconds"),
			NumTargetedNodes: readOptionalNumber[int, int32](launchConfigurationData, "num_targeted_nodes"),
			RebalancingOptions: &sdk.ScheduledrebalancingV1RebalancingOptions{
				MinNodes: readOptionalNumber[int, int32](launchConfigurationData, "rebalancing_min_nodes"),
			},
			Selector: selector,
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
	return nil
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
