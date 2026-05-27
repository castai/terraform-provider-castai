package castai

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	"github.com/google/uuid"
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
	CRI            *criConfigurationModel    `tfsdk:"cri"`
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

type criConfigurationModel struct {
	Socket types.String `tfsdk:"socket"`
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
			"cri": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "CRI (Container Runtime Interface) configuration for the edge node. Set this when you want kubelet to connect to a container runtime you have set up explicitly on the node. Currently only containerd is officially supported.",
				Attributes: map[string]schema.Attribute{
					"socket": schema.StringAttribute{
						Optional:    true,
						Description: "Path to an existing CRI socket. Example: unix:///run/containerd/containerd.sock",
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

	// Build the create request
	gcpConfig, d := r.toGCPConfiguration(ctx, plan.GCP)
	resp.Diagnostics.Append(d...)

	awsConfig, d := r.toAWSConfiguration(ctx, plan.AWS)
	resp.Diagnostics.Append(d...)

	ociConfig, d := r.toOCIConfiguration(ctx, plan.OCI)
	resp.Diagnostics.Append(d...)

	customConfig, d := r.toCustomConfiguration(ctx, plan.Custom)
	resp.Diagnostics.Append(d...)

	criConfig, d := r.toCRIConfiguration(ctx, plan.CRI)
	resp.Diagnostics.Append(d...)

	if resp.Diagnostics.HasError() {
		return
	}

	createReq := omni.EdgeConfiguration{
		Name:           plan.Name.ValueString(),
		Default:        lo.ToPtr(false),
		UserDataBase64: lo.ToPtr(plan.UserDataBase64.ValueString()),
		Gcp:            gcpConfig,
		Aws:            awsConfig,
		Oci:            ociConfig,
		Custom:         customConfig,
		Cri:            criConfig,
	}

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

	if apiResp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Failed to create edge configuration",
			"API response body is empty",
		)
		return
	}

	state := r.edgeConfigurationToTFModel(ctx, apiResp.JSON200, plan.OrganizationID, plan.ClusterID)
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

	if apiResp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Failed to read edge configuration",
			"API response body is empty",
		)
		return
	}

	state = r.edgeConfigurationToTFModel(ctx, apiResp.JSON200, state.OrganizationID, state.ClusterID)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
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

	// Build the update request
	gcpConfig, d := r.toGCPConfiguration(ctx, plan.GCP)
	resp.Diagnostics.Append(d...)

	awsConfig, d := r.toAWSConfiguration(ctx, plan.AWS)
	resp.Diagnostics.Append(d...)

	ociConfig, d := r.toOCIConfiguration(ctx, plan.OCI)
	resp.Diagnostics.Append(d...)

	customConfig, d := r.toCustomConfiguration(ctx, plan.Custom)
	resp.Diagnostics.Append(d...)

	criConfig, d := r.toCRIConfiguration(ctx, plan.CRI)
	resp.Diagnostics.Append(d...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := omni.EdgeConfigurationUpdate{
		Name:           lo.ToPtr(plan.Name.ValueString()),
		UserDataBase64: lo.ToPtr(plan.UserDataBase64.ValueString()),
		Gcp:            gcpConfig,
		Aws:            awsConfig,
		Oci:            ociConfig,
		Custom:         customConfig,
		Cri:            criConfig,
	}

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

	if apiResp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Failed to update edge configuration",
			"API response body is empty",
		)
		return
	}

	state := r.edgeConfigurationToTFModel(ctx, apiResp.JSON200, plan.OrganizationID, plan.ClusterID)

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

	// Fetch current config from API to check if it's the default.
	// State may be stale if another resource (e.g. edge_configuration_default)
	// changed the default flag after this resource was created.
	getResp, err := client.EdgeConfigurationsAPIGetEdgeConfigurationWithResponse(ctx, organizationID, clusterID, edgeLocationID, configurationID, nil)
	if err == nil && getResp.StatusCode() == http.StatusOK && getResp.JSON200 != nil && getResp.JSON200.Default != nil && *getResp.JSON200.Default {
		tflog.Info(ctx, "Skipping deletion of default edge configuration")
		return
	}
	if err == nil && getResp.StatusCode() == http.StatusNotFound {
		tflog.Info(ctx, "Edge configuration not found, treating as deleted")
		return
	}

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
	// Import format: organization_id/cluster_id/edge_location_id/{configuration id | name}
	ids := strings.Split(req.ID, "/")
	if len(ids) != 4 || ids[0] == "" || ids[1] == "" || ids[2] == "" || ids[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("expected: organization_id/cluster_id/edge_location_id/{configuration id | name}, got: %q", req.ID),
		)
		return
	}

	organizationID := ids[0]
	clusterID := ids[1]
	edgeLocationID := ids[2]
	configIDOrName := ids[3]

	// Check if the last part is a valid UUID - if so, use it directly
	if _, err := uuid.Parse(configIDOrName); err == nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), organizationID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), clusterID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("edge_location_id"), edgeLocationID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), configIDOrName)...)
		return
	}

	// Not a UUID - look up by name
	client := r.client.omniAPI
	params := &omni.EdgeConfigurationsAPIListEdgeConfigurationsParams{EdgeLocationId: &edgeLocationID}
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
		resp.Diagnostics.AddError("Failed to list edge configurations", "response body is empty")
		return
	}

	for _, cfg := range *apiResp.JSON200.Items {
		if cfg.Name == configIDOrName {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), organizationID)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), clusterID)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("edge_location_id"), edgeLocationID)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), lo.FromPtr(cfg.Id))...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"Failed to find edge configuration",
		fmt.Sprintf("no edge configuration found with name %q in edge location %q", configIDOrName, edgeLocationID),
	)
}

