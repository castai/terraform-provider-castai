package castai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	testingterraform "github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler_v2"
	mock_cluster_autoscaler_v2 "github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler_v2/mock"
)

const autoscalerPoliciesTestClusterID = "b6bfc074-a267-400f-b8f1-db0850c369b1"

func newAutoscalerPoliciesResourceWithMock(mockClient *mock_cluster_autoscaler_v2.MockClientWithResponsesInterface) *autoscalerPoliciesResource {
	return &autoscalerPoliciesResource{
		client: &ProviderConfig{
			clusterAutoscalerV2Client: mockClient,
		},
	}
}

func autoscalerPoliciesTestSchema(t *testing.T, r resource.Resource) *resource.SchemaResponse {
	t.Helper()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), "resource schema returned diagnostics: %v", schemaResp.Diagnostics)

	return schemaResp
}

func okHTTPResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}}
}

func notFoundHTTPResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusNotFound, Header: map[string][]string{"Content-Type": {"application/json"}}}
}

// expectUpdateCapture stubs the policies update call to return stored and
// returns the captured request body.
func expectUpdateCapture(t *testing.T, mockClient *mock_cluster_autoscaler_v2.MockClientWithResponsesInterface, stored *cluster_autoscaler_v2.PoliciesV2) *cluster_autoscaler_v2.PoliciesV2 {
	t.Helper()

	captured := &cluster_autoscaler_v2.PoliciesV2{}
	mockClient.EXPECT().
		PoliciesV2APIUpdateClusterPoliciesWithResponse(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ string, body cluster_autoscaler_v2.PoliciesV2, _ ...cluster_autoscaler_v2.RequestEditorFn) (*cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse, error) {
			*captured = body
			return &cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200:      stored,
			}, nil
		})
	return captured
}

func testAutoscalerPoliciesV2() *cluster_autoscaler_v2.PoliciesV2 {
	return &cluster_autoscaler_v2.PoliciesV2{
		Enabled:    lo.ToPtr(true),
		ScopedMode: lo.ToPtr(true),
		Version:    lo.ToPtr("v5"),
		ClusterLimits: &cluster_autoscaler_v2.ClusterLimitsPolicy{
			Enabled: lo.ToPtr(true),
			Cpu: &cluster_autoscaler_v2.ClusterLimitsCpu{
				MaxCores: 16,
				MinCores: lo.ToPtr(int32(2)),
			},
		},
		NodeDownscaler: &cluster_autoscaler_v2.NodeDownscalerPolicy{
			EmptyNodesDelay:   lo.ToPtr("3m"),
			EmptyNodesEnabled: lo.ToPtr(true),
		},
		UnschedulablePods: &cluster_autoscaler_v2.UnschedulablePodsPolicy{
			Enabled:                        lo.ToPtr(true),
			PartialTemplateMatchingEnabled: lo.ToPtr(true),
			PodPinner:                      &cluster_autoscaler_v2.PodPinner{Enabled: lo.ToPtr(true)},
		},
	}
}

// testClusterLimits / testNodeDownscaler / testUnschedulablePods build
// one-element section slices for tests, mirroring what Terraform core would
// plan when every inner field is declared.

func testClusterLimits(enabled bool, maxCores, minCores int64) []clusterLimitsModel {
	return []clusterLimitsModel{{
		Enabled: types.BoolValue(enabled),
		Cpu: []cpuModel{{
			MaxCores: types.Int64Value(maxCores),
			MinCores: types.Int64Value(minCores),
		}},
	}}
}

func testNodeDownscaler(enabled bool, delay string) []nodeDownscalerModel {
	return []nodeDownscalerModel{{
		EmptyNodesEnabled: types.BoolValue(enabled),
		EmptyNodesDelay:   types.StringValue(delay),
	}}
}

func testUnschedulablePods(enabled, partial bool, podPinnerEnabled bool) []unschedulablePodsModel {
	return []unschedulablePodsModel{{
		Enabled:                        types.BoolValue(enabled),
		PartialTemplateMatchingEnabled: types.BoolValue(partial),
		PodPinner:                      []podPinnerModel{{Enabled: types.BoolValue(podPinnerEnabled)}},
	}}
}

// testAutoscalerPoliciesPlanModel returns a full model covering every section,
// mirroring what Terraform core would plan. ID and version are unknown until
// the resource has been applied (the version changes on every write).
func testAutoscalerPoliciesPlanModel(clusterID string) autoscalerPoliciesModel {
	return autoscalerPoliciesModel{
		ID:                types.StringUnknown(),
		ClusterID:         types.StringValue(clusterID),
		Enabled:           types.BoolValue(true),
		ScopedMode:        types.BoolValue(true),
		Version:           types.StringUnknown(),
		ClusterLimits:     testClusterLimits(true, 16, 2),
		NodeDownscaler:    testNodeDownscaler(true, "3m"),
		UnschedulablePods: testUnschedulablePods(true, true, true),
	}
}

// autoscalerPoliciesPlanValue converts a typed model into the raw tftypes
// value the framework expects in requests, by round-tripping through
// tfsdk.Plan.Set: the framework knows how to encode typed nested structs via
// the schema, which types.ObjectValueFrom does not.
func autoscalerPoliciesPlanValue(t *testing.T, schemaResp *resource.SchemaResponse, model autoscalerPoliciesModel) tftypes.Value {
	t.Helper()

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	diags := plan.Set(context.Background(), model)
	require.False(t, diags.HasError(), "encoding plan: %v", diags)

	return plan.Raw
}

// autoscalerPoliciesNullValue returns the null raw value for the resource schema,
// the starting point for response states.
func autoscalerPoliciesNullValue(t *testing.T, schemaResp *resource.SchemaResponse) tftypes.Value {
	t.Helper()

	return tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), nil)
}

func TestResourceAutoscalerPolicies_Create(t *testing.T) {
	t.Parallel()

	clusterID := autoscalerPoliciesTestClusterID
	apiPolicies := testAutoscalerPoliciesV2()

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	capturedBody := expectUpdateCapture(t, mockClient, apiPolicies)
	mockClient.EXPECT().
		PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterID).
		Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
			HTTPResponse: okHTTPResponse(),
			JSON200:      apiPolicies,
		}, nil)

	r := newAutoscalerPoliciesResourceWithMock(mockClient)
	schemaResp := autoscalerPoliciesTestSchema(t, r)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Raw:    autoscalerPoliciesPlanValue(t, schemaResp, testAutoscalerPoliciesPlanModel(clusterID)),
			Schema: schemaResp.Schema,
		},
	}
	resp := resource.CreateResponse{
		State: tfsdk.State{
			Raw:    autoscalerPoliciesNullValue(t, schemaResp),
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "create diagnostics: %v", resp.Diagnostics)

	// The version is fetched from the API on create since the policies
	// already exist, and is included in the PUT for optimistic locking.
	require.NotNil(t, capturedBody.Version)
	require.Equal(t, "v5", *capturedBody.Version)

	// Field-level expand and flatten coverage lives in the policiesFromModel
	// and Read tests; this asserts the create plumbing.
	var state autoscalerPoliciesModel
	stateDiags := resp.State.Get(context.Background(), &state)
	require.False(t, stateDiags.HasError(), "state decode diagnostics: %v", stateDiags)

	require.Equal(t, clusterID, state.ID.ValueString())
	require.Equal(t, clusterID, state.ClusterID.ValueString())
	require.Equal(t, "v5", state.Version.ValueString())
	require.Len(t, state.ClusterLimits, 1)
	require.Len(t, state.NodeDownscaler, 1)
	require.Len(t, state.UnschedulablePods, 1)
}

