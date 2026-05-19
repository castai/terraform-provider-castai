package castai

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni"
)

var (
	_ resource.Resource                     = (*edgeConfigurationResource)(nil)
	_ resource.ResourceWithConfigure        = (*edgeConfigurationResource)(nil)
	_ resource.ResourceWithImportState      = (*edgeConfigurationResource)(nil)
	_ resource.ResourceWithConfigValidators = (*edgeConfigurationResource)(nil)
)

type edgeConfigurationResource struct {
	client *ProviderConfig
}

type edgeConfigurationModel struct {
	ID             types.String              `tfsdk:"id"`
	OrganizationID types.String              `tfsdk:"organization_id"`
	ClusterID      types.String              `tfsdk:"cluster_id"`
	EdgeLocationID types.String              `tfsdk:"edge_location_id"`
	Name           types.String              `tfsdk:"name"`
	Default        types.Bool                `tfsdk:"default"`
	UserDataBase64 types.String              `tfsdk:"user_data_base64"`
	GCP            *gcpConfigurationModel    `tfsdk:"gcp"`
	AWS            *awsConfigurationModel    `tfsdk:"aws"`
	OCI            *ociConfigurationModel    `tfsdk:"oci"`
	Custom         *customConfigurationModel `tfsdk:"custom"`
}

type gcpConfigurationModel struct {
	Labels          types.Map    `tfsdk:"labels"`
	ImageID         types.String `tfsdk:"image_id"`
	BootDiskSizeGiB types.Int64  `tfsdk:"boot_disk_size_gib"`
}

type awsConfigurationModel struct {
	Tags            types.Map    `tfsdk:"tags"`
	ImageID         types.String `tfsdk:"image_id"`
	BootDiskSizeGiB types.Int64  `tfsdk:"boot_disk_size_gib"`
}

type ociConfigurationModel struct {
	Tags            types.Map    `tfsdk:"tags"`
	ImageID         types.String `tfsdk:"image_id"`
	BootDiskSizeGiB types.Int64  `tfsdk:"boot_disk_size_gib"`
}

type customConfigurationModel struct {
	Custom types.Map `tfsdk:"custom"`
}

func newEdgeConfigurationResource() resource.Resource {
	return &edgeConfigurationResource{}
}

func (r *edgeConfigurationResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("gcp"),
			path.MatchRoot("aws"),
			path.MatchRoot("oci"),
			path.MatchRoot("custom"),
		),
	}
}

func (r *edgeConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_configuration"
}

func (r *edgeConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage CAST AI Edge Configuration for edge computing deployments",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Edge configuration ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:    true,
				Description: "CAST AI organization ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "CAST AI cluster ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"edge_location_id": schema.StringAttribute{
				Required:    true,
				Description: "Edge location ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the edge configuration",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"default": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this edge configuration is the default one",
			},
			"user_data_base64": schema.StringAttribute{
				Optional:    true,
				Description: "Base64 encoded user data to run on the edge as part of bootstrap. The payload must start with either `#cloud-config` (cloud-init YAML) or `#!` (shell script with a shebang)",
			},

			"gcp": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "GCP specific configuration",
				Attributes: map[string]schema.Attribute{
					"labels": schema.MapAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Instance/VM labels",
					},
					"image_id": schema.StringAttribute{
						Optional:    true,
						Description: "Exact image ID (for example 'projects/castai-public-339919/global/images/ubuntu-2404-lts-amd64-cuda') or image family (for example `projects/ubuntu-os-cloud/global/images/family/ubuntu-2404-lts-amd64`) to be used for edge creation",
					},
					"boot_disk_size_gib": schema.Int64Attribute{
						Optional:    true,
						Description: "Boot disk size in GiB",
					},
				},
			},
			"aws": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "AWS specific configuration",
				Attributes: map[string]schema.Attribute{
					"tags": schema.MapAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Instance/VM tags",
					},
					"image_id": schema.StringAttribute{
						Optional:    true,
						Description: "ImageID to be used for edge creation. It can be an AMI ID (for example 'ami-0abcdef1234567890') or a name filter (for example 'al2023-ami-ecs-hvm-*')",
					},
					"boot_disk_size_gib": schema.Int64Attribute{
						Optional:    true,
						Description: "Boot disk size in GiB",
					},
				},
			},
			"oci": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "OCI specific configuration",
				Attributes: map[string]schema.Attribute{
					"tags": schema.MapAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Instance/VM tags",
					},
					"image_id": schema.StringAttribute{
						Optional:    true,
						Description: "ImageID to be used for edge creation. It can be an AMI ID (for example 'ami-0abcdef1234567890') or a name filter (for example 'al2023-ami-ecs-hvm-*')",
					},
					"boot_disk_size_gib": schema.Int64Attribute{
						Optional:    true,
						Description: "Boot disk size in GiB",
					},
				},
			},
			"custom": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Custom cloud specific configuration",
				Attributes: map[string]schema.Attribute{
					"custom": schema.MapAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Custom cloud configuration as key-value pairs",
					},
				},
			},
		},
	}
}