func (r *edgeConfigurationResource) getOrganizationID(organizationID types.String) string {
	if !organizationID.IsNull() {
		return organizationID.ValueString()
	}
	return r.client.organizationID
}

func (r *edgeConfigurationResource) edgeConfigurationToTFModel(ctx context.Context, config *omni.EdgeConfiguration, organizationID types.String, clusterID types.String) edgeConfigurationModel {
	state := edgeConfigurationModel{
		ID:             types.StringValue(lo.FromPtr(config.Id)),
		OrganizationID: organizationID,
		ClusterID:      clusterID,
		Name:           types.StringValue(config.Name),
		EdgeLocationID: types.StringValue(lo.FromPtr(config.EdgeLocationId)),
		Default:        types.BoolValue(lo.FromPtr(config.Default)),
		UserDataBase64: normalizeStringPtr(config.UserDataBase64),
		GCP:            r.toGCPConfigurationModel(ctx, config.Gcp),
		AWS:            r.toAWSConfigurationModel(ctx, config.Aws),
		OCI:            r.toOCIConfigurationModel(ctx, config.Oci),
		Custom:         r.toCustomConfigurationModel(ctx, config.Custom),
		CRI:            r.toCRIConfigurationModel(ctx, config.Cri),
	}

	return state
}

func (r *edgeConfigurationResource) toGCPConfiguration(ctx context.Context, plan *gcpConfigurationModel) (*omni.GCPConfiguration, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan == nil {
		return nil, diags
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
		diags.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if !diags.HasError() {
			config.Labels = &labels
		}
	}

	return config, diags
}

func (r *edgeConfigurationResource) toGCPConfigurationModel(ctx context.Context, config *omni.GCPConfiguration) *gcpConfigurationModel {
	if config == nil {
		return nil
	}

	model := &gcpConfigurationModel{
		Labels:          types.MapNull(types.StringType),
		ImageID:         types.StringNull(),
		BootDiskSizeGiB: types.Int64Null(),
	}

	if config.ImageId != nil && *config.ImageId != "" {
		model.ImageID = types.StringValue(*config.ImageId)
	}

	if config.BootDiskSizeGib != nil {
		model.BootDiskSizeGiB = types.Int64Value(int64(*config.BootDiskSizeGib))
	}

	if config.Labels != nil && len(*config.Labels) > 0 {
		labels, diags := types.MapValueFrom(ctx, types.StringType, *config.Labels)
		if diags.HasError() {
			return model
		}
		model.Labels = labels
	}

	return model
}

func (r *edgeConfigurationResource) toAWSConfiguration(ctx context.Context, plan *awsConfigurationModel) (*omni.AWSConfiguration, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan == nil {
		return nil, diags
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
		diags.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if !diags.HasError() {
			config.Tags = &tags
		}
	}

	return config, diags
}

