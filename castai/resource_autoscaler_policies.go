package castai

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler_v2"
)

const (
	FieldAutoscalerPoliciesVersion           = "version"
	FieldAutoscalerPoliciesEnabled           = "enabled"
	FieldAutoscalerPoliciesScopedMode        = "scoped_mode"
	FieldAutoscalerPoliciesClusterLimits     = "cluster_limits"
	FieldAutoscalerPoliciesNodeDownscaler    = "node_downscaler"
	FieldAutoscalerPoliciesUnschedulablePods = "unschedulable_pods"

	FieldClusterLimitsEnabled     = "enabled"
	FieldClusterLimitsCPU         = "cpu"
	FieldClusterLimitsCPUMaxCores = "max_cores"
	FieldClusterLimitsCPUMinCores = "min_cores"

	FieldNodeDownscalerEmptyNodesDelay   = "empty_nodes_delay"
	FieldNodeDownscalerEmptyNodesEnabled = "empty_nodes_enabled"

	FieldUnschedulablePodsEnabled   = "enabled"
	FieldUnschedulablePodsPodPinner = "pod_pinner"

	FieldPodPinnerEnabled = "enabled"
)

func resourceAutoscalerPolicies() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceAutoscalerPoliciesRead,
		CreateContext: resourceAutoscalerPoliciesCreate,
		UpdateContext: resourceAutoscalerPoliciesUpdate,
		DeleteContext: resourceAutoscalerPoliciesDelete,
		Description:   "CAST AI autoscaler policies V2 resource to manage cluster autoscaling policies.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			FieldClusterId: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "CAST AI cluster id.",
			},
			FieldAutoscalerPoliciesEnabled: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable/disable all policies (global master switch).",
			},
			FieldAutoscalerPoliciesScopedMode: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Run the node autoscaler in scoped mode.",
			},
			FieldAutoscalerPoliciesClusterLimits: {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Defines minimum and maximum amount of CPU the cluster can have.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldClusterLimitsEnabled: {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Enable/disable cluster size limits policy.",
						},
						FieldClusterLimitsCPU: {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Defines the minimum and maximum amount of CPUs for cluster's worker nodes.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldClusterLimitsCPUMaxCores: {
										Type:             schema.TypeInt,
										Required:         true,
										Description:      "Defines the maximum allowed amount of vCPUs in the whole cluster.",
										ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(2)),
									},
									FieldClusterLimitsCPUMinCores: {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "Defines the minimum allowed amount of CPUs in the whole cluster. Deprecated: Min CPU limit is no longer enforced.",
										Deprecated:  "Min CPU limit is no longer enforced.",
									},
								},
							},
						},
					},
				},
			},
			FieldAutoscalerPoliciesNodeDownscaler: {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Node Downscaler defines policies for removing nodes based on the configured conditions.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldNodeDownscalerEmptyNodesDelay: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Period to wait before removing an empty node.",
						},
						FieldNodeDownscalerEmptyNodesEnabled: {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Enable/disable the empty worker nodes policy.",
						},
					},
				},
			},
			FieldAutoscalerPoliciesUnschedulablePods: {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Policy defining autoscaler's behavior when unschedulable pods were detected.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldUnschedulablePodsEnabled: {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Enable/disable unschedulable pods detection policy.",
						},
						FieldUnschedulablePodsPodPinner: {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Defines the CAST AI Pod Pinner component settings.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldPodPinnerEnabled: {
										Type:        schema.TypeBool,
										Optional:    true,
										Description: "Enable/disable the Pod Pinner policy.",
									},
								},
							},
						},
					},
				},
			},
			FieldAutoscalerPoliciesVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Policy version for optimistic locking.",
			},
		},
	}
}

func resourceAutoscalerPoliciesRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	client := meta.(*ProviderConfig).clusterAutoscalerV2Client
	resp, err := client.PoliciesV2APIGetClusterPoliciesWithResponse(ctx, clusterId)
	if err != nil {
		return diag.FromErr(err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[INFO] Autoscaler policies for cluster %s not found, removing from state", clusterId)
		data.SetId("")
		return nil
	}
	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("expected status code %d, received: status=%d body=%s", http.StatusOK, resp.StatusCode(), string(resp.Body)))
	}

	if resp.JSON200 == nil {
		return diag.FromErr(fmt.Errorf("received empty policies response for cluster %s", clusterId))
	}

	if err := setAutoscalerPoliciesState(data, resp.JSON200); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(clusterId)
	return nil
}

func resourceAutoscalerPoliciesCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	if err := upsertAutoscalerPolicies(ctx, data, meta, clusterId); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(clusterId)
	return resourceAutoscalerPoliciesRead(ctx, data, meta)
}

func resourceAutoscalerPoliciesUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	if err := upsertAutoscalerPolicies(ctx, data, meta, clusterId); err != nil {
		return diag.FromErr(err)
	}

	return resourceAutoscalerPoliciesRead(ctx, data, meta)
}

func resourceAutoscalerPoliciesDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Print("[INFO] Autoscaler policies V2 resource deletion is a no-op. Removing from state.")
	data.SetId("")
	return nil
}

func upsertAutoscalerPolicies(ctx context.Context, data *schema.ResourceData, meta interface{}, clusterId string) error {
	policies, err := toPoliciesV2(data)
	if err != nil {
		return fmt.Errorf("building policies V2 request: %w", err)
	}

	client := meta.(*ProviderConfig).clusterAutoscalerV2Client
	resp, err := client.PoliciesV2APIUpdateClusterPoliciesWithResponse(ctx, clusterId, *policies)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("expected status code %d, received: status=%d body=%s", http.StatusOK, resp.StatusCode(), string(resp.Body))
	}

	if resp.JSON200 == nil {
		return fmt.Errorf("received empty policies response after update for cluster %s", clusterId)
	}

	return nil
}

func toPoliciesV2(data *schema.ResourceData) (*cluster_autoscaler_v2.PoliciesV2, error) {
	policies := &cluster_autoscaler_v2.PoliciesV2{}

	if v, ok := data.GetOk(FieldAutoscalerPoliciesEnabled); ok {
		enabled := v.(bool)
		policies.Enabled = &enabled
	}

	if v, ok := data.GetOk(FieldAutoscalerPoliciesScopedMode); ok {
		scopedMode := v.(bool)
		policies.ScopedMode = &scopedMode
	}

	if v, ok := data.GetOk(FieldAutoscalerPoliciesClusterLimits); ok {
		clusterLimits, err := toClusterLimitsPolicy(v.([]interface{}))
		if err != nil {
			return nil, err
		}
		policies.ClusterLimits = clusterLimits
	}

	if v, ok := data.GetOk(FieldAutoscalerPoliciesNodeDownscaler); ok {
		nodeDownscaler, err := toNodeDownscalerPolicy(v.([]interface{}))
		if err != nil {
			return nil, err
		}
		policies.NodeDownscaler = nodeDownscaler
	}

	if v, ok := data.GetOk(FieldAutoscalerPoliciesUnschedulablePods); ok {
		unschedulablePods, err := toUnschedulablePodsPolicy(v.([]interface{}))
		if err != nil {
			return nil, err
		}
		policies.UnschedulablePods = unschedulablePods
	}

	// Include version from state for optimistic locking on updates.
	if v, ok := data.GetOk(FieldAutoscalerPoliciesVersion); ok {
		version := v.(string)
		policies.Version = &version
	}

	return policies, nil
}

func toClusterLimitsPolicy(in []interface{}) (*cluster_autoscaler_v2.ClusterLimitsPolicy, error) {
	if len(in) == 0 || in[0] == nil {
		return nil, nil
	}

	m := in[0].(map[string]interface{})
	out := &cluster_autoscaler_v2.ClusterLimitsPolicy{}

	if v, ok := m[FieldClusterLimitsEnabled]; ok {
		enabled := v.(bool)
		out.Enabled = &enabled
	}

	if v, ok := m[FieldClusterLimitsCPU]; ok {
		cpuList := v.([]interface{})
		if len(cpuList) > 0 && cpuList[0] != nil {
			cpuMap := cpuList[0].(map[string]interface{})
			cpu := &cluster_autoscaler_v2.ClusterLimitsCpu{}

			maxCores := cpuMap[FieldClusterLimitsCPUMaxCores].(int)
			cpu.MaxCores = int32(maxCores)

			if v, ok := cpuMap[FieldClusterLimitsCPUMinCores]; ok {
				minCores := int32(v.(int))
				cpu.MinCores = &minCores
			}

			out.Cpu = cpu
		}
	}

	return out, nil
}

func toNodeDownscalerPolicy(in []interface{}) (*cluster_autoscaler_v2.NodeDownscalerPolicy, error) {
	if len(in) == 0 || in[0] == nil {
		return nil, nil
	}

	m := in[0].(map[string]interface{})
	out := &cluster_autoscaler_v2.NodeDownscalerPolicy{}

	if v, ok := m[FieldNodeDownscalerEmptyNodesDelay]; ok {
		delay := v.(string)
		if delay != "" {
			out.EmptyNodesDelay = &delay
		}
	}

	if v, ok := m[FieldNodeDownscalerEmptyNodesEnabled]; ok {
		enabled := v.(bool)
		out.EmptyNodesEnabled = &enabled
	}

	return out, nil
}

