package castai

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*omniClusterDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*omniClusterDataSource)(nil)
)

type omniClusterDataSource struct {
	client *ProviderConfig
}

type omniClusterDataSourceModel struct {
	ID               types.String           `tfsdk:"id"`
	OrganizationID   types.String           `tfsdk:"organization_id"`
	ClusterID        types.String           `tfsdk:"cluster_id"`
	Name             types.String           `tfsdk:"name"`
	State            types.String           `tfsdk:"state"`
	ProviderType     types.String           `tfsdk:"provider_type"`
	ServiceAccountID types.String           `tfsdk:"service_account_id"`
	CastaiOidcConfig *castaiOidcConfigModel `tfsdk:"castai_oidc_config"`
}

type castaiOidcConfigModel struct {
	GcpServiceAccountEmail    types.String `tfsdk:"gcp_service_account_email"`
	GcpServiceAccountUniqueID types.String `tfsdk:"gcp_service_account_unique_id"`
}

func newOmniClusterDataSource() datasource.DataSource {
	return &omniClusterDataSource{}
}

func (d *omniClusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_omni_cluster"
}

func (d *omniClusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieve information about a CAST AI Omni cluster",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Cluster ID (same as cluster_id)",
			},
			"organization_id": schema.StringAttribute{
				Required:    true,
				Description: "CAST AI organization ID",
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "CAST AI cluster ID",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the cluster",
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "State of the cluster on API level",
			},
			"provider_type": schema.StringAttribute{
				Computed:    true,
				Description: "Provider type of the cluster (e.g. GKE, EKS)",
			},
			"service_account_id": schema.StringAttribute{
				Computed:    true,
				Description: "CAST AI service account ID associated with OMNI operations",
			},
			"castai_oidc_config": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "CAST AI OIDC configuration for service account impersonation",
				Attributes: map[string]schema.Attribute{
					"gcp_service_account_email": schema.StringAttribute{
						Computed:    true,
						Description: "CAST AI GCP service account email for impersonation",
					},
					"gcp_service_account_unique_id": schema.StringAttribute{
						Computed:    true,
						Description: "CAST AI GCP service account unique ID for impersonation",
					},
				},
			},
		},
	}
}

func (d *omniClusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderConfig, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *omniClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data omniClusterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.client.omniAPI
	organizationID := data.OrganizationID.ValueString()
	clusterID := data.ClusterID.ValueString()

	apiResp, err := client.ClustersAPIGetClusterWithResponse(ctx, organizationID, clusterID, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read omni cluster", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to read omni cluster",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	cluster := apiResp.JSON200

	data.ID = types.StringValue(clusterID)
	data.Name = types.StringPointerValue(cluster.Name)
	data.ServiceAccountID = types.StringPointerValue(cluster.ServiceAccountId)

	if cluster.State != nil {
		data.State = types.StringValue(string(*cluster.State))
	}
	if cluster.ProviderType != nil {
		data.ProviderType = types.StringValue(string(*cluster.ProviderType))
	}

	if cluster.CastaiOidcConfig != nil {
		data.CastaiOidcConfig = &castaiOidcConfigModel{
			GcpServiceAccountEmail:    types.StringPointerValue(cluster.CastaiOidcConfig.GcpServiceAccountEmail),
			GcpServiceAccountUniqueID: types.StringPointerValue(cluster.CastaiOidcConfig.GcpServiceAccountUniqueId),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