func TestResourceAutoscalerPolicies_Create_NoExistingPolicies(t *testing.T) {
	t.Parallel()

	clusterID := autoscalerPoliciesTestClusterID

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	mockClient.EXPECT().
		PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterID).
		Times(1).
		Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
			HTTPResponse: notFoundHTTPResponse(),
		}, nil)
	capturedBody := expectUpdateCapture(t, mockClient, testAutoscalerPoliciesV2())

	r := newAutoscalerPoliciesResourceWithMock(mockClient)

	plan := &autoscalerPoliciesModel{
		ClusterID: types.StringValue(clusterID),
		Enabled:   types.BoolValue(true),
	}
	policies, diags := r.upsert(context.Background(), clusterID, plan)

	require.New(t).False(diags.HasError())
	// No existing policies: no version to lock on, the PUT omits it.
	require.Nil(t, capturedBody.Version)
	require.NotNil(t, policies)
	require.Equal(t, "v5", *policies.Version)
}

func TestResourceAutoscalerPolicies_Update(t *testing.T) {
	t.Parallel()

	clusterID := autoscalerPoliciesTestClusterID

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	capturedBody := expectUpdateCapture(t, mockClient, testAutoscalerPoliciesV2())

	r := newAutoscalerPoliciesResourceWithMock(mockClient)

	plan := autoscalerPoliciesModel{
		ClusterID:  types.StringValue(clusterID),
		Enabled:    types.BoolValue(true),
		ScopedMode: types.BoolValue(false),
		Version:    types.StringValue("v5"),
	}
	policies, diags := r.upsert(context.Background(), clusterID, &plan)

	require.New(t).False(diags.HasError())
	// The version from the plan must be sent for optimistic locking on updates.
	require.NotNil(t, capturedBody.Version)
	require.Equal(t, "v5", *capturedBody.Version)
	require.NotNil(t, policies)
	require.NotNil(t, policies.Version)
	require.Equal(t, "v5", *policies.Version)
}

func TestResourceAutoscalerPolicies_Update_UsesStateVersion(t *testing.T) {
	t.Parallel()

	clusterID := autoscalerPoliciesTestClusterID

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	capturedBody := expectUpdateCapture(t, mockClient, testAutoscalerPoliciesV2())

	r := newAutoscalerPoliciesResourceWithMock(mockClient)
	schemaResp := autoscalerPoliciesTestSchema(t, r)

	// Plan carries an unknown version (it changes on every write); the prior
	// state holds "v5", which must be used for optimistic locking.
	planModel := testAutoscalerPoliciesPlanModel(clusterID)
	planModel.ID = types.StringValue(clusterID)

	stateModel := testAutoscalerPoliciesPlanModel(clusterID)
	stateModel.ID = types.StringValue(clusterID)
	stateModel.Version = types.StringValue("v5")

	req := resource.UpdateRequest{
		Plan: tfsdk.Plan{
			Raw:    autoscalerPoliciesPlanValue(t, schemaResp, planModel),
			Schema: schemaResp.Schema,
		},
		State: tfsdk.State{
			Raw:    autoscalerPoliciesPlanValue(t, schemaResp, stateModel),
			Schema: schemaResp.Schema,
		},
	}
	resp := resource.UpdateResponse{
		State: tfsdk.State{
			Raw:    autoscalerPoliciesNullValue(t, schemaResp),
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "update diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, capturedBody.Version)
	require.Equal(t, "v5", *capturedBody.Version)

	var state autoscalerPoliciesModel
	stateDiags := resp.State.Get(context.Background(), &state)
	require.False(t, stateDiags.HasError(), "state decode diagnostics: %v", stateDiags)
	require.Equal(t, "v5", state.Version.ValueString())
}

func TestResourceAutoscalerPolicies_Read(t *testing.T) {
	t.Parallel()

	clusterID := autoscalerPoliciesTestClusterID

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	mockClient.EXPECT().
		PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterID).
		Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
			HTTPResponse: okHTTPResponse(),
			JSON200:      testAutoscalerPoliciesV2(),
		}, nil)

	r := newAutoscalerPoliciesResourceWithMock(mockClient)

	policies, found, diags := r.readPolicies(context.Background(), clusterID)
	require.New(t).False(diags.HasError())
	require.True(t, found)

	prior := autoscalerPoliciesModel{
		ID:                types.StringValue(clusterID),
		ClusterID:         types.StringValue(clusterID),
		Enabled:           types.BoolValue(true),
		ScopedMode:        types.BoolValue(true),
		ClusterLimits:     testClusterLimits(true, 100, 50), // declared with placeholder values
		NodeDownscaler:    testNodeDownscaler(true, "placeholder"),
		UnschedulablePods: testUnschedulablePods(true, true, true),
	}
	state := refreshedState(clusterID, policies, prior)
	require.Equal(t, clusterID, state.ID.ValueString())
	require.Equal(t, clusterID, state.ClusterID.ValueString())
	require.True(t, state.Enabled.ValueBool())
	require.True(t, state.ScopedMode.ValueBool())
	require.Equal(t, "v5", state.Version.ValueString())

	require.Len(t, state.ClusterLimits, 1)
	require.True(t, state.ClusterLimits[0].Enabled.ValueBool())
	require.Len(t, state.ClusterLimits[0].Cpu, 1)
	require.Equal(t, int64(16), state.ClusterLimits[0].Cpu[0].MaxCores.ValueInt64())
	require.Equal(t, int64(2), state.ClusterLimits[0].Cpu[0].MinCores.ValueInt64())

	require.Len(t, state.NodeDownscaler, 1)
	require.True(t, state.NodeDownscaler[0].EmptyNodesEnabled.ValueBool())
	require.Equal(t, "3m", state.NodeDownscaler[0].EmptyNodesDelay.ValueString())

	require.Len(t, state.UnschedulablePods, 1)
	require.True(t, state.UnschedulablePods[0].Enabled.ValueBool())
	require.True(t, state.UnschedulablePods[0].PartialTemplateMatchingEnabled.ValueBool())
	require.Len(t, state.UnschedulablePods[0].PodPinner, 1)
	require.True(t, state.UnschedulablePods[0].PodPinner[0].Enabled.ValueBool())
}

func TestResourceAutoscalerPolicies_Read_NotFound(t *testing.T) {
	t.Parallel()

	clusterID := autoscalerPoliciesTestClusterID

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	mockClient.EXPECT().
		PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterID).
		Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
			HTTPResponse: notFoundHTTPResponse(),
		}, nil)

	r := newAutoscalerPoliciesResourceWithMock(mockClient)

	_, found, diags := r.readPolicies(context.Background(), clusterID)

	require.New(t).False(diags.HasError())
	require.False(t, found)
}

