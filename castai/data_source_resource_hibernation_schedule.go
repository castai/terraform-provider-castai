package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHibernationSchedule() *schema.Resource {
	dataSourceHibernationSchedule := &schema.Resource{
		Description: "Retrieve Hibernation Schedule ",
		ReadContext: dataSourceHibernationScheduleRead,
		Schema:      map[string]*schema.Schema{},
	}

	resourceHibernationSchedule := resourceHibernationSchedule()
	for key, value := range resourceHibernationSchedule.Schema {
		dataSourceHibernationSchedule.Schema[key] = value
		if key != FieldHibernationScheduleName && key != FieldHibernationScheduleOrganizationID {
			// only name and optionally organization id are provided in terraform configuration by user
			// other parameters are "computed" from existing hibernation schedule
			dataSourceHibernationSchedule.Schema[key].Computed = true
			dataSourceHibernationSchedule.Schema[key].Required = false
			//  MaxItems is for configurable attributes, there's nothing to configure on computed-only field
			dataSourceHibernationSchedule.Schema[key].MaxItems = 0
		}
	}
	return dataSourceHibernationSchedule
}

func dataSourceHibernationScheduleRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	organizationID, err := getHibernationScheduleOrganizationID(ctx, data, meta)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error retrieving hibernation schedule organization id: %w", err))
	}

	scheduleName := data.Get(FieldHibernationScheduleName).(string)
	schedule, err := getHibernationScheduleByName(ctx, meta, organizationID, scheduleName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error retrieving hibernation schedule: %w", err))
	}

	if err := hibernationScheduleToState(schedule, data); err != nil {
		return diag.FromErr(fmt.Errorf("error converting schdeure to terraform state: %w", err))
	}
	return nil
}
