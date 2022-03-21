package castai

import (
	"context"
	"fmt"
	"io/ioutil"
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
	vpnTypeWireGuardCrossLocationMesh = "wireguard_cross_location_mesh"
	vpnTypeWireGuardFullMesh          = "wireguard_full_mesh"
	vpnTypeCloudProvider              = "cloud_provider"
)

const (
	ClusterFieldName        = "name"
	ClusterFieldStatus      = "status"
	ClusterFieldRegion      = "region"
	ClusterFieldCredentials = "credentials"
	ClusterFieldKubeconfig  = "kubeconfig"

	ClusterFieldVPNType = "vpn_type"

	ClusterFieldInitializeParams = "initialize_params"
	ClusterFieldNodes            = "nodes"
	ClusterFieldNodesCloud       = "cloud"
	ClusterFieldNodesRole        = "role"
	ClusterFieldNodesShape       = "shape"

	PolicyFieldEnabled                               = "enabled"
	PolicyFieldAutoscalerPolicies                    = "autoscaler_policies"
	PolicyFieldClusterLimits                         = "cluster_limits"
	PolicyFieldClusterLimitsCPU                      = "cpu"
	PolicyFieldClusterLimitsCPUmax                   = "max_cores"
	PolicyFieldClusterLimitsCPUmin                   = "min_cores"
	PolicyFieldNodeDownscaler                        = "node_downscaler"
	PolicyFieldNodeDownscalerEmptyNodes              = "empty_nodes"
	PolicyFieldNodeDownscalerEmptyNodesDelay         = "delay_seconds"
	PolicyFieldSpotInstances                         = "spot_instances"
	PolicyFieldSpotInstancesClouds                   = "clouds"
	PolicyFieldUnschedulablePods                     = "unschedulable_pods"
	PolicyFieldUnschedulablePodsHeadroom             = "headroom"
	PolicyFieldUnschedulablePodsHeadroomCPUp         = "cpu_percentage"
	PolicyFieldUnschedulablePodsHeadroomRAMp         = "memory_percentage"
	PolicyFieldUnschedulablePodsNodeConstraint       = "node_constraints"
	PolicyFieldUnschedulablePodsNodeConstraintMaxCPU = "max_node_cpu_cores"
	PolicyFieldUnschedulablePodsNodeConstraintMaxRAM = "max_node_ram_gib"
	PolicyFieldUnschedulablePodsNodeConstraintMinCPU = "min_node_cpu_cores"
	PolicyFieldUnschedulablePodsNodeConstraintMinRAM = "min_node_ram_gib"
)

func resourceCastaiCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiClusterCreate,
		ReadContext:   resourceCastaiClusterRead,
		UpdateContext: resourceCastaiClusterUpdate,
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
			ClusterFieldVPNType: {
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Optional:         true,
				ForceNew:         false,
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
						PolicyFieldEnabled: {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						PolicyFieldClusterLimits: {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									PolicyFieldEnabled: {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  true,
									},
									PolicyFieldClusterLimitsCPU: {
										Type:     schema.TypeList,
										MaxItems: 1,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												PolicyFieldClusterLimitsCPUmax: {
													Type:     schema.TypeInt,
													Optional: true,
													Default:  21,
												},
												PolicyFieldClusterLimitsCPUmin: {
													Type:     schema.TypeInt,
													Optional: true,
													Default:  2,
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
									PolicyFieldNodeDownscalerEmptyNodes: {
										Type:     schema.TypeList,
										MaxItems: 1,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												PolicyFieldEnabled: {
													Type:     schema.TypeBool,
													Optional: true,
													Default:  false,
												},
												PolicyFieldNodeDownscalerEmptyNodesDelay: {
													Type:     schema.TypeInt,
													Optional: true,
													Default:  0,
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
									PolicyFieldSpotInstancesClouds: {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									PolicyFieldEnabled: {
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
									PolicyFieldEnabled: {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									PolicyFieldUnschedulablePodsHeadroom: {
										Type:     schema.TypeList,
										MaxItems: 1,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												PolicyFieldEnabled: {
													Type:     schema.TypeBool,
													Optional: true,
													Default:  false,
												},
												PolicyFieldUnschedulablePodsHeadroomCPUp: {
													Type:     schema.TypeInt,
													Optional: true,
													Default:  20,
												},
												PolicyFieldUnschedulablePodsHeadroomRAMp: {
													Type:     schema.TypeInt,
													Optional: true,
													Default:  2,
												},
											},
										},
									},
									PolicyFieldUnschedulablePodsNodeConstraint: {
										Type:     schema.TypeList,
										MaxItems: 1,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												PolicyFieldEnabled: {
													Type:     schema.TypeBool,
													Optional: true,
													Default:  false,
												},
												PolicyFieldUnschedulablePodsNodeConstraintMaxCPU: {
													Type:     schema.TypeInt,
													Optional: true,
													Default:  32,
												},
												PolicyFieldUnschedulablePodsNodeConstraintMaxRAM: {
													Type:     schema.TypeInt,
													Optional: true,
													Default:  262144,
												},
												PolicyFieldUnschedulablePodsNodeConstraintMinCPU: {
													Type:     schema.TypeInt,
													Optional: true,
													Default:  2,
												},
												PolicyFieldUnschedulablePodsNodeConstraintMinRAM: {
													Type:     schema.TypeInt,
													Optional: true,
													Default:  2048,
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

func resourceCastaiClusterCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	var nodes []sdk.NewNode
	for _, val := range data.Get(ClusterFieldInitializeParams + ".0." + ClusterFieldNodes).([]interface{}) {
		nodeData := val.(map[string]interface{})
		nodeShape := sdk.NodeShape(nodeData[ClusterFieldNodesShape].(string))
		nodes = append(nodes, sdk.NewNode{
			Role:  sdk.NodeType(nodeData[ClusterFieldNodesRole].(string)),
			Cloud: sdk.CloudType(nodeData[ClusterFieldNodesCloud].(string)),
			Shape: &nodeShape,
		})
	}

	cluster := sdk.CreateNewClusterJSONRequestBody{
		Name:                data.Get(ClusterFieldName).(string),
		Region:              data.Get(ClusterFieldRegion).(string),
		CloudCredentialsIDs: convertStringArr(data.Get(ClusterFieldCredentials).(*schema.Set).List()),
		Nodes:               nodes,
		Network:             toClusterNetwork(data.Get(ClusterFieldVPNType).(string)),
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

	log.Printf("[DEBUG] Cluster %q setting autoscaling policies", data.Id())
	autoscalerParams, ok := data.Get(PolicyFieldAutoscalerPolicies).([]interface{})
	if !ok || len(autoscalerParams) == 0 || autoscalerParams[0] == nil {
		log.Printf("[DEBUG] Reading Policies `autoscaler_policies` empty parameters")
		return resourceCastaiClusterRead(ctx, data, meta)
	}
	updatePolicies(ctx, client, data.Id(), autoscalerParams[0].(map[string]interface{}))

	return resourceCastaiClusterRead(ctx, data, meta)
}

func resourceCastaiClusterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	resp, err := client.GetClusterWithResponse(ctx, sdk.ClusterId(data.Id()))
	if err != nil {
		return diag.FromErr(err)
	} else if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Removing cluster %s from state because it no longer exists in CAST.AI", data.Id())
		data.SetId("")
		return nil
	}

	data.Set(ClusterFieldName, resp.JSON200.Name)
	data.Set(ClusterFieldRegion, resp.JSON200.Region)
	data.Set(ClusterFieldStatus, resp.JSON200.Status)
	data.Set(ClusterFieldCredentials, resp.JSON200.CloudCredentialsIDs)

	// Set vpn type from network.
	net := resp.JSON200.Network
	if net != nil {
		vpnType := vpnTypeCloudProvider
		if net.Vpn.WireGuard != nil {
			switch net.Vpn.WireGuard.Topology {
			case "crossLocationMesh":
				vpnType = vpnTypeWireGuardCrossLocationMesh
			case "fullMesh":
				vpnType = vpnTypeWireGuardFullMesh
			}
		}
		data.Set(ClusterFieldVPNType, vpnType)
	}

	kubeconfig, err := client.GetClusterKubeconfigWithResponse(ctx, sdk.ClusterId(data.Id()))
	if checkErr := sdk.CheckGetResponse(kubeconfig, err); checkErr == nil {
		kubecfg, err := flattenKubeConfig(string(kubeconfig.Body))
		if err != nil {
			return diag.Errorf("parsing kubeconfig: %v", err)
		}
		data.Set(ClusterFieldKubeconfig, kubecfg)
	} else {
		log.Printf("[WARN] kubeconfig is not available for cluster %q: %v", data.Id(), checkErr)
		data.Set(ClusterFieldKubeconfig, []interface{}{})
	}

	policies, err := client.PoliciesAPIGetClusterPoliciesWithResponse(ctx, data.Id())
	if checkErr := sdk.CheckGetResponse(policies, err); checkErr == nil {
		log.Printf("[INFO] Autoscaling policies for cluster %q", data.Id())
		data.Set(PolicyFieldAutoscalerPolicies, flattenAutoscalerPolicies(policies.JSON200))
	} else {
		log.Printf("[WARN] autoscaling policies are not available for cluster %q: %v", data.Id(), checkErr)
	}

	return nil
}

func resourceCastaiClusterUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if data.HasChange(ClusterFieldCredentials) {
		creds, ok := data.Get(ClusterFieldCredentials).(*schema.Set)
		if ok {
			log.Printf("[DEBUG] Cluster %q credentials update", data.Id())
			if err := updateCluster(ctx, client, data.Id(), data.Get(ClusterFieldVPNType), creds.List()); err != nil {
				return err
			}
		}
	}

	if data.HasChange(PolicyFieldAutoscalerPolicies) {
		autoscalerParams, ok := data.Get(PolicyFieldAutoscalerPolicies).([]interface{})

		//fmt.Printf("update policies %#v", autoscalerParams[0].(map[string]interface{}))

		if ok && len(autoscalerParams) > 0 {
			log.Printf("[DEBUG] Cluster %q autoscaling policies update", data.Id())
			if err := updatePolicies(ctx, client, data.Id(), autoscalerParams[0].(map[string]interface{})); err != nil {
				return err
			}
		}
	}

	return resourceCastaiClusterRead(ctx, data, meta)
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

func expandAutoscalerPolicies(pc map[string]interface{}) sdk.PoliciesAPIUpsertClusterPoliciesJSONRequestBody {
	var clusterLimits sdk.PoliciesV1ClusterLimitsPolicy
	for _, val := range pc[PolicyFieldClusterLimits].([]interface{}) {
		limitData := val.(map[string]interface{})
		for _, valn := range limitData[PolicyFieldClusterLimitsCPU].([]interface{}) {
			cpuData := valn.(map[string]interface{})
			clusterLimits = sdk.PoliciesV1ClusterLimitsPolicy{
				Enabled: toBoolPtr(limitData[PolicyFieldEnabled].(bool)),
				Cpu: &sdk.PoliciesV1ClusterLimitsCpu{
					MaxCores: toInt32Ptr(cpuData[PolicyFieldClusterLimitsCPUmax].(int32)),
					MinCores: toInt32Ptr(cpuData[PolicyFieldClusterLimitsCPUmin].(int32)),
				},
			}
		}
	}

	var nodeDownscalerPolicy sdk.PoliciesV1NodeDownscaler
	for _, val := range pc[PolicyFieldNodeDownscaler].([]interface{}) {
		ndData := val.(map[string]interface{})
		for _, valn := range ndData[PolicyFieldNodeDownscalerEmptyNodes].([]interface{}) {
			ndData := valn.(map[string]interface{})
			nodeDownscalerDelay := toInt32Ptr(ndData[PolicyFieldNodeDownscalerEmptyNodesDelay].(int32))
			nodeDownscalerEnabled := ndData[PolicyFieldEnabled].(bool)
			nodeDownscalerPolicy = sdk.PoliciesV1NodeDownscaler{
				EmptyNodes: &sdk.PoliciesV1NodeDownscalerEmptyNodes{
					DelaySeconds: nodeDownscalerDelay,
					Enabled:      &nodeDownscalerEnabled,
				},
			}
		}
	}

	var spotInstancesPolicy sdk.PoliciesV1SpotInstances
	for _, val := range pc[PolicyFieldSpotInstances].([]interface{}) {
		siData := val.(map[string]interface{})

		clouds := toCastaiClouds(siData[PolicyFieldSpotInstancesClouds].([]interface{}))
		spotInstancesPolicy = sdk.PoliciesV1SpotInstances{
			Enabled: toBoolPtr(siData[PolicyFieldEnabled].(bool)),
			Clouds:  &clouds,
		}
	}

	var unschedulablePodsPolicy sdk.PoliciesV1UnschedulablePodsPolicy
	var headroomPol sdk.PoliciesV1Headroom
	var nodeConstraintPol sdk.PoliciesV1NodeConstraints

	for _, val := range pc[PolicyFieldUnschedulablePods].([]interface{}) {
		upData := val.(map[string]interface{})
		for _, valn := range upData[PolicyFieldUnschedulablePodsHeadroom].([]interface{}) {
			hpData := valn.(map[string]interface{})

			hpEnabled := hpData[PolicyFieldEnabled].(bool)
			headroomPol = sdk.PoliciesV1Headroom{
				Enabled:          &hpEnabled,
				CpuPercentage:    toInt32Ptr(hpData[PolicyFieldUnschedulablePodsHeadroomCPUp].(int32)),
				MemoryPercentage: toInt32Ptr(hpData[PolicyFieldUnschedulablePodsHeadroomRAMp].(int32)),
			}
		}

		for _, valn := range upData[PolicyFieldUnschedulablePodsNodeConstraint].([]interface{}) {
			ncData := valn.(map[string]interface{})

			nodeConstraintPol = sdk.PoliciesV1NodeConstraints{
				Enabled:     toBoolPtr(ncData[PolicyFieldEnabled].(bool)),
				MaxCpuCores: toInt32Ptr(ncData[PolicyFieldUnschedulablePodsNodeConstraintMaxCPU].(int32)),
				MaxRamMib:   toInt32Ptr(ncData[PolicyFieldUnschedulablePodsNodeConstraintMaxRAM].(int32) * 1024),
				MinCpuCores: toInt32Ptr(ncData[PolicyFieldUnschedulablePodsNodeConstraintMinCPU].(int32)),
				MinRamMib:   toInt32Ptr(ncData[PolicyFieldUnschedulablePodsNodeConstraintMinRAM].(int32) * 1024),
			}
		}

		unschedulablePodsPolicy = sdk.PoliciesV1UnschedulablePodsPolicy{
			Enabled:         toBoolPtr(upData[PolicyFieldEnabled].(bool)),
			Headroom:        &headroomPol,
			NodeConstraints: &nodeConstraintPol,
		}
	}

	autoscalerConfig := sdk.PoliciesAPIUpsertClusterPoliciesJSONRequestBody{
		ClusterLimits:     &clusterLimits,
		Enabled:           toBoolPtr(pc[PolicyFieldEnabled].(bool)),
		NodeDownscaler:    &nodeDownscalerPolicy,
		SpotInstances:     &spotInstancesPolicy,
		UnschedulablePods: &unschedulablePodsPolicy,
	}

	log.Printf("[DEBUG] Reading autoscaler Policies #{autoscalerConfig}")
	return autoscalerConfig
}

func toClusterNetwork(vpnType interface{}) *sdk.Network {
	defaultNetwork := &sdk.Network{Vpn: &sdk.VpnConfig{IpSec: &sdk.IpSecConfig{}}}
	vpnTypeString, ok := vpnType.(string)
	if !ok {
		vpnTypeString = vpnTypeCloudProvider
	}
	switch vpnTypeString {
	case vpnTypeCloudProvider:
		return defaultNetwork
	case vpnTypeWireGuardCrossLocationMesh:
		return &sdk.Network{Vpn: &sdk.VpnConfig{WireGuard: &sdk.WireGuardConfig{Topology: "crossLocationMesh"}}}
	case vpnTypeWireGuardFullMesh:
		return &sdk.Network{Vpn: &sdk.VpnConfig{WireGuard: &sdk.WireGuardConfig{Topology: "fullMesh"}}}
	}
	return defaultNetwork
}

func updateCluster(ctx context.Context, client *sdk.ClientWithResponses, clusterID string, vpnType interface{}, creds []interface{}) diag.Diagnostics {
	ids := make([]string, 0, len(creds))
	for _, cred := range creds {
		ids = append(ids, cred.(string))
	}
	// TODO: We cannot use UpdateClusterWithResponse as api response spec is broken and returns different results.
	resp, err := client.UpdateCluster(ctx, sdk.ClusterId(clusterID), sdk.UpdateClusterJSONRequestBody{
		CloudCredentialsIDs: ids,
		Network:             toClusterNetwork(vpnType),
	})
	if err != nil {
		return diag.FromErr(err)
	}
	defer resp.Body.Close()
	if code := resp.StatusCode; code != http.StatusOK {
		errMsg, _ := ioutil.ReadAll(resp.Body)
		return diag.Errorf("expected status %d, got %d, err=%s", http.StatusOK, code, string(errMsg))
	}
	return nil
}

func updatePolicies(ctx context.Context, client *sdk.ClientWithResponses, clusterID string, policiesConfig map[string]interface{}) diag.Diagnostics {

	resppol, err := client.PoliciesAPIUpsertClusterPoliciesWithResponse(ctx, clusterID, expandAutoscalerPolicies(policiesConfig))
	if checkErr := sdk.CheckGetResponse(resppol, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	return nil
}