func TestResourceAutoscalerPolicies_Read_NilNestedFields(t *testing.T) {
	t.Parallel()

	// Sections the API omits or returns empty flatten to nil slices; a
	// section serialized without its fields decodes all-nil.
	tests := map[string]*cluster_autoscaler_v2.PoliciesV2{
		"no sections": {},
		"all-nil section fields": {
			ClusterLimits:     &cluster_autoscaler_v2.ClusterLimitsPolicy{},
			NodeDownscaler:    &cluster_autoscaler_v2.NodeDownscalerPolicy{},
			UnschedulablePods: &cluster_autoscaler_v2.UnschedulablePodsPolicy{},
		},
	}

	var emptySection cluster_autoscaler_v2.PoliciesV2
	require.NoError(t, json.Unmarshal([]byte(`{"unschedulablePods":{}}`), &emptySection))
	tests["empty serialized section"] = &emptySection

	r := newAutoscalerPoliciesResourceWithMock(nil)

	for name, policies := range tests {
		t.Run(name, func(t *testing.T) {
			state := r.policiesToModel(autoscalerPoliciesTestClusterID, policies)

			// Top-level bools are concrete values, not null, to prevent drift.
			require.False(t, state.Enabled.ValueBool())
			require.False(t, state.ScopedMode.ValueBool())
			require.Empty(t, state.ClusterLimits)
			require.Empty(t, state.NodeDownscaler)
			require.Empty(t, state.UnschedulablePods)
		})
	}
}

