package castai

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"gopkg.in/yaml.v2"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func dataSourceCastaiCluster() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCastaiClusterRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"region": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"credentials": {
				Type:     schema.TypeSet,
				Set:      schema.HashString,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"kubeconfig": schemaKubeconfig(),
			PolicyFieldAutoscalerPolicies: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						PolicyFieldEnabled: {
							Type:     schema.TypeBool,
							Computed: true,
						},
						PolicyFieldClusterLimits: {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									PolicyFieldEnabled: {
										Type:     schema.TypeBool,
										Computed: true,
									},
									PolicyFieldClusterLimitsCPU: {
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												PolicyFieldClusterLimitsCPUmax: {
													Type:     schema.TypeInt,
													Computed: true,
												},
												PolicyFieldClusterLimitsCPUmin: {
													Type:     schema.TypeInt,
													Computed: true,
												},
											},
										},
									},
								},
							},
						},
						PolicyFieldNodeDownscaler: {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									PolicyFieldNodeDownscalerEmptyNodes: {
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												PolicyFieldEnabled: {
													Type:     schema.TypeBool,
													Computed: true,
												},
												PolicyFieldNodeDownscalerEmptyNodesDelay: {
													Type:     schema.TypeInt,
													Computed: true,
												},
											},
										},
									},
								},
							},
						},
						PolicyFieldSpotInstances: {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									PolicyFieldSpotInstancesClouds: {
										Type:     schema.TypeList,
										Computed: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									PolicyFieldEnabled: {
										Type:     schema.TypeBool,
										Computed: true,
									},
								},
							},
						},
						PolicyFieldUnschedulablePods: {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									PolicyFieldEnabled: {
										Type:     schema.TypeBool,
										Computed: true,
									},
									PolicyFieldUnschedulablePodsHeadroom: {
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												PolicyFieldEnabled: {
													Type:     schema.TypeBool,
													Computed: true,
												},
												PolicyFieldUnschedulablePodsHeadroomCPUp: {
													Type:     schema.TypeInt,
													Computed: true,
												},
												PolicyFieldUnschedulablePodsHeadroomRAMp: {
													Type:     schema.TypeInt,
													Computed: true,
												},
											},
										},
									},
									PolicyFieldUnschedulablePodsNodeConstraint: {
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												PolicyFieldEnabled: {
													Type:     schema.TypeBool,
													Computed: true,
												},
												PolicyFieldUnschedulablePodsNodeConstraintMaxCPU: {
													Type:     schema.TypeInt,
													Computed: true,
												},
												PolicyFieldUnschedulablePodsNodeConstraintMaxRAM: {
													Type:     schema.TypeInt,
													Computed: true,
												},
												PolicyFieldUnschedulablePodsNodeConstraintMinCPU: {
													Type:     schema.TypeInt,
													Computed: true,
												},
												PolicyFieldUnschedulablePodsNodeConstraintMinRAM: {
													Type:     schema.TypeInt,
													Computed: true,
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

func schemaKubeconfig() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"raw_config": {
					Type:      schema.TypeString,
					Computed:  true,
					Sensitive: true,
				},
				"host": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"client_certificate": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"client_key": {
					Type:      schema.TypeString,
					Computed:  true,
					Sensitive: true,
				},
				"cluster_ca_certificate": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	}
}

func dataSourceCastaiClusterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	id := data.Get("id").(string)

	response, err := client.GetClusterWithResponse(ctx, sdk.ClusterId(id))
	if checkErr := sdk.CheckGetResponse(response, err); checkErr != nil {
		return diag.Errorf("fetching cluster by id=%s: %v", id, checkErr)
	}

	log.Printf("[INFO] found cluster: %v", response.JSON200)

	data.SetId(response.JSON200.Id)
	data.Set("name", response.JSON200.Name)
	data.Set("status", response.JSON200.Status)
	data.Set("region", response.JSON200.Region.Name)
	data.Set("credentials", response.JSON200.CloudCredentialsIDs)

	kubeconfig, err := client.GetClusterKubeconfigWithResponse(ctx, sdk.ClusterId(data.Id()))
	if checkErr := sdk.CheckGetResponse(kubeconfig, err); checkErr == nil {
		log.Printf("[INFO] kubeconfig is available for cluster %q", id)
		kubecfg, err := flattenKubeConfig(string(kubeconfig.Body))
		if err != nil {
			return diag.Errorf("parsing kubeconfig: %v", err)
		}
		data.Set(ClusterFieldKubeconfig, kubecfg)
	} else {
		log.Printf("[WARN] kubeconfig is not available for cluster %q: %v", id, checkErr)
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

func flattenAutoscalerPolicies(readPol *sdk.PoliciesV1Policies) []map[string]interface{} {
	return []map[string]interface{}{
		{
			PolicyFieldEnabled: toBool(readPol.Enabled),
			PolicyFieldClusterLimits: []map[string]interface{}{
				{
					PolicyFieldEnabled: toBool(readPol.ClusterLimits.Enabled),
					PolicyFieldClusterLimitsCPU: []map[string]interface{}{
						{
							PolicyFieldClusterLimitsCPUmax: toInt32(readPol.ClusterLimits.Cpu.MaxCores),
							PolicyFieldClusterLimitsCPUmin: toInt32(readPol.ClusterLimits.Cpu.MinCores),
						},
					},
				},
			},
			PolicyFieldNodeDownscaler: []map[string]interface{}{
				{
					PolicyFieldNodeDownscalerEmptyNodes: []map[string]interface{}{
						{
							PolicyFieldEnabled:                       toBool(readPol.NodeDownscaler.EmptyNodes.Enabled),
							PolicyFieldNodeDownscalerEmptyNodesDelay: toInt32(readPol.NodeDownscaler.EmptyNodes.DelaySeconds),
						},
					},
				},
			},
			PolicyFieldSpotInstances: []map[string]interface{}{
				{
					PolicyFieldEnabled:             toBool(readPol.SpotInstances.Enabled),
					PolicyFieldSpotInstancesClouds: toCloudsStringSlice(readPol.SpotInstances.Clouds),
				},
			},
			PolicyFieldUnschedulablePods: []map[string]interface{}{
				{
					PolicyFieldEnabled: toBool(readPol.UnschedulablePods.Enabled),
					PolicyFieldUnschedulablePodsHeadroom: []map[string]interface{}{
						{
							PolicyFieldEnabled:                       toBool(readPol.UnschedulablePods.Headroom.Enabled),
							PolicyFieldUnschedulablePodsHeadroomCPUp: toInt32(readPol.UnschedulablePods.Headroom.CpuPercentage),
							PolicyFieldUnschedulablePodsHeadroomRAMp: toInt32(readPol.UnschedulablePods.Headroom.MemoryPercentage),
						},
					},
					PolicyFieldUnschedulablePodsNodeConstraint: []map[string]interface{}{
						{
							PolicyFieldEnabled: toBool(readPol.UnschedulablePods.NodeConstraints.Enabled),
							PolicyFieldUnschedulablePodsNodeConstraintMaxCPU: toInt32(readPol.UnschedulablePods.NodeConstraints.MaxCpuCores),
							PolicyFieldUnschedulablePodsNodeConstraintMaxRAM: toInt32(readPol.UnschedulablePods.NodeConstraints.MaxRamMib) / 1024.0,
							PolicyFieldUnschedulablePodsNodeConstraintMinCPU: toInt32(readPol.UnschedulablePods.NodeConstraints.MinCpuCores),
							PolicyFieldUnschedulablePodsNodeConstraintMinRAM: toInt32(readPol.UnschedulablePods.NodeConstraints.MinRamMib) / 1024.0,
						},
					},
				},
			},
		},
	}
}

func flattenKubeConfig(rawKubeconfig string) ([]interface{}, error) {
	var cfg kubernetesConfig
	err := yaml.Unmarshal([]byte(rawKubeconfig), &cfg)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling raw kubeconfig: %w", err)
	}
	if len(cfg.Clusters) == 0 {
		return nil, errors.New("kubeconfig should contain cluster")
	}
	if len(cfg.Users) == 0 {
		return nil, errors.New("kubeconfig should contain user")
	}

	res := map[string]interface{}{
		"raw_config":             rawKubeconfig,
		"host":                   cfg.Clusters[0].Cluster.Server,
		"cluster_ca_certificate": cfg.Clusters[0].Cluster.CertificateAuthorityData,
		"client_certificate":     cfg.Users[0].User.ClientCertificateData,
		"client_key":             cfg.Users[0].User.ClientKeyData,
	}

	return []interface{}{res}, nil
}

type kubernetesConfig struct {
	APIVersion     string                    `yaml:"apiVersion"`
	Kind           string                    `yaml:"kind"`
	Clusters       []kubernetesConfigCluster `yaml:"clusters"`
	Contexts       []kubernetesConfigContext `yaml:"contexts"`
	CurrentContext string                    `yaml:"current-context"`
	Users          []kubernetesConfigUser    `yaml:"users"`
}

type kubernetesConfigCluster struct {
	Cluster kubernetesConfigClusterData `yaml:"cluster"`
	Name    string                      `yaml:"name"`
}
type kubernetesConfigClusterData struct {
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
	Server                   string `yaml:"server"`
}

type kubernetesConfigContext struct {
	Context kubernetesConfigContextData `yaml:"context"`
	Name    string                      `yaml:"name"`
}

type kubernetesConfigContextData struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

type kubernetesConfigUser struct {
	Name string                   `yaml:"name"`
	User kubernetesConfigUserData `yaml:"user"`
}

type kubernetesConfigUserData struct {
	ClientKeyData         string `yaml:"client-key-data,omitempty"`
	ClientCertificateData string `yaml:"client-certificate-data,omitempty"`
	Token                 string `yaml:"token"`
}
