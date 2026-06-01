package castai

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni"
)

var (
	_ resource.Resource                = (*edgeConfigurationDefaultResource)(nil)
	_ resource.ResourceWithConfigure   = (*edgeConfigurationDefaultResource)(nil)
	_ resource.ResourceWithImportState = (*edgeConfigurationDefaultResource)(nil)
)

type edgeConfigurationDefaultResource struct {
	client *ProviderConfig
}

type edgeConfigurationDefaultModel struct {
	ID              types.String `tfsdk:"id"`
	OrganizationID  types.String `tfsdk:"organization_id"`
	ClusterID       types.String `tfsdk:"cluster_id"`
	EdgeLocationID  types.String `tfsdk:"edge_location_id"`
	ConfigurationID types.String `tfsdk:"configuration_id"`
	Name            types.String `tfsdk:"name"`
	CloudProvider   types.String `tfsdk:"cloud_provider"`
}

func newEdgeConfigurationDefaultResource() resource.Resource {
	return &edgeConfigurationDefaultResource{}
}

func (r *edgeConfigurationDefaultResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_configuration_default"
}

func (r *edgeConfigurationDefaultResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the default CAST AI Edge Configuration for an edge location. " +
			"This resource sets an existing edge configuration as the default for the specified edge location.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Terraform resource ID (the configuration_id)",
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
			"configuration_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the edge configuration to set as default",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the edge configuration",
			},
			"cloud_provider": schema.StringAttribute{
				Computed:    true,
				Description: "Cloud provider type (gcp, aws, oci, custom)",
			},
		},
	}
}

func (r *edgeConfigurationDefaultResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *edgeConfigurationDefaultResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan edgeConfigurationDefaultModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.omniAPI
	organizationID := r.getOrganizationID(plan.OrganizationID)
	clusterID := plan.ClusterID.ValueString()
	edgeLocationID := plan.EdgeLocationID.ValueString()
	configurationID := plan.ConfigurationID.ValueString()

	tflog.Info(ctx, "Setting edge configuration as default",
		map[string]interface{}{
			"organization_id":  organizationID,
			"cluster_id":       clusterID,
			"edge_location_id": edgeLocationID,
			"configuration_id": configurationID,
		},
	)

	// Update the configuration to set it as default
	updateReq := omni.EdgeConfigurationUpdate{
		Default: lo.ToPtr(true),
	}

	apiResp, err := client.EdgeConfigurationsAPIUpdateEdgeConfigurationWithResponse(ctx, organizationID, clusterID, edgeLocationID, configurationID, nil, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to set edge configuration as default", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to set edge configuration as default",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	if apiResp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Failed to set edge configuration as default",
			"API response body is empty",
		)
		return
	}

	// Set the ID as just the configuration_id
	plan.ID = types.StringValue(configurationID)

	// Set computed fields from the configuration we just made default
	plan.Name = types.StringValue(apiResp.JSON200.Name)
	plan.CloudProvider = r.getCloudProviderType(apiResp.JSON200)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *edgeConfigurationDefaultResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state edgeConfigurationDefaultModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.omniAPI
	organizationID := r.getOrganizationID(state.OrganizationID)
	clusterID := state.ClusterID.ValueString()
	edgeLocationID := state.EdgeLocationID.ValueString()
	configurationID := state.ConfigurationID.ValueString()

	tflog.Info(ctx, "Reading edge configuration default",
		map[string]interface{}{
			"organization_id":  organizationID,
			"cluster_id":       clusterID,
			"edge_location_id": edgeLocationID,
			"configuration_id": configurationID,
		},
	)

	apiResp, err := client.EdgeConfigurationsAPIGetEdgeConfigurationWithResponse(ctx, organizationID, clusterID, edgeLocationID, configurationID, nil)
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

	// Verify the configuration is still marked as default
	if apiResp.JSON200.Default == nil || !*apiResp.JSON200.Default {
		// The configuration is no longer the default, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Update computed fields from the API response
	state.Name = types.StringValue(apiResp.JSON200.Name)
	state.CloudProvider = r.getCloudProviderType(apiResp.JSON200)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *edgeConfigurationDefaultResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This method is never called because all attributes have RequiresReplace plan modifier.
	// The resource is always destroyed and recreated instead of being updated.
	// Keeping this method to satisfy the resource.Resource interface.
}

func (r *edgeConfigurationDefaultResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// This is a no-op. The edge_configuration_default resource is just a marker
	// that sets a configuration as default. When this resource is destroyed,
	// we don't want to unset the default status - we just remove it from state.
	// The default status will remain on the configuration until another
	// configuration is explicitly set as default.
}

func (r *edgeConfigurationDefaultResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: organization_id/cluster_id/edge_location_id/configuration_id
	ids := strings.Split(req.ID, "/")
	if len(ids) != 4 || ids[0] == "" || ids[1] == "" || ids[2] == "" || ids[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("expected: organization_id/cluster_id/edge_location_id/configuration_id, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), ids[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), ids[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("edge_location_id"), ids[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("configuration_id"), ids[3])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ids[3])...)
}

func (r *edgeConfigurationDefaultResource) getCloudProviderType(config *omni.EdgeConfiguration) types.String {
	if config.Gcp != nil {
		return types.StringValue("gcp")
	}
	if config.Aws != nil {
		return types.StringValue("aws")
	}
	if config.Oci != nil {
		return types.StringValue("oci")
	}
	if config.Custom != nil {
		return types.StringValue("custom")
	}
	return types.StringValue("unknown")
}

func (r *edgeConfigurationDefaultResource) getOrganizationID(organizationID types.String) string {
	if !organizationID.IsNull() {
		return organizationID.ValueString()
	}
	return r.client.organizationID
}
