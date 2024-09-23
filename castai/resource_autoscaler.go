package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mitchellh/mapstructure"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/types"
)

const (
	FieldAutoscalerPoliciesJSON                   = "autoscaler_policies_json"
	FieldAutoscalerPolicies                       = "autoscaler_policies"
	FieldAutoscalerSettings                       = "autoscaler_settings"
	FieldEnabled                                  = "enabled"
	FieldIsScopedMode                             = "is_scoped_mode"
	FieldNodeTemplatesPartialMatchingEnabled      = "node_templates_partial_matching_enabled"
	FieldUnschedulablePods                        = "unschedulable_pods"
	FieldHeadroom                                 = "headroom"
	FieldCPUPercentage                            = "cpu_percentage"
	FieldMemoryPercentage                         = "memory_percentage"
	FieldHeadroomSpot                             = "headroom_spot"
	FieldNodeConstraints                          = "node_constraints"
	FieldMinCPUCores                              = "min_cpu_cores"
	FieldMaxCPUCores                              = "max_cpu_cores"
	FieldMinRAMMiB                                = "min_ram_mib"
	FieldMaxRAMMiB                                = "max_ram_mib"
	FieldCustomInstancesEnabled                   = "custom_instances_enabled"
	FieldClusterLimits                            = "cluster_limits"
	FieldCPU                                      = "cpu"
	FieldMinCores                                 = "min_cores"
	FieldMaxCores                                 = "max_cores"
	FieldSpotInstances                            = "spot_instances"
	FieldMaxReclaimRate                           = "max_reclaim_rate"
	FieldSpotBackups                              = "spot_backups"
	FieldSpotDiversityEnabled                     = "spot_diversity_enabled"
	FieldSpotDiversityPriceIncreaseLimit          = "spot_diversity_price_increase_limit"
	FieldSpotInterruptionPredictions              = "spot_interruption_predictions"
	FieldSpotBackupRestoreRateSeconds             = "spot_backup_restore_rate_seconds"
	FieldSpotInterruptionPredictionsType          = "spot_interruption_predictions_type"
	FieldNodeDownscaler                           = "node_downscaler"
	FieldEmptyNodes                               = "empty_nodes"
	FieldDelaySeconds                             = "delay_seconds"
	FieldEvictor                                  = "evictor"
	FieldEvictorDryRun                            = "dry_run"
	FieldEvictorAggressiveMode                    = "aggressive_mode"
	FieldEvictorScopedMode                        = "scoped_mode"
	FieldEvictorCycleInterval                     = "cycle_interval"
	FieldEvictorNodeGracePeriodMinutes            = "node_grace_period_minutes"
	FieldEvictorPodEvictionFailureBackOffInterval = "pod_eviction_failure_back_off_interval"
	FieldEvictorIgnorePodDisruptionBudgets        = "ignore_pod_disruption_budgets"
	FieldPodPinner                                = "pod_pinner"
)

