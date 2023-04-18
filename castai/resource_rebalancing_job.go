package castai

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"strings"
	"time"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/samber/lo"
)

func resourceRebalancingJob() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRebalancingJobCreate,
		ReadContext:   resourceRebalancingJobRead,
		DeleteContext: resourceRebalancingJobDelete,
		UpdateContext: resourceRebalancingJobUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: rebalancingJobStateImporter,
		},
		Description: "TODO",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldClusterId: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "CAST AI cluster id.",
			},
			"rebalancing_schedule_id": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "rebalancing schedule of this job",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "TODO",
			},
		},
	}
}

func resourceRebalancingJobCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	job, err := stateToRebalancingJob(d)
	if err != nil {
		return diag.FromErr(err)
	}

	req := sdk.ScheduledRebalancingAPICreateRebalancingJobJSONRequestBody{
		Id:                    job.Id,
		ClusterId:             job.ClusterId,
		RebalancingScheduleId: job.RebalancingScheduleId,
		Enabled:               job.Enabled,
	}

	resp, err := client.ScheduledRebalancingAPICreateRebalancingJobWithResponse(ctx, *job.ClusterId, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	d.SetId(*resp.JSON200.Id)

	return resourceRebalancingJobRead(ctx, d, meta)
}
func resourceRebalancingJobRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	clusterID := d.Get("cluster_id").(string)
	job, err := getRebalancingJobById(ctx, meta.(*ProviderConfig).api, clusterID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if !d.IsNewResource() && job == nil {
		tflog.Warn(ctx, "Rebalancing job not found, removing from state", map[string]any{"id": d.Id()})
		d.SetId("")
		return nil
	}

	if err := rebalancingJobToState(job, d); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceRebalancingJobUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	job, err := stateToRebalancingJob(d)
	if err != nil {
		return diag.FromErr(err)
	}

	req := sdk.ScheduledRebalancingAPIUpdateRebalancingJobJSONRequestBody{
		Enabled: job.Enabled,
	}

	resp, err := client.ScheduledRebalancingAPIUpdateRebalancingJobWithResponse(ctx, *job.ClusterId, *job.Id, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}
	return resourceRebalancingJobRead(ctx, d, meta)
}

func resourceRebalancingJobDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	clusterID := d.Get("cluster_id").(string)

	resp, err := client.ScheduledRebalancingAPIDeleteRebalancingJobWithResponse(ctx, clusterID, d.Id())
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}
	return nil
}

func rebalancingJobStateImporter(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	client := meta.(*ProviderConfig).api

	// if importing by UUID, nothing to do; if importing by name, fetch job ID and set that as resource ID
	if _, err := uuid.Parse(d.Id()); err != nil {
		tflog.Info(ctx, "provided job ID is not a UUID, will import by cluster ID/schedule ID combination")
		ids := strings.Split(d.Id(), "/")
		if len(ids) != 2 {
			return nil, fmt.Errorf("expected ID format to be 'clusterID/scheduleID'")
		}
		job, err := getRebalancingJobByScheduleID(ctx, client, ids[0], ids[1])
		if err != nil {
			return nil, err
		}
		d.SetId(lo.FromPtr(job.Id))
	}

	return []*schema.ResourceData{d}, nil
}

func stateToRebalancingJob(d *schema.ResourceData) (*sdk.ScheduledrebalancingV1RebalancingJob, error) {
	result := sdk.ScheduledrebalancingV1RebalancingJob{
		Id:                    lo.ToPtr(d.Id()),
		Enabled:               lo.ToPtr(d.Get("enabled").(bool)),
		ClusterId:             lo.ToPtr(d.Get("cluster_id").(string)),
		RebalancingScheduleId: lo.ToPtr(d.Get("rebalancing_schedule_id").(string)),
	}

	return &result, nil
}

func rebalancingJobToState(job *sdk.ScheduledrebalancingV1RebalancingJob, d *schema.ResourceData) error {
	d.SetId(*job.Id)
	if err := d.Set("schedule_id", job.RebalancingScheduleId); err != nil {
		return err
	}
	if err := d.Set("cluster_id", job.ClusterId); err != nil {
		return err
	}
	if err := d.Set("enabled", job.Enabled); err != nil {
		return err
	}

	return nil
}

func getRebalancingJobByScheduleID(ctx context.Context, client *sdk.ClientWithResponses, clusterID string, scheduleID string) (*sdk.ScheduledrebalancingV1RebalancingJob, error) {
	resp, err := client.ScheduledRebalancingAPIListRebalancingJobsWithResponse(ctx, clusterID)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return nil, checkErr
	}

	for _, job := range *resp.JSON200.Jobs {
		if *job.RebalancingScheduleId == scheduleID {
			return &job, nil
		}
	}

	return nil, fmt.Errorf("rebalancing job for schedule %q was not found", scheduleID)
}

func getRebalancingJobById(ctx context.Context, client *sdk.ClientWithResponses, clusterID string, id string) (*sdk.ScheduledrebalancingV1RebalancingJob, error) {
	resp, err := client.ScheduledRebalancingAPIGetRebalancingJobWithResponse(ctx, clusterID, id)
	if err != nil {
		return nil, err
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
