package castai

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk/workload_eviction"
)

const (
	FieldEvictorEnabled                      = "enabled"
	FieldEvictorArm64Supported               = "arm64_supported"
	FieldEvictorDrainRollbackTimeout         = "drain_rollback_timeout"
	FieldEvictorDrainTimeout                 = "drain_timeout"
	FieldEvictorEmitNodeRelatedPodEvents     = "emit_node_related_pod_events"
	FieldEvictorForceDisableKarpenterMode    = "force_disable_karpenter_mode"
	FieldEvictorForceDisableLiveMigration    = "force_disable_live_migration"
	FieldEvictorForceDisablePodMutations     = "force_disable_pod_mutations"
	FieldEvictorForceDisableWoop             = "force_disable_woop"
	FieldEvictorMaxTargetNodesPerCycle       = "max_target_nodes_per_cycle"
	FieldEvictorMinTargetNodesPerCycle       = "min_target_nodes_per_cycle"
	FieldEvictorPricingAwarenessEnabled      = "pricing_awareness_enabled"
	FieldEvictorPricingModel                 = "pricing_model"
	FieldEvictorBaseCpuCost                  = "base_cpu_cost"
	FieldEvictorBaseMemCost                  = "base_mem_cost"
	FieldEvictorSpotDiscount                 = "spot_discount"
	FieldEvictorSoftTainting                 = "soft_tainting"
	FieldEvictorStatus                       = "status"
	FieldEvictorTargetNodePercentage         = "target_node_percentage"
	FieldEvictorWindows                      = "windows"
)

func resourceEvictor() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceCastaiEvictorRead,
		CreateContext: resourceCastaiEvictorCreate,
		UpdateContext: resourceCastaiEvictorUpdate,
		DeleteContext: resourceCastaiEvictorDelete,
		Description:   "CAST AI evictor resource to manage evictor configuration",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
		},

		Importer: &schema.ResourceImporter{
			StateContext: evictorStateImporter,
		},

		Schema: map[string]*schema.Schema{
			FieldClusterId: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "CAST AI cluster id.",
			},
			FieldEvictorEnabled: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable/disable the Evictor policy. This will either install or uninstall the Evictor component in your cluster.",
			},
			FieldEvictorDryRun: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable/disable dry-run. This property allows you to prevent the Evictor from carrying any operations out and preview the actions it would take.",
			},
			FieldEvictorAggressiveMode: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable/disable aggressive mode. By default, Evictor does not target nodes that are running unreplicated pods. This mode will make the Evictor start considering application with just a single replica.",
			},
			FieldEvictorScopedMode: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable/disable scoped mode. By default, Evictor targets all nodes in the cluster. This mode will constrain it to just the nodes which were created by CAST AI.",
			},
			FieldEvictorCycleInterval: {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "1m",
				ValidateFunc:     validateDuration,
				Description:      "Configure the interval duration between Evictor operations. This property can be used to lower or raise the frequency of the Evictor's find-and-drain operations.",
			},
			FieldEvictorNodeGracePeriodMinutes: {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     5,
				Description: "Configure the node grace period which controls the duration which must pass after a node has been created before Evictor starts considering that node.",
			},
			FieldEvictorPodEvictionFailureBackOffInterval: {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "5s",
				ValidateFunc:     validateDuration,
				Description:      "Configure the pod eviction failure back off interval. If pod eviction fails then Evictor will attempt to evict it again after the amount of time specified here.",
			},
			FieldEvictorIgnorePodDisruptionBudgets: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If enabled then Evictor will attempt to evict pods that have pod disruption budgets configured.",
			},
			FieldEvictorStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status provides indication whether the Evictor policy will be synced with Evictor helm chart - it will only be done if status is \"Compatible\".",
			},
			FieldEvictorSoftTainting: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If enabled, Evictor will use soft tainting (PreferNoSchedule) instead of hard cordoning after eviction.",
			},
			FieldEvictorEmitNodeRelatedPodEvents: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If enabled, Evictor emits Kubernetes events on pods when they are evicted as part of a node drain.",
			},
			FieldEvictorDrainTimeout: {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "10m",
				ValidateFunc:     validateDuration,
				Description:      "Maximum time the evictor waits for a node to fully drain before giving up.",
			},
			FieldEvictorDrainRollbackTimeout: {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "1m",
				ValidateFunc:     validateDuration,
				Description:      "How long the evictor waits before rolling back a cordon when a drain attempt fails.",
			},
			FieldEvictorWindows: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If enabled, Evictor will evict pods and nodes running Windows workloads.",
			},
			FieldEvictorForceDisableLiveMigration: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If enabled, disables LIVE pod migration even when the cluster capability is detected.",
			},
			FieldEvictorForceDisableWoop: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If enabled, disables WOOP scheduling recommendations even when the cluster capability is detected.",
			},
			FieldEvictorForceDisablePodMutations: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If enabled, disables PodMutation CR application during scheduling simulation even when the cluster capability is detected.",
			},
			FieldEvictorForceDisableKarpenterMode: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If enabled, disables Karpenter-aware scheduling simulation even when Karpenter is detected.",
			},
			FieldEvictorMaxTargetNodesPerCycle: {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Upper bound on nodes drained in a single cycle.",
			},
			FieldEvictorMinTargetNodesPerCycle: {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Lower bound on nodes considered per cycle.",
			},
			FieldEvictorTargetNodePercentage: {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Percentage of eligible nodes to consider per cycle, applied after min/max bounds.",
			},
			FieldEvictorPricingAwarenessEnabled: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If enabled, the evictor factors node cost into drain candidate selection, preferring to drain more expensive nodes first.",
			},
			FieldEvictorPricingModel: {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Overrides the default cost coefficients used when pricing_awareness_enabled is true.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldEvictorBaseCpuCost: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Cost coefficient per CPU core used in node price estimation. Serialized as a Kubernetes resource.Quantity string.",
						},
						FieldEvictorBaseMemCost: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Cost coefficient per GiB of memory used in node price estimation. Serialized as a Kubernetes resource.Quantity string.",
						},
						FieldEvictorSpotDiscount: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Fractional discount applied to spot instance price relative to on-demand (e.g. \"0.5\" means spot is 50% cheaper). Serialized as a Kubernetes resource.Quantity string.",
						},
					},
				},
			},
			FieldEvictorArm64Supported: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If enabled, indicates arm64 nodes are present so the scheduling simulation includes them as valid bin-packing targets.",
			},
		},
	}
}