func resourceAutoscaler() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceCastaiAutoscalerRead,
		CreateContext: resourceCastaiAutoscalerCreate,
		UpdateContext: resourceCastaiAutoscalerUpdate,
		DeleteContext: resourceCastaiAutoscalerDelete,
		CustomizeDiff: resourceCastaiAutoscalerDiff,
		Description:   "CAST AI autoscaler resource to manage autoscaler settings",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldClusterId: {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "CAST AI cluster id",
			},
			FieldAutoscalerPoliciesJSON: {
				Type:             schema.TypeString,
				Description:      "autoscaler policies JSON string to override current autoscaler settings",
				Optional:         true,
				ValidateDiagFunc: validateAutoscalerPolicyJSON(),
				Deprecated:       "use autoscaler_settings instead. See README for example: https://github.com/castai/terraform-provider-castai?tab=readme-ov-file#migrating-from-6xx-to-7xx",
			},
			FieldAutoscalerPolicies: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "computed value to store full policies configuration",
			},
			FieldAutoscalerSettings: {
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				Description:   "autoscaler policy definitions to override current autoscaler settings",
				ConflictsWith: []string{FieldAutoscalerPoliciesJSON},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldEnabled: {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "enable/disable autoscaler policies",
						},
						FieldIsScopedMode: {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "run autoscaler in scoped mode. Only marked pods and nodes will be considered.",
						},
						FieldNodeTemplatesPartialMatchingEnabled: {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "marks whether partial matching should be used when deciding which custom node template to select.",
						},
						FieldUnschedulablePods: {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "policy defining autoscaler's behavior when unschedulable pods were detected.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldEnabled: {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "enable/disable unschedulable pods detection policy.",
									},
									FieldHeadroom: {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "additional headroom based on cluster's total available capacity for on-demand nodes.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldCPUPercentage: {
													Type:             schema.TypeInt,
													Optional:         true,
													Default:          10,
													Description:      "defines percentage of additional CPU capacity to be added.",
													ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 100)),
												},
												FieldMemoryPercentage: {
													Type:             schema.TypeInt,
													Optional:         true,
													Default:          10,
													Description:      "defines percentage of additional memory capacity to be added.",
													ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 100)),
												},
												FieldEnabled: {
													Type:        schema.TypeBool,
													Optional:    true,
													Default:     true,
													Description: "enable/disable headroom policy.",
												},
											},
										},
									},
									FieldHeadroomSpot: {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "additional headroom based on cluster's total available capacity for spot nodes.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldCPUPercentage: {
													Type:             schema.TypeInt,
													Optional:         true,
													Default:          10,
													Description:      "defines percentage of additional CPU capacity to be added.",
													ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 100)),
												},
												FieldMemoryPercentage: {
													Type:             schema.TypeInt,
													Optional:         true,
													Default:          10,
													Description:      "defines percentage of additional memory capacity to be added.",
													ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 100)),
												},
												FieldEnabled: {
													Type:        schema.TypeBool,
													Optional:    true,
													Default:     true,
													Description: "enable/disable headroom_spot policy.",
												},
											},
										},
									},
									FieldNodeConstraints: {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "defines the node constraints that will be applied when autoscaling with Unschedulable Pods policy.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldMinCPUCores: {
													Type:        schema.TypeInt,
													Optional:    true,
													Default:     0,
													Description: "defines min CPU cores for the node to pick.",
												},
												FieldMaxCPUCores: {
													Type:        schema.TypeInt,
													Optional:    true,
													Default:     32,
													Description: "defines max CPU cores for the node to pick.",
												},
												FieldMinRAMMiB: {
													Type:        schema.TypeInt,
													Optional:    true,
													Default:     2048,
													Description: "defines min RAM in MiB for the node to pick.",
												},
												FieldMaxRAMMiB: {
													Type:        schema.TypeInt,
													Optional:    true,
													Default:     262144,
													Description: "defines max RAM in MiB for the node to pick.",
												},
												FieldEnabled: {
													Type:        schema.TypeBool,
													Optional:    true,
													Default:     false,
													Description: "enable/disable node constraints policy.",
												},
											},
										},
									},
									FieldPodPinner: {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "defines the Cast AI Pod Pinner components settings.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldEnabled: {
													Type:        schema.TypeBool,
													Default:     true,
													Optional:    true,
													Description: "enable/disable the Pod Pinner component's automatic management in your cluster. Default: enabled.",
												},
											},
										},
									},
									FieldCustomInstancesEnabled: {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "enable/disable custom instances policy.",
									},
								},
							},
						},
						FieldClusterLimits: {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "defines minimum and maximum amount of CPU the cluster can have.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldEnabled: {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     true,
										Description: "enable/disable cluster size limits policy.",
									},
									FieldCPU: {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "defines the minimum and maximum amount of CPUs for cluster's worker nodes.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldMinCores: {
													Type:             schema.TypeInt,
													Optional:         true,
													Default:          1,
													Description:      "defines the minimum allowed amount of CPUs in the whole cluster.",
													ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
												},
												FieldMaxCores: {
													Type:             schema.TypeInt,
													Optional:         true,
													Default:          20,
													Description:      "defines the maximum allowed amount of vCPUs in the whole cluster.",
													ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(2)),
												},
											},
										},
									},
								},
							},
						},
						FieldSpotInstances: {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "policy defining whether autoscaler can use spot instances for provisioning additional workloads.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldEnabled: {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "enable/disable spot instances policy.",
									},
									FieldMaxReclaimRate: {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     0,
										Description: "max allowed reclaim rate when choosing spot instance type. E.g. if the value is 10%, instance types having 10% or higher reclaim rate will not be considered. Set to zero to use all instance types regardless of reclaim rate.",
									},
									FieldSpotBackups: {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "policy defining whether autoscaler can use spot backups instead of spot instances when spot instances are not available.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldEnabled: {
													Type:        schema.TypeBool,
													Optional:    true,
													Default:     false,
													Description: "enable/disable spot backups policy.",
												},
												FieldSpotBackupRestoreRateSeconds: {
													Type:             schema.TypeInt,
													Optional:         true,
													Default:          1800,
													Description:      "defines interval on how often spot backups restore to real spot should occur.",
													ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(60)),
												},
											},
										},
									},
									FieldSpotDiversityEnabled: {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
										Description: "enable/disable spot diversity policy. When enabled, autoscaler will try to balance between diverse and cost optimal instance types.",
									},
									FieldSpotDiversityPriceIncreaseLimit: {
										Type:             schema.TypeInt,
										Optional:         true,
										Default:          20,
										Description:      "allowed node configuration price increase when diversifying instance types. E.g. if the value is 10%, then the overall price of diversified instance types can be 10% higher than the price of the optimal configuration.",
										ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
									},
									FieldSpotInterruptionPredictions: {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "configure the handling of SPOT interruption predictions.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldEnabled: {
													Type:        schema.TypeBool,
													Optional:    true,
													Default:     false,
													Description: "enable/disable spot interruption predictions.",
												},
												FieldSpotInterruptionPredictionsType: {
													Type:             schema.TypeString,
													Optional:         true,
													Default:          "AWSRebalanceRecommendations",
													Description:      "define the type of the spot interruption prediction to handle. Allowed values are AWSRebalanceRecommendations, CASTAIInterruptionPredictions.",
													ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"AWSRebalanceRecommendations", "CASTAIInterruptionPredictions"}, false)),
												},
											},
										},
									},
								},
							},
						},
						FieldNodeDownscaler: {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "node downscaler defines policies for removing nodes based on the configured conditions.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldEnabled: {
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     true,
										Description: "enable/disable node downscaler policy.",
									},
									FieldEmptyNodes: {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "defines whether Node Downscaler should opt in for removing empty worker nodes when possible.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldEnabled: {
													Type:        schema.TypeBool,
													Optional:    true,
													Default:     false,
													Description: "enable/disable the empty worker nodes policy.",
												},
												FieldDelaySeconds: {
													Type:        schema.TypeInt,
													Optional:    true,
													Default:     300,
													Description: "period (in seconds) to wait before removing the node. Might be useful to control the aggressiveness of the downscaler.",
												},
											},
										},
									},
									FieldEvictor: {
										Type:        schema.TypeList,
										Optional:    true,
										MaxItems:    1,
										Description: "defines the CAST AI Evictor component settings. Evictor watches the pods running in your cluster and looks for ways to compact them into fewer nodes, making nodes empty, which will be removed by the empty worker nodes policy.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldEnabled: {
													Type:        schema.TypeBool,
													Optional:    true,
													Default:     false,
													Description: "enable/disable the Evictor policy. This will either install or uninstall the Evictor component in your cluster.",
												},
												FieldEvictorDryRun: {
													Type:        schema.TypeBool,
													Optional:    true,
													Default:     false,
													Description: "enable/disable dry-run. This property allows you to prevent the Evictor from carrying any operations out and preview the actions it would take.",
												},
												FieldEvictorAggressiveMode: {
													Type:        schema.TypeBool,
													Optional:    true,
													Default:     false,
													Description: "enable/disable aggressive mode. By default, Evictor does not target nodes that are running unreplicated pods. This mode will make the Evictor start considering application with just a single replica.",
												},
												FieldEvictorScopedMode: {
													Type:        schema.TypeBool,
													Optional:    true,
													Default:     false,
													Description: "enable/disable scoped mode. By default, Evictor targets all nodes in the cluster. This mode will constrain it to just the nodes which were created by CAST AI.",
												},
												FieldEvictorCycleInterval: {
													Type:        schema.TypeString,
													Optional:    true,
													Default:     "1m",
													Description: "configure the interval duration between Evictor operations. This property can be used to lower or raise the frequency of the Evictor's find-and-drain operations.",
												},
												FieldEvictorNodeGracePeriodMinutes: {
													Type:        schema.TypeInt,
													Optional:    true,
													Default:     5,
													Description: "configure the node grace period which controls the duration which must pass after a node has been created before Evictor starts considering that node.",
												},
												FieldEvictorPodEvictionFailureBackOffInterval: {
													Type:        schema.TypeString,
													Optional:    true,
													Default:     "5s",
													Description: "configure the pod eviction failure back off interval. If pod eviction fails then Evictor will attempt to evict it again after the amount of time specified here.",
												},
												FieldEvictorIgnorePodDisruptionBudgets: {
													Type:        schema.TypeBool,
													Optional:    true,
													Default:     false,
													Description: "if enabled then Evictor will attempt to evict pods that have pod disruption budgets configured.",
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
		},
	}
}

func resourceCastaiAutoscalerDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	err := upsertPolicies(ctx, meta, clusterId, `{"enabled":false}`)
	if err != nil {
		log.Printf("[ERROR] Failed to disable autoscaler policies: %v", err)
		return diag.FromErr(err)
	}

	return nil
}

func resourceCastaiAutoscalerDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	clusterId := getClusterId(d)
	if clusterId == "" {
		return nil
	}

	policies, err := getChangedPolicies(ctx, d, meta, clusterId)
	if err != nil {
		return err
	}
	if policies == nil {
		return nil
	}

	return d.SetNew(FieldAutoscalerPolicies, string(policies))
}

func resourceCastaiAutoscalerRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := readAutoscalerPolicies(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCastaiAutoscalerCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	err := updateAutoscalerPolicies(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(getClusterId(data))
	return nil
}

func resourceCastaiAutoscalerUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := updateAutoscalerPolicies(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(getClusterId(data))
	return nil
}

func getCurrentPolicies(ctx context.Context, client *sdk.ClientWithResponses, clusterId string) ([]byte, error) {
	log.Printf("[INFO] Getting cluster autoscaler information.")

	resp, err := client.PoliciesAPIGetClusterPolicies(ctx, clusterId)
	if err != nil {
		return nil, err
	} else if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("cluster %s policies do not exist at CAST AI", clusterId)
	}

	responseBytes, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	log.Printf("[DEBUG] Read autoscaler policies for cluster %s:\n%v\n", clusterId, string(responseBytes))

	return normalizeJSON(responseBytes)
}

