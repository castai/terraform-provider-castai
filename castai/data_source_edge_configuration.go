package castai

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	CRI            *criConfigurationModel `tfsdk:"cri"`
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
			"cri": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "CRI (Container Runtime Interface) configuration",
				Attributes: map[string]schema.Attribute{
					"socket": schema.StringAttribute{
						Computed:    true,
						Description: "Path to an existing CRI socket",
					},
				},
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
	data.UserDataBase64 = normalizeStringPtr(config.UserDataBase64)

	// AWS configuration
	if config.Aws != nil {
		data.Aws = &awsConfigurationModel{
			ImageID:         types.StringPointerValue(config.Aws.ImageId),
			BootDiskSizeGiB: types.Int64Null(),
			Tags:            types.MapNull(types.StringType),
		}
		if config.Aws.BootDiskSizeGib != nil {
			data.Aws.BootDiskSizeGiB = types.Int64Value(int64(*config.Aws.BootDiskSizeGib))
		}
		if config.Aws.Tags != nil && len(*config.Aws.Tags) > 0 {
			tags, diags := types.MapValueFrom(ctx, types.StringType, *config.Aws.Tags)
			if !diags.HasError() {
				data.Aws.Tags = tags
			} else {
				resp.Diagnostics.Append(diags...)
			}
		}
	}

	// GCP configuration
	if config.Gcp != nil {
		data.Gcp = &gcpConfigurationModel{
			ImageID:         types.StringPointerValue(config.Gcp.ImageId),
			BootDiskSizeGiB: types.Int64Null(),
			Labels:          types.MapNull(types.StringType),
		}
		if config.Gcp.BootDiskSizeGib != nil {
			data.Gcp.BootDiskSizeGiB = types.Int64Value(int64(*config.Gcp.BootDiskSizeGib))
		}
		if config.Gcp.Labels != nil && len(*config.Gcp.Labels) > 0 {
			labels, diags := types.MapValueFrom(ctx, types.StringType, *config.Gcp.Labels)
			if !diags.HasError() {
				data.Gcp.Labels = labels
			} else {
				resp.Diagnostics.Append(diags...)
			}
		}
	}

	// OCI configuration
	if config.Oci != nil {
		data.Oci = &ociConfigurationModel{
			ImageID:         types.StringPointerValue(config.Oci.ImageId),
			BootDiskSizeGiB: types.Int64Null(),
			Tags:            types.MapNull(types.StringType),
		}
		if config.Oci.BootDiskSizeGib != nil {
			data.Oci.BootDiskSizeGiB = types.Int64Value(int64(*config.Oci.BootDiskSizeGib))
		}
		if config.Oci.Tags != nil && len(*config.Oci.Tags) > 0 {
			tags, diags := types.MapValueFrom(ctx, types.StringType, *config.Oci.Tags)
			if !diags.HasError() {
				data.Oci.Tags = tags
			} else {
				resp.Diagnostics.Append(diags...)
			}
		}
	}

	// CRI configuration
	if config.Cri != nil {
		data.CRI = &criConfigurationModel{
			Socket: types.StringPointerValue(config.Cri.Socket),
		}
	}

	// Custom configuration
	if config.Custom != nil && len(*config.Custom) > 0 {
		custom, diags := types.MapValueFrom(ctx, types.StringType, *config.Custom)
		if !diags.HasError() {
			data.Custom = custom
		} else {
			resp.Diagnostics.Append(diags...)
		}
	} else {
		data.Custom = types.MapNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
