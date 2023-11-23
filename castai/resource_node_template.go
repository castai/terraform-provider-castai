package castai

import (
	"context"
	"fmt"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"
	"log"
	"strings"
	"time"
)

const (
	FieldNodeTemplateArchitectures                          = "architectures"
	FieldNodeTemplateComputeOptimized                       = "compute_optimized"
	FieldNodeTemplateConfigurationId                        = "configuration_id"
	FieldNodeTemplateConstraints                            = "constraints"
	FieldNodeTemplateCustomInstancesEnabled                 = "custom_instances_enabled"
	FieldNodeTemplateCustomLabels                           = "custom_labels"
	FieldNodeTemplateCustomTaints                           = "custom_taints"
	FieldNodeTemplateEnableSpotDiversity                    = "enable_spot_diversity"
	FieldNodeTemplateExclude                                = "exclude"
	FieldNodeTemplateExcludeNames                           = "exclude_names"
	FieldNodeTemplateFallbackRestoreRateSeconds             = "fallback_restore_rate_seconds"
	FieldNodeTemplateGpu                                    = "gpu"
	FieldNodeTemplateInclude                                = "include"
	FieldNodeTemplateIncludeNames                           = "include_names"
	FieldNodeTemplateInstanceFamilies                       = "instance_families"
	FieldNodeTemplateIsDefault                              = "is_default"
	FieldNodeTemplateIsEnabled                              = "is_enabled"
	FieldNodeTemplateIsGpuOnly                              = "is_gpu_only"
	FieldNodeTemplateManufacturers                          = "manufacturers"
	FieldNodeTemplateMaxCount                               = "max_count"
	FieldNodeTemplateMaxCpu                                 = "max_cpu"
	FieldNodeTemplateMaxMemory                              = "max_memory"
	FieldNodeTemplateMinCount                               = "min_count"
	FieldNodeTemplateMinCpu                                 = "min_cpu"
	FieldNodeTemplateMinMemory                              = "min_memory"
	FieldNodeTemplateName                                   = "name"
	FieldNodeTemplateOnDemand                               = "on_demand"
	FieldNodeTemplateOs                                     = "os"
	FieldNodeTemplateRebalancingConfigMinNodes              = "rebalancing_config_min_nodes"
	FieldNodeTemplateShouldTaint                            = "should_taint"
	FieldNodeTemplateSpot                                   = "spot"
	FieldNodeTemplateSpotDiversityPriceIncreaseLimitPercent = "spot_diversity_price_increase_limit_percent"
	FieldNodeTemplateSpotInterruptionPredictionsEnabled     = "spot_interruption_predictions_enabled"
	FieldNodeTemplateSpotInterruptionPredictionsType        = "spot_interruption_predictions_type"
	FieldNodeTemplateStorageOptimized                       = "storage_optimized"
	FieldNodeTemplateUseSpotFallbacks                       = "use_spot_fallbacks"
)

const (
	TaintEffectNoSchedule = "NoSchedule"
	TaintEffectNoExecute  = "NoExecute"
)

const (
	ArchAMD64 = "amd64"
	ArchARM64 = "arm64"
	OsLinux   = "linux"
	OsWindows = "windows"
)

