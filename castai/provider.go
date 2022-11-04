package castai

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_url": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsURLWithHTTPS),
				DefaultFunc:      schema.EnvDefaultFunc("CASTAI_API_URL", "https://api.cast.ai"),
				Description:      "CAST.AI API url.",
			},
			"api_token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CASTAI_API_TOKEN", nil),
				Description: "The token used to connect to CAST AI API.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"castai_eks_cluster":                resourceEKSCluster(),
			"castai_gke_cluster":                resourceGKECluster(),
			"castai_aks_cluster":                resourceAKSCluster(),
			"castai_autoscaler":                 resourceAutoscaler(),
			"castai_cluster_token":              resourceClusterToken(),
			"castai_node_configuration":         resourceNodeConfiguration(),
			"castai_node_configuration_default": resourceNodeConfigurationDefault(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"castai_eks_settings":      dataSourceEKSSettings(),
			"castai_eks_clusterid":     dataSourceEKSClusterID(),
			"castai_eks_user_arn":      dataSourceEKSClusterUserARN(),
			"castai_gke_user_policies": dataSourceGKEPolicies(),
		},

		ConfigureContextFunc: providerConfigure(),
	}

	return p
}

func providerConfigure() schema.ConfigureContextFunc {
	return func(ctx context.Context, data *schema.ResourceData) (interface{}, diag.Diagnostics) {
		config := Config{
			ApiUrl:   data.Get("api_url").(string),
			ApiToken: data.Get("api_token").(string),
		}

		meta, err := config.configureProvider()
		if err != nil {
			return nil, diag.FromErr(err)
		}

		return meta, nil
	}
}
