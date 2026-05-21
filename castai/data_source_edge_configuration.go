package castai

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni"
)

var (
	_ datasource.DataSource              = (*edgeConfigurationDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*edgeConfigurationDataSource)(nil)
)

// edgeConfigurationDataSource retrieves information about a specific CAST AI edge configuration.
type edgeConfigurationDataSource struct {
	client *ProviderConfig
}

// edgeConfigurationSingleDataModel describes a single edge configuration data source.
type edgeConfigurationSingleDataModel struct {
	ID             types.String           `tfsdk:"id"`
	OrganizationID types.String           `tfsdk:"organization_id"`
	ClusterID      types.String           `tfsdk:"cluster_id"`
	EdgeLocationID types.String           `tfsdk:"edge_location_id"`
	Name           types.String           `tfsdk:"name"`
	Default        types.Bool             `tfsdk:"default"`
	UserDataBase64 types.String           `tfsdk:"user_data_base64"`
	Aws            *awsConfigurationModel `tfsdk:"aws"`
	Gcp            *gcpConfigurationModel `tfsdk:"gcp"`
	Oci            *ociConfigurationModel `tfsdk:"oci"`
	Custom         types.Map              `tfsdk:"custom"`
}

// findEdgeConfigurationByName searches for an edge configuration by name in the list response.
func findEdgeConfigurationByName(configs *[]omni.EdgeConfiguration, name string) *omni.EdgeConfiguration {
	if configs == nil {
		return nil
	}
	for _, config := range *configs {
		if config.Name == name {
			return &config
		}
	}
	return nil
}

func newEdgeConfigurationDataSource() datasource.DataSource {
	return &edgeConfigurationDataSource{}
}

func (d *edgeConfigurationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_configuration"
}

func (d *edgeConfigurationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieve information about a CAST AI edge configuration",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the edge configuration",
			},
			"organization_id": schema.StringAttribute{
				Required:    true,
				Description: "CAST AI organization ID",
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "CAST AI cluster ID",
			},
			"edge_location_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the edge location",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the edge configuration",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"default": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is the default configuration",
			},
			"user_data_base64": schema.StringAttribute{
				Computed:    true,
				Description: "Base64 encoded user data for edge bootstrap",
			},
			"aws": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "AWS specific configuration",
				Attributes: map[string]schema.Attribute{
					"image_id": schema.StringAttribute{
						Computed:    true,
						Description: "AWS AMI ID or name filter for edge creation",
					},
					"boot_disk_size_gib": schema.Int64Attribute{
						Computed:    true,
						Description: "Boot disk size in GiB",
					},
					"tags": schema.MapAttribute{
						Computed:    true,
						Description: "Instance/VM tags",
						ElementType: types.StringType,
					},
				},
			},
			"gcp": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "GCP specific configuration",
				Attributes: map[string]schema.Attribute{
					"image_id": schema.StringAttribute{
						Computed:    true,
						Description: "GCP image ID or family for edge creation",
					},
					"boot_disk_size_gib": schema.Int64Attribute{
						Computed:    true,
						Description: "Boot disk size in GiB",
					},
					"labels": schema.MapAttribute{
						Computed:    true,
						Description: "Instance/VM labels",
						ElementType: types.StringType,
					},
				},
			},
			"oci": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "OCI specific configuration",
				Attributes: map[string]schema.Attribute{
					"image_id": schema.StringAttribute{
						Computed:    true,
						Description: "OCI image ID or name filter for edge creation",
					},
					"boot_disk_size_gib": schema.Int64Attribute{
						Computed:    true,
						Description: "Boot disk size in GiB",
					},
					"tags": schema.MapAttribute{
						Computed:    true,
						Description: "Instance/VM tags",
						ElementType: types.StringType,
					},
				},
			},
			"custom": schema.MapAttribute{
				Computed:    true,
				Description: "Custom cloud specific configuration tags",
				ElementType: types.StringType,
			},
		},
	}
}

