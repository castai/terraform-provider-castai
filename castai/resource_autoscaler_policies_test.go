package castai

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler_v2"
	mock_cluster_autoscaler_v2 "github.com/castai/terraform-provider-castai/castai/sdk/cluster_autoscaler_v2/mock"
)

func newAutoscalerPoliciesResourceWithMock(mockClient *mock_cluster_autoscaler_v2.MockClientWithResponsesInterface) *autoscalerPoliciesResource {
	return &autoscalerPoliciesResource{
		client: &ProviderConfig{
			clusterAutoscalerV2Client: mockClient,
		},
	}
}

func autoscalerPoliciesTestSchema(t *testing.T, r resource.Resource) (*resource.SchemaResponse, tftypes.Type) {
	t.Helper()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), "resource schema returned diagnostics: %v", schemaResp.Diagnostics)

	return schemaResp, schemaResp.Schema.Type().TerraformType(context.Background())
}

func okHTTPResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}}
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

// autoscalerPoliciesFullPlanValue builds a full plan covering every block as a
// raw tftypes value, mirroring what Terraform core would send to Create.
func autoscalerPoliciesFullPlanValue(t *testing.T, schemaType tftypes.Type, clusterID string) tftypes.Value {
	t.Helper()

	objType := schemaType.(tftypes.Object)
	attrTypes := objType.AttributeTypes

	limitsType := attrTypes[FieldAutoscalerPoliciesClusterLimits].(tftypes.List).ElementType.(tftypes.Object)
	cpuType := limitsType.AttributeTypes[FieldClusterLimitsCPU].(tftypes.List).ElementType.(tftypes.Object)
	downscalerType := attrTypes[FieldAutoscalerPoliciesNodeDownscaler].(tftypes.List).ElementType.(tftypes.Object)
	unschedulableType := attrTypes[FieldAutoscalerPoliciesUnschedulablePods].(tftypes.List).ElementType.(tftypes.Object)
	podPinnerType := unschedulableType.AttributeTypes[FieldUnschedulablePodsPodPinner].(tftypes.List).ElementType.(tftypes.Object)

	b := func(v bool) tftypes.Value { return tftypes.NewValue(tftypes.Bool, v) }
	n := func(v int64) tftypes.Value { return tftypes.NewValue(tftypes.Number, float64(v)) }
	s := func(v string) tftypes.Value { return tftypes.NewValue(tftypes.String, v) }
	list := func(el tftypes.Type, vals ...tftypes.Value) tftypes.Value {
		return tftypes.NewValue(tftypes.List{ElementType: el}, vals)
	}
	object := func(obj tftypes.Object, attrs map[string]tftypes.Value) tftypes.Value {
		return tftypes.NewValue(obj, attrs)
	}

	return tftypes.NewValue(objType, map[string]tftypes.Value{
		FieldAutoscalerPoliciesID:         tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		FieldClusterId:                    s(clusterID),
		FieldAutoscalerPoliciesEnabled:    b(true),
		FieldAutoscalerPoliciesScopedMode: b(true),
		FieldAutoscalerPoliciesVersion:    tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		FieldAutoscalerPoliciesClusterLimits: list(limitsType, object(limitsType, map[string]tftypes.Value{
			FieldClusterLimitsEnabled: b(true),
			FieldClusterLimitsCPU: list(cpuType, object(cpuType, map[string]tftypes.Value{
				FieldClusterLimitsCPUMaxCores: n(16),
				FieldClusterLimitsCPUMinCores: n(2),
			})),
		})),
		FieldAutoscalerPoliciesNodeDownscaler: list(downscalerType, object(downscalerType, map[string]tftypes.Value{
			FieldNodeDownscalerEmptyNodesDelay:   s("3m"),
			FieldNodeDownscalerEmptyNodesEnabled: b(true),
		})),
		FieldAutoscalerPoliciesUnschedulablePods: list(unschedulableType, object(unschedulableType, map[string]tftypes.Value{
			FieldUnschedulablePodsEnabled:                 b(true),
			FieldUnschedulablePodsPartialTemplateMatching: b(true),
			FieldUnschedulablePodsPodPinner: list(podPinnerType, object(podPinnerType, map[string]tftypes.Value{
				FieldPodPinnerEnabled: b(true),
			})),
		})),
	})
}

