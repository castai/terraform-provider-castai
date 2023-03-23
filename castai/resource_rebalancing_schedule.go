package castai

import (
	"context"
	"fmt"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/samber/lo"
	"time"
)

func resourceRebalancingSchedule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRebalancingScheduleCreate,
		ReadContext:   resourceRebalancingScheduleRead,
		DeleteContext: resourceRebalancingScheduleDelete,
		// UpdateContext: resourceRebalancingScheduleUpdate,
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

		Schema: map[string]*schema.Schema{},
	}
}

func rebalancingScheduleStateImporter(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	return nil, fmt.Errorf("TODO")
}

func resourceRebalancingScheduleCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	req := sdk.ScheduledRebalancingAPICreateRebalancingScheduleJSONRequestBody{
		TriggerConditions: &sdk.ScheduledrebalancingV1TriggerConditions{
			SavingsPercentage: lo.ToPtr[float32](1.15),
		},
	}

	resp, err := client.ScheduledRebalancingAPICreateRebalancingScheduleWithResponse(ctx, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	d.SetId(*resp.JSON200.Id)

	return resourceRebalancingScheduleRead(ctx, d, meta)
}

func resourceRebalancingScheduleRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return nil
}

func resourceRebalancingScheduleUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return nil
}

func resourceRebalancingScheduleDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return nil
}
