package castai

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
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

// The sections are Optional + Computed plain list-of-object attributes rather
// than nested blocks or nested attributes: the API always returns a fully
// materialized object, so a configuration that omits a section must adopt the
// reported value instead of planning its removal, while the existing block
// syntax keeps working (Terraform accepts block syntax for object-typed
// attributes since 1.1.5). Declared values still win; removing a section from
// the configuration stops managing it rather than resetting it.
var (
	clusterLimitsCPUType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			FieldClusterLimitsCPUMaxCores: types.Int64Type,
			FieldClusterLimitsCPUMinCores: types.Int64Type,
		},
	}

	clusterLimitsType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			FieldClusterLimitsEnabled: types.BoolType,
			FieldClusterLimitsCPU:     types.ListType{ElemType: clusterLimitsCPUType},
		},
	}

	nodeDownscalerType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			FieldNodeDownscalerEmptyNodesEnabled: types.BoolType,
			FieldNodeDownscalerEmptyNodesDelay:   types.StringType,
		},
	}

	podPinnerType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			FieldPodPinnerEnabled: types.BoolType,
		},
	}

	unschedulablePodsType = types.ObjectType{
		AttrTypes: map[string]attr.Type{
			FieldUnschedulablePodsEnabled:                 types.BoolType,
			FieldUnschedulablePodsPartialTemplateMatching: types.BoolType,
			FieldUnschedulablePodsPodPinner:               types.ListType{ElemType: podPinnerType},
		},
	}
)

var (
	_ resource.Resource                = (*autoscalerPoliciesResource)(nil)
	_ resource.ResourceWithConfigure   = (*autoscalerPoliciesResource)(nil)
	_ resource.ResourceWithImportState = (*autoscalerPoliciesResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*autoscalerPoliciesResource)(nil)
)

// autoscalerPoliciesResource implements the castai_autoscaler_policies resource
// using the terraform-plugin-framework.
type autoscalerPoliciesResource struct {
	client *ProviderConfig
}

