package castai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler"
)

const (
	FieldHibernationScheduleOrganizationID     = "organization_id"
	FieldHibernationScheduleName               = "name"
	FieldHibernationScheduleEnabled            = "enabled"
	FieldHibernationSchedulePauseConfig        = "pause_config"
	FieldHibernationScheduleResumeConfig       = "resume_config"
	FieldHibernationScheduleClusterAssignments = "cluster_assignments"
	FieldHibernationScheduleSchedule           = "schedule"
	FieldHibernationScheduleJobConfig          = "job_config"
	FieldHibernationScheduleNodeConfig         = "node_config"
	FieldHibernationScheduleInstanceType       = "instance_type"
	FieldHibernationScheduleConfigId           = "config_id"
	FieldHibernationScheduleConfigName         = "config_name"
	FieldHibernationScheduleGpuConfig          = "gpu_config"
	FieldHibernationScheduleKubernetesLabels   = "kubernetes_labels"
	FieldHibernationScheduleKubernetesTaints   = "kubernetes_taints"
	FieldHibernationScheduleNodeAffinity       = "node_affinity"
	FieldHibernationScheduleSpotConfig         = "spot_config"
	FieldHibernationScheduleSubnetId           = "subnet_id"
	FieldHibernationScheduleVolume             = "volume"
	FieldHibernationScheduleZone               = "zone"
	FieldHibernationScheduleCount              = "count"
	FieldHibernationScheduleType               = "type"
	FieldHibernationScheduleKey                = "key"
	FieldHibernationScheduleValue              = "value"
	FieldHibernationScheduleEffect             = "effect"
	FieldHibernationScheduleDedicatedGroup     = "dedicated_group"
	FieldHibernationScheduleAffinity           = "affinity"
	FieldHibernationScheduleOperator           = "operator"
	FieldHibernationScheduleValues             = "values"
	FieldHibernationSchedulePriceHourly        = "price_hourly"
	FieldHibernationScheduleSpot               = "spot"
	FieldHibernationScheduleRaidConfig         = "raid_config"
	FieldHibernationScheduleChunkSizeKb        = "chunk_size_kb"
	FieldHibernationScheduleSizeGib            = "size_gib"
	FieldHibernationScheduleAssignment         = "assignment"
	FieldHibernationScheduleClusterID          = "cluster_id"
	FieldHibernationScheduleCronExpression     = "cron_expression"
)

