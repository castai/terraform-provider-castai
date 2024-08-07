package castai

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

type ProviderConfig struct {
	api *sdk.ClientWithResponses
}

func Provider(version string) *schema.Provider {
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
			"castai_eks_clusterid":              resourceEKSClusterID(),
			"castai_gke_cluster":                resourceGKECluster(),
			"castai_aks_cluster":                resourceAKSCluster(),
			"castai_autoscaler":                 resourceAutoscaler(),
			"castai_evictor_advanced_config":    resourceEvictionConfig(),
			"castai_node_template":              resourceNodeTemplate(),
			"castai_rebalancing_schedule":       resourceRebalancingSchedule(),
			"castai_rebalancing_job":            resourceRebalancingJob(),
			"castai_node_configuration":         resourceNodeConfiguration(),
			"castai_node_configuration_default": resourceNodeConfigurationDefault(),
			"castai_eks_user_arn":               resourceEKSClusterUserARN(),
			"castai_reservations":               resourceReservations(),
			"castai_commitments":                resourceCommitments(),
			"castai_organization_members":       resourceOrganizationMembers(),
			"castai_sso_connection":             resourceSSOConnection(),
			"castai_workload_scaling_policy":    resourceWorkloadScalingPolicy(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"castai_eks_settings":      dataSourceEKSSettings(),
			"castai_gke_user_policies": dataSourceGKEPolicies(),
			"castai_organization":      dataSourceOrganization(),

			// TODO: remove in next major release
			"castai_eks_user_arn": dataSourceEKSClusterUserARN(),
		},

		ConfigureContextFunc: providerConfigure(version),
	}

	return p
}

func providerConfigure(version string) schema.ConfigureContextFunc {
	return func(ctx context.Context, data *schema.ResourceData) (interface{}, diag.Diagnostics) {
		apiURL := data.Get("api_url").(string)
		apiToken := data.Get("api_token").(string)

		agent := fmt.Sprintf("castai-terraform-provider/%v", version)
		if addUA := os.Getenv("CASTAI_ADDITIONAL_USER_AGENT"); addUA != "" {
			agent = fmt.Sprintf("%s %s", agent, addUA)
		}

		client, err := sdk.CreateClient(apiURL, apiToken, agent)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		return &ProviderConfig{api: client}, nil
	}
}