func validateDuration(i interface{}, k string) (warnings []string, errors []error) {
	v, ok := i.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("expected type of %s to be string", k))
		return
	}
	if _, err := time.ParseDuration(v); err != nil {
		errors = append(errors, fmt.Errorf("expected %s to be a valid duration, got %q: %w", k, v, err))
	}
	return
}

func resourceCastaiEvictorRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	client := meta.(*ProviderConfig).workloadEvictionClient

	resp, err := client.EvictorAPIGetConfigWithResponse(ctx, clusterId)
	if err != nil {
		return diag.FromErr(fmt.Errorf("getting evictor config for cluster %s: %w", clusterId, err))
	}
	if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[INFO] Evictor config for cluster %s not found, removing from state", clusterId)
		data.SetId("")
		return nil
	}
	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("getting evictor config for cluster %s: expected status code %d, received: status=%d body=%s", clusterId, http.StatusOK, resp.StatusCode(), string(resp.Body)))
	}
	if resp.JSON200 == nil {
		return diag.FromErr(fmt.Errorf("getting evictor config for cluster %s: empty response", clusterId))
	}

	if err := flattenEvictorConfig(data, resp.JSON200); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(clusterId)
	return nil
}

func resourceCastaiEvictorCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := upsertEvictorConfig(ctx, data, meta); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(getClusterId(data))
	return nil
}

func resourceCastaiEvictorUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := upsertEvictorConfig(ctx, data, meta); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(getClusterId(data))
	return nil
}

func resourceCastaiEvictorDelete(_ context.Context, data *schema.ResourceData, _ interface{}) diag.Diagnostics {
	log.Printf("[INFO] Evictor config for cluster %s removed from Terraform state. No API delete is performed.", getClusterId(data))
	data.SetId("")
	return nil
}