func (r *edgeConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderConfig, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *edgeConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan edgeConfigurationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.omniAPI
	organizationID := r.getOrganizationID(plan.OrganizationID)
	clusterID := plan.ClusterID.ValueString()
	edgeLocationID := plan.EdgeLocationID.ValueString()

	tflog.Info(ctx, "Creating edge configuration",
		map[string]interface{}{
			"organization_id":  organizationID,
			"cluster_id":       clusterID,
			"edge_location_id": edgeLocationID,
		},
	)

	createReq := r.edgeConfigurationToSDK(plan)

	apiResp, err := client.EdgeConfigurationsAPICreateEdgeConfigurationWithResponse(ctx, organizationID, clusterID, edgeLocationID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create edge configuration", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to create edge configuration",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	state := r.edgeConfigurationToTFModel(apiResp.JSON200, plan.OrganizationID, plan.ClusterID)
	state.EdgeLocationID = plan.EdgeLocationID

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *edgeConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state edgeConfigurationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.omniAPI
	organizationID := r.getOrganizationID(state.OrganizationID)
	clusterID := state.ClusterID.ValueString()
	edgeLocationID := state.EdgeLocationID.ValueString()

	tflog.Info(ctx, "Reading edge configuration",
		map[string]interface{}{
			"organization_id":  organizationID,
			"cluster_id":       clusterID,
			"edge_location_id": edgeLocationID,
			"configuration_id": state.ID.ValueString(),
		},
	)

	apiResp, err := client.EdgeConfigurationsAPIGetEdgeConfigurationWithResponse(ctx, organizationID, clusterID, edgeLocationID, state.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read edge configuration", err.Error())
		return
	}

	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to read edge configuration",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	plan := r.edgeConfigurationToTFModel(apiResp.JSON200, state.OrganizationID, state.ClusterID)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *edgeConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan edgeConfigurationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.omniAPI
	organizationID := r.getOrganizationID(plan.OrganizationID)
	clusterID := plan.ClusterID.ValueString()
	edgeLocationID := plan.EdgeLocationID.ValueString()

	tflog.Info(ctx, "Updating edge configuration",
		map[string]interface{}{
			"organization_id":  organizationID,
			"cluster_id":       clusterID,
			"edge_location_id": edgeLocationID,
			"configuration_id": plan.ID.ValueString(),
		},
	)

	updateReq := r.edgeConfigurationUpdateToSDK(plan)

	apiResp, err := client.EdgeConfigurationsAPIUpdateEdgeConfigurationWithResponse(ctx, organizationID, clusterID, edgeLocationID, plan.ID.ValueString(), nil, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update edge configuration", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to update edge configuration",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	state := r.edgeConfigurationToTFModel(apiResp.JSON200, plan.OrganizationID, plan.ClusterID)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *edgeConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state edgeConfigurationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.omniAPI
	organizationID := r.getOrganizationID(state.OrganizationID)
	clusterID := state.ClusterID.ValueString()
	edgeLocationID := state.EdgeLocationID.ValueString()
	configurationID := state.ID.ValueString()

	tflog.Info(ctx, "Deleting edge configuration - state values",
		map[string]interface{}{
			"state_organization_id":    state.OrganizationID.ValueString(),
			"state_cluster_id":         state.ClusterID.ValueString(),
			"state_edge_location_id":   state.EdgeLocationID.ValueString(),
			"state_configuration_id":   state.ID.ValueString(),
			"computed_organization_id": organizationID,
		},
	)

	apiResp, err := client.EdgeConfigurationsAPIDeleteEdgeConfigurationWithResponse(ctx, organizationID, clusterID, edgeLocationID, configurationID)

	tflog.Info(ctx, "Edge configuration delete response",
		map[string]interface{}{
			"status_code": apiResp.StatusCode(),
			"body":        string(apiResp.Body),
			"error":       err,
		},
	)

	if err != nil {
		resp.Diagnostics.AddError("Failed to delete edge configuration", err.Error())
		return
	}

	if apiResp.StatusCode() == http.StatusNotFound {
		tflog.Info(ctx, "Edge configuration not found, treating as deleted")
		return
	}

	if apiResp.StatusCode() != http.StatusOK && apiResp.StatusCode() != http.StatusNoContent {
		resp.Diagnostics.AddError(
			"Failed to delete edge configuration",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	tflog.Info(ctx, "Edge configuration deleted successfully")
}

func (r *edgeConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: organization_id/cluster_id/edge_location_id/id
	ids := strings.Split(req.ID, "/")
	if len(ids) != 4 || ids[0] == "" || ids[1] == "" || ids[2] == "" || ids[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("expected: organization_id/cluster_id/edge_location_id/id, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), ids[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), ids[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("edge_location_id"), ids[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ids[3])...)
}

func (r *edgeConfigurationResource) getOrganizationID(organizationID types.String) string {
	if !organizationID.IsNull() {
		return organizationID.ValueString()
	}
	return r.client.organizationID
}

func (r *edgeConfigurationResource) edgeConfigurationToTFModel(config *omni.EdgeConfiguration, organizationID types.String, clusterID types.String) *edgeConfigurationModel {
	state := &edgeConfigurationModel{
		ID:             types.StringValue(lo.FromPtr(config.Id)),
		OrganizationID: organizationID,
		ClusterID:      clusterID,
		Name:           types.StringValue(config.Name),
		EdgeLocationID: types.StringValue(lo.FromPtr(config.EdgeLocationId)),
		Default:        types.BoolValue(lo.FromPtr(config.Default)),
		UserDataBase64: types.StringValue(lo.FromPtr(config.UserDataBase64)),
	}

	if config.Gcp != nil {
		state.GCP = r.toGCPConfigurationModel(config.Gcp)
	}
	if config.Aws != nil {
		state.AWS = r.toAWSConfigurationModel(config.Aws)
	}
	if config.Oci != nil {
		state.OCI = r.toOCIConfigurationModel(config.Oci)
	}
	if config.Custom != nil {
		state.Custom = r.toCustomConfigurationModel(config.Custom)
	}

	return state
}

func (r *edgeConfigurationResource) edgeConfigurationToSDK(plan edgeConfigurationModel) omni.EdgeConfigurationsAPICreateEdgeConfigurationJSONRequestBody {
	createReq := omni.EdgeConfiguration{
		Name:           plan.Name.ValueString(),
		EdgeLocationId: lo.ToPtr(plan.EdgeLocationID.ValueString()),
	}

	createReq.Default = lo.ToPtr(false)

	if !plan.UserDataBase64.IsNull() {
		createReq.UserDataBase64 = lo.ToPtr(plan.UserDataBase64.ValueString())
	}

	if plan.GCP != nil {
		createReq.Gcp = r.toGCPConfiguration(plan.GCP)
	}
	if plan.AWS != nil {
		createReq.Aws = r.toAWSConfiguration(plan.AWS)
	}
	if plan.OCI != nil {
		createReq.Oci = r.toOCIConfiguration(plan.OCI)
	}
	if plan.Custom != nil {
		createReq.Custom = r.toCustomConfiguration(plan.Custom)
	}

	return createReq
}

func (r *edgeConfigurationResource) edgeConfigurationUpdateToSDK(plan edgeConfigurationModel) omni.EdgeConfigurationsAPIUpdateEdgeConfigurationJSONRequestBody {
	updateReq := omni.EdgeConfigurationUpdate{
		Name: lo.ToPtr(plan.Name.ValueString()),
	}

	updateReq.Default = lo.ToPtr(false)

	if !plan.UserDataBase64.IsNull() {
		updateReq.UserDataBase64 = lo.ToPtr(plan.UserDataBase64.ValueString())
	}

	if plan.GCP != nil {
		updateReq.Gcp = r.toGCPConfiguration(plan.GCP)
	}
	if plan.AWS != nil {
		updateReq.Aws = r.toAWSConfiguration(plan.AWS)
	}
	if plan.OCI != nil {
		updateReq.Oci = r.toOCIConfiguration(plan.OCI)
	}
	if plan.Custom != nil {
		updateReq.Custom = r.toCustomConfiguration(plan.Custom)
	}

	return updateReq
}

func (r *edgeConfigurationResource) toGCPConfiguration(plan *gcpConfigurationModel) *omni.GCPConfiguration {
	if plan == nil {
		return nil
	}

	config := &omni.GCPConfiguration{}

	if !plan.ImageID.IsNull() {
		config.ImageId = lo.ToPtr(plan.ImageID.ValueString())
	}

	if !plan.BootDiskSizeGiB.IsNull() {
		config.BootDiskSizeGib = lo.ToPtr(int32(plan.BootDiskSizeGiB.ValueInt64()))
	}

	if !plan.Labels.IsNull() {
		labels := make(map[string]string)
		plan.Labels.ElementsAs(context.Background(), &labels, false)
		config.Labels = &labels
	}

	return config
}

func (r *edgeConfigurationResource) toGCPConfigurationModel(config *omni.GCPConfiguration) *gcpConfigurationModel {
	if config == nil {
		return nil
	}

	model := &gcpConfigurationModel{
		Labels:          types.MapNull(types.StringType),
		ImageID:         types.StringNull(),
		BootDiskSizeGiB: types.Int64Null(),
	}

	if config.ImageId != nil {
		model.ImageID = types.StringValue(*config.ImageId)
	}

	if config.BootDiskSizeGib != nil {
		model.BootDiskSizeGiB = types.Int64Value(int64(*config.BootDiskSizeGib))
	}

	if config.Labels != nil {
		labels, diags := types.MapValueFrom(context.Background(), types.StringType, *config.Labels)
		if diags.HasError() {
			return model
		}
		model.Labels = labels
	}

	return model
}

func (r *edgeConfigurationResource) toAWSConfiguration(plan *awsConfigurationModel) *omni.AWSConfiguration {
	if plan == nil {
		return nil
	}

	config := &omni.AWSConfiguration{}

	if !plan.ImageID.IsNull() {
		config.ImageId = lo.ToPtr(plan.ImageID.ValueString())
	}

	if !plan.BootDiskSizeGiB.IsNull() {
		config.BootDiskSizeGib = lo.ToPtr(int32(plan.BootDiskSizeGiB.ValueInt64()))
	}

	if !plan.Tags.IsNull() {
		tags := make(map[string]string)
		plan.Tags.ElementsAs(context.Background(), &tags, false)
		config.Tags = &tags
	}

	return config
}

func (r *edgeConfigurationResource) toAWSConfigurationModel(config *omni.AWSConfiguration) *awsConfigurationModel {
	if config == nil {
		return nil
	}

	model := &awsConfigurationModel{
		Tags:            types.MapNull(types.StringType),
		ImageID:         types.StringNull(),
		BootDiskSizeGiB: types.Int64Null(),
	}

	if config.ImageId != nil {
		model.ImageID = types.StringValue(*config.ImageId)
	}

	if config.BootDiskSizeGib != nil {
		model.BootDiskSizeGiB = types.Int64Value(int64(*config.BootDiskSizeGib))
	}

	if config.Tags != nil {
		tags, diags := types.MapValueFrom(context.Background(), types.StringType, *config.Tags)
		if diags.HasError() {
			return model
		}
		model.Tags = tags
	}

	return model
}

func (r *edgeConfigurationResource) toOCIConfiguration(plan *ociConfigurationModel) *omni.OCIConfiguration {
	if plan == nil {
		return nil
	}

	config := &omni.OCIConfiguration{}

	if !plan.ImageID.IsNull() {
		config.ImageId = lo.ToPtr(plan.ImageID.ValueString())
	}

	if !plan.BootDiskSizeGiB.IsNull() {
		config.BootDiskSizeGib = lo.ToPtr(int32(plan.BootDiskSizeGiB.ValueInt64()))
	}

	if !plan.Tags.IsNull() {
		tags := make(map[string]string)
		plan.Tags.ElementsAs(context.Background(), &tags, false)
		config.Tags = &tags
	}

	return config
}

func (r *edgeConfigurationResource) toOCIConfigurationModel(config *omni.OCIConfiguration) *ociConfigurationModel {
	if config == nil {
		return nil
	}

	model := &ociConfigurationModel{
		Tags:            types.MapNull(types.StringType),
		ImageID:         types.StringNull(),
		BootDiskSizeGiB: types.Int64Null(),
	}

	if config.ImageId != nil {
		model.ImageID = types.StringValue(*config.ImageId)
	}

	if config.BootDiskSizeGib != nil {
		model.BootDiskSizeGiB = types.Int64Value(int64(*config.BootDiskSizeGib))
	}

	if config.Tags != nil {
		tags, diags := types.MapValueFrom(context.Background(), types.StringType, *config.Tags)
		if diags.HasError() {
			return model
		}
		model.Tags = tags
	}

	return model
}

func (r *edgeConfigurationResource) toCustomConfiguration(plan *customConfigurationModel) *omni.CustomCloudConfiguration {
	if plan == nil {
		return nil
	}

	if plan.Custom.IsNull() {
		return nil
	}

	custom := make(map[string]interface{})
	plan.Custom.ElementsAs(context.Background(), &custom, false)

	return &custom
}

func (r *edgeConfigurationResource) toCustomConfigurationModel(config *omni.CustomCloudConfiguration) *customConfigurationModel {
	if config == nil {
		return nil
	}

	custom, diags := types.MapValueFrom(context.Background(), types.StringType, *config)
	if diags.HasError() {
		return &customConfigurationModel{
			Custom: types.MapNull(types.StringType),
		}
	}

	return &customConfigurationModel{
		Custom: custom,
	}
}
