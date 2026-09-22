package castai

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler_v2"
)

// uuidRegex matches canonical 8-4-4-4-12 hexadecimal UUID strings.
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Field name constants for the castai_autoscaler_policies resource. Struct
// tags cannot reference constants, so tfsdk tags are the only place the names
// appear as string literals.
const (
	FieldAutoscalerPoliciesID                = "id"
	FieldAutoscalerPoliciesVersion           = "version"
	FieldAutoscalerPoliciesEnabled           = "enabled"
	FieldAutoscalerPoliciesScopedMode        = "scoped_mode"
	FieldAutoscalerPoliciesClusterLimits     = "cluster_limits"
	FieldAutoscalerPoliciesNodeDownscaler    = "node_downscaler"
	FieldAutoscalerPoliciesUnschedulablePods = "unschedulable_pods"

	FieldClusterLimitsEnabled     = "enabled"
	FieldClusterLimitsCPU         = "cpu"
	FieldClusterLimitsCPUMaxCores = "max_cores"
	FieldClusterLimitsCPUMinCores = "min_cores"

	FieldNodeDownscalerEmptyNodesDelay   = "empty_nodes_delay"
	FieldNodeDownscalerEmptyNodesEnabled = "empty_nodes_enabled"

	FieldUnschedulablePodsEnabled                 = "enabled"
	FieldUnschedulablePodsPartialTemplateMatching = "partial_template_matching_enabled"
	FieldUnschedulablePodsPodPinner               = "pod_pinner"

	FieldPodPinnerEnabled = "enabled"
)

var (
	_ resource.Resource                = (*autoscalerPoliciesResource)(nil)
	_ resource.ResourceWithConfigure   = (*autoscalerPoliciesResource)(nil)
	_ resource.ResourceWithImportState = (*autoscalerPoliciesResource)(nil)
)

// autoscalerPoliciesResource implements the castai_autoscaler_policies resource
// using the terraform-plugin-framework.
type autoscalerPoliciesResource struct {
	client *ProviderConfig
}

type autoscalerPoliciesModel struct {
	ID                types.String             `tfsdk:"id"`
	ClusterID         types.String             `tfsdk:"cluster_id"`
	Enabled           types.Bool               `tfsdk:"enabled"`
	ScopedMode        types.Bool               `tfsdk:"scoped_mode"`
	Version           types.String             `tfsdk:"version"`
	ClusterLimits     []clusterLimitsModel     `tfsdk:"cluster_limits"`
	NodeDownscaler    []nodeDownscalerModel    `tfsdk:"node_downscaler"`
	UnschedulablePods []unschedulablePodsModel `tfsdk:"unschedulable_pods"`
}

type clusterLimitsModel struct {
	Enabled types.Bool              `tfsdk:"enabled"`
	CPU     []clusterLimitsCPUModel `tfsdk:"cpu"`
}

type clusterLimitsCPUModel struct {
	MaxCores types.Int64 `tfsdk:"max_cores"`
	MinCores types.Int64 `tfsdk:"min_cores"`
}

type nodeDownscalerModel struct {
	EmptyNodesDelay   types.String `tfsdk:"empty_nodes_delay"`
	EmptyNodesEnabled types.Bool   `tfsdk:"empty_nodes_enabled"`
}

type unschedulablePodsModel struct {
	Enabled                        types.Bool       `tfsdk:"enabled"`
	PartialTemplateMatchingEnabled types.Bool       `tfsdk:"partial_template_matching_enabled"`
	PodPinner                      []podPinnerModel `tfsdk:"pod_pinner"`
}

type podPinnerModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

func newAutoscalerPoliciesResource() resource.Resource {
	return &autoscalerPoliciesResource{}
}

func (r *autoscalerPoliciesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_autoscaler_policies"
}

