package castai

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	ClusterFieldName        = "name"
	ClusterFieldStatus      = "status"
	ClusterFieldRegion      = "region"
	ClusterFieldCredentials = "credentials"
	ClusterFieldKubeconfig  = "kubeconfig"

	ClusterFieldInitializeParams = "initialize_params"
	ClusterFieldNodes            = "nodes"
	ClusterFieldNodesCloud       = "cloud"
	ClusterFieldNodesRole        = "role"
	ClusterFieldNodesShape       = "shape"

	PolicyFieldAutoscalerPolicies = "autoscaler_policies"
	PolicyFieldClusterLimits 	  = "cluster_limits"
	PolicyFieldNodeDownscaler     = "node_downscaler"
	PolicyFieldSpotInstances      = "spot_instances"
	PolicyFieldUnschedulablePods  = "unschedulable_pods"

)

func resourceCastaiCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiClusterCreateOrUpdate,
		ReadContext:   resourceCastaiClusterRead,
		UpdateContext: resourceCastaiClusterCreateOrUpdate,
		DeleteContext: resourceCastaiClusterDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(45 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			ClusterFieldName: {
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Required:         true,
				ForceNew:         true,
			},
			ClusterFieldRegion: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			ClusterFieldCredentials: {
				Type:     schema.TypeSet,
				Set:      schema.HashString,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
			},
			ClusterFieldStatus: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			ClusterFieldInitializeParams: {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						ClusterFieldNodes: {
							Type:     schema.TypeList,
							MinItems: 1,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									ClusterFieldNodesCloud: {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(sdk.SupportedClouds(), false)),
									},
									ClusterFieldNodesRole: {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"master", "worker"}, false)),
									},
									ClusterFieldNodesShape: {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"x-small", "small", "medium", "large", "x-large", "2x-large"}, false)),
									},
								},
							},
						},
					},
				},
			},
			ClusterFieldKubeconfig: schemaKubeconfig(),
			PolicyFieldAutoscalerPolicies: {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default: true,
						},
						PolicyFieldClusterLimits: {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  true,
									},
									"cpu": {
										Type:     schema.TypeList,
										MaxItems: 1,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"max_cores": {
													Type:     schema.TypeInt,
													Optional: true,
													Default: 21,
												},
												"min_cores": {
													Type:     schema.TypeInt,
													Optional: true,
													Default: 2,
												},
											},
										},
									},
								},
							},
						},
						PolicyFieldNodeDownscaler: {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"empty_nodes": {
										Type:     schema.TypeList,
										MaxItems: 1,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"enabled": {
													Type:     schema.TypeBool,
													Optional: true,
													Default:  false,
												},
											},
										},
									},
								},
							},
						},
						PolicyFieldSpotInstances: {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"clouds": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"enabled": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
								},
							},
						},
						PolicyFieldUnschedulablePods: {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"headroom": {
										Type:     schema.TypeList,
										MaxItems: 1,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"cpu_percentage": {
													Type:     schema.TypeInt,
													Optional: true,
													Default: 20,
												},
												"memory_percentage": {
													Type:     schema.TypeInt,
													Optional: true,
													Default: 2,
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

func resourceCastaiClusterCreateOrUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	var nodes []sdk.Node
	for _, val := range data.Get(ClusterFieldInitializeParams + ".0." + ClusterFieldNodes).([]interface{}) {
		nodeData := val.(map[string]interface{})
		nodes = append(nodes, sdk.Node{
			Role:  sdk.NodeType(nodeData[ClusterFieldNodesRole].(string)),
			Cloud: sdk.CloudType(nodeData[ClusterFieldNodesCloud].(string)),
			Shape: sdk.NodeShape(nodeData[ClusterFieldNodesShape].(string)),
		})
	}

	cluster := sdk.CreateNewClusterJSONRequestBody{
		Name:                data.Get(ClusterFieldName).(string),
		Region:              data.Get(ClusterFieldRegion).(string),
		CloudCredentialsIDs: convertStringArr(data.Get(ClusterFieldCredentials).(*schema.Set).List()),
		Nodes:               nodes,
	}

	log.Printf("[INFO] Creating new cluster: %#v", cluster)

	response, err := client.CreateNewClusterWithResponse(ctx, cluster)
	if checkErr := sdk.CheckCreateResponse(response, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	data.SetId(response.JSON201.Id)

	log.Printf("[DEBUG] Waiting for cluster to reach `ready` status, id=%q name=%q", data.Id(), data.Get(ClusterFieldName))
	err = resource.RetryContext(ctx, data.Timeout(schema.TimeoutCreate), waitForClusterToReachCreatedFunc(ctx, client, data.Id()))
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Cluster %q has reached `ready` status", data.Id())

	log.Printf("[DEBUG] Cluster %q autoscaling policies update", data.Id())
	autoscalerParams, ok := data.Get(PolicyFieldAutoscalerPolicies).([]interface{})
	if !ok || autoscalerParams == nil || len(autoscalerParams) == 0 || autoscalerParams[0] == nil {
		println("[DEBUG] Reading Policies `autoscaler_policies` empty parameters %v")
		return resourceCastaiClusterRead(ctx, data, meta)
	}
	updatePolicies(ctx, client, response.JSON201.Id, autoscalerParams[0].(map[string]interface{}))

	return resourceCastaiClusterRead(ctx, data, meta)
}

func resourceCastaiClusterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	response, err := client.GetClusterWithResponse(ctx, sdk.ClusterId(data.Id()))
	if err != nil {
		return diag.FromErr(err)
	} else if response.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Removing cluster %s from state because it no longer exists in CAST.AI", data.Id())
		data.SetId("")
		return nil
	}

	data.Set(ClusterFieldName, response.JSON200.Name)
	data.Set(ClusterFieldRegion, response.JSON200.Region)
	data.Set(ClusterFieldStatus, response.JSON200.Status)
	data.Set(ClusterFieldCredentials, response.JSON200.CloudCredentialsIDs)

	kubeconfig, err := client.GetClusterKubeconfigWithResponse(ctx, sdk.ClusterId(data.Id()))
	if checkErr := sdk.CheckGetResponse(kubeconfig, err); checkErr == nil {
		kubecfg, err := flattenKubeConfig(string(kubeconfig.Body))
		if err != nil {
			return nil
		}
		data.Set(ClusterFieldKubeconfig, kubecfg)
	} else {
		log.Printf("[WARN] kubeconfig is not available for cluster %q: %v", data.Id(), checkErr)
		data.Set(ClusterFieldKubeconfig, []interface{}{})
	}

	policies, err := client.GetPoliciesWithResponse(ctx,sdk.ClusterId(data.Id()))
	if checkErr := sdk.CheckGetResponse(policies, err); checkErr == nil {
		if err := data.Set(PolicyFieldAutoscalerPolicies, flattenAutoscalerPolicies(policies.JSON200)); err != nil {
			return nil
		}
	} else {
		log.Printf("[WARN] autoscaling policies are not available for cluster %q: %v", data.Id(), checkErr)
	}

	return nil
}

func flattenAutoscalerPolicies(readPol *sdk.PoliciesConfig) []map[string]interface{} {

	p := make(map[string]interface{})
	if readPol == nil {
		p["enabled"] = false
		return []map[string]interface{}{p}
	}

	p["enabled"] =  readPol.Enabled
	p["cluster_limits"] = readPol.ClusterLimits
	p["node_downscaler"] = readPol.NodeDownscaler
	p["spot_instances"] = readPol.SpotInstances
	p["unschedulable_pods"] = readPol.UnschedulablePods

	return []map[string]interface{}{p}
}


func resourceCastaiClusterDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if err := sdk.CheckDeleteResponse(client.DeleteClusterWithResponse(ctx, sdk.ClusterId(data.Id()))); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Waiting for cluster to reach `deleted` status, id=%q name=%q", data.Id(), data.Get(ClusterFieldName))
	err := resource.RetryContext(ctx, data.Timeout(schema.TimeoutDelete), waitForClusterStatusDeletedFunc(ctx, client, data.Id()))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func waitForClusterToReachCreatedFunc(ctx context.Context, client *sdk.ClientWithResponses, id string) resource.RetryFunc {
	return waitForClusterToReachStatusFunc(ctx, client, id, "ready", []string{"creating", "warning"})
}

func waitForClusterStatusDeletedFunc(ctx context.Context, client *sdk.ClientWithResponses, id string) resource.RetryFunc {
	return waitForClusterToReachStatusFunc(ctx, client, id, "deleted", []string{"deleting", "warning"})
}

func waitForClusterToReachStatusFunc(ctx context.Context, client *sdk.ClientWithResponses, id string, targetStatus string, retryableStatuses []string) resource.RetryFunc {
	return func() *resource.RetryError {
		response, err := client.GetClusterWithResponse(ctx, sdk.ClusterId(id))
		if err != nil || response.JSON200 == nil {
			return resource.NonRetryableError(err)
		}

		cluster := response.JSON200

		if cluster.Status == targetStatus {
			return nil
		}

		for _, retryableStatus := range retryableStatuses {
			if cluster.Status == retryableStatus {
				return resource.RetryableError(fmt.Errorf("waiting for cluster to reach %q status, id=%q name=%q, status=%s", targetStatus, cluster.Id, cluster.Name, cluster.Status))
			}
		}
		return resource.NonRetryableError(fmt.Errorf("cluster has reached unexpected status, id=%q name=%q, status=%s", cluster.Id, cluster.Name, cluster.Status))
	}
}

func expandCPUlimit(params interface{}) sdk.ClusterLimitsCpu {
	var clusterLimitsCPU sdk.ClusterLimitsCpu
	for _, val := range params.([]interface{}) {
		cpuData := val.(map[string]interface{})
		clusterLimitsCPU = sdk.ClusterLimitsCpu{
			MaxCores:  int64(cpuData["max_cores"].(int)),
			MinCores:  int64(cpuData["min_cores"].(int)),
		}
	}
	return clusterLimitsCPU
}

func expandNodeDownscaler(params interface{}) *sdk.NodeDownscalerEmptyNodes {
	var nodeDownscalerBool sdk.NodeDownscalerEmptyNodes
	for _, val := range params.([]interface{}) {
		ndData := val.(map[string]interface{})
		nodeDownscalerEnabled := ndData["enabled"].(bool)
		nodeDownscalerBool = sdk.NodeDownscalerEmptyNodes{
			Enabled:  &nodeDownscalerEnabled,
		}
	}
	return &nodeDownscalerBool
}

func expandHeadroom(params interface{}) sdk.Headroom {
	var headroomPercentage sdk.Headroom
	for _, val := range params.([]interface{}) {
		hpData := val.(map[string]interface{})
		headroomPercentage = sdk.Headroom{
			CpuPercentage:  	hpData["cpu_percentage"].(int),
			MemoryPercentage:   hpData["memory_percentage"].(int),
		}
	}
	return headroomPercentage
}

func updatePolicies(ctx context.Context, client *sdk.ClientWithResponses, clusterID string, pc map[string]interface{}) diag.Diagnostics {
	var clusterLimits sdk.ClusterLimitsPolicy
	for _, val := range pc[PolicyFieldClusterLimits].([]interface{}) {
		limitData := val.(map[string]interface{})

		clusterLimits = sdk.ClusterLimitsPolicy{
			Enabled:  limitData["enabled"].(bool),
			Cpu: expandCPUlimit(limitData["cpu"].(interface{})),
		}
	}
	log.Printf("[DEBUG] Reading Policies `cluster_limits` parameterEnabled=%v", clusterLimits)

	var nodeDownscaler sdk.NodeDownscaler
	for _, val := range pc[PolicyFieldNodeDownscaler].([]interface{}) {
		ndData := val.(map[string]interface{})
		nodeDownscaler = sdk.NodeDownscaler{
			EmptyNodes: expandNodeDownscaler(ndData["empty_nodes"].(interface{})),
		}
	}
	log.Printf("[DEBUG] Reading Policies `node_downscaler` parameterEnabled=%v", &nodeDownscaler.EmptyNodes.Enabled)

	var spotInstances sdk.SpotInstances
	for _, val := range pc[PolicyFieldSpotInstances].([]interface{}) {
		siData := val.(map[string]interface{})

		spotInstances = sdk.SpotInstances{
			Enabled:  siData["enabled"].(bool),
			Clouds: convertStringArr(siData["clouds"].([]interface{})),
		}
	}
	log.Printf("[DEBUG] Reading Policies `spot_instances` parameterEnabled=%v", spotInstances)

	var unschedulablePods sdk.UnschedulablePodsPolicy
	for _, val := range pc[PolicyFieldUnschedulablePods].([]interface{}) {
		upData := val.(map[string]interface{})

		unschedulablePods = sdk.UnschedulablePodsPolicy{
			Enabled:  upData["enabled"].(bool),
			Headroom: expandHeadroom(upData["headroom"].(interface{})),
		}
	}
	log.Printf("[DEBUG] Reading Policies `unschedulable_pods` parameterEnabled=%v", unschedulablePods)

	autoscalerConfig := sdk.UpsertPoliciesJSONRequestBody{
		ClusterLimits:     clusterLimits,
		Enabled:		   pc["enabled"].(bool),
		NodeDownscaler:    &nodeDownscaler,
		SpotInstances:     spotInstances,
		UnschedulablePods: unschedulablePods,
	}

	resppol, err := client.UpsertPoliciesWithResponse(ctx, sdk.ClusterId(clusterID), autoscalerConfig)
	if checkErr := sdk.CheckGetResponse(resppol, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	return nil
}
