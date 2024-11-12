package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRebalancingSchedule() *schema.Resource {
	dataSourceRebalancingSchedule := &schema.Resource{
		Description: "Retrieve Rebalancing Schedule ",
		ReadContext: dataSourceRebalancingScheduleRead,
		Schema:      map[string]*schema.Schema{},
	}

	resourceRebalancingSchedule := resourceRebalancingSchedule()
	for key, value := range resourceRebalancingSchedule.Schema {
		dataSourceRebalancingSchedule.Schema[key] = value
		if key != "name" {
			// only name is provided in terraform configuration by user
			// other parameters are "computed" from existing rebalancing schedule
			dataSourceRebalancingSchedule.Schema[key].Computed = true
			dataSourceRebalancingSchedule.Schema[key].Required = false
		}
	}
	return dataSourceRebalancingSchedule
}

func dataSourceRebalancingScheduleRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	rebalancingScheduleName := data.Get("name").(string)
	client := meta.(*ProviderConfig).api
	schedule, err := getRebalancingScheduleByName(ctx, client, rebalancingScheduleName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error retrieving rebalancing schedule: %w", err))
	}

	if err := scheduleToState(schedule, data); err != nil {
		return diag.FromErr(fmt.Errorf("error converting schdeure to terraform state: %w", err))
	}
	return nil
}