func resourceNodeTemplate() *schema.Resource {
	supportedArchitectures := []string{ArchAMD64, ArchARM64}
	supportedOs := []string{OsLinux, OsWindows}

	return &schema.Resource{
		CreateContext: resourceNodeTemplateCreate,
		ReadContext:   resourceNodeTemplateRead,
		DeleteContext: resourceNodeTemplateDelete,
		UpdateContext: resourceNodeTemplateUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: nodeTemplateStateImporter,
		},
		Description: "CAST AI node template resource to manage node templates",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldClusterId: {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "CAST AI cluster id.",
			},
			FieldNodeTemplateName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "Name of the node template.",
			},
			FieldNodeTemplateIsEnabled: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Flag whether the node template is enabled and considered for autoscaling.",
			},
			FieldNodeTemplateIsDefault: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag whether the node template is default.",
			},
			FieldNodeTemplateConfigurationId: {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "CAST AI node configuration id to be used for node template.",
			},
			FieldNodeTemplateShouldTaint: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Marks whether the templated nodes will have a taint.",
			},
			FieldNodeTemplateConstraints: {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldNodeTemplateSpot: {
							Type:        schema.TypeBool,
							Default:     false,
							Optional:    true,
							Description: "Should include spot instances in the considered pool.",
						},
						FieldNodeTemplateOnDemand: {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "Should include on-demand instances in the considered pool.",
						},
						FieldNodeTemplateUseSpotFallbacks: {
							Type:        schema.TypeBool,
							Default:     false,
							Optional:    true,
							Description: "Spot instance fallback constraint - when true, on-demand instances will be created, when spots are unavailable.",
						},
						FieldNodeTemplateFallbackRestoreRateSeconds: {
							Type:        schema.TypeInt,
							Default:     0,
							Optional:    true,
							Description: "Fallback restore rate in seconds: defines how much time should pass before spot fallback should be attempted to be restored to real spot.",
						},
						FieldNodeTemplateEnableSpotDiversity: {
							Type:        schema.TypeBool,
							Default:     false,
							Optional:    true,
							Description: "Enable/disable spot diversity policy. When enabled, autoscaler will try to balance between diverse and cost optimal instance types.",
						},
						FieldNodeTemplateSpotDiversityPriceIncreaseLimitPercent: {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Allowed node configuration price increase when diversifying instance types. E.g. if the value is 10%, then the overall price of diversified instance types can be 10% higher than the price of the optimal configuration.",
						},
						FieldNodeTemplateSpotInterruptionPredictionsEnabled: {
							Type:        schema.TypeBool,
							Default:     false,
							Optional:    true,
							Description: "Enable/disable spot interruption predictions.",
						},
						FieldNodeTemplateSpotInterruptionPredictionsType: {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "Spot interruption predictions type. Can be either \"aws-rebalance-recommendations\" or \"interruption-predictions\".",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"aws-rebalance-recommendations", "interruption-predictions"}, false)),
						},
						FieldNodeTemplateMinCpu: {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Min CPU cores per node.",
						},
						FieldNodeTemplateMaxCpu: {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Max CPU cores per node.",
						},
						FieldNodeTemplateMinMemory: {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Min Memory (Mib) per node.",
						},
						FieldNodeTemplateMaxMemory: {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Max Memory (Mib) per node.",
						},
						FieldNodeTemplateStorageOptimized: {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Storage optimized instance constraint - will only pick storage optimized nodes if true",
						},
						FieldNodeTemplateIsGpuOnly: {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "GPU instance constraint - will only pick nodes with GPU if true",
						},
						FieldNodeTemplateComputeOptimized: {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Compute optimized instance constraint - will only pick compute optimized nodes if true.",
						},
						FieldNodeTemplateInstanceFamilies: {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldNodeTemplateInclude: {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
										Description: "Instance families to exclude when filtering (includes all other families).",
									},
									FieldNodeTemplateExclude: {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
										Description: "Instance families to include when filtering (excludes all other families).",
									},
								},
							},
						},
						FieldNodeTemplateGpu: {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldNodeTemplateManufacturers: {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
										Description: "Manufacturers of the gpus to select - NVIDIA, AMD.",
									},
									FieldNodeTemplateIncludeNames: {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
										Description: "Instance families to include when filtering (excludes all other families).",
									},
									FieldNodeTemplateExcludeNames: {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
										Description: "Names of the GPUs to exclude.",
									},
									FieldNodeTemplateMinCount: {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "Min GPU count for the instance type to have.",
									},
									FieldNodeTemplateMaxCount: {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "Max GPU count for the instance type to have.",
									},
								},
							},
						},
						FieldNodeTemplateArchitectures: {
							Type:     schema.TypeList,
							MaxItems: 2,
							MinItems: 1,
							Optional: true,
							Computed: true,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(supportedArchitectures, false)),
							},
							DefaultFunc: func() (interface{}, error) {
								return []string{ArchAMD64}, nil
							},
							Description: fmt.Sprintf("List of acceptable instance CPU architectures, the default is %s. Allowed values: %s.", ArchAMD64, strings.Join(supportedArchitectures, ", ")),
						},
						FieldNodeTemplateOs: {
							Type:     schema.TypeList,
							MaxItems: 2,
							MinItems: 1,
							Optional: true,
							Computed: true,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(supportedOs, false)),
							},
							DefaultFunc: func() (interface{}, error) {
								return []string{OsLinux}, nil
							},
							Description: fmt.Sprintf("List of acceptable instance Operating Systems, the default is %s. Allowed values: %s.", OsLinux, strings.Join(supportedOs, ", ")),
						},
					},
				},
			},
			FieldNodeTemplateCustomLabels: {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Custom labels to be added to nodes created from this template. " +
					"If the field `custom_label` is present, the value of `custom_labels` will be ignored.",
			},
			FieldNodeTemplateCustomTaints: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldKey: {
							Required:         true,
							Type:             schema.TypeString,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
							Description:      "Key of a taint to be added to nodes created from this template.",
						},
						FieldValue: {
							Required:         true,
							Type:             schema.TypeString,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
							Description:      "Value of a taint to be added to nodes created from this template.",
						},
						FieldEffect: {
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
				Description: "Custom taints to be added to the nodes created from this template. " +
					"`shouldTaint` has to be `true` in order to create/update the node template with custom taints. " +
					"If `shouldTaint` is `true`, but no custom taints are provided, the nodes will be tainted with the default node template taint.",
			},
			FieldNodeTemplateRebalancingConfigMinNodes: {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          0,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
				Description:      "Minimum nodes that will be kept when rebalancing nodes using this node template.",
			},
			FieldNodeTemplateCustomInstancesEnabled: {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				Description: "Marks whether custom instances should be used when deciding which parts of inventory are available. " +
					"Custom instances are only supported in GCP.",
			},
		},
	}
}

func resourceNodeTemplateRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	log.Printf("[INFO] List Node Templates get call start")
	defer log.Printf("[INFO] List Node Templates get call end")

	clusterID := getClusterId(d)
	if clusterID == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	nodeTemplate, err := getNodeTemplateByName(ctx, d, meta, clusterID)
	if err != nil {
		return diag.FromErr(err)
	}
	if !d.IsNewResource() && nodeTemplate == nil {
		log.Printf("[WARN] Node template (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err := d.Set(FieldNodeTemplateName, nodeTemplate.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %w", err))
	}
	if err := d.Set(FieldNodeTemplateIsEnabled, nodeTemplate.IsEnabled); err != nil {
		return diag.FromErr(fmt.Errorf("setting is enabled: %w", err))
	}
	if err := d.Set(FieldNodeTemplateIsDefault, nodeTemplate.IsDefault); err != nil {
		return diag.FromErr(fmt.Errorf("setting is default: %w", err))
	}
	if err := d.Set(FieldNodeTemplateConfigurationId, nodeTemplate.ConfigurationId); err != nil {
		return diag.FromErr(fmt.Errorf("setting configuration id: %w", err))
	}
	if err := d.Set(FieldNodeTemplateShouldTaint, nodeTemplate.ShouldTaint); err != nil {
		return diag.FromErr(fmt.Errorf("setting should taint: %w", err))
	}
	if nodeTemplate.RebalancingConfig != nil {
		if err := d.Set(FieldNodeTemplateRebalancingConfigMinNodes, nodeTemplate.RebalancingConfig.MinNodes); err != nil {
			return diag.FromErr(fmt.Errorf("setting configuration id: %w", err))
		}
	}
	if nodeTemplate.Constraints != nil {
		constraints, err := flattenConstraints(nodeTemplate.Constraints)
		if err != nil {
			return diag.FromErr(fmt.Errorf("flattening constraints: %w", err))
		}

		if err := d.Set(FieldNodeTemplateConstraints, constraints); err != nil {
			return diag.FromErr(fmt.Errorf("setting constraints: %w", err))
		}
	}
	if err := d.Set(FieldNodeTemplateCustomLabels, nodeTemplate.CustomLabels.AdditionalProperties); err != nil {
		return diag.FromErr(fmt.Errorf("setting custom labels: %w", err))
	}
	if err := d.Set(FieldNodeTemplateCustomTaints, flattenCustomTaints(nodeTemplate.CustomTaints)); err != nil {
		return diag.FromErr(fmt.Errorf("setting custom taints: %w", err))
	}
	if err := d.Set(FieldNodeTemplateCustomInstancesEnabled, lo.FromPtrOr(nodeTemplate.CustomInstancesEnabled, false)); err != nil {
		return diag.FromErr(fmt.Errorf("setting custom instances enabled: %w", err))
	}

	return nil
}

