package castai

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

func newAutoscalerPoliciesResourceWithMock(mockClient *mock_cluster_autoscaler_v2.MockClientWithResponsesInterface) *autoscalerPoliciesResource {
	return &autoscalerPoliciesResource{
		client: &ProviderConfig{
			clusterAutoscalerV2Client: mockClient,
		},
	}
}

func autoscalerPoliciesTestSchema(t *testing.T, r resource.Resource) (*resource.SchemaResponse, attr.Type) {
	t.Helper()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), "resource schema returned diagnostics: %v", schemaResp.Diagnostics)

	return schemaResp, schemaResp.Schema.Type()
}

func okHTTPResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}}
}

func testAutoscalerPoliciesV2() *cluster_autoscaler_v2.PoliciesV2 {
	return &cluster_autoscaler_v2.PoliciesV2{
		Enabled:    new(true),
		ScopedMode: new(true),
		Version:    new("v5"),
		ClusterLimits: &cluster_autoscaler_v2.ClusterLimitsPolicy{
			Enabled: new(true),
			Cpu: &cluster_autoscaler_v2.ClusterLimitsCpu{
				MaxCores: 16,
				MinCores: new(int32(2)),
			},
		},
		NodeDownscaler: &cluster_autoscaler_v2.NodeDownscalerPolicy{
			EmptyNodesDelay:   new("3m"),
			EmptyNodesEnabled: new(true),
		},
		UnschedulablePods: &cluster_autoscaler_v2.UnschedulablePodsPolicy{
			Enabled:                        new(true),
			PartialTemplateMatchingEnabled: new(true),
			PodPinner:                      &cluster_autoscaler_v2.PodPinner{Enabled: new(true)},
		},
	}
}

// testAutoscalerPoliciesPlanModel returns a full model covering every block,
// mirroring what Terraform core would plan. ID and version are unknown until
// the resource has been applied (the version changes on every write).
func testAutoscalerPoliciesPlanModel(clusterID string) autoscalerPoliciesModel {
	return autoscalerPoliciesModel{
		ID:         types.StringUnknown(),
		ClusterID:  types.StringValue(clusterID),
		Enabled:    types.BoolValue(true),
		ScopedMode: types.BoolValue(true),
		Version:    types.StringUnknown(),
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
}

// autoscalerPoliciesPlanValue converts a typed model into the raw tftypes
// value the framework expects in requests, so tests can declare plans as
// ordinary model structs instead of hand-built tftypes values.
func autoscalerPoliciesPlanValue(t *testing.T, schemaType attr.Type, model autoscalerPoliciesModel) tftypes.Value {
	t.Helper()

	obj, diags := types.ObjectValueFrom(context.Background(), schemaType.(types.ObjectType).AttributeTypes(), model)
	require.False(t, diags.HasError(), "converting model to object value: %v", diags)

	value, err := obj.ToTerraformValue(context.Background())
	require.NoError(t, err)

	return value
}

// autoscalerPoliciesNullValue returns the null raw value for the resource schema,
// the starting point for response states.
func autoscalerPoliciesNullValue(t *testing.T, schemaType attr.Type) tftypes.Value {
	t.Helper()

	return tftypes.NewValue(schemaType.TerraformType(context.Background()), nil)
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
			Raw:    autoscalerPoliciesPlanValue(t, schemaType, testAutoscalerPoliciesPlanModel(clusterID)),
			Schema: schemaResp.Schema,
		},
	}
	resp := resource.CreateResponse{
		State: tfsdk.State{
			Raw:    autoscalerPoliciesNullValue(t, schemaType),
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "create diagnostics: %v", resp.Diagnostics)

	// Assert the payload sent to the API (expand path). The version is
	// fetched from the API on create since the policies already exist, and is
	// included in the PUT for optimistic locking.
	require.NotNil(t, capturedBody.Version)
	require.Equal(t, "v5", *capturedBody.Version)
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

func TestResourceAutoscalerPolicies_Create_NoExistingPolicies(t *testing.T) {
	t.Parallel()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	var capturedBody cluster_autoscaler_v2.PoliciesV2

	mockClient := mock_cluster_autoscaler_v2.NewMockClientWithResponsesInterface(t)

	// First GET: no policies exist yet. Second GET: read-back after the PUT.
	getCalls := 0
	mockClient.EXPECT().
		PoliciesV2APIGetClusterPoliciesWithResponse(mock.Anything, clusterID).
		RunAndReturn(func(_ context.Context, _ string, _ ...cluster_autoscaler_v2.RequestEditorFn) (*cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse, error) {
			getCalls++
			if getCalls == 1 {
				return &cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
					HTTPResponse: &http.Response{StatusCode: http.StatusNotFound, Header: map[string][]string{"Content-Type": {"application/json"}}},
					JSON200:      nil,
				}, nil
			}
			return &cluster_autoscaler_v2.PoliciesV2APIGetClusterPoliciesResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200:      testAutoscalerPoliciesV2(),
			}, nil
		})
	mockClient.EXPECT().
		PoliciesV2APIUpdateClusterPoliciesWithResponse(mock.Anything, clusterID, mock.Anything).
		RunAndReturn(func(_ context.Context, _ string, body cluster_autoscaler_v2.PoliciesV2, _ ...cluster_autoscaler_v2.RequestEditorFn) (*cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse, error) {
			capturedBody = body
			return &cluster_autoscaler_v2.PoliciesV2APIUpdateClusterPoliciesResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200:      testAutoscalerPoliciesV2(),
			}, nil
		})

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

func TestResourceAutoscalerPolicies_Update_UsesStateVersion(t *testing.T) {
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
	schemaResp, schemaType := autoscalerPoliciesTestSchema(t, r)

	// Plan carries an unknown version (it changes on every write); the prior
	// state holds "v5", which must be used for optimistic locking.
	planModel := testAutoscalerPoliciesPlanModel(clusterID)
	planModel.ID = types.StringValue(clusterID)

	stateModel := testAutoscalerPoliciesPlanModel(clusterID)
	stateModel.ID = types.StringValue(clusterID)
	stateModel.Version = types.StringValue("v5")

	req := resource.UpdateRequest{
		Plan: tfsdk.Plan{
			Raw:    autoscalerPoliciesPlanValue(t, schemaType, planModel),
			Schema: schemaResp.Schema,
		},
		State: tfsdk.State{
			Raw:    autoscalerPoliciesPlanValue(t, schemaType, stateModel),
			Schema: schemaResp.Schema,
		},
	}
	resp := resource.UpdateResponse{
		State: tfsdk.State{
			Raw:    autoscalerPoliciesNullValue(t, schemaType),
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
			Enabled: new(true),
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
			Raw:    autoscalerPoliciesNullValue(t, schemaType),
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
			Raw:    autoscalerPoliciesNullValue(t, schemaType),
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
// cluster as the V1 autoscaler acceptance test ("cost-terraform" by default,
// overridable via CLUSTER_NAME), and both resources manage autoscaling policies
// on that cluster, so it must not run in parallel with the V1 test: neither
// test calls t.Parallel(), which keeps them sequential within the package.
func TestAccEKS_ResourceAutoscalerPolicies_basic(t *testing.T) {
	rName := fmt.Sprintf("%v-policies-%v", ResourcePrefix, acctest.RandString(8))
	clusterName, _ := lo.Coalesce(os.Getenv("CLUSTER_NAME"), "cost-terraform")

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