type autoscalerPoliciesModel struct {
	ID                types.String `tfsdk:"id"`
	ClusterID         types.String `tfsdk:"cluster_id"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	ScopedMode        types.Bool   `tfsdk:"scoped_mode"`
	Version           types.String `tfsdk:"version"`
	ClusterLimits     types.List   `tfsdk:"cluster_limits"`
	NodeDownscaler    types.List   `tfsdk:"node_downscaler"`
	UnschedulablePods types.List   `tfsdk:"unschedulable_pods"`
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
			},
			FieldAutoscalerPoliciesClusterLimits: schema.ListAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Defines minimum and maximum amount of CPU the cluster can have. cluster_limits { enabled = true, cpu { max_cores = 100, min_cores = 1 } }.",
				ElementType:   clusterLimitsType,
				Validators:    []validator.List{listvalidator.SizeAtMost(1), clusterLimitsValidator{}},
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown(), fillNullsFromState{}},
			},
			FieldAutoscalerPoliciesNodeDownscaler: schema.ListAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Node Downscaler defines policies for removing nodes based on the configured conditions. node_downscaler { empty_nodes_enabled = true, empty_nodes_delay = \"5m\" }.",
				ElementType:   nodeDownscalerType,
				Validators:    []validator.List{listvalidator.SizeAtMost(1)},
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown(), fillNullsFromState{}},
			},
			FieldAutoscalerPoliciesUnschedulablePods: schema.ListAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Policy defining autoscaler's behavior when unschedulable pods were detected. unschedulable_pods { enabled = true, partial_template_matching_enabled = false, pod_pinner { enabled = true } }.",
				ElementType:   unschedulablePodsType,
				Validators:    []validator.List{listvalidator.SizeAtMost(1), unschedulablePodsValidator{}},
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown(), fillNullsFromState{}},
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

	state := r.preserveSectionPresence(r.policiesToModel(clusterID, policies), plan)
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
		resp.Diagnostics.AddError(
			"Missing cluster id",
			"Cannot read autoscaler policies: both cluster_id and id are missing from state. "+
				"The state may be corrupted; consider re-importing the resource.",
		)
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

	prior := state
	state = r.preserveSectionPresence(r.policiesToModel(clusterID, policies), prior)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *autoscalerPoliciesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan autoscalerPoliciesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state autoscalerPoliciesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The version changes on every write, so it stays unknown in the plan.
	// Carry the version observed in the prior state into the update request
	// for optimistic locking.
	if plan.Version.IsNull() || plan.Version.ValueString() == "" {
		plan.Version = state.Version
	}

	clusterID := plan.ClusterID.ValueString()

	policies, diags := r.upsert(ctx, clusterID, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newState := r.preserveSectionPresence(r.policiesToModel(clusterID, policies), plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *autoscalerPoliciesResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Autoscaler policies V2 resource deletion is a no-op. Removing from state.")
	resp.State.RemoveResource(ctx)
}

func (r *autoscalerPoliciesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root(FieldAutoscalerPoliciesID), req, resp)
}

// ModifyPlan settles the version when nothing else changes: the framework
// marks computed attributes with null configuration as unknown whenever it
// detects any change between the proposed and prior state — including the
// transient inner-field changes the section plan modifiers settle
// afterwards. Without settling it back, a converged configuration would
// plan a change on every run, even though the server does not bump the
// version for identical content.
func (r *autoscalerPoliciesResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var plan, state autoscalerPoliciesModel
	if diags := req.Plan.Get(ctx, &plan); diags.HasError() {
		return
	}
	if diags := req.State.Get(ctx, &state); diags.HasError() {
		return
	}

	if plan.Version.IsUnknown() &&
		plan.ID.Equal(state.ID) &&
		plan.ClusterID.Equal(state.ClusterID) &&
		plan.Enabled.Equal(state.Enabled) &&
		plan.ScopedMode.Equal(state.ScopedMode) &&
		plan.ClusterLimits.Equal(state.ClusterLimits) &&
		plan.NodeDownscaler.Equal(state.NodeDownscaler) &&
		plan.UnschedulablePods.Equal(state.UnschedulablePods) {
		plan.Version = state.Version
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
	}
}

// upsert pushes the plan to the API and returns the stored policies from
// the update response. The version for optimistic locking comes from the
// plan on updates and is fetched first on create, where the server rejects
// a duplicate insert without it.
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

	// The update response carries the stored policies, including the new
	// version; no read-back is needed.
	return apiResp.JSON200, diags
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

	policies.ClusterLimits = clusterLimitsFromValue(m.ClusterLimits)
	policies.NodeDownscaler = nodeDownscalerFromValue(m.NodeDownscaler)
	policies.UnschedulablePods = unschedulablePodsFromValue(m.UnschedulablePods)

	// Include version from plan for optimistic locking on updates.
	if !m.Version.IsNull() && m.Version.ValueString() != "" {
		policies.Version = lo.ToPtr(m.Version.ValueString())
	}

	return policies
}

func clusterLimitsFromValue(v types.List) *cluster_autoscaler_v2.ClusterLimitsPolicy {
	if v.IsNull() || len(v.Elements()) == 0 {
		return nil
	}

	attrs := v.Elements()[0].(types.Object).Attributes()
	out := &cluster_autoscaler_v2.ClusterLimitsPolicy{}

	if e, ok := attrs[FieldClusterLimitsEnabled].(types.Bool); ok && !e.IsNull() {
		out.Enabled = lo.ToPtr(e.ValueBool())
	}
	if c, ok := attrs[FieldClusterLimitsCPU].(types.List); ok {
		out.Cpu = cpuFromValue(c)
	}

	return out
}

func cpuFromValue(v types.List) *cluster_autoscaler_v2.ClusterLimitsCpu {
	if v.IsNull() || len(v.Elements()) == 0 {
		return nil
	}

	attrs := v.Elements()[0].(types.Object).Attributes()
	cpu := &cluster_autoscaler_v2.ClusterLimitsCpu{}

	if m, ok := attrs[FieldClusterLimitsCPUMaxCores].(types.Int64); ok && !m.IsNull() {
		cpu.MaxCores = int32(m.ValueInt64())
	}
	if m, ok := attrs[FieldClusterLimitsCPUMinCores].(types.Int64); ok && !m.IsNull() {
		cpu.MinCores = lo.ToPtr(int32(m.ValueInt64()))
	}

	return cpu
}

func nodeDownscalerFromValue(v types.List) *cluster_autoscaler_v2.NodeDownscalerPolicy {
	if v.IsNull() || len(v.Elements()) == 0 {
		return nil
	}

	attrs := v.Elements()[0].(types.Object).Attributes()
	out := &cluster_autoscaler_v2.NodeDownscalerPolicy{}

	if d, ok := attrs[FieldNodeDownscalerEmptyNodesDelay].(types.String); ok && !d.IsNull() && d.ValueString() != "" {
		out.EmptyNodesDelay = lo.ToPtr(d.ValueString())
	}
	if e, ok := attrs[FieldNodeDownscalerEmptyNodesEnabled].(types.Bool); ok && !e.IsNull() {
		out.EmptyNodesEnabled = lo.ToPtr(e.ValueBool())
	}

	return out
}

func unschedulablePodsFromValue(v types.List) *cluster_autoscaler_v2.UnschedulablePodsPolicy {
	if v.IsNull() || len(v.Elements()) == 0 {
		return nil
	}

	attrs := v.Elements()[0].(types.Object).Attributes()
	out := &cluster_autoscaler_v2.UnschedulablePodsPolicy{}

	if e, ok := attrs[FieldUnschedulablePodsEnabled].(types.Bool); ok && !e.IsNull() {
		out.Enabled = lo.ToPtr(e.ValueBool())
	}
	if p, ok := attrs[FieldUnschedulablePodsPartialTemplateMatching].(types.Bool); ok && !p.IsNull() {
		out.PartialTemplateMatchingEnabled = lo.ToPtr(p.ValueBool())
	}
	if p, ok := attrs[FieldUnschedulablePodsPodPinner].(types.List); ok {
		out.PodPinner = podPinnerFromValue(p)
	}

	return out
}

func podPinnerFromValue(v types.List) *cluster_autoscaler_v2.PodPinner {
	if v.IsNull() || len(v.Elements()) == 0 {
		return nil
	}

	attrs := v.Elements()[0].(types.Object).Attributes()
	podPinner := &cluster_autoscaler_v2.PodPinner{}

	if e, ok := attrs[FieldPodPinnerEnabled].(types.Bool); ok && !e.IsNull() {
		podPinner.Enabled = lo.ToPtr(e.ValueBool())
	}

	return podPinner
}

// policiesToModel converts the SDK policies response to the Terraform state
// model. Fields the API omits flatten to null so a configuration that omits
// them adopts the stored value.
func (r *autoscalerPoliciesResource) policiesToModel(clusterID string, policies *cluster_autoscaler_v2.PoliciesV2) autoscalerPoliciesModel {
	return autoscalerPoliciesModel{
		ID:         types.StringValue(clusterID),
		ClusterID:  types.StringValue(clusterID),
		Enabled:    boolPtrToValue(policies.Enabled),
		ScopedMode: boolPtrToValue(policies.ScopedMode),
		Version:    stringPtrToValue(policies.Version),

		ClusterLimits:     clusterLimitsToValue(policies.ClusterLimits),
		NodeDownscaler:    nodeDownscalerToValue(policies.NodeDownscaler),
		UnschedulablePods: unschedulablePodsToValue(policies.UnschedulablePods),
	}
}

// preserveSectionPresence carries sections the flatten did not produce over
// from the prior model, so an apply never loses a declared section to a sparse
// response. Sections are carried verbatim, never fabricated.
func (r *autoscalerPoliciesResource) preserveSectionPresence(state, prior autoscalerPoliciesModel) autoscalerPoliciesModel {
	if state.ClusterLimits.IsNull() && !prior.ClusterLimits.IsNull() {
		state.ClusterLimits = prior.ClusterLimits
	}
	if state.NodeDownscaler.IsNull() && !prior.NodeDownscaler.IsNull() {
		state.NodeDownscaler = prior.NodeDownscaler
	}
	if state.UnschedulablePods.IsNull() && !prior.UnschedulablePods.IsNull() {
		state.UnschedulablePods = prior.UnschedulablePods
	}
	return state
}

func clusterLimitsToValue(in *cluster_autoscaler_v2.ClusterLimitsPolicy) types.List {
	if in == nil || (in.Enabled == nil && in.Cpu == nil) {
		return types.ListNull(clusterLimitsType)
	}

	return objectListValue(clusterLimitsType, map[string]attr.Value{
		FieldClusterLimitsEnabled: boolPtrToAttr(in.Enabled),
		FieldClusterLimitsCPU:     cpuToValue(in.Cpu),
	})
}

func cpuToValue(in *cluster_autoscaler_v2.ClusterLimitsCpu) types.List {
	if in == nil {
		return types.ListNull(clusterLimitsCPUType)
	}

	min := types.Int64Null()
	if in.MinCores != nil {
		min = types.Int64Value(int64(*in.MinCores))
	}

	return objectListValue(clusterLimitsCPUType, map[string]attr.Value{
		FieldClusterLimitsCPUMaxCores: types.Int64Value(int64(in.MaxCores)),
		FieldClusterLimitsCPUMinCores: min,
	})
}

func nodeDownscalerToValue(in *cluster_autoscaler_v2.NodeDownscalerPolicy) types.List {
	if in == nil || (in.EmptyNodesDelay == nil && in.EmptyNodesEnabled == nil) {
		return types.ListNull(nodeDownscalerType)
	}

	delay := types.StringNull()
	if in.EmptyNodesDelay != nil {
		delay = types.StringValue(*in.EmptyNodesDelay)
	}

	return objectListValue(nodeDownscalerType, map[string]attr.Value{
		FieldNodeDownscalerEmptyNodesEnabled: boolPtrToAttr(in.EmptyNodesEnabled),
		FieldNodeDownscalerEmptyNodesDelay:   delay,
	})
}

func unschedulablePodsToValue(in *cluster_autoscaler_v2.UnschedulablePodsPolicy) types.List {
	if in == nil || (in.Enabled == nil && in.PartialTemplateMatchingEnabled == nil && in.PodPinner == nil) {
		return types.ListNull(unschedulablePodsType)
	}

	return objectListValue(unschedulablePodsType, map[string]attr.Value{
		FieldUnschedulablePodsEnabled:                 boolPtrToAttr(in.Enabled),
		FieldUnschedulablePodsPartialTemplateMatching: boolPtrToAttr(in.PartialTemplateMatchingEnabled),
		FieldUnschedulablePodsPodPinner:               podPinnerToValue(in.PodPinner),
	})
}

func podPinnerToValue(in *cluster_autoscaler_v2.PodPinner) types.List {
	if in == nil || in.Enabled == nil {
		return types.ListNull(podPinnerType)
	}

	return objectListValue(podPinnerType, map[string]attr.Value{
		FieldPodPinnerEnabled: boolPtrToAttr(in.Enabled),
	})
}

func objectListValue(objType types.ObjectType, attrs map[string]attr.Value) types.List {
	obj := types.ObjectValueMust(objType.AttrTypes, attrs)
	return types.ListValueMust(objType, []attr.Value{obj})
}

func boolPtrToAttr(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
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