func (r *edgeConfigurationResource) toAWSConfigurationModel(ctx context.Context, config *omni.AWSConfiguration) *awsConfigurationModel {
	if config == nil {
		return nil
	}

	model := &awsConfigurationModel{
		Tags:            types.MapNull(types.StringType),
		ImageID:         types.StringNull(),
		BootDiskSizeGiB: types.Int64Null(),
	}

	if config.ImageId != nil && *config.ImageId != "" {
		model.ImageID = types.StringValue(*config.ImageId)
	}

	if config.BootDiskSizeGib != nil {
		model.BootDiskSizeGiB = types.Int64Value(int64(*config.BootDiskSizeGib))
	}

	if config.Tags != nil && len(*config.Tags) > 0 {
		tags, diags := types.MapValueFrom(ctx, types.StringType, *config.Tags)
		if diags.HasError() {
			return model
		}
		model.Tags = tags
	}

	return model
}

func (r *edgeConfigurationResource) toOCIConfiguration(ctx context.Context, plan *ociConfigurationModel) (*omni.OCIConfiguration, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan == nil {
		return nil, diags
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
		diags.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if !diags.HasError() {
			config.Tags = &tags
		}
	}

	return config, diags
}

func (r *edgeConfigurationResource) toOCIConfigurationModel(ctx context.Context, config *omni.OCIConfiguration) *ociConfigurationModel {
	if config == nil {
		return nil
	}

	model := &ociConfigurationModel{
		Tags:            types.MapNull(types.StringType),
		ImageID:         types.StringNull(),
		BootDiskSizeGiB: types.Int64Null(),
	}

	if config.ImageId != nil && *config.ImageId != "" {
		model.ImageID = types.StringValue(*config.ImageId)
	}

	if config.BootDiskSizeGib != nil {
		model.BootDiskSizeGiB = types.Int64Value(int64(*config.BootDiskSizeGib))
	}

	if config.Tags != nil && len(*config.Tags) > 0 {
		tags, diags := types.MapValueFrom(ctx, types.StringType, *config.Tags)
		if diags.HasError() {
			return model
		}
		model.Tags = tags
	}

	return model
}

func (r *edgeConfigurationResource) toCustomConfiguration(ctx context.Context, plan *customConfigurationModel) (*omni.CustomCloudConfiguration, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan == nil {
		return nil, diags
	}

	if plan.Custom.IsNull() {
		return nil, diags
	}

	custom := make(map[string]interface{})
	diags.Append(plan.Custom.ElementsAs(ctx, &custom, false)...)
	if diags.HasError() {
		return nil, diags
	}

	return &custom, diags
}

func (r *edgeConfigurationResource) toCustomConfigurationModel(ctx context.Context, config *omni.CustomCloudConfiguration) *customConfigurationModel {
	if config == nil {
		return nil
	}

	if len(*config) == 0 {
		return &customConfigurationModel{
			Custom: types.MapNull(types.StringType),
		}
	}

	custom, diags := types.MapValueFrom(ctx, types.StringType, *config)
	if diags.HasError() {
		return &customConfigurationModel{
			Custom: types.MapNull(types.StringType),
		}
	}

	return &customConfigurationModel{
		Custom: custom,
	}
}

func (r *edgeConfigurationResource) toCRIConfiguration(_ context.Context, plan *criConfigurationModel) (*omni.EdgeConfigurationCRIConfiguration, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan == nil {
		// Return an empty struct (not nil) so the request JSON includes
		// cri: {} instead of omitting the field. This tells the API to
		// explicitly clear any existing CRI configuration.
		return &omni.EdgeConfigurationCRIConfiguration{}, diags
	}

	config := &omni.EdgeConfigurationCRIConfiguration{}

	if !plan.Socket.IsNull() {
		config.Socket = lo.ToPtr(plan.Socket.ValueString())
	}

	return config, diags
}

func (r *edgeConfigurationResource) toCRIConfigurationModel(_ context.Context, config *omni.EdgeConfigurationCRIConfiguration) *criConfigurationModel {
	if config == nil {
		return nil
	}

	// Normalize: a CRI object with no socket is semantically equivalent
	// to no CRI configuration at all.
	if config.Socket == nil || *config.Socket == "" {
		return nil
	}

	return &criConfigurationModel{
		Socket: types.StringValue(*config.Socket),
	}
}

func normalizeStringPtr(s *string) types.String {
	if s == nil || *s == "" {
		return types.StringNull()
	}
	return types.StringValue(*s)
}