func toUnschedulablePodsPolicy(in []interface{}) (*cluster_autoscaler_v2.UnschedulablePodsPolicy, error) {
	if len(in) == 0 || in[0] == nil {
		return nil, nil
	}

	m := in[0].(map[string]interface{})
	out := &cluster_autoscaler_v2.UnschedulablePodsPolicy{}

	if v, ok := m[FieldUnschedulablePodsEnabled]; ok {
		enabled := v.(bool)
		out.Enabled = &enabled
	}

	if v, ok := m[FieldUnschedulablePodsPodPinner]; ok {
		podPinnerList := v.([]interface{})
		if len(podPinnerList) > 0 && podPinnerList[0] != nil {
			podPinnerMap := podPinnerList[0].(map[string]interface{})
			podPinner := &cluster_autoscaler_v2.PodPinner{}

			if v, ok := podPinnerMap[FieldPodPinnerEnabled]; ok {
				enabled := v.(bool)
				podPinner.Enabled = &enabled
			}

			out.PodPinner = podPinner
		}
	}

	return out, nil
}

func setAutoscalerPoliciesState(data *schema.ResourceData, policies *cluster_autoscaler_v2.PoliciesV2) error {
	// Always call data.Set to prevent state drift when API omits fields.
	var enabled bool
	if policies.Enabled != nil {
		enabled = *policies.Enabled
	}
	if err := data.Set(FieldAutoscalerPoliciesEnabled, enabled); err != nil {
		return err
	}

	var scopedMode bool
	if policies.ScopedMode != nil {
		scopedMode = *policies.ScopedMode
	}
	if err := data.Set(FieldAutoscalerPoliciesScopedMode, scopedMode); err != nil {
		return err
	}

	var version string
	if policies.Version != nil {
		version = *policies.Version
	}
	if err := data.Set(FieldAutoscalerPoliciesVersion, version); err != nil {
		return err
	}

	if err := data.Set(FieldAutoscalerPoliciesClusterLimits, flattenClusterLimitsPolicy(policies.ClusterLimits)); err != nil {
		return err
	}

	if err := data.Set(FieldAutoscalerPoliciesNodeDownscaler, flattenNodeDownscalerPolicy(policies.NodeDownscaler)); err != nil {
		return err
	}

	if err := data.Set(FieldAutoscalerPoliciesUnschedulablePods, flattenUnschedulablePodsPolicy(policies.UnschedulablePods)); err != nil {
		return err
	}

	return nil
}

func flattenClusterLimitsPolicy(in *cluster_autoscaler_v2.ClusterLimitsPolicy) []map[string]interface{} {
	if in == nil {
		return nil
	}
	out := map[string]interface{}{}

	if in.Enabled != nil {
		out[FieldClusterLimitsEnabled] = *in.Enabled
	}

	if in.Cpu != nil {
		cpu := map[string]interface{}{
			FieldClusterLimitsCPUMaxCores: int(in.Cpu.MaxCores),
		}
		if in.Cpu.MinCores != nil {
			cpu[FieldClusterLimitsCPUMinCores] = int(*in.Cpu.MinCores)
		}
		out[FieldClusterLimitsCPU] = []map[string]interface{}{cpu}
	}

	if len(out) == 0 {
		return nil
	}

	return []map[string]interface{}{out}
}

func flattenNodeDownscalerPolicy(in *cluster_autoscaler_v2.NodeDownscalerPolicy) []map[string]interface{} {
	if in == nil {
		return nil
	}
	out := map[string]interface{}{}

	if in.EmptyNodesDelay != nil {
		out[FieldNodeDownscalerEmptyNodesDelay] = *in.EmptyNodesDelay
	}

	if in.EmptyNodesEnabled != nil {
		out[FieldNodeDownscalerEmptyNodesEnabled] = *in.EmptyNodesEnabled
	}

	if len(out) == 0 {
		return nil
	}

	return []map[string]interface{}{out}
}

func flattenUnschedulablePodsPolicy(in *cluster_autoscaler_v2.UnschedulablePodsPolicy) []map[string]interface{} {
	if in == nil {
		return nil
	}
	out := map[string]interface{}{}

	if in.Enabled != nil {
		out[FieldUnschedulablePodsEnabled] = *in.Enabled
	}

	if in.PodPinner != nil {
		out[FieldUnschedulablePodsPodPinner] = flattenPodPinner(in.PodPinner)
	}

	if len(out) == 0 {
		return nil
	}

	return []map[string]interface{}{out}
}

func flattenPodPinner(in *cluster_autoscaler_v2.PodPinner) []map[string]interface{} {
	if in == nil {
		return nil
	}
	out := map[string]interface{}{}

	if in.Enabled != nil {
		out[FieldPodPinnerEnabled] = *in.Enabled
	}

	if len(out) == 0 {
		return nil
	}

	return []map[string]interface{}{out}
}
