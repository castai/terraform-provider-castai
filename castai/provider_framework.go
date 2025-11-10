package castai

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	tfprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler"
	omnisdk "github.com/castai/terraform-provider-castai/castai/sdk/omni"
	"github.com/castai/terraform-provider-castai/castai/sdk/organization_management"
)

var _ tfprovider.Provider = (*frameworkProvider)(nil)

type frameworkProvider struct {
	version string
}

type frameworkProviderModel struct {
	APIUrl   types.String `tfsdk:"api_url"`
	APIToken types.String `tfsdk:"api_token"`
}

func NewFrameworkProvider(version string) tfprovider.Provider {
	return &frameworkProvider{
		version: version,
	}
}

func (p *frameworkProvider) Metadata(_ context.Context, _ tfprovider.MetadataRequest, resp *tfprovider.MetadataResponse) {
	resp.TypeName = "castai"
}

func (p *frameworkProvider) Schema(_ context.Context, _ tfprovider.SchemaRequest, resp *tfprovider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Required:    true,
				Description: "CAST.AI API url.",
			},
			"api_token": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The token used to connect to CAST AI API.",
			},
		},
	}
}

func (p *frameworkProvider) Configure(ctx context.Context, req tfprovider.ConfigureRequest, resp *tfprovider.ConfigureResponse) {
	var config frameworkProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiURL := config.APIUrl.ValueString()
	if apiURL == "" {
		apiURL = os.Getenv("CASTAI_API_URL")
		if apiURL == "" {
			apiURL = "https://api.cast.ai"
		}
	}

	apiToken := config.APIToken.ValueString()
	if apiToken == "" {
		apiToken = os.Getenv("CASTAI_API_TOKEN")
	}

	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API Token Configuration",
			"The provider cannot create the CAST AI API client as there is a missing or empty value for the API token. "+
				"Set the api_token value in the configuration or use the CASTAI_API_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
		return
	}

	agent := fmt.Sprintf("castai-terraform-provider/%v", p.version)
	if addUA := os.Getenv("CASTAI_ADDITIONAL_USER_AGENT"); addUA != "" {
		agent = fmt.Sprintf("%s %s", agent, addUA)
	}

	client, err := sdk.CreateClient(apiURL, apiToken, agent)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API client", err.Error())
		return
	}

	clusterAutoscalerClient, err := cluster_autoscaler.CreateClient(apiURL, apiToken, agent)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create cluster autoscaler client", err.Error())
		return
	}

	organizationManagementClient, err := organization_management.CreateClient(apiURL, apiToken, agent)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create organization management client", err.Error())
		return
	}

	omniClient, err := omnisdk.CreateClient(apiURL, apiToken, agent)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create omni client", err.Error())
		return
	}

	providerConfig := &ProviderConfig{
		api:                          client,
		clusterAutoscalerClient:      clusterAutoscalerClient,
		organizationManagementClient: organizationManagementClient,
		OmniAPI:                      omniClient,
	}

	resp.DataSourceData = providerConfig
	resp.ResourceData = providerConfig
}

func (p *frameworkProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewEdgeLocationResource,
		NewOmniClusterResource,
	}
}

func (p *frameworkProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
