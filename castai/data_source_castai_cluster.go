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
			return nil
		}
		data.Set(ClusterFieldKubeconfig, kubecfg)
	} else {
		log.Printf("[WARN] kubeconfig is not available for cluster %q: %v", id, checkErr)
		data.Set(ClusterFieldKubeconfig, []interface{}{})
	}

	return nil
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