func TestResourceAutoscalerPolicies_Read_MissingClusterID(t *testing.T) {
	t.Parallel()

	r := newAutoscalerPoliciesResourceWithMock(nil)
	schemaResp := autoscalerPoliciesTestSchema(t, r)

	// A state with neither cluster_id nor id is corrupt: the read must
	// surface an error instead of silently doing nothing.
	req := resource.ReadRequest{
		State: tfsdk.State{
			Raw:    autoscalerPoliciesNullValue(t, schemaResp),
			Schema: schemaResp.Schema,
		},
	}
	resp := resource.ReadResponse{
		State: tfsdk.State{
			Raw:    autoscalerPoliciesNullValue(t, schemaResp),
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, &resp)

	require.True(t, resp.Diagnostics.HasError())
}

func TestResourceAutoscalerPolicies_Read_UnschedulablePodsPartialMatchingOmitted(t *testing.T) {
	t.Parallel()

	// Server returns unschedulable_pods with only `enabled` set; the prior
	// state declared the section but left `partial_template_matching_enabled`
	// and `pod_pinner` unset. The refresh filters per-field, so the unset
	// fields stay at their declared null values (which the framework then
	// plans as the schema default on the next apply).
	policies := &cluster_autoscaler_v2.PoliciesV2{
		UnschedulablePods: &cluster_autoscaler_v2.UnschedulablePodsPolicy{
			Enabled: lo.ToPtr(true),
		},
	}
	prior := autoscalerPoliciesModel{
		ID: types.StringValue(autoscalerPoliciesTestClusterID),
		UnschedulablePods: []unschedulablePodsModel{{
			Enabled:                        types.BoolValue(true),
			PartialTemplateMatchingEnabled: types.BoolNull(),
			PodPinner:                      nil, // omitted in prior — should not appear
		}},
	}

	state := refreshedState(autoscalerPoliciesTestClusterID, policies, prior)

	require.Len(t, state.UnschedulablePods, 1)
	require.True(t, state.UnschedulablePods[0].Enabled.ValueBool())
	// API omitted the field: state stays null so an omitted configuration
	// adopts the stored value without drift.
	require.True(t, state.UnschedulablePods[0].PartialTemplateMatchingEnabled.IsNull())
	// pod_pinner was not in the prior either: should stay absent.
	require.Empty(t, state.UnschedulablePods[0].PodPinner)
}

func TestResourceAutoscalerPolicies_Read_MaterializedDefaults(t *testing.T) {
	t.Parallel()

	policies := &cluster_autoscaler_v2.PoliciesV2{
		Enabled:    lo.ToPtr(true),
		ScopedMode: lo.ToPtr(false),
		Version:    lo.ToPtr("v7"),
		ClusterLimits: &cluster_autoscaler_v2.ClusterLimitsPolicy{
			Enabled: lo.ToPtr(false),
			Cpu: &cluster_autoscaler_v2.ClusterLimitsCpu{
				MinCores: lo.ToPtr(int32(1)),
				MaxCores: 100,
			},
		},
		NodeDownscaler: &cluster_autoscaler_v2.NodeDownscalerPolicy{
			EmptyNodesEnabled: lo.ToPtr(false),
			EmptyNodesDelay:   lo.ToPtr("5m0s"),
		},
		UnschedulablePods: &cluster_autoscaler_v2.UnschedulablePodsPolicy{
			Enabled:                        lo.ToPtr(false),
			PartialTemplateMatchingEnabled: lo.ToPtr(false),
			PodPinner:                      &cluster_autoscaler_v2.PodPinner{Enabled: lo.ToPtr(false)},
		},
	}
	prior := autoscalerPoliciesModel{
		ID: types.StringValue(autoscalerPoliciesTestClusterID),
		// Declare every section so the refresh adopts the response values
		// for the fields the user is managing.
		ClusterLimits:     testClusterLimits(true, 0, 0),
		NodeDownscaler:    testNodeDownscaler(true, ""),
		UnschedulablePods: testUnschedulablePods(true, true, true),
	}

	state := refreshedState(autoscalerPoliciesTestClusterID, policies, prior)

	require.True(t, state.Enabled.ValueBool())

	require.Len(t, state.ClusterLimits, 1)
	require.False(t, state.ClusterLimits[0].Enabled.ValueBool())
	require.Len(t, state.ClusterLimits[0].Cpu, 1)
	require.Equal(t, int64(1), state.ClusterLimits[0].Cpu[0].MinCores.ValueInt64())
	require.Equal(t, int64(100), state.ClusterLimits[0].Cpu[0].MaxCores.ValueInt64())

	require.Len(t, state.NodeDownscaler, 1)
	require.Equal(t, "5m0s", state.NodeDownscaler[0].EmptyNodesDelay.ValueString())

	require.Len(t, state.UnschedulablePods, 1)
	require.False(t, state.UnschedulablePods[0].Enabled.ValueBool())
	require.Len(t, state.UnschedulablePods[0].PodPinner, 1)
}

func TestResourceAutoscalerPolicies_Delete(t *testing.T) {
	t.Parallel()

	r := newAutoscalerPoliciesResourceWithMock(nil)
	schemaResp := autoscalerPoliciesTestSchema(t, r)

	resp := resource.DeleteResponse{
		State: tfsdk.State{
			Raw:    autoscalerPoliciesNullValue(t, schemaResp),
			Schema: schemaResp.Schema,
		},
	}

	r.Delete(context.Background(), resource.DeleteRequest{}, &resp)

	require.New(t).False(resp.Diagnostics.HasError())
}

func TestResourceAutoscalerPolicies_Import(t *testing.T) {
	t.Parallel()

	clusterID := autoscalerPoliciesTestClusterID

	r := newAutoscalerPoliciesResourceWithMock(nil)
	schemaResp := autoscalerPoliciesTestSchema(t, r)

	req := resource.ImportStateRequest{ID: clusterID}
	resp := resource.ImportStateResponse{
		State: tfsdk.State{
			Raw:    autoscalerPoliciesNullValue(t, schemaResp),
			Schema: schemaResp.Schema,
		},
	}

	r.ImportState(context.Background(), req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "import diagnostics: %v", resp.Diagnostics)

	var state autoscalerPoliciesModel
	stateDiags := resp.State.Get(context.Background(), &state)
	require.False(t, stateDiags.HasError(), "state decode diagnostics: %v", stateDiags)
	require.Equal(t, clusterID, state.ID.ValueString())
}

func TestResourceAutoscalerPolicies_policiesFromModel(t *testing.T) {
	t.Parallel()

	model := &autoscalerPoliciesModel{
		ClusterID:         types.StringValue(autoscalerPoliciesTestClusterID),
		Enabled:           types.BoolValue(true),
		ScopedMode:        types.BoolValue(false),
		Version:           types.StringValue("v5"),
		ClusterLimits:     testClusterLimits(true, 16, 2),
		NodeDownscaler:    testNodeDownscaler(true, "3m"),
		UnschedulablePods: testUnschedulablePods(true, true, true),
	}

	policies := policiesFromModel(model)

	r := require.New(t)
	r.NotNil(policies)
	r.NotNil(policies.Enabled)
	r.True(*policies.Enabled)
	r.NotNil(policies.ScopedMode)
	r.False(*policies.ScopedMode)
	r.NotNil(policies.Version)
	r.Equal("v5", *policies.Version)
	r.NotNil(policies.ClusterLimits)
	r.NotNil(policies.ClusterLimits.Enabled)
	r.True(*policies.ClusterLimits.Enabled)
	r.NotNil(policies.ClusterLimits.Cpu)
	r.EqualValues(16, policies.ClusterLimits.Cpu.MaxCores)
	r.NotNil(policies.ClusterLimits.Cpu.MinCores)
	r.EqualValues(2, *policies.ClusterLimits.Cpu.MinCores)
	r.NotNil(policies.NodeDownscaler)
	r.NotNil(policies.NodeDownscaler.EmptyNodesDelay)
	r.Equal("3m", *policies.NodeDownscaler.EmptyNodesDelay)
	r.True(*policies.NodeDownscaler.EmptyNodesEnabled)
	r.NotNil(policies.UnschedulablePods)
	r.True(*policies.UnschedulablePods.Enabled)
	r.True(*policies.UnschedulablePods.PartialTemplateMatchingEnabled)
	r.NotNil(policies.UnschedulablePods.PodPinner)
	r.True(*policies.UnschedulablePods.PodPinner.Enabled)
}

func TestResourceAutoscalerPolicies_policiesFromModel_NilSections(t *testing.T) {
	t.Parallel()

	// A model with no sections omits them from the payload: the server
	// merge keeps the stored value.
	model := &autoscalerPoliciesModel{
		ClusterID: types.StringValue(autoscalerPoliciesTestClusterID),
	}

	policies := policiesFromModel(model)

	r := require.New(t)
	r.NotNil(policies)
	r.Nil(policies.Enabled)
	r.Nil(policies.ScopedMode)
	r.Nil(policies.Version)
	r.Nil(policies.ClusterLimits)
	r.Nil(policies.NodeDownscaler)
	r.Nil(policies.UnschedulablePods)
}

func TestResourceAutoscalerPolicies_policiesFromModel_SectionWithNullFields(t *testing.T) {
	t.Parallel()

	// Fields the configuration omits are not sent: the server merge keeps
	// the stored value.
	model := &autoscalerPoliciesModel{
		ClusterID: types.StringValue(autoscalerPoliciesTestClusterID),
		NodeDownscaler: []nodeDownscalerModel{{
			EmptyNodesEnabled: types.BoolValue(true),
			EmptyNodesDelay:   types.StringNull(),
		}},
	}

	policies := policiesFromModel(model)

	r := require.New(t)
	r.NotNil(policies.NodeDownscaler)
	r.NotNil(policies.NodeDownscaler.EmptyNodesEnabled)
	r.True(*policies.NodeDownscaler.EmptyNodesEnabled)
	r.Nil(policies.NodeDownscaler.EmptyNodesDelay)
}

// TestResourceAutoscalerPolicies_Schema_CPUMaxCoresRequired guards against a
// cpu block that omits max_cores passing plan: the V2 policies API requires
// max_cores (>= 1) and the SDK always serializes it, so an omitted value is
// sent as 0 and rejected at apply time. Required lets Terraform core reject
// the configuration at validate time instead.
func TestResourceAutoscalerPolicies_Schema_CPUMaxCoresRequired(t *testing.T) {
	t.Parallel()

	schemaResp := autoscalerPoliciesTestSchema(t, newAutoscalerPoliciesResource())

	r := require.New(t)
	clusterLimits, ok := schemaResp.Schema.Blocks[FieldAutoscalerPoliciesClusterLimits].(schema.ListNestedBlock)
	r.True(ok, "cluster_limits must be a list nested block")
	cpu, ok := clusterLimits.NestedObject.Blocks[FieldClusterLimitsCPU].(schema.ListNestedBlock)
	r.True(ok, "cluster_limits.cpu must be a list nested block")
	maxCores, ok := cpu.NestedObject.Attributes[FieldClusterLimitsCPUMaxCores]
	r.True(ok, "cluster_limits.cpu.max_cores must be declared")

	r.True(maxCores.IsRequired(), "cluster_limits.cpu.max_cores must be required")
	r.False(maxCores.IsComputed(), "cluster_limits.cpu.max_cores must not be computed: an unset value would stay unknown after apply")
}

// TestResourceAutoscalerPolicies_refreshedState_ObservedAPIResponse pins the
// refresh against a payload observed from the V2 policies endpoint. Prior
// state declares every section so the refresh adopts the response values.
func TestResourceAutoscalerPolicies_refreshedState_ObservedAPIResponse(t *testing.T) {
	t.Parallel()

	// Payload captured from GET /cluster-autoscaler/v2/clusters/{id}/policies.
	body := `{"enabled":true,"unschedulablePods":{"enabled":false,"podPinner":{"enabled":false,"status":"POD_PINNER_STATUS_COMPATIBLE"},"partialTemplateMatchingEnabled":false},"clusterLimits":{"enabled":true,"cpu":{"minCores":0,"maxCores":20}},"nodeDownscaler":{"emptyNodesEnabled":true,"emptyNodesDelay":"300s"},"scopedMode":false,"version":"3"}`

	var policies cluster_autoscaler_v2.PoliciesV2
	require.NoError(t, json.Unmarshal([]byte(body), &policies))

	prior := autoscalerPoliciesModel{
		ID:                types.StringValue(autoscalerPoliciesTestClusterID),
		ClusterLimits:     testClusterLimits(true, 0, 0),
		NodeDownscaler:    testNodeDownscaler(true, ""),
		UnschedulablePods: testUnschedulablePods(true, true, true),
	}
	state := refreshedState(autoscalerPoliciesTestClusterID, &policies, prior)

	require.True(t, state.Enabled.ValueBool())
	require.False(t, state.ScopedMode.ValueBool())
	require.Equal(t, "3", state.Version.ValueString())

	require.Len(t, state.ClusterLimits, 1)
	require.True(t, state.ClusterLimits[0].Enabled.ValueBool())
	require.Len(t, state.ClusterLimits[0].Cpu, 1)
	require.Equal(t, int64(20), state.ClusterLimits[0].Cpu[0].MaxCores.ValueInt64())
	require.Equal(t, int64(0), state.ClusterLimits[0].Cpu[0].MinCores.ValueInt64())

	require.Len(t, state.NodeDownscaler, 1)
	require.True(t, state.NodeDownscaler[0].EmptyNodesEnabled.ValueBool())
	require.Equal(t, "300s", state.NodeDownscaler[0].EmptyNodesDelay.ValueString())

	require.Len(t, state.UnschedulablePods, 1)
	require.False(t, state.UnschedulablePods[0].Enabled.ValueBool())
	require.False(t, state.UnschedulablePods[0].PartialTemplateMatchingEnabled.ValueBool())
	require.Len(t, state.UnschedulablePods[0].PodPinner, 1)
	require.False(t, state.UnschedulablePods[0].PodPinner[0].Enabled.ValueBool())
}

// TestResourceAutoscalerPolicies_ModifyPlan_SettlesVersionWhenConverged: when
// everything but the version matches the state, the version must stay known
// so a converged configuration plans to no changes; when anything else
// changes, the version stays unknown for the apply to report the new value.
func TestResourceAutoscalerPolicies_ModifyPlan_SettlesVersionWhenConverged(t *testing.T) {
	t.Parallel()

	clusterID := autoscalerPoliciesTestClusterID

	r := newAutoscalerPoliciesResourceWithMock(nil)
	schemaResp := autoscalerPoliciesTestSchema(t, r)

	state := testAutoscalerPoliciesPlanModel(clusterID)
	state.ID = types.StringValue(clusterID)
	state.Version = types.StringValue("v5")

	t.Run("settles the version when converged", func(t *testing.T) {
		plan := state
		plan.Version = types.StringUnknown()

		req := resource.ModifyPlanRequest{
			Plan:  tfsdk.Plan{Raw: autoscalerPoliciesPlanValue(t, schemaResp, plan), Schema: schemaResp.Schema},
			State: tfsdk.State{Raw: autoscalerPoliciesPlanValue(t, schemaResp, state), Schema: schemaResp.Schema},
		}
		resp := resource.ModifyPlanResponse{Plan: req.Plan}

		r.ModifyPlan(context.Background(), req, &resp)

		require.False(t, resp.Diagnostics.HasError(), "modify plan diagnostics: %v", resp.Diagnostics)

		var modified autoscalerPoliciesModel
		diags := resp.Plan.Get(context.Background(), &modified)
		require.False(t, diags.HasError(), "plan decode diagnostics: %v", diags)
		require.Equal(t, types.StringValue("v5"), modified.Version)
	})

	t.Run("keeps the version unknown when something changed", func(t *testing.T) {
		plan := state
		plan.Version = types.StringUnknown()
		plan.Enabled = types.BoolValue(false)

		req := resource.ModifyPlanRequest{
			Plan:  tfsdk.Plan{Raw: autoscalerPoliciesPlanValue(t, schemaResp, plan), Schema: schemaResp.Schema},
			State: tfsdk.State{Raw: autoscalerPoliciesPlanValue(t, schemaResp, state), Schema: schemaResp.Schema},
		}
		resp := resource.ModifyPlanResponse{Plan: req.Plan}

		r.ModifyPlan(context.Background(), req, &resp)

		require.False(t, resp.Diagnostics.HasError(), "modify plan diagnostics: %v", resp.Diagnostics)

		var modified autoscalerPoliciesModel
		diags := resp.Plan.Get(context.Background(), &modified)
		require.False(t, diags.HasError(), "plan decode diagnostics: %v", diags)
		require.True(t, modified.Version.IsUnknown())
	})
}

// TestResourceAutoscalerPolicies_PreservesBlocksMissingFromResponse: the API
// response lacks the unschedulable_pods section the configuration declares;
// the section must survive in state on every entry point.
func TestResourceAutoscalerPolicies_PreservesBlocksMissingFromResponse(t *testing.T) {
	t.Parallel()

	clusterID := autoscalerPoliciesTestClusterID

	// Response lacks the unschedulable_pods section.
	responseWithoutSection := &cluster_autoscaler_v2.PoliciesV2{
		Enabled:    lo.ToPtr(true),
		ScopedMode: lo.ToPtr(false),
		Version:    lo.ToPtr("v6"),
		ClusterLimits: &cluster_autoscaler_v2.ClusterLimitsPolicy{
			Enabled: lo.ToPtr(true),
			Cpu:     &cluster_autoscaler_v2.ClusterLimitsCpu{MaxCores: 20, MinCores: lo.ToPtr(int32(0))},
		},
		NodeDownscaler: &cluster_autoscaler_v2.NodeDownscalerPolicy{
			EmptyNodesDelay:   lo.ToPtr("300s"),
			EmptyNodesEnabled: lo.ToPtr(true),
		},
	}

	configuredModel := autoscalerPoliciesModel{
		ClusterID:         types.StringValue(clusterID),
		Enabled:           types.BoolValue(true),
		ScopedMode:        types.BoolValue(false),
		ClusterLimits:     testClusterLimits(true, 20, 0),
		NodeDownscaler:    testNodeDownscaler(true, "300s"),
		UnschedulablePods: []unschedulablePodsModel{{Enabled: types.BoolValue(false), PartialTemplateMatchingEnabled: types.BoolValue(false), PodPinner: nil}},
	}

	assertState := func(t *testing.T, state tfsdk.State) {
		t.Helper()

		var m autoscalerPoliciesModel
		diags := state.Get(context.Background(), &m)
		require.False(t, diags.HasError(), "state decode diagnostics: %v", diags)

		require.Equal(t, "v6", m.Version.ValueString())
		require.Len(t, m.ClusterLimits, 1)
		require.Len(t, m.NodeDownscaler, 1)
		// The unschedulable_pods section survives, carried over verbatim.
		require.Len(t, m.UnschedulablePods, 1)
		require.False(t, m.UnschedulablePods[0].Enabled.ValueBool())
		require.False(t, m.UnschedulablePods[0].PartialTemplateMatchingEnabled.ValueBool())
	}

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
		mockClient.EXPECT().
			PoliciesV2APIUpdateClusterPoliciesWithResponse(mock.Anything, clusterID, mock.Anything).
			Return(&cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200:      responseWithoutSection,
			}, nil)

		r := newAutoscalerPoliciesResourceWithMock(mockClient)
		schemaResp := autoscalerPoliciesTestSchema(t, r)

		stateModel := configuredModel
		stateModel.ID = types.StringValue(clusterID)
		stateModel.Version = types.StringValue("v5")

		req := resource.UpdateRequest{
			Plan:  tfsdk.Plan{Raw: autoscalerPoliciesPlanValue(t, schemaResp, configuredModel), Schema: schemaResp.Schema},
			State: tfsdk.State{Raw: autoscalerPoliciesPlanValue(t, schemaResp, stateModel), Schema: schemaResp.Schema},
		}
		resp := resource.UpdateResponse{
			State: tfsdk.State{Raw: autoscalerPoliciesNullValue(t, schemaResp), Schema: schemaResp.Schema},
		}

		r.Update(context.Background(), req, &resp)

		require.False(t, resp.Diagnostics.HasError(), "update diagnostics: %v", resp.Diagnostics)
		assertState(t, resp.State)
	})

	t.Run("create", func(t *testing.T) {
		t.Parallel()

		mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
		mockClient.EXPECT().
			PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterID).
			Times(1).
			Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200:      responseWithoutSection,
			}, nil)
		mockClient.EXPECT().
			PoliciesV2APIUpdateClusterPoliciesWithResponse(mock.Anything, clusterID, mock.Anything).
			Return(&cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200:      responseWithoutSection,
			}, nil)

		r := newAutoscalerPoliciesResourceWithMock(mockClient)
		schemaResp := autoscalerPoliciesTestSchema(t, r)

		planModel := configuredModel
		planModel.ID = types.StringNull()

		req := resource.CreateRequest{
			Plan: tfsdk.Plan{Raw: autoscalerPoliciesPlanValue(t, schemaResp, planModel), Schema: schemaResp.Schema},
		}
		resp := resource.CreateResponse{
			State: tfsdk.State{Raw: autoscalerPoliciesNullValue(t, schemaResp), Schema: schemaResp.Schema},
		}

		r.Create(context.Background(), req, &resp)

		require.False(t, resp.Diagnostics.HasError(), "create diagnostics: %v", resp.Diagnostics)
		assertState(t, resp.State)
	})

	t.Run("read", func(t *testing.T) {
		t.Parallel()

		mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
		mockClient.EXPECT().
			PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterID).
			Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200:      responseWithoutSection,
			}, nil)

		r := newAutoscalerPoliciesResourceWithMock(mockClient)
		schemaResp := autoscalerPoliciesTestSchema(t, r)

		priorState := configuredModel
		priorState.ID = types.StringValue(clusterID)
		priorState.Version = types.StringValue("v5")

		req := resource.ReadRequest{
			State: tfsdk.State{Raw: autoscalerPoliciesPlanValue(t, schemaResp, priorState), Schema: schemaResp.Schema},
		}
		resp := resource.ReadResponse{
			State: tfsdk.State{Raw: autoscalerPoliciesNullValue(t, schemaResp), Schema: schemaResp.Schema},
		}

		r.Read(context.Background(), req, &resp)

		require.False(t, resp.Diagnostics.HasError(), "read diagnostics: %v", resp.Diagnostics)
		assertState(t, resp.State)
	})
}