var supportedAffinityOperators = []string{
	string(cluster_autoscaler.DOESNOTEXIST),
	string(cluster_autoscaler.EXISTS),
	string(cluster_autoscaler.GT),
	string(cluster_autoscaler.IN),
	string(cluster_autoscaler.LT),
	string(cluster_autoscaler.NOTIN),
}
var scheduleSchema = &schema.Schema{
	Type:     schema.TypeList,
	Required: true,
	MaxItems: 1,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			FieldHibernationScheduleCronExpression: {
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
}

func resourceHibernationSchedule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceHibernationScheduleCreate,
		ReadContext:   resourceHibernationScheduleRead,
		DeleteContext: resourceHibernationScheduleDelete,
		UpdateContext: resourceHibernationScheduleUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: hibernationScheduleStateImporter,
		},
		Description: "CAST AI hibernation schedule resource to manage hibernation schedules.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldHibernationScheduleOrganizationID: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "ID of the organization. If not provided, then will attempt to infer it using CAST AI API client.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldHibernationScheduleName: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "Name of the schedule.",
			},
			FieldHibernationScheduleEnabled: {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Enables or disables the schedule.",
			},
			FieldHibernationSchedulePauseConfig: {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldHibernationScheduleEnabled: {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Enables or disables the pause configuration.",
						},
						FieldHibernationScheduleSchedule: scheduleSchema,
					},
				},
			},
			FieldHibernationScheduleResumeConfig: {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldHibernationScheduleEnabled: {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Enables or disables the pause configuration.",
						},
						FieldHibernationScheduleSchedule: scheduleSchema,
						FieldHibernationScheduleJobConfig: {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldHibernationScheduleNodeConfig: {
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldHibernationScheduleInstanceType: {
													Type:             schema.TypeString,
													Required:         true,
													ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
													Description:      "Instance type.",
												},
												FieldHibernationScheduleConfigId: {
													Type:             schema.TypeString,
													Optional:         true,
													ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
													Description:      "ID reference of Node Configuration to be used for node creation. Supersedes 'config_name' parameter.",
												},
												FieldHibernationScheduleConfigName: {
													Type:             schema.TypeString,
													Optional:         true,
													ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
													Description:      "Name reference of Node Configuration to be used for node creation. Superseded if 'config_id' parameter is provided.",
												},
												FieldHibernationScheduleGpuConfig: {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															FieldHibernationScheduleCount: {
																Type:        schema.TypeInt,
																Required:    true,
																Description: "Number of GPUs.",
															},
															FieldHibernationScheduleType: {
																Type:             schema.TypeString,
																Optional:         true,
																ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
																Description:      "GPU type.",
															},
														},
													},
												},
												FieldHibernationScheduleKubernetesLabels: {
													Type:     schema.TypeMap,
													Optional: true,
													Elem: &schema.Schema{
														Type: schema.TypeString,
													},
													Description: "Custom labels to be added to the node.",
												},
												FieldHibernationScheduleKubernetesTaints: {
													Type:     schema.TypeList,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															FieldHibernationScheduleKey: {
																Required:         true,
																Type:             schema.TypeString,
																ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
																Description:      "Key of a taint to be added to nodes created from this template.",
															},
															FieldHibernationScheduleValue: {
																Optional:    true,
																Type:        schema.TypeString,
																Description: "Value of a taint to be added to nodes created from this template.",
															},
															FieldHibernationScheduleEffect: {
																Optional: true,
																Type:     schema.TypeString,
																Default:  TaintEffectNoSchedule,
																ValidateDiagFunc: validation.ToDiagFunc(
																	validation.StringInSlice([]string{TaintEffectNoSchedule, TaintEffectNoExecute}, false),
																),
																Description: fmt.Sprintf("Effect of a taint to be added to nodes created from this template, the default is %s. Allowed values: %s.", TaintEffectNoSchedule, strings.Join([]string{TaintEffectNoSchedule, TaintEffectNoExecute}, ", ")),
															},
														},
													},
													Description: "Custom taints to be added to the node created from this configuration.",
												},
												FieldHibernationScheduleNodeAffinity: {
													Type:     schema.TypeList,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															FieldHibernationScheduleDedicatedGroup: {
																Required:         true,
																Type:             schema.TypeString,
																ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
																Description:      "Key of a taint to be added to nodes created from this template.",
															},
															FieldHibernationScheduleAffinity: {
																Optional: true,
																Type:     schema.TypeList,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		FieldHibernationScheduleKey: {
																			Required:    true,
																			Type:        schema.TypeString,
																			Description: "Key of the node affinity selector.",
																		},
																		FieldHibernationScheduleOperator: {
																			Required:         true,
																			Type:             schema.TypeString,
																			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(supportedAffinityOperators, false)),
																			Description:      fmt.Sprintf("Operator of the node affinity selector. Allowed values: %s.", strings.Join(supportedAffinityOperators, ", ")),
																		},
																		FieldHibernationScheduleValues: {
																			Required: true,
																			Type:     schema.TypeList,
																			Elem: &schema.Schema{
																				Type: schema.TypeString,
																			},
																			Description: "Values of the node affinity selector.",
																		},
																	},
																},
															},
														},
													},
													Description: "Custom taints to be added to the node created from this configuration.",
												},
												FieldHibernationScheduleSpotConfig: {
													Type:     schema.TypeList,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															FieldHibernationSchedulePriceHourly: {
																Optional:         true,
																Type:             schema.TypeString,
																ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
																Description:      "Spot instance price. Applicable only for AWS nodes.",
															},
															FieldHibernationScheduleSpot: {
																Type:        schema.TypeBool,
																Optional:    true,
																Description: "Whether node should be created as spot instance.",
															},
														},
													},
													Description: "Custom taints to be added to the node created from this configuration.",
												},
												FieldHibernationScheduleSubnetId: {
													Type:             schema.TypeString,
													Optional:         true,
													ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
													Description:      "Node subnet ID.",
												},
												FieldHibernationScheduleVolume: {
													Type:     schema.TypeList,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															FieldHibernationScheduleRaidConfig: {
																Type:     schema.TypeList,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		FieldHibernationScheduleChunkSizeKb: {
																			Type:        schema.TypeInt,
																			Optional:    true,
																			Description: "Specify the RAID0 chunk size in kilobytes, this parameter affects the read/write in the disk array and must be tailored for the type of data written by the workloads in the node. If not provided it will default to 64KB",
																		},
																	},
																},
															},
															FieldHibernationScheduleSizeGib: {
																Type:        schema.TypeInt,
																Optional:    true,
																Description: "Volume size in GiB.",
															},
														},
													},
												},
												FieldHibernationScheduleZone: {
													Type:             schema.TypeString,
													Optional:         true,
													ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
													Description:      "Zone of the node.",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			FieldHibernationScheduleClusterAssignments: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldHibernationScheduleAssignment: {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldHibernationScheduleClusterID: {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
										Description:      "ID of the cluster.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func hibernationScheduleStateImporter(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	organizationID, id := parseImportID(d)
	if organizationID == "" {
		stateOrganizationID, err := getHibernationScheduleOrganizationID(ctx, d, meta)
		if err != nil {
			return nil, err
		}

		organizationID = stateOrganizationID
	}

	// if importing by UUID, nothing to do; if importing by name, fetch schedule ID and set that as resource ID
	if _, err := uuid.Parse(id); err != nil {
		tflog.Info(ctx, "provided schedule ID is not a UUID, will import by name")
		schedule, err := getHibernationScheduleByName(ctx, meta, organizationID, id)
		if err != nil {
			return nil, err
		} else if schedule == nil {
			return nil, fmt.Errorf("could not find schedule by name: %s", id)
		}

		d.SetId(lo.FromPtr(schedule.Id))
		if err = d.Set(FieldHibernationScheduleOrganizationID, lo.FromPtr(schedule.OrganizationId)); err != nil {
			return nil, err
		}
	}

	return []*schema.ResourceData{d}, nil
}

func resourceHibernationScheduleUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).clusterAutoscalerClient

	organizationID, err := getHibernationScheduleOrganizationID(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	schedule, err := stateToHibernationSchedule(d)
	if err != nil {
		return diag.FromErr(err)
	}

	req := cluster_autoscaler.HibernationSchedulesAPIUpdateHibernationScheduleJSONRequestBody{
		Name:               lo.ToPtr(schedule.Name),
		Enabled:            lo.ToPtr(schedule.Enabled),
		ResumeConfig:       lo.ToPtr(schedule.ResumeConfig),
		PauseConfig:        lo.ToPtr(schedule.PauseConfig),
		ClusterAssignments: lo.ToPtr(schedule.ClusterAssignments),
	}

	resp, err := client.HibernationSchedulesAPIUpdateHibernationScheduleWithResponse(ctx, organizationID, d.Id(), req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(fmt.Errorf("could not update hibernation schedule in organization %s: %v", organizationID, checkErr))
	}

	return readHibernationScheduleIntoState(ctx, d, meta, organizationID, d.Id())
}

func resourceHibernationScheduleDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).clusterAutoscalerClient

	organizationID, err := getHibernationScheduleOrganizationID(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := client.HibernationSchedulesAPIDeleteHibernationScheduleWithResponse(ctx, organizationID, d.Id())
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}
	return nil
}

func resourceHibernationScheduleCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).clusterAutoscalerClient

	organizationID, err := getHibernationScheduleOrganizationID(ctx, d, meta)
	if err != nil {
		return diag.FromErr(fmt.Errorf("could not determine organization id: %v", err))
	}

	schedule, err := stateToHibernationSchedule(d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("could not map state to hibernation schedule: %v", err))
	}

	resp, err := client.HibernationSchedulesAPICreateHibernationScheduleWithResponse(ctx, organizationID, *schedule)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(fmt.Errorf("could not create hibernation schedule in organization %s: %v", organizationID, checkErr))
	}

	d.SetId(*resp.JSON200.Id)

	return readHibernationScheduleIntoState(ctx, d, meta, organizationID, d.Id())
}

func resourceHibernationScheduleRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	organizationID, err := getHibernationScheduleOrganizationID(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	return readHibernationScheduleIntoState(ctx, d, meta, organizationID, d.Id())
}

func readHibernationScheduleIntoState(ctx context.Context, d *schema.ResourceData, meta any, organizationID, id string) diag.Diagnostics {
	schedule, err := getHibernationScheduleById(ctx, meta, organizationID, id)
	if err != nil {
		return diag.FromErr(fmt.Errorf("could not retrieve hibernation schedule by id in organization %s: %v", organizationID, err))
	}
	if !d.IsNewResource() && schedule == nil {
		tflog.Warn(ctx, "Hibernation schedule not found, removing from state", map[string]any{"id": d.Id()})
		d.SetId("")
		return nil
	}

	if err := hibernationScheduleToState(schedule, d); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func hibernationScheduleToState(schedule *cluster_autoscaler.HibernationSchedule, d *schema.ResourceData) error {
	d.SetId(*schedule.Id)
	if err := d.Set(FieldHibernationScheduleName, schedule.Name); err != nil {
		return err
	}
	if err := d.Set(FieldHibernationScheduleOrganizationID, schedule.OrganizationId); err != nil {
		return err
	}
	if err := d.Set(FieldHibernationScheduleEnabled, schedule.Enabled); err != nil {
		return err
	}

	clusterAssignments := []map[string]any{
		{
			FieldHibernationScheduleAssignment: lo.Map(schedule.ClusterAssignments.Items, func(item cluster_autoscaler.ClusterAssignment, _ int) map[string]any {
				return map[string]any{
					FieldHibernationScheduleClusterID: item.ClusterId,
				}
			}),
		},
	}

	if err := d.Set(FieldHibernationScheduleClusterAssignments, clusterAssignments); err != nil {
		return err
	}

	pauseConfig := []map[string]any{
		{
			FieldHibernationScheduleEnabled: schedule.PauseConfig.Enabled,
			FieldHibernationScheduleSchedule: []map[string]any{
				{
					FieldHibernationScheduleCronExpression: schedule.PauseConfig.Schedule.CronExpression,
				},
			},
		},
	}

	if err := d.Set(FieldHibernationSchedulePauseConfig, pauseConfig); err != nil {
		return err
	}

	nodeConfig := schedule.ResumeConfig.JobConfig.NodeConfig

	var gpuConfig []map[string]any
	if nodeConfig.GpuConfig != nil {
		gpuConfig = []map[string]any{
			{
				FieldHibernationScheduleCount: nodeConfig.GpuConfig.Count,
				FieldHibernationScheduleType:  nodeConfig.GpuConfig.Type,
			},
		}
	}

	var kubernetesLabels map[string]string
	if nodeConfig.KubernetesLabels != nil && len(*nodeConfig.KubernetesLabels) != 0 {
		kubernetesLabels = make(map[string]string, len(*nodeConfig.KubernetesLabels))
		for key, value := range *nodeConfig.KubernetesLabels {
			kubernetesLabels[key] = value
		}
	}

	var kubernetesTaints []map[string]any
	if nodeConfig.KubernetesTaints != nil && len(*nodeConfig.KubernetesTaints) != 0 {
		kubernetesTaints = make([]map[string]any, 0, len(*nodeConfig.KubernetesTaints))
		for _, taint := range *nodeConfig.KubernetesTaints {
			kubernetesTaints = append(kubernetesTaints, map[string]any{
				FieldHibernationScheduleKey:    taint.Key,
				FieldHibernationScheduleValue:  taint.Value,
				FieldHibernationScheduleEffect: taint.Effect,
			})
		}
	}

	var nodeAffinity []map[string]any
	if nodeConfig.NodeAffinity != nil {
		var affinities []map[string]any
		if nodeConfig.NodeAffinity.Affinity != nil && len(*nodeConfig.NodeAffinity.Affinity) != 0 {
			affinities = make([]map[string]any, 0, len(*nodeConfig.NodeAffinity.Affinity))
			for _, affinity := range *nodeConfig.NodeAffinity.Affinity {
				affinities = append(affinities, map[string]any{
					FieldHibernationScheduleKey:      affinity.Key,
					FieldHibernationScheduleOperator: affinity.Operator,
					FieldHibernationScheduleValues:   affinity.Values,
				})
			}
		}

		nodeAffinity = []map[string]any{
			{
				FieldHibernationScheduleDedicatedGroup: nodeConfig.NodeAffinity.DedicatedGroup,
				FieldHibernationScheduleAffinity:       affinities,
			},
		}
	}

	var spotConfig []map[string]any
	if nodeConfig.SpotConfig != nil {
		spotConfig = []map[string]any{
			{
				FieldHibernationSchedulePriceHourly: nodeConfig.SpotConfig.PriceHourly,
				FieldHibernationScheduleSpot:        nodeConfig.SpotConfig.Spot,
			},
		}
	}

	var volume []map[string]any
	if nodeConfig.Volume != nil {
		var raidConfig []map[string]any
		if nodeConfig.Volume.RaidConfig != nil {
			raidConfig = []map[string]any{
				{
					FieldHibernationScheduleChunkSizeKb: nodeConfig.Volume.RaidConfig.ChunkSizeKb,
				},
			}
		}

		volume = []map[string]any{
			{
				FieldHibernationScheduleSizeGib:    nodeConfig.Volume.SizeGib,
				FieldHibernationScheduleRaidConfig: raidConfig,
			},
		}
	}

	resumeConfig := []map[string]any{
		{
			FieldHibernationScheduleEnabled: schedule.ResumeConfig.Enabled,
			FieldHibernationScheduleSchedule: []map[string]any{
				{
					FieldHibernationScheduleCronExpression: schedule.ResumeConfig.Schedule.CronExpression,
				},
			},
			FieldHibernationScheduleJobConfig: []map[string]any{
				{
					FieldHibernationScheduleNodeConfig: []map[string]any{
						{
							FieldHibernationScheduleInstanceType:     nodeConfig.InstanceType,
							FieldHibernationScheduleConfigId:         nodeConfig.ConfigId,
							FieldHibernationScheduleConfigName:       nodeConfig.ConfigName,
							FieldHibernationScheduleGpuConfig:        gpuConfig,
							FieldHibernationScheduleKubernetesLabels: kubernetesLabels,
							FieldHibernationScheduleKubernetesTaints: kubernetesTaints,
							FieldHibernationScheduleNodeAffinity:     nodeAffinity,
							FieldHibernationScheduleSpotConfig:       spotConfig,
							FieldHibernationScheduleSubnetId:         nodeConfig.SubnetId,
							FieldHibernationScheduleZone:             nodeConfig.Zone,
							FieldHibernationScheduleVolume:           volume,
						},
					},
				},
			},
		},
	}

	if err := d.Set(FieldHibernationScheduleResumeConfig, resumeConfig); err != nil {
		return err
	}

	return nil
}

func sectionToHibernationScheduleConfig(section map[string]interface{}) cluster_autoscaler.Schedule {
	return cluster_autoscaler.Schedule{
		CronExpression: section[FieldHibernationScheduleCronExpression].(string),
	}
}

func nodeConfigSectionToGPUConfig(section map[string]interface{}) *cluster_autoscaler.GPUConfig {
	gpuConfigEntry, ok := section[FieldHibernationScheduleGpuConfig]
	if !ok {
		return nil
	}

	gpuConfigEntryList := gpuConfigEntry.([]interface{})
	if len(gpuConfigEntryList) == 0 {
		return nil
	}

	gpuConfigMap := gpuConfigEntryList[0].(map[string]interface{})
	if len(gpuConfigMap) == 0 {
		return nil
	}

	gpuCount := int32(readOptionalValueOrDefault[int](gpuConfigMap, FieldHibernationScheduleCount, 0))
	gpuType := readOptionalValueOrDefault[string](gpuConfigMap, FieldHibernationScheduleType, "")

	return &cluster_autoscaler.GPUConfig{
		Count: lo.Ternary(gpuCount != 0, &gpuCount, nil),
		Type:  lo.Ternary(gpuType != "", &gpuType, nil),
	}
}

func nodeConfigSectionToNodeAffinity(section map[string]interface{}) *cluster_autoscaler.NodeAffinity {
	nodeAffinityEntry, ok := section[FieldHibernationScheduleNodeAffinity]
	if !ok {
		return nil
	}

	nodeAffinityEntryList := nodeAffinityEntry.([]interface{})
	if len(nodeAffinityEntryList) == 0 {
		return nil
	}

	nodeAffinityMap := nodeAffinityEntryList[0].(map[string]interface{})
	if len(nodeAffinityMap) == 0 {
		return nil
	}

	nodeAffinity := &cluster_autoscaler.NodeAffinity{
		DedicatedGroup: readOptionalValue[string](nodeAffinityMap, FieldHibernationScheduleDedicatedGroup),
	}

	affinityEntry, ok := nodeAffinityMap[FieldHibernationScheduleAffinity]
	if !ok {
		return nil
	}

	affinityList := affinityEntry.([]interface{})
	if len(affinityList) == 0 {
		return nil
	}

	affinities := make([]cluster_autoscaler.KubernetesNodeAffinity, 0, len(affinityList))

	for _, affinityListEntry := range affinityList {
		affinityMap := affinityListEntry.(map[string]interface{})
		affinities = append(affinities, cluster_autoscaler.KubernetesNodeAffinity{
			Key:      affinityMap[FieldHibernationScheduleKey].(string),
			Operator: cluster_autoscaler.KubernetesNodeAffinityOperator(affinityMap[FieldHibernationScheduleOperator].(string)),
			Values:   toStringList(affinityMap[FieldHibernationScheduleValues].([]interface{})),
		})
	}

	nodeAffinity.Affinity = &affinities
	return nodeAffinity
}

func nodeConfigSectionToKubernetesTaints(section map[string]interface{}) *[]cluster_autoscaler.Taint {
	kubernetesTaintsEntry, ok := section[FieldHibernationScheduleKubernetesTaints]
	if !ok {
		return nil
	}

	kubernetesTaintsList := kubernetesTaintsEntry.([]interface{})

	if len(kubernetesTaintsList) == 0 {
		return nil
	}

	taints := make([]cluster_autoscaler.Taint, 0, len(kubernetesTaintsList))

	for _, taintListEntry := range kubernetesTaintsList {
		taintMap := taintListEntry.(map[string]interface{})

		taints = append(taints, cluster_autoscaler.Taint{
			Key:    taintMap[FieldHibernationScheduleKey].(string),
			Value:  taintMap[FieldHibernationScheduleValue].(string),
			Effect: taintMap[FieldHibernationScheduleEffect].(string),
		})
	}

	return &taints
}

func nodeConfigSectionToSpotConfig(section map[string]interface{}) *cluster_autoscaler.NodeSpotConfig {
	spotConfigEntry, ok := section[FieldHibernationScheduleSpotConfig]
	if !ok {
		return nil
	}

	spotConfigEntryList := spotConfigEntry.([]interface{})
	if len(spotConfigEntryList) == 0 {
		return nil
	}

	spotConfigMap := spotConfigEntryList[0].(map[string]interface{})
	if len(spotConfigMap) == 0 {
		return nil
	}

	priceHourly := readOptionalValueOrDefault[string](spotConfigMap, FieldHibernationSchedulePriceHourly, "")
	spot := readOptionalValue[bool](spotConfigMap, FieldHibernationScheduleSpot)

	return &cluster_autoscaler.NodeSpotConfig{
		PriceHourly: lo.Ternary(priceHourly != "", &priceHourly, nil),
		Spot:        spot,
	}
}

func nodeConfigSectionToVolume(section map[string]interface{}) *cluster_autoscaler.NodeVolume {
	volumeEntry, ok := section[FieldHibernationScheduleVolume]
	if !ok {
		return nil
	}

	volumeEntryList := volumeEntry.([]interface{})
	if len(volumeEntryList) == 0 {
		return nil
	}

	volumeMap := volumeEntryList[0].(map[string]interface{})
	if len(volumeMap) == 0 {
		return nil
	}

	sizeGib := int32(readOptionalValueOrDefault[int](volumeMap, FieldHibernationScheduleSizeGib, 0))

	volume := &cluster_autoscaler.NodeVolume{
		SizeGib: lo.Ternary(sizeGib != 0, &sizeGib, nil),
	}

	if raidConfigEntry, ok := volumeMap[FieldHibernationScheduleRaidConfig]; ok {
		raidConfigEntryList := raidConfigEntry.([]interface{})
		if len(raidConfigEntryList) == 0 {
			return nil
		}

		raidConfigMap := raidConfigEntryList[0].(map[string]interface{})
		if len(raidConfigMap) == 0 {
			return nil
		}

		if len(raidConfigMap) != 0 {
			chunkSizeKb := int32(readOptionalValueOrDefault[int](raidConfigMap, FieldHibernationScheduleChunkSizeKb, 0))

			volume.RaidConfig = &cluster_autoscaler.RaidConfig{
				ChunkSizeKb: lo.Ternary(chunkSizeKb != 0, &chunkSizeKb, nil),
			}
		}
	}

	return volume
}

func sectionToNodeConfig(section map[string]interface{}) cluster_autoscaler.NodeConfig {
	configId := readOptionalValueOrDefault[string](section, FieldHibernationScheduleConfigId, "")
	configName := readOptionalValueOrDefault[string](section, FieldHibernationScheduleConfigName, "")
	subnetId := readOptionalValueOrDefault[string](section, FieldHibernationScheduleSubnetId, "")
	zone := readOptionalValueOrDefault[string](section, FieldHibernationScheduleZone, "")
	kubernetesLabels := toStringMap(section[FieldHibernationScheduleKubernetesLabels].(map[string]interface{}))

	nodeConfig := cluster_autoscaler.NodeConfig{
		InstanceType:     section[FieldHibernationScheduleInstanceType].(string),
		ConfigId:         lo.Ternary(configId != "", &configId, nil),
		ConfigName:       lo.Ternary(configName != "", &configName, nil),
		SubnetId:         lo.Ternary(subnetId != "", &subnetId, nil),
		Zone:             lo.Ternary(zone != "", &zone, nil),
		KubernetesLabels: lo.Ternary(len(kubernetesLabels) != 0, &kubernetesLabels, nil),
		GpuConfig:        nodeConfigSectionToGPUConfig(section),
		NodeAffinity:     nodeConfigSectionToNodeAffinity(section),
		KubernetesTaints: nodeConfigSectionToKubernetesTaints(section),
		SpotConfig:       nodeConfigSectionToSpotConfig(section),
		Volume:           nodeConfigSectionToVolume(section),
	}

	return nodeConfig
}

func sectionToClusterAssignments(section map[string]interface{}) cluster_autoscaler.ClusterAssignments {
	clusterAssignments := cluster_autoscaler.ClusterAssignments{
		Items: []cluster_autoscaler.ClusterAssignment{},
	}

	if items, ok := section[FieldHibernationScheduleAssignment]; ok {
		for _, item := range items.([]interface{}) {
			itemMap := item.(map[string]interface{})
			clusterAssignments.Items = append(clusterAssignments.Items, cluster_autoscaler.ClusterAssignment{
				ClusterId: itemMap[FieldHibernationScheduleClusterID].(string),
			})
		}
	}

	return clusterAssignments
}

func stateToHibernationSchedule(d *schema.ResourceData) (*cluster_autoscaler.HibernationSchedule, error) {
	pauseConfigData := toSection(d, FieldHibernationSchedulePauseConfig)
	pauseConfigScheduleData := toNestedSection(d, FieldHibernationSchedulePauseConfig, "0", FieldHibernationScheduleSchedule)

	resumeConfigData := toSection(d, FieldHibernationScheduleResumeConfig)
	resumeConfigScheduleData := toNestedSection(d, FieldHibernationScheduleResumeConfig, "0", FieldHibernationScheduleSchedule)
	resumeConfigNodeConfig := toNestedSection(d, FieldHibernationScheduleResumeConfig, "0", FieldHibernationScheduleJobConfig, "0", FieldHibernationScheduleNodeConfig)

	clusterAssignmentsData := toSection(d, FieldHibernationScheduleClusterAssignments)

	result := cluster_autoscaler.HibernationSchedule{
		Id:      lo.ToPtr(d.Id()),
		Name:    d.Get(FieldHibernationScheduleName).(string),
		Enabled: d.Get(FieldHibernationScheduleEnabled).(bool),
		PauseConfig: cluster_autoscaler.PauseConfig{
			Enabled:  pauseConfigData[FieldHibernationScheduleEnabled].(bool),
			Schedule: sectionToHibernationScheduleConfig(pauseConfigScheduleData),
		},
		ResumeConfig: cluster_autoscaler.ResumeConfig{
			Enabled:  resumeConfigData[FieldHibernationScheduleEnabled].(bool),
			Schedule: sectionToHibernationScheduleConfig(resumeConfigScheduleData),
			JobConfig: cluster_autoscaler.ResumeJobConfig{
				NodeConfig: sectionToNodeConfig(resumeConfigNodeConfig),
			},
		},
		ClusterAssignments: sectionToClusterAssignments(clusterAssignmentsData),
	}

	return &result, nil
}

func getHibernationScheduleOrganizationID(ctx context.Context, data *schema.ResourceData, meta interface{}) (string, error) {
	var organizationID string
	var err error

	organizationID = data.Get(FieldHibernationScheduleOrganizationID).(string)
	if organizationID == "" {
		organizationID, err = getDefaultOrganizationId(ctx, meta)
		if err != nil {
			return "", fmt.Errorf("getting organization ID: %w", err)
		}
	}

	return organizationID, nil
}

func getHibernationScheduleById(ctx context.Context, meta interface{}, organizationID, id string) (*cluster_autoscaler.HibernationSchedule, error) {
	client := meta.(*ProviderConfig).clusterAutoscalerClient

	resp, err := client.HibernationSchedulesAPIGetHibernationScheduleWithResponse(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

func getHibernationScheduleByName(ctx context.Context, meta interface{}, organizationID, name string) (*cluster_autoscaler.HibernationSchedule, error) {
	client := meta.(*ProviderConfig).clusterAutoscalerClient

	params := &cluster_autoscaler.HibernationSchedulesAPIListHibernationSchedulesParams{
		PageLimit: lo.ToPtr("500"),
	}
	resp, err := client.HibernationSchedulesAPIListHibernationSchedulesWithResponse(ctx, organizationID, params)
	if err != nil {
		return nil, err
	}

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return nil, err
	}

	for _, item := range resp.JSON200.Items {
		if item.Name == name {
			return &item, nil
		}
	}

	return nil, nil
}

func parseImportID(d *schema.ResourceData) (string, string) {
	id := d.Id()

	if strings.Contains(id, "/") {
		if parts := strings.Split(id, "/"); len(parts) > 1 {
			return parts[0], parts[1]
		}
	}

	return "", id
}