func updateAutoscalerPolicies(ctx context.Context, data *schema.ResourceData, meta interface{}) error {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	policies, err := getChangedPolicies(ctx, data, meta, clusterId)
	if err != nil {
		return err
	}

	if policies == nil {
		log.Printf("[DEBUG] changed policies json not calculated. Skipping autoscaler policies changes")
		return nil
	}

	changedPoliciesJSON := string(policies)
	if changedPoliciesJSON == "" {
		log.Printf("[DEBUG] changed policies json not found. Skipping autoscaler policies changes")
		return nil
	}

	return upsertPolicies(ctx, meta, clusterId, changedPoliciesJSON)
}

func upsertPolicies(ctx context.Context, meta interface{}, clusterId string, changedPoliciesJSON string) error {
	client := meta.(*ProviderConfig).api

	resp, err := client.PoliciesAPIUpsertClusterPoliciesWithBodyWithResponse(ctx, clusterId, "application/json", bytes.NewReader([]byte(changedPoliciesJSON)))
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return checkErr
	}

	return nil
}

func readAutoscalerPolicies(ctx context.Context, data *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] AUTOSCALER policies get call start")
	defer log.Printf("[INFO] AUTOSCALER policies get call end")

	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	client := meta.(*ProviderConfig).api
	currentPolicies, err := getCurrentPolicies(ctx, client, clusterId)
	if err != nil {
		return err
	}

	err = data.Set(FieldAutoscalerPolicies, string(currentPolicies))
	if err != nil {
		log.Printf("[ERROR] Failed to set field: %v", err)
		return err
	}

	return nil
}

