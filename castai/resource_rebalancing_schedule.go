package castai

import (
	"context"
	"fmt"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"
	"time"
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
		},
	}
}

func resourceRebalancingScheduleCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	schedule := resourceToRebalancingSchedule(d)

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

func resourceToRebalancingSchedule(d *schema.ResourceData) *sdk.ScheduledrebalancingV1RebalancingSchedule {
	scheduleData := d.Get("schedule").([]interface{})[0].(map[string]interface{})
	result := sdk.ScheduledrebalancingV1RebalancingSchedule{
		Id:   lo.ToPtr(d.Id()),
		Name: d.Get("name").(string),
		Schedule: sdk.ScheduledrebalancingV1Schedule{
			Cron: scheduleData["cron"].(string),
		},
		LaunchConfiguration: sdk.ScheduledrebalancingV1LaunchConfiguration{},
		TriggerConditions: sdk.ScheduledrebalancingV1TriggerConditions{
			SavingsPercentage: lo.ToPtr[float32](1.15),
		},
	}
	return &result
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

	if err := setStateFromSchedule(schedule, d); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %w", err))
	}

	return nil
}

func setStateFromSchedule(schedule *sdk.ScheduledrebalancingV1RebalancingSchedule, d *schema.ResourceData) error {
	d.SetId(*schedule.Id)
	if err := d.Set("name", schedule.Name); err != nil {
		return err
	}
	if err := d.Set("schedule", []map[string]interface{}{
		{
			"cron": schedule.Schedule.Cron,
		},
	}); err != nil {
		return err
	}
	return nil
}

func resourceRebalancingScheduleUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	schedule := resourceToRebalancingSchedule(d)

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
	return nil
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