func (r *autoscalerPoliciesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "CAST AI autoscaler policies V2 resource to manage cluster autoscaling policies.",
		Attributes: map[string]schema.Attribute{
			FieldAutoscalerPoliciesID: schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this resource, equal to the cluster id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			FieldClusterId: schema.StringAttribute{
				Required:    true,
				Description: "CAST AI cluster id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex, "cluster_id must be a valid UUID"),
				},
			},
			FieldAutoscalerPoliciesEnabled: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable/disable all policies (global master switch).",
				Default:     booldefault.StaticBool(false),
			},
			FieldAutoscalerPoliciesScopedMode: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Run the node autoscaler in scoped mode.",
				Default:     booldefault.StaticBool(false),
			},
			FieldAutoscalerPoliciesVersion: schema.StringAttribute{
				Computed:    true,
				Description: "Policy version for optimistic locking.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			FieldAutoscalerPoliciesClusterLimits: schema.ListNestedBlock{
				Description: "Defines minimum and maximum amount of CPU the cluster can have.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						FieldClusterLimitsEnabled: schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Enable/disable cluster size limits policy.",
							Default:     booldefault.StaticBool(false),
						},
					},
					Blocks: map[string]schema.Block{
						FieldClusterLimitsCPU: schema.ListNestedBlock{
							Description: "Defines the minimum and maximum amount of CPUs for cluster's worker nodes.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									FieldClusterLimitsCPUMaxCores: schema.Int64Attribute{
										Required:    true,
										Description: "Defines the maximum allowed amount of vCPUs in the whole cluster.",
										Validators: []validator.Int64{
											int64validator.AtLeast(2),
										},
									},
									FieldClusterLimitsCPUMinCores: schema.Int64Attribute{
										Optional:           true,
										Computed:           true,
										Description:        "Defines the minimum allowed amount of CPUs in the whole cluster. Deprecated: Min CPU limit is no longer enforced.",
										DeprecationMessage: "Min CPU limit is no longer enforced.",
										Default:            int64default.StaticInt64(0),
									},
								},
							},
						},
					},
				},
			},
			FieldAutoscalerPoliciesNodeDownscaler: schema.ListNestedBlock{
				Description: "Node Downscaler defines policies for removing nodes based on the configured conditions.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						FieldNodeDownscalerEmptyNodesDelay: schema.StringAttribute{
							Optional:    true,
							Description: "Period to wait before removing an empty node.",
						},
						FieldNodeDownscalerEmptyNodesEnabled: schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Enable/disable the empty worker nodes policy.",
							Default:     booldefault.StaticBool(false),
						},
					},
				},
			},
			FieldAutoscalerPoliciesUnschedulablePods: schema.ListNestedBlock{
				Description: "Policy defining autoscaler's behavior when unschedulable pods were detected.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						FieldUnschedulablePodsEnabled: schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Enable/disable unschedulable pods detection policy.",
							Default:     booldefault.StaticBool(false),
						},
						FieldUnschedulablePodsPartialTemplateMatching: schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Marks whether partial matching should be used when deciding which custom node template to select.",
							Default:     booldefault.StaticBool(false),
						},
					},
					Blocks: map[string]schema.Block{
						FieldUnschedulablePodsPodPinner: schema.ListNestedBlock{
							Description: "Defines the CAST AI Pod Pinner component settings.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									FieldPodPinnerEnabled: schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Description: "Enable/disable the Pod Pinner policy.",
										Default:     booldefault.StaticBool(false),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *autoscalerPoliciesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *autoscalerPoliciesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan autoscalerPoliciesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID := plan.ClusterID.ValueString()

	policies, diags := r.upsert(ctx, clusterID, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := r.policiesToModel(clusterID, policies)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *autoscalerPoliciesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state autoscalerPoliciesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fall back to the resource id (import passthrough) when cluster_id is not
	// yet in state; the id equals the cluster id.
	clusterID := state.ClusterID.ValueString()
	if clusterID == "" {
		clusterID = state.ID.ValueString()
	}
	if clusterID == "" {
		tflog.Info(ctx, "ClusterId is missing. Will skip operation.")
		return
	}

	policies, found, diags := r.readPolicies(ctx, clusterID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !found {
		tflog.Info(ctx, "Autoscaler policies not found, removing from state", map[string]interface{}{"cluster_id": clusterID})
		resp.State.RemoveResource(ctx)
		return
	}

	state = r.policiesToModel(clusterID, policies)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *autoscalerPoliciesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan autoscalerPoliciesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID := plan.ClusterID.ValueString()

	policies, diags := r.upsert(ctx, clusterID, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := r.policiesToModel(clusterID, policies)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *autoscalerPoliciesResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Autoscaler policies V2 resource deletion is a no-op. Removing from state.")
	resp.State.RemoveResource(ctx)
}

func (r *autoscalerPoliciesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root(FieldAutoscalerPoliciesID), req, resp)
}

// upsert builds the policies payload from the plan and pushes it to the API,
// then reads the resulting policies back. The version is included for
// optimistic locking: on updates it comes from the plan (state), and on
// create it is fetched from the API first, since the policies most likely
// already exist for the cluster and the server rejects an insert of a
// duplicate record.
func (r *autoscalerPoliciesResource) upsert(ctx context.Context, clusterID string, plan *autoscalerPoliciesModel) (*cluster_autoscaler_v2.PoliciesV2, diag.Diagnostics) {
	var diags diag.Diagnostics

	policies := policiesFromModel(plan)

	if policies.Version == nil {
		current, found, readDiags := r.readPolicies(ctx, clusterID)
		diags.Append(readDiags...)
		if diags.HasError() {
			return nil, diags
		}
		if found && current.Version != nil {
			policies.Version = current.Version
		}
	}

	client := r.client.clusterAutoscalerV2Client
	apiResp, err := client.PoliciesV2APIUpdateClusterPoliciesWithResponse(ctx, clusterID, *policies)
	if err != nil {
		diags.AddError("Failed to update autoscaler policies", err.Error())
		return nil, diags
	}
	if apiResp.StatusCode() != http.StatusOK {
		diags.AddError(
			"Failed to update autoscaler policies",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return nil, diags
	}
	if apiResp.JSON200 == nil {
		diags.AddError(
			"Failed to update autoscaler policies",
			fmt.Sprintf("received empty policies response after update for cluster %s", clusterID),
		)
		return nil, diags
	}

	// Read the policies back so state reflects what the API actually stored,
	// mirroring the SDKv2 create/update behavior.
	result, found, readDiags := r.readPolicies(ctx, clusterID)
	diags.Append(readDiags...)
	if diags.HasError() {
		return nil, diags
	}
	if !found {
		diags.AddError(
			"Failed to read autoscaler policies",
			fmt.Sprintf("policies for cluster %s not found after update", clusterID),
		)
		return nil, diags
	}

	return result, diags
}

// readPolicies fetches the cluster policies. The second return value reports
// whether the policies were found (false on 404).
func (r *autoscalerPoliciesResource) readPolicies(ctx context.Context, clusterID string) (*cluster_autoscaler_v2.PoliciesV2, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	client := r.client.clusterAutoscalerV2Client
	apiResp, err := client.PoliciesV2APIGetClusterPoliciesWithResponse(ctx, clusterID)
	if err != nil {
		diags.AddError("Failed to read autoscaler policies", err.Error())
		return nil, false, diags
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		return nil, false, diags
	}
	if apiResp.StatusCode() != http.StatusOK {
		diags.AddError(
			"Failed to read autoscaler policies",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return nil, false, diags
	}
	if apiResp.JSON200 == nil {
		diags.AddError(
			"Failed to read autoscaler policies",
			fmt.Sprintf("received empty policies response for cluster %s", clusterID),
		)
		return nil, false, diags
	}

	return apiResp.JSON200, true, diags
}

// policiesFromModel converts the Terraform plan model to the SDK policies payload.
func policiesFromModel(m *autoscalerPoliciesModel) *cluster_autoscaler_v2.PoliciesV2 {
	policies := &cluster_autoscaler_v2.PoliciesV2{}

	if !m.Enabled.IsNull() {
		policies.Enabled = lo.ToPtr(m.Enabled.ValueBool())
	}

	if !m.ScopedMode.IsNull() {
		policies.ScopedMode = lo.ToPtr(m.ScopedMode.ValueBool())
	}

	if len(m.ClusterLimits) > 0 {
		policies.ClusterLimits = clusterLimitsFromModel(&m.ClusterLimits[0])
	}

	if len(m.NodeDownscaler) > 0 {
		policies.NodeDownscaler = nodeDownscalerFromModel(&m.NodeDownscaler[0])
	}

	if len(m.UnschedulablePods) > 0 {
		policies.UnschedulablePods = unschedulablePodsFromModel(&m.UnschedulablePods[0])
	}

	// Include version from plan for optimistic locking on updates.
	if !m.Version.IsNull() && m.Version.ValueString() != "" {
		policies.Version = lo.ToPtr(m.Version.ValueString())
	}

	return policies
}

func clusterLimitsFromModel(m *clusterLimitsModel) *cluster_autoscaler_v2.ClusterLimitsPolicy {
	out := &cluster_autoscaler_v2.ClusterLimitsPolicy{}

	if !m.Enabled.IsNull() {
		out.Enabled = lo.ToPtr(m.Enabled.ValueBool())
	}

	if len(m.CPU) > 0 {
		cpu := &cluster_autoscaler_v2.ClusterLimitsCpu{
			MaxCores: int32(m.CPU[0].MaxCores.ValueInt64()),
		}
		if !m.CPU[0].MinCores.IsNull() {
			cpu.MinCores = lo.ToPtr(int32(m.CPU[0].MinCores.ValueInt64()))
		}
		out.Cpu = cpu
	}

	return out
}

func nodeDownscalerFromModel(m *nodeDownscalerModel) *cluster_autoscaler_v2.NodeDownscalerPolicy {
	out := &cluster_autoscaler_v2.NodeDownscalerPolicy{}

	if !m.EmptyNodesDelay.IsNull() && m.EmptyNodesDelay.ValueString() != "" {
		out.EmptyNodesDelay = lo.ToPtr(m.EmptyNodesDelay.ValueString())
	}

	if !m.EmptyNodesEnabled.IsNull() {
		out.EmptyNodesEnabled = lo.ToPtr(m.EmptyNodesEnabled.ValueBool())
	}

	return out
}

func unschedulablePodsFromModel(m *unschedulablePodsModel) *cluster_autoscaler_v2.UnschedulablePodsPolicy {
	out := &cluster_autoscaler_v2.UnschedulablePodsPolicy{}

	if !m.Enabled.IsNull() {
		out.Enabled = lo.ToPtr(m.Enabled.ValueBool())
	}

	if !m.PartialTemplateMatchingEnabled.IsNull() {
		out.PartialTemplateMatchingEnabled = lo.ToPtr(m.PartialTemplateMatchingEnabled.ValueBool())
	}

	if len(m.PodPinner) > 0 {
		podPinner := &cluster_autoscaler_v2.PodPinner{}
		if !m.PodPinner[0].Enabled.IsNull() {
			podPinner.Enabled = lo.ToPtr(m.PodPinner[0].Enabled.ValueBool())
		}
		out.PodPinner = podPinner
	}

	return out
}

// policiesToModel converts the SDK policies response to the Terraform state
// model. Bool fields with static defaults are always written with concrete
// values (false when the API omits them) to prevent state drift.
func (r *autoscalerPoliciesResource) policiesToModel(clusterID string, policies *cluster_autoscaler_v2.PoliciesV2) autoscalerPoliciesModel {
	model := autoscalerPoliciesModel{
		ID:         types.StringValue(clusterID),
		ClusterID:  types.StringValue(clusterID),
		Enabled:    boolPtrToValue(policies.Enabled),
		ScopedMode: boolPtrToValue(policies.ScopedMode),
		Version:    stringPtrToValue(policies.Version),
	}

	if limits := clusterLimitsToModel(policies.ClusterLimits); limits != nil {
		model.ClusterLimits = []clusterLimitsModel{*limits}
	}

	if downscaler := nodeDownscalerToModel(policies.NodeDownscaler); downscaler != nil {
		model.NodeDownscaler = []nodeDownscalerModel{*downscaler}
	}

	if unschedulable := unschedulablePodsToModel(policies.UnschedulablePods); unschedulable != nil {
		model.UnschedulablePods = []unschedulablePodsModel{*unschedulable}
	}

	return model
}

func clusterLimitsToModel(in *cluster_autoscaler_v2.ClusterLimitsPolicy) *clusterLimitsModel {
	if in == nil {
		return nil
	}
	if in.Enabled == nil && in.Cpu == nil {
		return nil
	}

	out := &clusterLimitsModel{
		Enabled: boolPtrToValue(in.Enabled),
	}

	if in.Cpu != nil {
		cpu := clusterLimitsCPUModel{
			MaxCores: types.Int64Value(int64(in.Cpu.MaxCores)),
		}
		if in.Cpu.MinCores != nil {
			cpu.MinCores = types.Int64Value(int64(*in.Cpu.MinCores))
		} else {
			cpu.MinCores = types.Int64Value(0)
		}
		out.CPU = []clusterLimitsCPUModel{cpu}
	}

	return out
}

func nodeDownscalerToModel(in *cluster_autoscaler_v2.NodeDownscalerPolicy) *nodeDownscalerModel {
	if in == nil {
		return nil
	}
	if in.EmptyNodesDelay == nil && in.EmptyNodesEnabled == nil {
		return nil
	}

	out := &nodeDownscalerModel{
		EmptyNodesEnabled: boolPtrToValue(in.EmptyNodesEnabled),
	}

	if in.EmptyNodesDelay != nil {
		out.EmptyNodesDelay = types.StringValue(*in.EmptyNodesDelay)
	}

	return out
}

func unschedulablePodsToModel(in *cluster_autoscaler_v2.UnschedulablePodsPolicy) *unschedulablePodsModel {
	if in == nil {
		return nil
	}
	if in.Enabled == nil && in.PartialTemplateMatchingEnabled == nil && in.PodPinner == nil {
		return nil
	}

	out := &unschedulablePodsModel{
		Enabled:                        boolPtrToValue(in.Enabled),
		PartialTemplateMatchingEnabled: boolPtrToValue(in.PartialTemplateMatchingEnabled),
	}

	if podPinner := podPinnerToModel(in.PodPinner); podPinner != nil {
		out.PodPinner = []podPinnerModel{*podPinner}
	}

	return out
}

func podPinnerToModel(in *cluster_autoscaler_v2.PodPinner) *podPinnerModel {
	if in == nil || in.Enabled == nil {
		return nil
	}

	return &podPinnerModel{
		Enabled: boolPtrToValue(in.Enabled),
	}
}

func boolPtrToValue(b *bool) types.Bool {
	if b == nil {
		return types.BoolValue(false)
	}
	return types.BoolValue(*b)
}

func stringPtrToValue(s *string) types.String {
	if s == nil {
		return types.StringValue("")
	}
	return types.StringValue(*s)
}