func TestResourceAutoscalerPolicies_Create(t *testing.T) {
	t.Parallel()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	apiPolicies := testAutoscalerPoliciesV2()
	var capturedBody cluster_autoscaler_v2.PoliciesV2

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)

	mockClient.EXPECT().
		PoliciesV2APIUpdateClusterPoliciesWithResponse(mock.Anything, clusterID, mock.Anything).
		RunAndReturn(func(_ context.Context, _ string, body cluster_autoscaler_v2.PoliciesV2, _ ...cluster_autoscaler_v2.RequestEditorFn) (*cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse, error) {
			capturedBody = body
			return &cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200:      apiPolicies,
			}, nil
		})
	mockClient.EXPECT().
		PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterID).
		Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
			HTTPResponse: okHTTPResponse(),
			JSON200:      apiPolicies,
		}, nil)

	r := newAutoscalerPoliciesResourceWithMock(mockClient)
	schemaResp, schemaType := autoscalerPoliciesTestSchema(t, r)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Raw:    autoscalerPoliciesFullPlanValue(t, schemaType, clusterID),
			Schema: schemaResp.Schema,
		},
	}
	resp := resource.CreateResponse{
		State: tfsdk.State{
			Raw:    tftypes.NewValue(schemaType, nil),
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "create diagnostics: %v", resp.Diagnostics)

	// Assert the payload sent to the API (expand path).
	require.NotNil(t, capturedBody.Enabled)
	require.True(t, *capturedBody.Enabled)
	require.NotNil(t, capturedBody.ScopedMode)
	require.True(t, *capturedBody.ScopedMode)
	require.NotNil(t, capturedBody.ClusterLimits)
	require.NotNil(t, capturedBody.ClusterLimits.Enabled)
	require.True(t, *capturedBody.ClusterLimits.Enabled)
	require.NotNil(t, capturedBody.ClusterLimits.Cpu)
	require.EqualValues(t, 16, capturedBody.ClusterLimits.Cpu.MaxCores)
	require.NotNil(t, capturedBody.ClusterLimits.Cpu.MinCores)
	require.EqualValues(t, 2, *capturedBody.ClusterLimits.Cpu.MinCores)
	require.NotNil(t, capturedBody.NodeDownscaler)
	require.NotNil(t, capturedBody.NodeDownscaler.EmptyNodesDelay)
	require.Equal(t, "3m", *capturedBody.NodeDownscaler.EmptyNodesDelay)
	require.NotNil(t, capturedBody.NodeDownscaler.EmptyNodesEnabled)
	require.True(t, *capturedBody.NodeDownscaler.EmptyNodesEnabled)
	require.NotNil(t, capturedBody.UnschedulablePods)
	require.NotNil(t, capturedBody.UnschedulablePods.Enabled)
	require.True(t, *capturedBody.UnschedulablePods.Enabled)
	require.NotNil(t, capturedBody.UnschedulablePods.PartialTemplateMatchingEnabled)
	require.True(t, *capturedBody.UnschedulablePods.PartialTemplateMatchingEnabled)
	require.NotNil(t, capturedBody.UnschedulablePods.PodPinner)
	require.NotNil(t, capturedBody.UnschedulablePods.PodPinner.Enabled)
	require.True(t, *capturedBody.UnschedulablePods.PodPinner.Enabled)

	// Assert the state written after the read-back (flatten path).
	var state autoscalerPoliciesModel
	stateDiags := resp.State.Get(context.Background(), &state)
	require.False(t, stateDiags.HasError(), "state decode diagnostics: %v", stateDiags)

	require.Equal(t, clusterID, state.ID.ValueString())
	require.Equal(t, clusterID, state.ClusterID.ValueString())
	require.True(t, state.Enabled.ValueBool())
	require.True(t, state.ScopedMode.ValueBool())
	require.Equal(t, "v5", state.Version.ValueString())
	require.Len(t, state.ClusterLimits, 1)
	require.True(t, state.ClusterLimits[0].Enabled.ValueBool())
	require.Len(t, state.ClusterLimits[0].CPU, 1)
	require.EqualValues(t, 16, state.ClusterLimits[0].CPU[0].MaxCores.ValueInt64())
	require.EqualValues(t, 2, state.ClusterLimits[0].CPU[0].MinCores.ValueInt64())
	require.Len(t, state.NodeDownscaler, 1)
	require.Equal(t, "3m", state.NodeDownscaler[0].EmptyNodesDelay.ValueString())
	require.True(t, state.NodeDownscaler[0].EmptyNodesEnabled.ValueBool())
	require.Len(t, state.UnschedulablePods, 1)
	require.True(t, state.UnschedulablePods[0].Enabled.ValueBool())
	require.True(t, state.UnschedulablePods[0].PartialTemplateMatchingEnabled.ValueBool())
	require.Len(t, state.UnschedulablePods[0].PodPinner, 1)
	require.True(t, state.UnschedulablePods[0].PodPinner[0].Enabled.ValueBool())
}