// TestResourceAutoscalerPolicies_stateFromResponse pins the post-apply
// state shape: sections come from the plan verbatim (Terraform core enforces
// plan block count == apply result block count), computed attributes come
// from the server response.
func TestResourceAutoscalerPolicies_stateFromResponse(t *testing.T) {
	t.Parallel()

	t.Run("sections mirror the plan verbatim", func(t *testing.T) {
		plan := autoscalerPoliciesModel{
			Enabled:           types.BoolValue(true),
			ScopedMode:        types.BoolValue(false),
			ClusterLimits:     testClusterLimits(true, 100, 0),
			NodeDownscaler:    testNodeDownscaler(true, "120s"),
			UnschedulablePods: testUnschedulablePods(true, false, false),
		}
		response := testAutoscalerPoliciesV2()

		got := stateFromResponse(autoscalerPoliciesTestClusterID, response, plan)

		require.Equal(t, plan.ClusterLimits, got.ClusterLimits)
		require.Equal(t, plan.NodeDownscaler, got.NodeDownscaler)
		require.Equal(t, plan.UnschedulablePods, got.UnschedulablePods)
		// Computed attributes come from the response.
		require.Equal(t, autoscalerPoliciesTestClusterID, got.ID.ValueString())
		require.Equal(t, autoscalerPoliciesTestClusterID, got.ClusterID.ValueString())
		require.Equal(t, "v5", got.Version.ValueString())
		require.True(t, got.Enabled.ValueBool())
	})

	t.Run("omitted sections stay absent regardless of response", func(t *testing.T) {
		plan := autoscalerPoliciesModel{Enabled: types.BoolValue(true)}
		response := testAutoscalerPoliciesV2() // server materializes every section

		got := stateFromResponse(autoscalerPoliciesTestClusterID, response, plan)

		require.Empty(t, got.ClusterLimits)
		require.Empty(t, got.NodeDownscaler)
		require.Empty(t, got.UnschedulablePods)
	})
}

