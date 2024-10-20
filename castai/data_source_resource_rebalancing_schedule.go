package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRebalancingSchedule() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieve Rebalancing Schedule ",
		ReadContext: dataSourceRebalancingScheduleRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the Rebalancing schedule",
			},
			"schedule": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cron": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cron expression defining when the schedule should trigger.",
						},
					},
				},
			},
			"trigger_conditions": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"savings_percentage": {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "Defines the minimum percentage of savings expected.",
						},
						"ignore_savings": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "If true, the savings percentage will be ignored and the rebalancing will be triggered regardless of the savings percentage.",
						},
					},
				},
			},
			"launch_configuration": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"node_ttl_seconds": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Specifies amount of time since node creation before the node is allowed to be considered for automated rebalancing.",
						},
						"num_targeted_nodes": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Maximum number of nodes that will be selected for rebalancing.",
						},
						"rebalancing_min_nodes": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Minimum number of nodes that should be kept in the cluster after rebalancing.",
						},
						"keep_drain_timeout_nodes": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Defines whether the nodes that failed to get drained until a predefined timeout, will be kept with a rebalancing.cast.ai/status=drain-failed annotation instead of forcefully drained.",
						},
						"aggressive_mode": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "When enabled rebalancing will also consider problematic pods (pods without controller, job pods, pods with removal-disabled annotation) as not-problematic.",
						},
						"execution_conditions": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Enables or disables the execution conditions.",
									},
									"achieved_savings_percentage": {
										Type:     schema.TypeInt,
										Computed: true,
										Description: "The percentage of the predicted savings that must be achieved in order to fully execute the plan." +
											"If the savings are not achieved after creating the new nodes, the plan will fail and delete the created nodes.",
									},
								},
							},
						},
						"selector": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Node selector in JSON format.",
						},
						"target_node_selection_algorithm": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Defines the algorithm used to select the target nodes for rebalancing.",
						},
					},
				},
			},
		},
	}
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
