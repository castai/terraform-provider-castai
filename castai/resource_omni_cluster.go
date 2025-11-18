package castai

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = (*omniClusterResource)(nil)
	_ resource.ResourceWithConfigure = (*omniClusterResource)(nil)
)

type omniClusterResource struct {
	client *ProviderConfig
}

type omniClusterModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ClusterID      types.String `tfsdk:"cluster_id"`
}

func newOmniClusterResource() resource.Resource {
	return &omniClusterResource{}
}

func (r *omniClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_omni_cluster"
}

func (r *omniClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Omni cluster resource allows registering a cluster with CAST AI Omni provider.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource ID (same as cluster_id)",
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
				Description: "CAST AI cluster ID to register",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *omniClusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *omniClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan omniClusterModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.omniAPI
	organizationID := plan.OrganizationID.ValueString()
	clusterID := plan.ClusterID.ValueString()

	apiResp, err := client.ClustersAPIRegisterClusterWithResponse(ctx, organizationID, clusterID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to register omni cluster", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to register omni cluster",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	plan.ID = types.StringValue(clusterID)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *omniClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state omniClusterModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.omniAPI
	organizationID := state.OrganizationID.ValueString()
	clusterID := state.ID.ValueString()

	apiResp, err := client.ClustersAPIGetClusterWithResponse(ctx, organizationID, clusterID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read omni cluster", err.Error())
		return
	}

	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to read omni cluster",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	state.ClusterID = types.StringValue(*apiResp.JSON200.Id)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *omniClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update operation - all fields require replacement
}

func (r *omniClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state omniClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	client := r.client.omniAPI
	organizationID := state.OrganizationID.ValueString()
	clusterID := state.ID.ValueString()

	apiResp, err := client.ClustersAPIDeleteClusterWithResponse(ctx, organizationID, clusterID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete omni cluster", err.Error())
		return
	}

	if apiResp.StatusCode() == http.StatusNotFound {
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to delete omni cluster",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
	}
}