// TestResourceAutoscalerPolicies_refreshedState pins the post-Read state
// shape: the always-materialized server response is filtered against the
// prior state's declared shape so server defaults don't leak in for
// unmanaged sections.
func TestResourceAutoscalerPolicies_refreshedState(t *testing.T) {
	t.Parallel()

	t.Run("undeclared sections are dropped from the materialized response", func(t *testing.T) {
		prior := autoscalerPoliciesModel{
			Enabled: types.BoolValue(true),
		}
		response := testAutoscalerPoliciesV2() // materializes all three

		got := refreshedState(autoscalerPoliciesTestClusterID, response, prior)

		require.Empty(t, got.ClusterLimits)
		require.Empty(t, got.NodeDownscaler)
		require.Empty(t, got.UnschedulablePods)
	})

	t.Run("declared section adopts response values", func(t *testing.T) {
		prior := autoscalerPoliciesModel{
			ClusterLimits: testClusterLimits(true, 100, 5), // declared with min_cores=5
		}
		// Server responds with min_cores=10, simulating server-side drift.
		response := &cluster_autoscaler_v2.PoliciesV2{
			Version: lo.ToPtr("v5"),
			ClusterLimits: &cluster_autoscaler_v2.ClusterLimitsPolicy{
				Enabled: lo.ToPtr(true),
				Cpu: &cluster_autoscaler_v2.ClusterLimitsCpu{
					MaxCores: 100,
					MinCores: lo.ToPtr(int32(10)),
				},
			},
		}

		got := refreshedState(autoscalerPoliciesTestClusterID, response, prior)

		require.Len(t, got.ClusterLimits, 1)
		require.Equal(t, int64(10), got.ClusterLimits[0].Cpu[0].MinCores.ValueInt64())
		require.Empty(t, got.NodeDownscaler)
		require.Empty(t, got.UnschedulablePods)
	})

	t.Run("sparse response: declared section missing from response carries over", func(t *testing.T) {
		prior := autoscalerPoliciesModel{
			ClusterLimits: testClusterLimits(true, 100, 5),
		}
		// Server returns only a non-section field.
		response := &cluster_autoscaler_v2.PoliciesV2{
			Version: lo.ToPtr("v5"),
		}

		got := refreshedState(autoscalerPoliciesTestClusterID, response, prior)

		require.Len(t, got.ClusterLimits, 1)
		require.Equal(t, prior.ClusterLimits, got.ClusterLimits)
	})

	t.Run("nested pod_pinner declared, omitted in response: carries over verbatim", func(t *testing.T) {
		prior := autoscalerPoliciesModel{
			UnschedulablePods: testUnschedulablePods(true, true, true),
		}
		// Server returns the section but omits pod_pinner.
		response := &cluster_autoscaler_v2.PoliciesV2{
			Version: lo.ToPtr("v5"),
			UnschedulablePods: &cluster_autoscaler_v2.UnschedulablePodsPolicy{
				Enabled:                        lo.ToPtr(true),
				PartialTemplateMatchingEnabled: lo.ToPtr(true),
				// PodPinner intentionally absent.
			},
		}

		got := refreshedState(autoscalerPoliciesTestClusterID, response, prior)

		require.Len(t, got.UnschedulablePods, 1)
		require.Len(t, got.UnschedulablePods[0].PodPinner, 1)
		require.Equal(t, prior.UnschedulablePods[0].PodPinner, got.UnschedulablePods[0].PodPinner)
	})

	t.Run("import state: id-only prior adopts the full response", func(t *testing.T) {
		prior := autoscalerPoliciesModel{
			ID: types.StringValue(autoscalerPoliciesTestClusterID),
			// ClusterID, sections: all absent (the post-import shape).
		}
		response := testAutoscalerPoliciesV2()

		got := refreshedState(autoscalerPoliciesTestClusterID, response, prior)

		require.Equal(t, autoscalerPoliciesTestClusterID, got.ID.ValueString())
		require.Equal(t, autoscalerPoliciesTestClusterID, got.ClusterID.ValueString())
		require.Len(t, got.ClusterLimits, 1)
		require.Len(t, got.NodeDownscaler, 1)
		require.Len(t, got.UnschedulablePods, 1)
	})
}