func TestResourceAutoscalerPolicies_Update(t *testing.T) {
	t.Parallel()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	var capturedBody cluster_autoscaler_v2.PoliciesV2

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)

	mockClient.EXPECT().
		PoliciesV2APIUpdateClusterPoliciesWithResponse(mock.Anything, clusterID, mock.Anything).
		RunAndReturn(func(_ context.Context, _ string, body cluster_autoscaler_v2.PoliciesV2, _ ...cluster_autoscaler_v2.RequestEditorFn) (*cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse, error) {
			capturedBody = body
			return &cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200:      testAutoscalerPoliciesV2(),
			}, nil
		})
	mockClient.EXPECT().
		PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterID).
		Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
			HTTPResponse: okHTTPResponse(),
			JSON200:      testAutoscalerPoliciesV2(),
		}, nil)

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

func TestResourceAutoscalerPolicies_Read(t *testing.T) {
	t.Parallel()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

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

	state := r.policiesToModel(clusterID, policies)
	require.Equal(t, clusterID, state.ID.ValueString())
	require.Equal(t, clusterID, state.ClusterID.ValueString())
	require.True(t, state.Enabled.ValueBool())
	require.True(t, state.ScopedMode.ValueBool())
	require.Equal(t, "v5", state.Version.ValueString())
	require.Len(t, state.ClusterLimits, 1)
	require.True(t, state.ClusterLimits[0].Enabled.ValueBool())
	require.EqualValues(t, 16, state.ClusterLimits[0].CPU[0].MaxCores.ValueInt64())
	require.EqualValues(t, 2, state.ClusterLimits[0].CPU[0].MinCores.ValueInt64())
	require.Len(t, state.NodeDownscaler, 1)
	require.Equal(t, "3m", state.NodeDownscaler[0].EmptyNodesDelay.ValueString())
	require.True(t, state.NodeDownscaler[0].EmptyNodesEnabled.ValueBool())
	require.Len(t, state.UnschedulablePods, 1)
	require.True(t, state.UnschedulablePods[0].Enabled.ValueBool())
	require.True(t, state.UnschedulablePods[0].PartialTemplateMatchingEnabled.ValueBool())
	require.Len(t, state.UnschedulablePods[0].PodPinner, 1)
	require.True(t, state.UnschedulablePods[0].PodPinner[0].Enabled.ValueBool())
}

func TestResourceAutoscalerPolicies_Read_NotFound(t *testing.T) {
	t.Parallel()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)
	mockClient.EXPECT().
		PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterID).
		Return(&cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusNotFound, Header: map[string][]string{"Content-Type": {"application/json"}}},
			JSON200:      nil,
		}, nil)

	r := newAutoscalerPoliciesResourceWithMock(mockClient)

	_, found, diags := r.readPolicies(context.Background(), clusterID)

	require.New(t).False(diags.HasError())
	require.False(t, found)
}

func TestResourceAutoscalerPolicies_Read_NilNestedFields(t *testing.T) {
	t.Parallel()

	r := newAutoscalerPoliciesResourceWithMock(nil)

	state := r.policiesToModel("b6bfc074-a267-400f-b8f1-db0850c369b1", &cluster_autoscaler_v2.PoliciesV2{})

	// Bool defaults are concrete values, not null, to prevent state drift.
	require.False(t, state.Enabled.ValueBool())
	require.False(t, state.ScopedMode.ValueBool())
	require.Equal(t, "", state.Version.ValueString())
	require.Empty(t, state.ClusterLimits)
	require.Empty(t, state.NodeDownscaler)
	require.Empty(t, state.UnschedulablePods)
}

func TestResourceAutoscalerPolicies_Read_UnschedulablePodsPartialMatchingOmitted(t *testing.T) {
	t.Parallel()

	r := newAutoscalerPoliciesResourceWithMock(nil)

	policies := &cluster_autoscaler_v2.PoliciesV2{
		UnschedulablePods: &cluster_autoscaler_v2.UnschedulablePodsPolicy{
			Enabled: lo.ToPtr(true),
		},
	}

	state := r.policiesToModel("b6bfc074-a267-400f-b8f1-db0850c369b1", policies)

	require.Len(t, state.UnschedulablePods, 1)
	require.True(t, state.UnschedulablePods[0].Enabled.ValueBool())
	// API omitted the field: state stays at the static default (false), no drift.
	require.False(t, state.UnschedulablePods[0].PartialTemplateMatchingEnabled.ValueBool())
}

