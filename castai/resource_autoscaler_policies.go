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
	"reflect"

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
	_ resource.ResourceWithModifyPlan  = (*autoscalerPoliciesResource)(nil)
)

// autoscalerPoliciesResource implements the castai_autoscaler_policies resource
// using the terraform-plugin-framework.
//
// Each policy section is a ListNestedBlock with SizeAtMost(1), decoding to
// a slice on the typed model: nil slice means absent, a single-element slice
// means declared. Per-field Default / Required / validators replace the
// hand-written plan modifier and custom validators used in earlier versions.
// The List-of-one shape is preserved on the wire and in state, so existing
// state files load without upgraders.
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
	Enabled types.Bool `tfsdk:"enabled"`
	Cpu     []cpuModel `tfsdk:"cpu"`
}

type cpuModel struct {
	MaxCores types.Int64 `tfsdk:"max_cores"`
	MinCores types.Int64 `tfsdk:"min_cores"`
}

type nodeDownscalerModel struct {
	EmptyNodesEnabled types.Bool   `tfsdk:"empty_nodes_enabled"`
	EmptyNodesDelay   types.String `tfsdk:"empty_nodes_delay"`
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
			},
		},
		Blocks: map[string]schema.Block{
			FieldAutoscalerPoliciesClusterLimits: schema.ListNestedBlock{
				Description: "Defines minimum and maximum amount of CPU the cluster can have. cluster_limits { enabled = true, cpu { max_cores = 100, min_cores = 1 } }.",
				Validators:  []validator.List{listvalidator.SizeAtMost(1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						FieldClusterLimitsEnabled: schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "Enable/disable the cluster_limits policy.",
						},
					},
					Blocks: map[string]schema.Block{
						FieldClusterLimitsCPU: schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									FieldClusterLimitsCPUMaxCores: schema.Int64Attribute{
										Required:    true,
										Description: "Maximum vCPUs allowed cluster-wide.",
										Validators:  []validator.Int64{int64validator.AtLeast(2)},
									},
									FieldClusterLimitsCPUMinCores: schema.Int64Attribute{
										Optional:    true,
										Computed:    true,
										Default:     int64default.StaticInt64(0),
										Description: "Minimum vCPUs allowed cluster-wide.",
									},
								},
							},
						},
					},
				},
			},
			FieldAutoscalerPoliciesNodeDownscaler: schema.ListNestedBlock{
				Description: "Node Downscaler defines policies for removing nodes based on the configured conditions. node_downscaler { empty_nodes_enabled = true, empty_nodes_delay = \"5m\" }.",
				Validators:  []validator.List{listvalidator.SizeAtMost(1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						FieldNodeDownscalerEmptyNodesEnabled: schema.BoolAttribute{
							Required:    true,
							Description: "Enable downscaling of empty nodes.",
						},
						FieldNodeDownscalerEmptyNodesDelay: schema.StringAttribute{
							Required:    true,
							Description: "How long a node must be empty before it becomes eligible for downscaling (e.g. \"5m\").",
						},
					},
				},
			},
			FieldAutoscalerPoliciesUnschedulablePods: schema.ListNestedBlock{
				Description: "Policy defining autoscaler's behavior when unschedulable pods were detected. unschedulable_pods { enabled = true, partial_template_matching_enabled = false, pod_pinner { enabled = true } }.",
				Validators:  []validator.List{listvalidator.SizeAtMost(1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						FieldUnschedulablePodsEnabled: schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "Enable the unschedulable pods policy.",
						},
						FieldUnschedulablePodsPartialTemplateMatching: schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "Use partial template matching when deciding which custom node template to select.",
						},
					},
					Blocks: map[string]schema.Block{
						FieldUnschedulablePodsPodPinner: schema.ListNestedBlock{
							Description: "Pod Pinner component settings.",
							Validators:  []validator.List{listvalidator.SizeAtMost(1)},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									FieldPodPinnerEnabled: schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(false),
										Description: "Enable the Pod Pinner component.",
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
		equalNestedSections(plan.ClusterLimits, state.ClusterLimits) &&
		equalNestedSections(plan.NodeDownscaler, state.NodeDownscaler) &&
		equalNestedSections(plan.UnschedulablePods, state.UnschedulablePods) {
		plan.Version = state.Version
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
	}
}

// equalNestedSections returns true when both slice-shaped sections are equal
// in length and content. Used in ModifyPlan to compare typed nested structs
// without dereferencing nil pointers (the typed model uses slices since the
// schema uses ListNestedBlock). The inner fields are framework types
// (types.Bool, types.Int64, etc.) and slices of nested structs, all of which
// reflect.DeepEqual handles correctly.
func equalNestedSections[T any](a, b []T) bool {
	return reflect.DeepEqual(a, b)
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
// Empty / nil section slices produce nil SDK pointers; defaulted inner fields
// stay populated because the framework fills them at plan time.
func policiesFromModel(m *autoscalerPoliciesModel) *cluster_autoscaler_v2.PoliciesV2 {
	policies := &cluster_autoscaler_v2.PoliciesV2{}

	if !m.Enabled.IsNull() {
		policies.Enabled = lo.ToPtr(m.Enabled.ValueBool())
	}

	if !m.ScopedMode.IsNull() {
		policies.ScopedMode = lo.ToPtr(m.ScopedMode.ValueBool())
	}

	policies.ClusterLimits = clusterLimitsFromModel(m.ClusterLimits)
	policies.NodeDownscaler = nodeDownscalerFromModel(m.NodeDownscaler)
	policies.UnschedulablePods = unschedulablePodsFromModel(m.UnschedulablePods)

	// Include version from plan for optimistic locking on updates.
	if !m.Version.IsNull() && m.Version.ValueString() != "" {
		policies.Version = lo.ToPtr(m.Version.ValueString())
	}

	return policies
}

func clusterLimitsFromModel(s []clusterLimitsModel) *cluster_autoscaler_v2.ClusterLimitsPolicy {
	if len(s) == 0 {
		return nil
	}
	m := s[0]
	out := &cluster_autoscaler_v2.ClusterLimitsPolicy{}

	if !m.Enabled.IsNull() {
		out.Enabled = lo.ToPtr(m.Enabled.ValueBool())
	}
	out.Cpu = cpuFromModel(m.Cpu)

	return out
}

func cpuFromModel(s []cpuModel) *cluster_autoscaler_v2.ClusterLimitsCpu {
	if len(s) == 0 {
		return nil
	}
	m := s[0]
	cpu := &cluster_autoscaler_v2.ClusterLimitsCpu{}

	if !m.MaxCores.IsNull() {
		cpu.MaxCores = int32(m.MaxCores.ValueInt64())
	}
	if !m.MinCores.IsNull() {
		cpu.MinCores = lo.ToPtr(int32(m.MinCores.ValueInt64()))
	}

	return cpu
}

func nodeDownscalerFromModel(s []nodeDownscalerModel) *cluster_autoscaler_v2.NodeDownscalerPolicy {
	if len(s) == 0 {
		return nil
	}
	m := s[0]
	out := &cluster_autoscaler_v2.NodeDownscalerPolicy{}

	if !m.EmptyNodesEnabled.IsNull() {
		out.EmptyNodesEnabled = lo.ToPtr(m.EmptyNodesEnabled.ValueBool())
	}
	if !m.EmptyNodesDelay.IsNull() && m.EmptyNodesDelay.ValueString() != "" {
		out.EmptyNodesDelay = lo.ToPtr(m.EmptyNodesDelay.ValueString())
	}

	return out
}

func unschedulablePodsFromModel(s []unschedulablePodsModel) *cluster_autoscaler_v2.UnschedulablePodsPolicy {
	if len(s) == 0 {
		return nil
	}
	m := s[0]
	out := &cluster_autoscaler_v2.UnschedulablePodsPolicy{}

	if !m.Enabled.IsNull() {
		out.Enabled = lo.ToPtr(m.Enabled.ValueBool())
	}
	if !m.PartialTemplateMatchingEnabled.IsNull() {
		out.PartialTemplateMatchingEnabled = lo.ToPtr(m.PartialTemplateMatchingEnabled.ValueBool())
	}
	out.PodPinner = podPinnerFromModel(m.PodPinner)

	return out
}

func podPinnerFromModel(s []podPinnerModel) *cluster_autoscaler_v2.PodPinner {
	if len(s) == 0 {
		return nil
	}
	m := s[0]
	out := &cluster_autoscaler_v2.PodPinner{}

	if !m.Enabled.IsNull() {
		out.Enabled = lo.ToPtr(m.Enabled.ValueBool())
	}

	return out
}

// policiesToModel converts the SDK policies response to the Terraform state
// model. Nil SDK pointers flatten to nil section slices, so a section the
// configuration omits becomes absent and a section the API omits is dropped
// from state — preserveSectionPresence carries prior sections over for the
// latter case.
func (r *autoscalerPoliciesResource) policiesToModel(clusterID string, policies *cluster_autoscaler_v2.PoliciesV2) autoscalerPoliciesModel {
	return autoscalerPoliciesModel{
		ID:                types.StringValue(clusterID),
		ClusterID:         types.StringValue(clusterID),
		Enabled:           boolPtrToValue(policies.Enabled),
		ScopedMode:        boolPtrToValue(policies.ScopedMode),
		Version:           stringPtrToValue(policies.Version),
		ClusterLimits:     clusterLimitsToModel(policies.ClusterLimits),
		NodeDownscaler:    nodeDownscalerToModel(policies.NodeDownscaler),
		UnschedulablePods: unschedulablePodsToModel(policies.UnschedulablePods),
	}
}

// preserveSectionPresence carries sections the flatten did not produce over
// from the prior model, so an apply never loses a declared section to a
// sparse response. Sections are carried verbatim, never fabricated.
func (r *autoscalerPoliciesResource) preserveSectionPresence(state, prior autoscalerPoliciesModel) autoscalerPoliciesModel {
	if len(state.ClusterLimits) == 0 && len(prior.ClusterLimits) > 0 {
		state.ClusterLimits = prior.ClusterLimits
	}
	if len(state.NodeDownscaler) == 0 && len(prior.NodeDownscaler) > 0 {
		state.NodeDownscaler = prior.NodeDownscaler
	}
	if len(state.UnschedulablePods) == 0 && len(prior.UnschedulablePods) > 0 {
		state.UnschedulablePods = prior.UnschedulablePods
	}
	return state
}

func clusterLimitsToModel(in *cluster_autoscaler_v2.ClusterLimitsPolicy) []clusterLimitsModel {
	if in == nil || (in.Enabled == nil && in.Cpu == nil) {
		return nil
	}
	return []clusterLimitsModel{{
		Enabled: boolPtrToAttr(in.Enabled),
		Cpu:     cpuToModel(in.Cpu),
	}}
}

func cpuToModel(in *cluster_autoscaler_v2.ClusterLimitsCpu) []cpuModel {
	if in == nil {
		return nil
	}
	return []cpuModel{{
		MaxCores: types.Int64Value(int64(in.MaxCores)),
		MinCores: int32PtrToInt64Value(in.MinCores),
	}}
}

func nodeDownscalerToModel(in *cluster_autoscaler_v2.NodeDownscalerPolicy) []nodeDownscalerModel {
	if in == nil || (in.EmptyNodesDelay == nil && in.EmptyNodesEnabled == nil) {
		return nil
	}
	return []nodeDownscalerModel{{
		EmptyNodesEnabled: boolPtrToAttr(in.EmptyNodesEnabled),
		EmptyNodesDelay:   stringPtrToAttr(in.EmptyNodesDelay),
	}}
}

func unschedulablePodsToModel(in *cluster_autoscaler_v2.UnschedulablePodsPolicy) []unschedulablePodsModel {
	if in == nil || (in.Enabled == nil && in.PartialTemplateMatchingEnabled == nil && in.PodPinner == nil) {
		return nil
	}
	return []unschedulablePodsModel{{
		Enabled:                        boolPtrToAttr(in.Enabled),
		PartialTemplateMatchingEnabled: boolPtrToAttr(in.PartialTemplateMatchingEnabled),
		PodPinner:                      podPinnerToModel(in.PodPinner),
	}}
}

func podPinnerToModel(in *cluster_autoscaler_v2.PodPinner) []podPinnerModel {
	if in == nil {
		return nil
	}
	return []podPinnerModel{{
		Enabled: boolPtrToAttr(in.Enabled),
	}}
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

func stringPtrToAttr(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

func stringPtrToValue(s *string) types.String {
	if s == nil {
		return types.StringValue("")
	}
	return types.StringValue(*s)
}

func int32PtrToInt64Value(v *int32) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}