// TestResourceAutoscalerPolicies_Read_AfterImport covers the documented
// import-then-plan flow: state has only id (cluster id passthrough), and the
// first Read adopts the server's materialized response so the user can run
// `terraform plan` and see a stable diff against their config.
func TestResourceAutoscalerPolicies_Read_AfterImport(t *testing.T) {
	t.Parallel()

	clusterID := autoscalerPoliciesTestClusterID

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	mockClient.EXPECT().
		PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterID).
		Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
			HTTPResponse: okHTTPResponse(),
			JSON200:      testAutoscalerPoliciesV2(),
		}, nil)

	r := newAutoscalerPoliciesResourceWithMock(mockClient)
	schemaResp := autoscalerPoliciesTestSchema(t, r)

	// Post-ImportStatePassthroughID: id is set, everything else is null/empty.
	importedState := autoscalerPoliciesModel{
		ID: types.StringValue(clusterID),
	}

	req := resource.ReadRequest{
		State: tfsdk.State{
			Raw:    autoscalerPoliciesPlanValue(t, schemaResp, importedState),
			Schema: schemaResp.Schema,
		},
	}
	resp := resource.ReadResponse{
		State: tfsdk.State{
			Raw:    autoscalerPoliciesNullValue(t, schemaResp),
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "read diagnostics: %v", resp.Diagnostics)

	var state autoscalerPoliciesModel
	stateDiags := resp.State.Get(context.Background(), &state)
	require.False(t, stateDiags.HasError(), "state decode diagnostics: %v", stateDiags)
	require.Equal(t, clusterID, state.ID.ValueString())
	require.Equal(t, clusterID, state.ClusterID.ValueString())
	require.Len(t, state.ClusterLimits, 1)
	require.Len(t, state.NodeDownscaler, 1)
	require.Len(t, state.UnschedulablePods, 1)
}

// autoscalerPoliciesAccSettings holds the knobs for the V2 policies acceptance
// test configs.
type autoscalerPoliciesAccSettings struct {
	Enabled                  bool
	ScopedMode               bool
	ClusterLimitsEnabled     bool
	MinCores                 int
	MaxCores                 int
	EmptyNodesEnabled        bool
	EmptyNodesDelay          string
	UnschedulablePodsEnabled bool
	PartialTemplateMatching  bool
	PodPinnerEnabled         bool
}

// TestAccEKS_ResourceAutoscalerPolicies_basic covers the V2 autoscaler policies
// resource end-to-end against a real cluster. It registers the same EKS
// cluster as the V1 autoscaler acceptance test ("core-tf-acc-21-08-2025" by default,
// overridable via CLUSTER_NAME), and both resources manage autoscaling policies
// on that cluster, so it must not run in parallel with the V1 test: neither
// test calls t.Parallel(), which keeps them sequential within the package.
func TestAccEKS_ResourceAutoscalerPolicies_basic(t *testing.T) {
	rName := fmt.Sprintf("%v-policies-%v", ResourcePrefix, acctest.RandString(8))
	clusterName, _ := lo.Coalesce(os.Getenv("CLUSTER_NAME"), "core-tf-acc-21-08-2025")

	initial := autoscalerPoliciesAccSettings{
		Enabled:                  true,
		ScopedMode:               false,
		ClusterLimitsEnabled:     true,
		MinCores:                 1,
		MaxCores:                 100,
		EmptyNodesEnabled:        true,
		EmptyNodesDelay:          "120s",
		UnschedulablePodsEnabled: true,
		PartialTemplateMatching:  true,
		PodPinnerEnabled:         false,
	}
	updated := autoscalerPoliciesAccSettings{
		Enabled:                  false,
		ScopedMode:               true,
		ClusterLimitsEnabled:     false,
		MinCores:                 2,
		MaxCores:                 200,
		EmptyNodesEnabled:        false,
		EmptyNodesDelay:          "300s",
		UnschedulablePodsEnabled: false,
		PartialTemplateMatching:  false,
		PodPinnerEnabled:         true,
	}

	tfresource.Test(t, tfresource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]tfresource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "~> 5.0",
			},
		},
		Steps: []tfresource.TestStep{
			// Step 1: Apply initial V2 policies.
			{
				Config: testAccAutoscalerPoliciesConfig(rName, clusterName, initial),
				Check:  testAccCheckAutoscalerPolicies(initial),
			},
			// Step 2: Update every policy block.
			{
				Config: testAccAutoscalerPoliciesConfig(rName, clusterName, updated),
				Check:  testAccCheckAutoscalerPolicies(updated),
			},
			// Step 3: Import the resource by cluster id.
			{
				ResourceName: "castai_autoscaler_policies.test",
				ImportStateIdFunc: func(s *testingterraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["castai_eks_cluster.test"]
					if !ok {
						return "", fmt.Errorf("castai_eks_cluster.test not found in state")
					}
					return rs.Primary.ID, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 4: Re-apply the initial policies to verify updating back.
			{
				Config: testAccAutoscalerPoliciesConfig(rName, clusterName, initial),
				Check:  testAccCheckAutoscalerPolicies(initial),
			},
		},
	})
}