func TestResourceAutoscalerPolicies_Delete(t *testing.T) {
	t.Parallel()

	r := newAutoscalerPoliciesResourceWithMock(nil)
	schemaResp, schemaType := autoscalerPoliciesTestSchema(t, r)

	resp := resource.DeleteResponse{
		State: tfsdk.State{
			Raw:    tftypes.NewValue(schemaType, nil),
			Schema: schemaResp.Schema,
		},
	}

	r.Delete(context.Background(), resource.DeleteRequest{}, &resp)

	require.New(t).False(resp.Diagnostics.HasError())
}

func TestResourceAutoscalerPolicies_Import(t *testing.T) {
	t.Parallel()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	r := newAutoscalerPoliciesResourceWithMock(nil)
	schemaResp, schemaType := autoscalerPoliciesTestSchema(t, r)

	req := resource.ImportStateRequest{ID: clusterID}
	resp := resource.ImportStateResponse{
		State: tfsdk.State{
			Raw:    tftypes.NewValue(schemaType, nil),
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
		ClusterID:  types.StringValue("b6bfc074-a267-400f-b8f1-db0850c369b1"),
		Enabled:    types.BoolValue(true),
		ScopedMode: types.BoolValue(false),
		Version:    types.StringValue("v5"),
		ClusterLimits: []clusterLimitsModel{{
			Enabled: types.BoolValue(true),
			CPU:     []clusterLimitsCPUModel{{MaxCores: types.Int64Value(16), MinCores: types.Int64Value(2)}},
		}},
		NodeDownscaler: []nodeDownscalerModel{{
			EmptyNodesDelay:   types.StringValue("3m"),
			EmptyNodesEnabled: types.BoolValue(true),
		}},
		UnschedulablePods: []unschedulablePodsModel{{
			Enabled:                        types.BoolValue(true),
			PartialTemplateMatchingEnabled: types.BoolValue(true),
			PodPinner:                      []podPinnerModel{{Enabled: types.BoolValue(true)}},
		}},
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
	r.NotNil(policies.ClusterLimits.Cpu)
	r.EqualValues(16, policies.ClusterLimits.Cpu.MaxCores)
	r.NotNil(policies.ClusterLimits.Cpu.MinCores)
	r.EqualValues(2, *policies.ClusterLimits.Cpu.MinCores)
	r.NotNil(policies.NodeDownscaler)
	r.NotNil(policies.NodeDownscaler.EmptyNodesDelay)
	r.Equal("3m", *policies.NodeDownscaler.EmptyNodesDelay)
	r.NotNil(policies.UnschedulablePods)
	r.NotNil(policies.UnschedulablePods.PartialTemplateMatchingEnabled)
	r.True(*policies.UnschedulablePods.PartialTemplateMatchingEnabled)
	r.NotNil(policies.UnschedulablePods.PodPinner)
	r.True(*policies.UnschedulablePods.PodPinner.Enabled)
}

func TestResourceAutoscalerPolicies_policiesFromModel_NullFields(t *testing.T) {
	t.Parallel()

	// A minimal model with unset top-level fields omits them from the payload.
	model := &autoscalerPoliciesModel{
		ClusterID: types.StringValue("b6bfc074-a267-400f-b8f1-db0850c369b1"),
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

func TestResourceAutoscalerPolicies_policiesToModel_EmptyNestedPolicies(t *testing.T) {
	t.Parallel()

	r := newAutoscalerPoliciesResourceWithMock(nil)

	// Nested policies with all-nil fields flatten to absent blocks.
	state := r.policiesToModel("b6bfc074-a267-400f-b8f1-db0850c369b1", &cluster_autoscaler_v2.PoliciesV2{
		ClusterLimits:     &cluster_autoscaler_v2.ClusterLimitsPolicy{},
		NodeDownscaler:    &cluster_autoscaler_v2.NodeDownscalerPolicy{},
		UnschedulablePods: &cluster_autoscaler_v2.UnschedulablePodsPolicy{},
	})

	require.Empty(t, state.ClusterLimits)
	require.Empty(t, state.NodeDownscaler)
	require.Empty(t, state.UnschedulablePods)
}