func evictorStateImporter(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	clusterID := d.Id()
	if clusterID == "" {
		return nil, fmt.Errorf("expected import id to be a cluster id")
	}

	if err := d.Set(FieldClusterId, clusterID); err != nil {
		return nil, fmt.Errorf("setting cluster_id: %w", err)
	}

	client := meta.(*ProviderConfig).workloadEvictionClient
	resp, err := client.EvictorAPIGetConfigWithResponse(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("fetching evictor config for cluster %s: %w", clusterID, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("fetching evictor config for cluster %s: expected status code %d, received: status=%d body=%s", clusterID, http.StatusOK, resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("fetching evictor config for cluster %s: empty response", clusterID)
	}

	if err := flattenEvictorConfig(d, resp.JSON200); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func upsertEvictorConfig(ctx context.Context, data *schema.ResourceData, meta interface{}) error {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	config, err := toEvictorConfig(data)
	if err != nil {
		return fmt.Errorf("building evictor config: %w", err)
	}

	client := meta.(*ProviderConfig).workloadEvictionClient
	resp, err := client.EvictorAPIUpdateConfigWithResponse(ctx, clusterId, *config)
	if err != nil {
		return fmt.Errorf("updating evictor config for cluster %s: %w", clusterId, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("updating evictor config for cluster %s: expected status code %d, received: status=%d body=%s", clusterId, http.StatusOK, resp.StatusCode(), string(resp.Body))
	}
	if resp.JSON200 == nil {
		return fmt.Errorf("updating evictor config for cluster %s: empty response", clusterId)
	}

	if err := flattenEvictorConfig(data, resp.JSON200); err != nil {
		return err
	}

	return nil
}

func toEvictorConfig(data *schema.ResourceData) (*workload_eviction.Config, error) {
	cfg := &workload_eviction.Config{}

	cfg.Enabled = lo.ToPtr(data.Get(FieldEvictorEnabled).(bool))
	cfg.DryRun = lo.ToPtr(data.Get(FieldEvictorDryRun).(bool))
	cfg.AggressiveMode = lo.ToPtr(data.Get(FieldEvictorAggressiveMode).(bool))
	cfg.ScopedMode = lo.ToPtr(data.Get(FieldEvictorScopedMode).(bool))
	cfg.CycleInterval = lo.ToPtr(data.Get(FieldEvictorCycleInterval).(string))
	cfg.NodeGracePeriodMinutes = lo.ToPtr(int32(data.Get(FieldEvictorNodeGracePeriodMinutes).(int)))
	cfg.PodEvictionFailureBackOffInterval = lo.ToPtr(data.Get(FieldEvictorPodEvictionFailureBackOffInterval).(string))
	cfg.IgnorePodDisruptionBudgets = lo.ToPtr(data.Get(FieldEvictorIgnorePodDisruptionBudgets).(bool))
	cfg.SoftTainting = lo.ToPtr(data.Get(FieldEvictorSoftTainting).(bool))
	cfg.EmitNodeRelatedPodEvents = lo.ToPtr(data.Get(FieldEvictorEmitNodeRelatedPodEvents).(bool))
	cfg.DrainTimeout = lo.ToPtr(data.Get(FieldEvictorDrainTimeout).(string))
	cfg.DrainRollbackTimeout = lo.ToPtr(data.Get(FieldEvictorDrainRollbackTimeout).(string))
	cfg.Windows = lo.ToPtr(data.Get(FieldEvictorWindows).(bool))
	cfg.ForceDisableLiveMigration = lo.ToPtr(data.Get(FieldEvictorForceDisableLiveMigration).(bool))
	cfg.ForceDisableWoop = lo.ToPtr(data.Get(FieldEvictorForceDisableWoop).(bool))
	cfg.ForceDisablePodMutations = lo.ToPtr(data.Get(FieldEvictorForceDisablePodMutations).(bool))
	cfg.ForceDisableKarpenterMode = lo.ToPtr(data.Get(FieldEvictorForceDisableKarpenterMode).(bool))
	cfg.MaxTargetNodesPerCycle = lo.ToPtr(int32(data.Get(FieldEvictorMaxTargetNodesPerCycle).(int)))
	cfg.MinTargetNodesPerCycle = lo.ToPtr(int32(data.Get(FieldEvictorMinTargetNodesPerCycle).(int)))
	cfg.TargetNodePercentage = lo.ToPtr(int32(data.Get(FieldEvictorTargetNodePercentage).(int)))
	cfg.PricingAwarenessEnabled = lo.ToPtr(data.Get(FieldEvictorPricingAwarenessEnabled).(bool))
	cfg.Arm64Supported = lo.ToPtr(data.Get(FieldEvictorArm64Supported).(bool))

	pricingModel, err := toEvictorPricingModel(data)
	if err != nil {
		return nil, err
	}
	cfg.PricingModel = pricingModel

	return cfg, nil
}

func toEvictorPricingModel(data *schema.ResourceData) (*workload_eviction.PricingModel, error) {
	raw, ok := data.GetOk(FieldEvictorPricingModel)
	if !ok {
		return nil, nil
	}

	items, ok := raw.([]interface{})
	if !ok || len(items) == 0 {
		return nil, nil
	}

	m, ok := items[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid pricing_model block")
	}

	pm := &workload_eviction.PricingModel{}
	if v, ok := m[FieldEvictorBaseCpuCost].(string); ok && v != "" {
		pm.BaseCpuCost = lo.ToPtr(v)
	}
	if v, ok := m[FieldEvictorBaseMemCost].(string); ok && v != "" {
		pm.BaseMemCost = lo.ToPtr(v)
	}
	if v, ok := m[FieldEvictorSpotDiscount].(string); ok && v != "" {
		pm.SpotDiscount = lo.ToPtr(v)
	}

	return pm, nil
}

func flattenEvictorConfig(data *schema.ResourceData, cfg *workload_eviction.Config) error {
	if cfg == nil {
		return nil
	}

	setters := map[string]func() error{
		FieldEvictorEnabled:                      func() error { return data.Set(FieldEvictorEnabled, boolValue(cfg.Enabled)) },
		FieldEvictorDryRun:                       func() error { return data.Set(FieldEvictorDryRun, boolValue(cfg.DryRun)) },
		FieldEvictorAggressiveMode:               func() error { return data.Set(FieldEvictorAggressiveMode, boolValue(cfg.AggressiveMode)) },
		FieldEvictorScopedMode:                   func() error { return data.Set(FieldEvictorScopedMode, boolValue(cfg.ScopedMode)) },
		FieldEvictorCycleInterval:                func() error { return data.Set(FieldEvictorCycleInterval, evictorStringValue(cfg.CycleInterval)) },
		FieldEvictorNodeGracePeriodMinutes:       func() error { return data.Set(FieldEvictorNodeGracePeriodMinutes, intValue(cfg.NodeGracePeriodMinutes)) },
		FieldEvictorPodEvictionFailureBackOffInterval: func() error { return data.Set(FieldEvictorPodEvictionFailureBackOffInterval, evictorStringValue(cfg.PodEvictionFailureBackOffInterval)) },
		FieldEvictorIgnorePodDisruptionBudgets:   func() error { return data.Set(FieldEvictorIgnorePodDisruptionBudgets, boolValue(cfg.IgnorePodDisruptionBudgets)) },
		FieldEvictorSoftTainting:                 func() error { return data.Set(FieldEvictorSoftTainting, boolValue(cfg.SoftTainting)) },
		FieldEvictorEmitNodeRelatedPodEvents:     func() error { return data.Set(FieldEvictorEmitNodeRelatedPodEvents, boolValue(cfg.EmitNodeRelatedPodEvents)) },
		FieldEvictorDrainTimeout:                 func() error { return data.Set(FieldEvictorDrainTimeout, evictorStringValue(cfg.DrainTimeout)) },
		FieldEvictorDrainRollbackTimeout:         func() error { return data.Set(FieldEvictorDrainRollbackTimeout, evictorStringValue(cfg.DrainRollbackTimeout)) },
		FieldEvictorWindows:                      func() error { return data.Set(FieldEvictorWindows, boolValue(cfg.Windows)) },
		FieldEvictorForceDisableLiveMigration:    func() error { return data.Set(FieldEvictorForceDisableLiveMigration, boolValue(cfg.ForceDisableLiveMigration)) },
		FieldEvictorForceDisableWoop:             func() error { return data.Set(FieldEvictorForceDisableWoop, boolValue(cfg.ForceDisableWoop)) },
		FieldEvictorForceDisablePodMutations:     func() error { return data.Set(FieldEvictorForceDisablePodMutations, boolValue(cfg.ForceDisablePodMutations)) },
		FieldEvictorForceDisableKarpenterMode:    func() error { return data.Set(FieldEvictorForceDisableKarpenterMode, boolValue(cfg.ForceDisableKarpenterMode)) },
		FieldEvictorMaxTargetNodesPerCycle:       func() error { return data.Set(FieldEvictorMaxTargetNodesPerCycle, intValue(cfg.MaxTargetNodesPerCycle)) },
		FieldEvictorMinTargetNodesPerCycle:       func() error { return data.Set(FieldEvictorMinTargetNodesPerCycle, intValue(cfg.MinTargetNodesPerCycle)) },
		FieldEvictorTargetNodePercentage:         func() error { return data.Set(FieldEvictorTargetNodePercentage, intValue(cfg.TargetNodePercentage)) },
		FieldEvictorPricingAwarenessEnabled:      func() error { return data.Set(FieldEvictorPricingAwarenessEnabled, boolValue(cfg.PricingAwarenessEnabled)) },
		FieldEvictorArm64Supported:               func() error { return data.Set(FieldEvictorArm64Supported, boolValue(cfg.Arm64Supported)) },
		FieldEvictorStatus:                       func() error { return data.Set(FieldEvictorStatus, statusValue(cfg.Status)) },
		FieldEvictorPricingModel:                 func() error { return data.Set(FieldEvictorPricingModel, flattenPricingModel(cfg.PricingModel)) },
	}

	for field, setter := range setters {
		if err := setter(); err != nil {
			return fmt.Errorf("setting field %s: %w", field, err)
		}
	}

	return nil
}

func boolValue(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

func evictorStringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func intValue(v *int32) int {
	if v == nil {
		return 0
	}
	return int(*v)
}

func statusValue(v *workload_eviction.ConfigStatus) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func flattenPricingModel(pm *workload_eviction.PricingModel) []interface{} {
	if pm == nil {
		return nil
	}

	out := map[string]interface{}{
		FieldEvictorBaseCpuCost:  evictorStringValue(pm.BaseCpuCost),
		FieldEvictorBaseMemCost:  evictorStringValue(pm.BaseMemCost),
		FieldEvictorSpotDiscount: evictorStringValue(pm.SpotDiscount),
	}

	return []interface{}{out}
}