func flattenConstraints(c *sdk.NodetemplatesV1TemplateConstraints) ([]map[string]any, error) {
	if c == nil {
		return nil, nil
	}

	out := make(map[string]any)
	if c.Gpu != nil {
		out[FieldNodeTemplateGpu] = flattenGpu(c.Gpu)
	}
	if c.InstanceFamilies != nil {
		out[FieldNodeTemplateInstanceFamilies] = flattenInstanceFamilies(c.InstanceFamilies)
	}
	if c.ComputeOptimized != nil {
		out[FieldNodeTemplateComputeOptimized] = c.ComputeOptimized
	}
	if c.StorageOptimized != nil {
		out[FieldNodeTemplateStorageOptimized] = c.StorageOptimized
	}
	if c.Spot != nil {
		out[FieldNodeTemplateSpot] = c.Spot
	}
	if c.OnDemand != nil {
		out[FieldNodeTemplateOnDemand] = c.OnDemand
	}
	if c.IsGpuOnly != nil {
		out[FieldNodeTemplateIsGpuOnly] = c.IsGpuOnly
	}
	if c.UseSpotFallbacks != nil {
		out[FieldNodeTemplateUseSpotFallbacks] = c.UseSpotFallbacks
	}
	if c.FallbackRestoreRateSeconds != nil {
		out[FieldNodeTemplateFallbackRestoreRateSeconds] = c.FallbackRestoreRateSeconds
	}
	if c.EnableSpotDiversity != nil {
		out[FieldNodeTemplateEnableSpotDiversity] = c.EnableSpotDiversity
	}
	if c.SpotDiversityPriceIncreaseLimitPercent != nil {
		out[FieldNodeTemplateSpotDiversityPriceIncreaseLimitPercent] = c.SpotDiversityPriceIncreaseLimitPercent
	}
	if c.SpotInterruptionPredictionsEnabled != nil {
		out[FieldNodeTemplateSpotInterruptionPredictionsEnabled] = c.SpotInterruptionPredictionsEnabled
	}
	if c.SpotInterruptionPredictionsType != nil {
		out[FieldNodeTemplateSpotInterruptionPredictionsType] = c.SpotInterruptionPredictionsType
	}
	if c.MinMemory != nil {
		out[FieldNodeTemplateMinMemory] = c.MinMemory
	}
	if c.MaxMemory != nil {
		out[FieldNodeTemplateMaxMemory] = c.MaxMemory
	}
	if c.MinCpu != nil {
		out[FieldNodeTemplateMinCpu] = c.MinCpu
	}
	if c.MaxCpu != nil {
		out[FieldNodeTemplateMaxCpu] = c.MaxCpu
	}
	if c.Architectures != nil {
		out[FieldNodeTemplateArchitectures] = lo.FromPtr(c.Architectures)
	}
	if c.Os != nil {
		out[FieldNodeTemplateOs] = lo.FromPtr(c.Os)
	}
	return []map[string]any{out}, nil
}

func flattenInstanceFamilies(families *sdk.NodetemplatesV1TemplateConstraintsInstanceFamilyConstraints) []map[string][]string {
	if families == nil {
		return nil
	}
	out := map[string][]string{}
	if families.Exclude != nil {
		out[FieldNodeTemplateExclude] = lo.FromPtr(families.Exclude)
	}
	if families.Include != nil {
		out[FieldNodeTemplateInclude] = lo.FromPtr(families.Include)
	}
	return []map[string][]string{out}
}

func flattenGpu(gpu *sdk.NodetemplatesV1TemplateConstraintsGPUConstraints) []map[string]any {
	if gpu == nil {
		return nil
	}
	out := map[string]any{}
	if gpu.ExcludeNames != nil {
		out[FieldNodeTemplateExcludeNames] = gpu.ExcludeNames
	}
	if gpu.IncludeNames != nil {
		out[FieldNodeTemplateIncludeNames] = gpu.IncludeNames
	}
	if gpu.Manufacturers != nil {
		out[FieldNodeTemplateManufacturers] = gpu.Manufacturers
	}
	if gpu.MinCount != nil {
		out[FieldNodeTemplateMinCount] = gpu.MinCount
	}
	if gpu.MaxCount != nil {
		out[FieldNodeTemplateMaxCount] = gpu.MaxCount
	}
	return []map[string]any{out}
}

func resourceNodeTemplateDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)
	name := d.Get(FieldNodeTemplateName).(string)

	if isDefault, ok := d.Get(FieldNodeTemplateIsDefault).(bool); ok && isDefault {
		return diag.Diagnostics{
			{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Skipping delete of \"%s\" node template", name),
				Detail: "Default node templates cannot be deleted from CAST.ai. If you want to autoscaler to stop " +
					"considering this node template, you can disable it (either from UI or by setting `is_enabled` " +
					"flag to false).",
			},
		}
	}

	resp, err := client.NodeTemplatesAPIDeleteNodeTemplateWithResponse(ctx, clusterID, name)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	return nil
}

func resourceNodeTemplateUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return updateNodeTemplate(ctx, d, meta, false)
}

func updateNodeTemplate(ctx context.Context, d *schema.ResourceData, meta any, skipChangeCheck bool) diag.Diagnostics {
	if !skipChangeCheck && !d.HasChanges(
		FieldNodeTemplateName,
		FieldNodeTemplateShouldTaint,
		FieldNodeTemplateConfigurationId,
		FieldNodeTemplateRebalancingConfigMinNodes,
		FieldNodeTemplateCustomLabels,
		FieldNodeTemplateCustomTaints,
		FieldNodeTemplateCustomInstancesEnabled,
		FieldNodeTemplateConstraints,
		FieldNodeTemplateIsEnabled,
	) {
		log.Printf("[INFO] Nothing to update in node template")
		return nil
	}

	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)
	name := d.Get(FieldNodeTemplateName).(string)

	req := sdk.NodeTemplatesAPIUpdateNodeTemplateJSONRequestBody{}
	if v, ok := d.GetOk(FieldNodeTemplateIsDefault); ok {
		req.IsDefault = toPtr(v.(bool))
	}

	if v, ok := d.GetOk(FieldNodeTemplateIsEnabled); ok {
		req.IsEnabled = toPtr(v.(bool))
	}

	if v, ok := d.GetOk(FieldNodeTemplateConfigurationId); ok {
		req.ConfigurationId = toPtr(v.(string))
	}

	if req.CustomLabel == nil {
		if v, ok := d.Get(FieldNodeTemplateCustomLabels).(map[string]any); ok && len(v) > 0 {
			customLabels := map[string]string{}

			for k, v := range v {
				customLabels[k] = v.(string)
			}

			req.CustomLabels = &sdk.NodetemplatesV1UpdateNodeTemplate_CustomLabels{AdditionalProperties: customLabels}
		}
	}

	if v, _ := d.GetOk(FieldNodeTemplateShouldTaint); v != nil {
		req.ShouldTaint = toPtr(v.(bool))
	}

	if v, ok := d.Get(FieldNodeTemplateCustomTaints).([]any); ok && len(v) > 0 {
		ts := []map[string]any{}
		for _, val := range v {
			ts = append(ts, val.(map[string]any))
		}

		req.CustomTaints = toCustomTaintsWithOptionalEffect(ts)
	}

	if !(*req.ShouldTaint) && req.CustomTaints != nil && len(*req.CustomTaints) > 0 {
		return diag.FromErr(fmt.Errorf("shouldTaint must be true for the node template to get updated with custom taints"))
	}

	if v, _ := d.GetOk(FieldNodeTemplateRebalancingConfigMinNodes); v != nil {
		req.RebalancingConfig = &sdk.NodetemplatesV1RebalancingConfiguration{
			MinNodes: toPtr(int32(v.(int))),
		}
	}

	if v, ok := d.Get(FieldNodeTemplateConstraints).([]any); ok && len(v) > 0 {
		req.Constraints = toTemplateConstraints(v[0].(map[string]any))
	}

	if v, _ := d.GetOk(FieldNodeTemplateCustomInstancesEnabled); v != nil {
		req.CustomInstancesEnabled = lo.ToPtr(v.(bool))
	}

	resp, err := client.NodeTemplatesAPIUpdateNodeTemplateWithResponse(ctx, clusterID, name, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	return resourceNodeTemplateRead(ctx, d, meta)
}

func resourceNodeTemplateCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	log.Printf("[INFO] Create Node Template post call start")
	defer log.Printf("[INFO] Create Node Template post call end")
	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)

	// default node template is created by default in the background, therefore we need to use PUT instead of POST
	if d.Get(FieldNodeTemplateIsDefault).(bool) {
		return updateDefaultNodeTemplate(ctx, d, meta)
	}

	req := sdk.NodeTemplatesAPICreateNodeTemplateJSONRequestBody{
		Name:            lo.ToPtr(d.Get(FieldNodeTemplateName).(string)),
		IsDefault:       lo.ToPtr(d.Get(FieldNodeTemplateIsDefault).(bool)),
		ConfigurationId: lo.ToPtr(d.Get(FieldNodeTemplateConfigurationId).(string)),
		ShouldTaint:     lo.ToPtr(d.Get(FieldNodeTemplateShouldTaint).(bool)),
	}

	if v, ok := d.GetOk(FieldNodeTemplateIsEnabled); ok {
		req.IsEnabled = lo.ToPtr(v.(bool))
	}

	if v, ok := d.Get(FieldNodeTemplateRebalancingConfigMinNodes).(int32); ok {
		req.RebalancingConfig = &sdk.NodetemplatesV1RebalancingConfiguration{
			MinNodes: lo.ToPtr(v),
		}
	}

	if v, ok := d.Get(FieldNodeTemplateCustomLabels).(map[string]any); ok && len(v) > 0 {
		customLabels := map[string]string{}

		for k, v := range v {
			customLabels[k] = v.(string)
		}

		req.CustomLabels = &sdk.NodetemplatesV1NewNodeTemplate_CustomLabels{AdditionalProperties: customLabels}
	}

	if v, ok := d.Get(FieldNodeTemplateCustomTaints).([]any); ok && len(v) > 0 {
		ts := []map[string]any{}
		for _, val := range v {
			ts = append(ts, val.(map[string]any))
		}

		req.CustomTaints = toCustomTaintsWithOptionalEffect(ts)
	}

	if !(*req.ShouldTaint) && req.CustomTaints != nil && len(*req.CustomTaints) > 0 {
		return diag.FromErr(fmt.Errorf("shouldTaint must be true for the node template to get created with custom taints"))
	}

	if v, ok := d.Get(FieldNodeTemplateConstraints).([]any); ok && len(v) > 0 {
		req.Constraints = toTemplateConstraints(v[0].(map[string]any))
	}

	if v, _ := d.GetOk(FieldNodeTemplateCustomInstancesEnabled); v != nil {
		req.CustomInstancesEnabled = lo.ToPtr(v.(bool))
	}

	resp, err := client.NodeTemplatesAPICreateNodeTemplateWithResponse(ctx, clusterID, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	d.SetId(lo.FromPtr(resp.JSON200.Name))

	return resourceNodeTemplateRead(ctx, d, meta)
}

func updateDefaultNodeTemplate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	d.SetId(d.Get(FieldNodeTemplateName).(string))
	// make timeout 5 seconds less than the creation timeout
	timeout := d.Timeout(schema.TimeoutCreate) - 5*time.Second
	// handle situation when default node template is not created yet by autoscaler policy
	if err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		diagnostics := updateNodeTemplate(ctx, d, meta, true)

		for _, d := range diagnostics {
			if d.Severity == diag.Error {
				if strings.Contains(d.Summary, "node template not found") {
					return retry.RetryableError(fmt.Errorf(d.Summary))
				}
				return retry.NonRetryableError(fmt.Errorf(d.Summary))
			}
		}
		return nil
	}); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func getNodeTemplateByName(ctx context.Context, data *schema.ResourceData, meta any, clusterID string) (*sdk.NodetemplatesV1NodeTemplate, error) {
	client := meta.(*ProviderConfig).api
	nodeTemplateName := data.Id()

	log.Printf("[INFO] Getting current node templates")
	resp, err := client.NodeTemplatesAPIListNodeTemplatesWithResponse(ctx, clusterID, &sdk.NodeTemplatesAPIListNodeTemplatesParams{IncludeDefault: lo.ToPtr(true)})
	notFound := fmt.Errorf("node templates for cluster %q not found at CAST AI", clusterID)
	if err != nil {
		return nil, err
	}

	templates := resp.JSON200

	if templates == nil {
		return nil, notFound
	}

	if err != nil {
		log.Printf("[WARN] Getting current node template: %v", err)
		return nil, fmt.Errorf("failed to get current node template from API: %v", err)
	}

	t, ok := lo.Find[sdk.NodetemplatesV1NodeTemplateListItem](lo.FromPtr(templates.Items), func(t sdk.NodetemplatesV1NodeTemplateListItem) bool {
		return lo.FromPtr(t.Template.Name) == nodeTemplateName
	})

	if !ok {
		return nil, fmt.Errorf("failed to find node template with name: %v", nodeTemplateName)
	}

	if err != nil {
		log.Printf("[WARN] Failed merging node template changes: %v", err)
		return nil, fmt.Errorf("failed to merge node template changes: %v", err)
	}

	return t.Template, nil
}