// TestAccEKS_ResourceAutoscalerPolicies_omittedDefaults exercises CSU-6199's
// regression path end-to-end: declare a section while omitting the optional
// inner fields whose server-side defaults the resource must respect. The
// second plan must report no diff on the omitted fields (the framework
// populates them from the schema Defaults and they round-trip through state
// unchanged).
func TestAccEKS_ResourceAutoscalerPolicies_omittedDefaults(t *testing.T) {
	rName := fmt.Sprintf("%v-policies-omitted-%v", ResourcePrefix, acctest.RandString(8))
	clusterName, _ := lo.Coalesce(os.Getenv("CLUSTER_NAME"), "core-tf-acc-21-08-2025")

	tfresource.Test(t, tfresource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]tfresource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "~> 5.0",
			},
		},
		Steps: []tfresource.TestStep{
			{
				Config: testAccAutoscalerPoliciesOmittedDefaultsConfig(rName, clusterName),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("castai_autoscaler_policies.test", "cluster_limits.0.cpu.0.min_cores", "1"),
					tfresource.TestCheckResourceAttr("castai_autoscaler_policies.test", "unschedulable_pods.0.partial_template_matching_enabled", "false"),
					// An entirely omitted nested block stays absent from state; the
					// server response is filtered to the plan shape on Read.
					tfresource.TestCheckResourceAttr("castai_autoscaler_policies.test", "unschedulable_pods.0.pod_pinner.#", "0"),
				),
			},
			// Re-apply: the schema Defaults must match what the server returned,
			// so the second plan reports no changes on the omitted fields.
			{
				Config: testAccAutoscalerPoliciesOmittedDefaultsConfig(rName, clusterName),
			},
		},
	})
}

// TestAccEKS_ResourceAutoscalerPolicies_cpuRequiresMaxCores verifies that a
// cpu block without max_cores is rejected at validate time, before any API
// call. The configuration never reaches the provider's Configure, so no
// cluster or credentials are needed.
func TestAccEKS_ResourceAutoscalerPolicies_cpuRequiresMaxCores(t *testing.T) {
	missingMaxCores := regexp.MustCompile(`The argument "max_cores" is required`)

	tfresource.Test(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config:      testAccAutoscalerPoliciesCPUConfig("min_cores = 1"),
				PlanOnly:    true,
				ExpectError: missingMaxCores,
			},
			{
				Config:      testAccAutoscalerPoliciesCPUConfig(""),
				PlanOnly:    true,
				ExpectError: missingMaxCores,
			},
		},
	})
}

func testAccAutoscalerPoliciesCPUConfig(cpuBody string) string {
	return fmt.Sprintf(`
resource "castai_autoscaler_policies" "test" {
  cluster_id = %q

  cluster_limits {
    enabled = true

    cpu {
      %s
    }
  }
}
`, autoscalerPoliciesTestClusterID, cpuBody)
}

func testAccAutoscalerPoliciesConfig(rName, clusterName string, s autoscalerPoliciesAccSettings) string {
	return ConfigCompose(testAccEKSClusterConfig(rName, clusterName), fmt.Sprintf(`
resource "castai_autoscaler_policies" "test" {
  cluster_id = castai_eks_cluster.test.id

  enabled     = %t
  scoped_mode = %t

  cluster_limits {
    enabled = %t

    cpu {
      max_cores = %d
      min_cores = %d
    }
  }

  node_downscaler {
    empty_nodes_enabled = %t
    empty_nodes_delay   = %q
  }

  unschedulable_pods {
    enabled                           = %t
    partial_template_matching_enabled = %t

    pod_pinner {
      enabled = %t
    }
  }
}
`, s.Enabled, s.ScopedMode, s.ClusterLimitsEnabled, s.MaxCores, s.MinCores,
		s.EmptyNodesEnabled, s.EmptyNodesDelay, s.UnschedulablePodsEnabled, s.PartialTemplateMatching, s.PodPinnerEnabled))
}

// testAccAutoscalerPoliciesOmittedDefaultsConfig exercises the regression path
// from CSU-6199: the cluster_limits.cpu.min_cores, unschedulable_pods.partial_template_matching_enabled,
// and unschedulable_pods.pod_pinner.enabled fields are omitted from the config
// and rely on the schema's Default values to match the server's defaults.
func testAccAutoscalerPoliciesOmittedDefaultsConfig(rName, clusterName string) string {
	return ConfigCompose(testAccEKSClusterConfig(rName, clusterName), `
resource "castai_autoscaler_policies" "test" {
  cluster_id = castai_eks_cluster.test.id

  enabled = true

  cluster_limits {
    enabled = true

    cpu {
      max_cores = 100
    }
  }

  node_downscaler {
    empty_nodes_enabled = true
    empty_nodes_delay   = "120s"
  }

  unschedulable_pods {
    enabled = true
  }
}
`)
}

func testAccCheckAutoscalerPolicies(s autoscalerPoliciesAccSettings) tfresource.TestCheckFunc {
	resourceName := "castai_autoscaler_policies.test"

	return tfresource.ComposeTestCheckFunc(
		tfresource.TestCheckResourceAttrSet(resourceName, "cluster_id"),
		tfresource.TestCheckResourceAttrSet(resourceName, "id"),
		tfresource.TestCheckResourceAttrSet(resourceName, "version"),
		tfresource.TestCheckResourceAttr(resourceName, "enabled", strconv.FormatBool(s.Enabled)),
		tfresource.TestCheckResourceAttr(resourceName, "scoped_mode", strconv.FormatBool(s.ScopedMode)),
		tfresource.TestCheckResourceAttr(resourceName, "cluster_limits.0.enabled", strconv.FormatBool(s.ClusterLimitsEnabled)),
		tfresource.TestCheckResourceAttr(resourceName, "cluster_limits.0.cpu.0.max_cores", strconv.Itoa(s.MaxCores)),
		tfresource.TestCheckResourceAttr(resourceName, "cluster_limits.0.cpu.0.min_cores", strconv.Itoa(s.MinCores)),
		tfresource.TestCheckResourceAttr(resourceName, "node_downscaler.0.empty_nodes_enabled", strconv.FormatBool(s.EmptyNodesEnabled)),
		tfresource.TestCheckResourceAttr(resourceName, "node_downscaler.0.empty_nodes_delay", s.EmptyNodesDelay),
		tfresource.TestCheckResourceAttr(resourceName, "unschedulable_pods.0.enabled", strconv.FormatBool(s.UnschedulablePodsEnabled)),
		tfresource.TestCheckResourceAttr(resourceName, "unschedulable_pods.0.partial_template_matching_enabled", strconv.FormatBool(s.PartialTemplateMatching)),
		tfresource.TestCheckResourceAttr(resourceName, "unschedulable_pods.0.pod_pinner.0.enabled", strconv.FormatBool(s.PodPinnerEnabled)),
	)
}