func (d *edgeConfigurationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *edgeConfigurationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data edgeConfigurationSingleDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.client.omniAPI
	organizationID := data.OrganizationID.ValueString()
	clusterID := data.ClusterID.ValueString()
	edgeLocationID := data.EdgeLocationID.ValueString()
	configName := data.Name.ValueString()

	// Use ListEdgeConfigurations API and filter by name
	params := &omni.EdgeConfigurationsAPIListEdgeConfigurationsParams{
		EdgeLocationId: &edgeLocationID,
	}

	apiResp, err := client.EdgeConfigurationsAPIListEdgeConfigurationsWithResponse(ctx, organizationID, clusterID, params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list edge configurations", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to list edge configurations",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	if apiResp.JSON200 == nil || apiResp.JSON200.Items == nil {
		resp.Diagnostics.AddError(
			"Failed to list edge configurations",
			"API response body is empty",
		)
		return
	}

	// Find configuration by name
	config := findEdgeConfigurationByName(apiResp.JSON200.Items, configName)
	if config == nil {
		resp.Diagnostics.AddError(
			"Edge configuration not found",
			fmt.Sprintf("No edge configuration found with name: %s", configName),
		)
		return
	}

	// Populate the model
	data.ID = types.StringPointerValue(config.Id)
	data.EdgeLocationID = types.StringPointerValue(config.EdgeLocationId)

	if config.Default != nil {
		data.Default = types.BoolValue(*config.Default)
	}
	if config.UserDataBase64 != nil {
		data.UserDataBase64 = types.StringValue(*config.UserDataBase64)
	}

	// AWS configuration
	if config.Aws != nil {
		data.Aws = &awsConfigurationModel{
			ImageID:         types.StringPointerValue(config.Aws.ImageId),
			BootDiskSizeGiB: int32ToInt64(config.Aws.BootDiskSizeGib),
			Tags:            stringMapToTF(config.Aws.Tags),
		}
	}

	// GCP configuration
	if config.Gcp != nil {
		data.Gcp = &gcpConfigurationModel{
			ImageID:         types.StringPointerValue(config.Gcp.ImageId),
			BootDiskSizeGiB: int32ToInt64(config.Gcp.BootDiskSizeGib),
			Labels:          stringMapToTF(config.Gcp.Labels),
		}
	}

	// OCI configuration
	if config.Oci != nil {
		data.Oci = &ociConfigurationModel{
			ImageID:         types.StringPointerValue(config.Oci.ImageId),
			BootDiskSizeGiB: int32ToInt64(config.Oci.BootDiskSizeGib),
			Tags:            stringMapToTF(config.Oci.Tags),
		}
	}

	// Custom configuration
	if config.Custom != nil {
		data.Custom = customConfigToTF(config.Custom)
	} else {
		data.Custom = types.MapNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// int32ToInt64 converts *int32 to types.Int64
func int32ToInt64(v *int32) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}

// stringMapToTF converts map[string]string to types.Map
func stringMapToTF(m *map[string]string) types.Map {
	if m == nil {
		return types.MapNull(types.StringType)
	}

	elements := make(map[string]attr.Value, len(*m))
	for k, v := range *m {
		elements[k] = types.StringValue(v)
	}

	result, diags := types.MapValue(types.StringType, elements)
	if diags.HasError() {
		return types.MapNull(types.StringType)
	}
	return result
}

// customConfigToTF converts map[string]interface{} to types.Map
func customConfigToTF(m *omni.CustomCloudConfiguration) types.Map {
	if m == nil || *m == nil {
		return types.MapNull(types.StringType)
	}

	if len(*m) == 0 {
		return types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	elements := make(map[string]attr.Value, len(*m))
	for k, v := range *m {
		if strVal, ok := v.(string); ok {
			elements[k] = types.StringValue(strVal)
		} else if v == nil {
			elements[k] = types.StringNull()
		} else {
			elements[k] = types.StringValue(fmt.Sprintf("%v", v))
		}
	}

	result, diags := types.MapValue(types.StringType, elements)
	if diags.HasError() {
		return types.MapNull(types.StringType)
	}
	return result
}
