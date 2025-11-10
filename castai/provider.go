package castai

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler"
	"github.com/castai/terraform-provider-castai/castai/sdk/omni"
	"github.com/castai/terraform-provider-castai/castai/sdk/organization_management"
)

type ProviderConfig struct {
	api                          sdk.ClientWithResponsesInterface
	clusterAutoscalerClient      cluster_autoscaler.ClientWithResponsesInterface
	organizationManagementClient organization_management.ClientWithResponsesInterface
	OmniAPI                      *omni.ClientWithResponses
}

func Provider(version string) *schema.Provider {
	p := &schema.Provider{
		TerraformVersion: "1.11",
		Schema: map[string]*schema.Schema{
			"api_url": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsURLWithHTTPS),
				DefaultFunc:      schema.EnvDefaultFunc("CASTAI_API_URL", "https://api.cast.ai"),
				Description:      "CAST.AI API url.",
			},
			"api_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("CASTAI_API_TOKEN", nil),
				Description: "The token used to connect to CAST AI API.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"castai_eks_cluster":                   resourceEKSCluster(),
			"castai_eks_clusterid":                 resourceEKSClusterID(),
			"castai_gke_cluster":                   resourceGKECluster(),
			"castai_gke_cluster_id":                resourceGKEClusterId(),
			"castai_aks_cluster":                   resourceAKSCluster(),
			"castai_autoscaler":                    resourceAutoscaler(),
			"castai_evictor_advanced_config":       resourceEvictionConfig(),
			"castai_node_template":                 resourceNodeTemplate(),
			"castai_rebalancing_schedule":          resourceRebalancingSchedule(),
			"castai_rebalancing_job":               resourceRebalancingJob(),
			"castai_node_configuration":            resourceNodeConfiguration(),
			"castai_node_configuration_default":    resourceNodeConfigurationDefault(),
			"castai_eks_user_arn":                  resourceEKSClusterUserARN(),
			"castai_reservations":                  resourceReservations(),
			"castai_commitments":                   resourceCommitments(),
			"castai_organization_members":          resourceOrganizationMembers(),
			"castai_sso_connection":                resourceSSOConnection(),
			"castai_service_account":               resourceServiceAccount(),
			"castai_service_account_key":           resourceServiceAccountKey(),
			"castai_workload_scaling_policy":       resourceWorkloadScalingPolicy(),
			"castai_workload_scaling_policy_order": resourceWorkloadScalingPolicyOrder(),
			"castai_organization_group":            resourceOrganizationGroup(),
			"castai_role_bindings":                 resourceRoleBindings(),
			"castai_hibernation_schedule":          resourceHibernationSchedule(),
			"castai_security_runtime_rule":         resourceSecurityRuntimeRule(),
			"castai_allocation_group":              resourceAllocationGroup(),
			"castai_enterprise_group":              resourceEnterpriseGroup(),
			"castai_enterprise_role_binding":       resourceEnterpriseRoleBinding(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"castai_eks_settings":                  dataSourceEKSSettings(),
			"castai_gke_user_policies":             dataSourceGKEPolicies(),
			"castai_organization":                  dataSourceOrganization(),
			"castai_rebalancing_schedule":          dataSourceRebalancingSchedule(),
			"castai_hibernation_schedule":          dataSourceHibernationSchedule(),
			"castai_workload_scaling_policy_order": dataSourceWorkloadScalingPolicyOrder(),

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

		if apiToken == "" {
			return nil, diag.Errorf("api_token must be set either in provider configuration or via CASTAI_API_TOKEN environment variable")
		}

		agent := fmt.Sprintf("castai-terraform-provider/%v", version)
		if addUA := os.Getenv("CASTAI_ADDITIONAL_USER_AGENT"); addUA != "" {
			agent = fmt.Sprintf("%s %s", agent, addUA)
		}

		client, err := sdk.CreateClient(apiURL, apiToken, agent)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		clusterAutoscalerClient, err := cluster_autoscaler.CreateClient(apiURL, apiToken, agent)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		organizationManagementClient, err := organization_management.CreateClient(apiURL, apiToken, agent)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		omniClient, err := omni.CreateClient(apiURL, apiToken, agent)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		return &ProviderConfig{
			api:                          client,
			clusterAutoscalerClient:      clusterAutoscalerClient,
			organizationManagementClient: organizationManagementClient,
			OmniAPI:                      omniClient,
		}, nil
	}
}