func nodeTemplateStateImporter(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	ids := strings.Split(d.Id(), "/")
	if len(ids) != 2 || ids[0] == "" || ids[1] == "" {
		return nil, fmt.Errorf("expected import id with format: <cluster_id>/<node_template name or id>, got: %q", d.Id())
	}

	clusterID, id := ids[0], ids[1]
	if err := d.Set(FieldClusterID, clusterID); err != nil {
		return nil, fmt.Errorf("setting cluster id: %w", err)
	}
	d.SetId(id)

	// Return if node config ID provided.
	if _, err := uuid.Parse(id); err == nil {
		return []*schema.ResourceData{d}, nil
	}

	// Find node templates
	client := meta.(*ProviderConfig).api
	resp, err := client.NodeTemplatesAPIListNodeTemplatesWithResponse(ctx, clusterID, &sdk.NodeTemplatesAPIListNodeTemplatesParams{IncludeDefault: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}

	if resp.JSON200 != nil {
		for _, cfg := range *resp.JSON200.Items {
			name := toString(cfg.Template.Name)
			if name == id {
				d.SetId(name)
				return []*schema.ResourceData{d}, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to find node template with the following name: %v", id)
}

func toCustomLabel(obj map[string]any) *sdk.NodetemplatesV1Label {
	if obj == nil {
		return nil
	}

	out := &sdk.NodetemplatesV1Label{}
	if v, ok := obj[FieldKey]; ok && v != "" {
		out.Key = toPtr(v.(string))
	}
	if v, ok := obj[FieldValue]; ok && v != "" {
		out.Value = toPtr(v.(string))
	}

	return out
}

func toCustomTaintsWithOptionalEffect(objs []map[string]any) *[]sdk.NodetemplatesV1TaintWithOptionalEffect {
	if len(objs) == 0 {
		return nil
	}

	out := &[]sdk.NodetemplatesV1TaintWithOptionalEffect{}

	for _, taint := range objs {
		t := sdk.NodetemplatesV1TaintWithOptionalEffect{}

		if v, ok := taint[FieldKey]; ok && v != "" {
			t.Key = v.(string)
		}
		if v, ok := taint[FieldValue]; ok && v != "" {
			t.Value = toPtr(v.(string))
		}
		if v, ok := taint[FieldEffect]; ok && v != "" {
			t.Effect = toPtr(sdk.NodetemplatesV1TaintEffect(v.(string)))
		}

		*out = append(*out, t)
	}

	return out
}

func flattenCustomLabel(label *sdk.NodetemplatesV1Label) []map[string]string {
	if label == nil {
		return nil
	}

	m := map[string]string{}
	if v := label.Key; v != nil {
		m[FieldKey] = toString(v)
	}
	if v := label.Value; v != nil {
		m[FieldValue] = toString(v)
	}
	return []map[string]string{m}
}

func flattenCustomTaints(taints *[]sdk.NodetemplatesV1Taint) []map[string]string {
	if taints == nil {
		return nil
	}

	var ts []map[string]string
	for _, taint := range *taints {
		t := map[string]string{}
		if k := taint.Key; k != nil {
			t[FieldKey] = toString(k)
		}
		if v := taint.Value; v != nil {
			t[FieldValue] = toString(v)
		}
		if e := taint.Effect; e != nil {
			t[FieldEffect] = string(*e)
		}

		ts = append(ts, t)
	}

	return ts
}

func toTemplateConstraints(obj map[string]any) *sdk.NodetemplatesV1TemplateConstraints {
	if obj == nil {
		return nil
	}

	out := &sdk.NodetemplatesV1TemplateConstraints{}
	if v, ok := obj[FieldNodeTemplateComputeOptimized].(bool); ok {
		out.ComputeOptimized = toPtr(v)
	}
	if v, ok := obj[FieldNodeTemplateFallbackRestoreRateSeconds].(int); ok {
		out.FallbackRestoreRateSeconds = toPtr(int32(v))
	}
	if v, ok := obj[FieldNodeTemplateGpu].([]any); ok && len(v) > 0 {
		val, ok := v[0].(map[string]any)
		if ok {
			out.Gpu = toTemplateConstraintsGpuConstraints(val)
		}
	}
	if v, ok := obj[FieldNodeTemplateInstanceFamilies].([]any); ok && len(v) > 0 {
		val, ok := v[0].(map[string]any)
		if ok {
			out.InstanceFamilies = toTemplateConstraintsInstanceFamilies(val)
		}
	}
	if v, ok := obj[FieldNodeTemplateMaxCpu].(int); ok && v != 0 {
		out.MaxCpu = toPtr(int32(v))
	}
	if v, ok := obj[FieldNodeTemplateMaxMemory].(int); ok && v != 0 {
		out.MaxMemory = toPtr(int32(v))
	}
	if v, ok := obj[FieldNodeTemplateMinCpu].(int); ok {
		out.MinCpu = toPtr(int32(v))
	}
	if v, ok := obj[FieldNodeTemplateMinMemory].(int); ok {
		out.MinMemory = toPtr(int32(v))
	}
	if v, ok := obj[FieldNodeTemplateSpot].(bool); ok {
		out.Spot = toPtr(v)
	}
	if v, ok := obj[FieldNodeTemplateOnDemand].(bool); ok {
		out.OnDemand = toPtr(v)
	} else {
		if v, ok := obj[FieldNodeTemplateSpot].(bool); ok {
			out.Spot = toPtr(!v)
		}
	}
	if v, ok := obj[FieldNodeTemplateStorageOptimized].(bool); ok {
		out.StorageOptimized = toPtr(v)
	}
	if v, ok := obj[FieldNodeTemplateUseSpotFallbacks].(bool); ok {
		out.UseSpotFallbacks = toPtr(v)
	}
	if v, ok := obj[FieldNodeTemplateArchitectures].([]any); ok {
		out.Architectures = toPtr(toStringList(v))
	}
	if v, ok := obj[FieldNodeTemplateOs].([]any); ok {
		out.Os = toPtr(toStringList(v))
	}
	if v, ok := obj[FieldNodeTemplateIsGpuOnly].(bool); ok {
		out.IsGpuOnly = toPtr(v)
	}
	if v, ok := obj[FieldNodeTemplateEnableSpotDiversity].(bool); ok {
		out.EnableSpotDiversity = toPtr(v)
	}
	if v, ok := obj[FieldNodeTemplateSpotDiversityPriceIncreaseLimitPercent].(int); ok {
		out.SpotDiversityPriceIncreaseLimitPercent = toPtr(int32(v))
	}
	if v, ok := obj[FieldNodeTemplateSpotInterruptionPredictionsEnabled].(bool); ok {
		out.SpotInterruptionPredictionsEnabled = toPtr(v)
	}
	if v, ok := obj[FieldNodeTemplateSpotInterruptionPredictionsType].(string); ok {
		out.SpotInterruptionPredictionsType = toPtr(v)
	}

	return out
}

func toTemplateConstraintsInstanceFamilies(o map[string]any) *sdk.NodetemplatesV1TemplateConstraintsInstanceFamilyConstraints {
	if o == nil {
		return nil
	}

	out := &sdk.NodetemplatesV1TemplateConstraintsInstanceFamilyConstraints{}
	if v, ok := o[FieldNodeTemplateExclude].([]any); ok {
		out.Exclude = toPtr(toStringList(v))
	}
	if v, ok := o[FieldNodeTemplateInclude].([]any); ok {
		out.Include = toPtr(toStringList(v))
	}
	return out
}

func toTemplateConstraintsGpuConstraints(o map[string]any) *sdk.NodetemplatesV1TemplateConstraintsGPUConstraints {
	if o == nil {
		return nil
	}

	out := &sdk.NodetemplatesV1TemplateConstraintsGPUConstraints{}
	if v, ok := o[FieldNodeTemplateManufacturers].([]any); ok {
		out.Manufacturers = toPtr(toStringList(v))
	}

	if v, ok := o[FieldNodeTemplateExcludeNames].([]any); ok {
		out.ExcludeNames = toPtr(toStringList(v))
	}
	if v, ok := o[FieldNodeTemplateIncludeNames].([]any); ok {
		out.IncludeNames = toPtr(toStringList(v))
	}

	if v, ok := o[FieldNodeTemplateMinCount].(int); ok {
		out.MinCount = toPtr(int32(v))
	}
	if v, ok := o[FieldNodeTemplateMaxCount].(int); ok && v != 0 {
		out.MaxCount = toPtr(int32(v))
	}

	return out
}