func getClusterId(data types.ResourceProvider) string {
	value, found := data.GetOk(FieldClusterId)
	if !found {
		return ""
	}

	return value.(string)
}

func getChangedPolicies(ctx context.Context, data types.ResourceProvider, meta interface{}, clusterId string) ([]byte, error) {
	policyChangesJSON, isPoliciesJSONExist := data.GetOk(FieldAutoscalerPoliciesJSON)
	_, isPoliciesSettingsExist := data.GetOk(FieldAutoscalerSettings)

	var policyChanges []byte

	if isPoliciesSettingsExist {
		policy, err := toAutoscalerPolicy(data)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize policy definitions: %v", err)
		}

		data, err := json.Marshal(policy)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize policy definition: %v", err)
		}

		policyChanges = data
	} else if isPoliciesJSONExist {
		policyChanges = []byte(policyChangesJSON.(string))
	} else {
		log.Printf("[DEBUG] policies json not provided. Skipping autoscaler policies changes")
		return nil, nil
	}

	if !json.Valid(policyChanges) {
		log.Printf("[WARN] policies JSON invalid: %v", string(policyChanges))
		return nil, fmt.Errorf("policies JSON invalid")
	}

	client := meta.(*ProviderConfig).api

	currentPolicies, err := getCurrentPolicies(ctx, client, clusterId)
	if err != nil {
		log.Printf("[WARN] Getting current policies: %v", err)
		return nil, fmt.Errorf("failed to get policies from API: %v", err)
	}

	policies, err := jsonpatch.MergePatch(currentPolicies, policyChanges)
	if err != nil {
		log.Printf("[WARN] Failed to merge policy changes: %v", err)
		return nil, fmt.Errorf("failed to merge policies: %v", err)
	}

	return normalizeJSON(policies)
}

func validateAutoscalerPolicyJSON() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(func(i interface{}, k string) ([]string, []error) {
		v, ok := i.(string)
		if !ok {
			return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
		}
		policyMap := make(map[string]interface{})
		err := json.Unmarshal([]byte(v), &policyMap)
		if err != nil {
			return nil, []error{fmt.Errorf("failed to deserialize JSON: %v", err)}
		}
		errors := make([]error, 0)
		if _, found := policyMap["spotInstances"]; found {
			errors = append(errors, createValidationError("spotInstances", v))
		}
		if unschedulablePods, found := policyMap["unschedulablePods"]; found {
			if unschedulablePodsMap, ok := unschedulablePods.(map[string]interface{}); ok {
				if _, found := unschedulablePodsMap["customInstancesEnabled"]; found {
					errors = append(errors, createValidationError("customInstancesEnabled", v))
				}
				if _, found := unschedulablePodsMap["nodeConstraints"]; found {
					errors = append(errors, createValidationError("nodeConstraints", v))
				}
			}
		}

		return nil, errors
	})
}

func createValidationError(field, value string) error {
	return fmt.Errorf("'%s' field was removed from policies JSON in 5.0.0. "+
		"The configuration was migrated to default node template.\n\n"+
		"See: https://github.com/castai/terraform-provider-castai#migrating-from-4xx-to-5xx\n\n"+
		"Policy:\n%v", field, value)
}

// toAutoscalerPolicy converts FieldAutoscalerSettings to types.AutoscalerPolicy for given data.
func toAutoscalerPolicy(data types.ResourceProvider) (*types.AutoscalerPolicy, error) {
	out, ok := extractNestedValues(data, FieldAutoscalerSettings, true, true)
	if !ok {
		return nil, nil
	}

	var policy types.AutoscalerPolicy

	// This allows us to decode the map into a struct with weak type conversions
	// like "true" string into bool true or "1" string into int 1.
	err := mapstructure.WeakDecode(out, &policy)
	if err != nil {
		return nil, err
	}

	return &policy, nil
}
