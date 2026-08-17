package castai

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/workload_eviction"
	mock_workload_eviction "github.com/castai/terraform-provider-castai/castai/sdk/workload_eviction/mock"
)

func newEvictorProvider(mockClient *mock_workload_eviction.MockClientWithResponsesInterface) *ProviderConfig {
	return &ProviderConfig{
		workloadEvictionClient: mockClient,
	}
}

func TestResourceEvictor_ReadContext(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	status := workload_eviction.ConfigStatusEVICTORSTATUSCOMPATIBLE

	cfg := fullEvictorConfig(&status)

	mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)
	provider := newEvictorProvider(mockClient)

	mockClient.EXPECT().
		EvictorAPIGetConfigWithResponse(mock.Anything, clusterId).
		Return(&workload_eviction.EvictorAPIGetConfigResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}},
			JSON200:      cfg,
		}, nil)

	resource := resourceEvictor()
	stateValue := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	data := resource.Data(state)

	diags := resource.ReadContext(context.Background(), data, provider)

	r := require.New(t)
	r.False(diags.HasError())
	r.Equal(clusterId, data.Id())
	r.Equal(clusterId, data.Get(FieldClusterId))
	r.Equal(*cfg.Enabled, data.Get(FieldEvictorEnabled))
	r.Equal(*cfg.DryRun, data.Get(FieldEvictorDryRun))
	r.Equal(*cfg.AggressiveMode, data.Get(FieldEvictorAggressiveMode))
	r.Equal(*cfg.ScopedMode, data.Get(FieldEvictorScopedMode))
	r.Equal(*cfg.CycleInterval, data.Get(FieldEvictorCycleInterval))
	r.Equal(int(*cfg.NodeGracePeriodMinutes), data.Get(FieldEvictorNodeGracePeriodMinutes))
	r.Equal(*cfg.PodEvictionFailureBackOffInterval, data.Get(FieldEvictorPodEvictionFailureBackOffInterval))
	r.Equal(*cfg.IgnorePodDisruptionBudgets, data.Get(FieldEvictorIgnorePodDisruptionBudgets))
	r.Equal(*cfg.SoftTainting, data.Get(FieldEvictorSoftTainting))
	r.Equal(*cfg.EmitNodeRelatedPodEvents, data.Get(FieldEvictorEmitNodeRelatedPodEvents))
	r.Equal(*cfg.DrainTimeout, data.Get(FieldEvictorDrainTimeout))
	r.Equal(*cfg.DrainRollbackTimeout, data.Get(FieldEvictorDrainRollbackTimeout))
	r.Equal(*cfg.Windows, data.Get(FieldEvictorWindows))
	r.Equal(*cfg.ForceDisableLiveMigration, data.Get(FieldEvictorForceDisableLiveMigration))
	r.Equal(*cfg.ForceDisableWoop, data.Get(FieldEvictorForceDisableWoop))
	r.Equal(*cfg.ForceDisablePodMutations, data.Get(FieldEvictorForceDisablePodMutations))
	r.Equal(*cfg.ForceDisableKarpenterMode, data.Get(FieldEvictorForceDisableKarpenterMode))
	r.Equal(int(*cfg.MaxTargetNodesPerCycle), data.Get(FieldEvictorMaxTargetNodesPerCycle))
	r.Equal(int(*cfg.MinTargetNodesPerCycle), data.Get(FieldEvictorMinTargetNodesPerCycle))
	r.Equal(int(*cfg.TargetNodePercentage), data.Get(FieldEvictorTargetNodePercentage))
	r.Equal(*cfg.PricingAwarenessEnabled, data.Get(FieldEvictorPricingAwarenessEnabled))
	r.Equal(*cfg.Arm64Supported, data.Get(FieldEvictorArm64Supported))
	r.Equal(string(*cfg.Status), data.Get(FieldEvictorStatus))

	pricingModel := data.Get(FieldEvictorPricingModel).([]interface{})
	r.Len(pricingModel, 1)
	pm := pricingModel[0].(map[string]interface{})
	r.Equal(*cfg.PricingModel.BaseCpuCost, pm[FieldEvictorBaseCpuCost])
	r.Equal(*cfg.PricingModel.BaseMemCost, pm[FieldEvictorBaseMemCost])
	r.Equal(*cfg.PricingModel.SpotDiscount, pm[FieldEvictorSpotDiscount])
}

func TestResourceEvictor_CreateContext(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	status := workload_eviction.ConfigStatusEVICTORSTATUSCOMPATIBLE

	mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)
	provider := newEvictorProvider(mockClient)

	var capturedBody workload_eviction.Config
	mockClient.EXPECT().
		EvictorAPIUpdateConfigWithResponse(mock.Anything, clusterId, mock.Anything).
		RunAndReturn(func(_ context.Context, _ string, body workload_eviction.Config, _ ...workload_eviction.RequestEditorFn) (*workload_eviction.EvictorAPIUpdateConfigResponse, error) {
			capturedBody = body
			return &workload_eviction.EvictorAPIUpdateConfigResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}},
				JSON200:      fullEvictorConfig(&status),
			}, nil
		})

	resource := resourceEvictor()
	stateValue := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:                      cty.StringVal(clusterId),
		FieldEvictorEnabled:                 cty.BoolVal(true),
		FieldEvictorDryRun:                  cty.BoolVal(true),
		FieldEvictorAggressiveMode:          cty.BoolVal(true),
		FieldEvictorScopedMode:              cty.BoolVal(true),
		FieldEvictorCycleInterval:           cty.StringVal("2m"),
		FieldEvictorNodeGracePeriodMinutes:  cty.NumberIntVal(10),
		FieldEvictorPodEvictionFailureBackOffInterval: cty.StringVal("10s"),
		FieldEvictorIgnorePodDisruptionBudgets: cty.BoolVal(true),
		FieldEvictorSoftTainting:            cty.BoolVal(true),
		FieldEvictorEmitNodeRelatedPodEvents: cty.BoolVal(true),
		FieldEvictorDrainTimeout:            cty.StringVal("15m"),
		FieldEvictorDrainRollbackTimeout:    cty.StringVal("2m"),
		FieldEvictorWindows:                 cty.BoolVal(true),
		FieldEvictorForceDisableLiveMigration: cty.BoolVal(true),
		FieldEvictorForceDisableWoop:        cty.BoolVal(true),
		FieldEvictorForceDisablePodMutations: cty.BoolVal(true),
		FieldEvictorForceDisableKarpenterMode: cty.BoolVal(true),
		FieldEvictorMaxTargetNodesPerCycle:  cty.NumberIntVal(5),
		FieldEvictorMinTargetNodesPerCycle:  cty.NumberIntVal(1),
		FieldEvictorTargetNodePercentage:    cty.NumberIntVal(50),
		FieldEvictorPricingAwarenessEnabled: cty.BoolVal(true),
		FieldEvictorArm64Supported:          cty.BoolVal(true),
		FieldEvictorPricingModel: cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				FieldEvictorBaseCpuCost:  cty.StringVal("0.05"),
				FieldEvictorBaseMemCost:  cty.StringVal("0.01"),
				FieldEvictorSpotDiscount: cty.StringVal("0.6"),
			}),
		}),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	data := resource.Data(state)

	diags := resource.CreateContext(context.Background(), data, provider)

	r := require.New(t)
	r.False(diags.HasError())
	r.Equal(clusterId, data.Id())
	r.NotNil(capturedBody.Enabled)
	r.True(*capturedBody.Enabled)
	r.NotNil(capturedBody.DryRun)
	r.True(*capturedBody.DryRun)
	r.NotNil(capturedBody.AggressiveMode)
	r.True(*capturedBody.AggressiveMode)
	r.NotNil(capturedBody.ScopedMode)
	r.True(*capturedBody.ScopedMode)
	r.NotNil(capturedBody.CycleInterval)
	r.Equal("2m", *capturedBody.CycleInterval)
	r.NotNil(capturedBody.NodeGracePeriodMinutes)
	r.Equal(int32(10), *capturedBody.NodeGracePeriodMinutes)
	r.NotNil(capturedBody.PodEvictionFailureBackOffInterval)
	r.Equal("10s", *capturedBody.PodEvictionFailureBackOffInterval)
	r.NotNil(capturedBody.IgnorePodDisruptionBudgets)
	r.True(*capturedBody.IgnorePodDisruptionBudgets)
	r.NotNil(capturedBody.SoftTainting)
	r.True(*capturedBody.SoftTainting)
	r.NotNil(capturedBody.EmitNodeRelatedPodEvents)
	r.True(*capturedBody.EmitNodeRelatedPodEvents)
	r.NotNil(capturedBody.DrainTimeout)
	r.Equal("15m", *capturedBody.DrainTimeout)
	r.NotNil(capturedBody.DrainRollbackTimeout)
	r.Equal("2m", *capturedBody.DrainRollbackTimeout)
	r.NotNil(capturedBody.Windows)
	r.True(*capturedBody.Windows)
	r.NotNil(capturedBody.ForceDisableLiveMigration)
	r.True(*capturedBody.ForceDisableLiveMigration)
	r.NotNil(capturedBody.ForceDisableWoop)
	r.True(*capturedBody.ForceDisableWoop)
	r.NotNil(capturedBody.ForceDisablePodMutations)
	r.True(*capturedBody.ForceDisablePodMutations)
	r.NotNil(capturedBody.ForceDisableKarpenterMode)
	r.True(*capturedBody.ForceDisableKarpenterMode)
	r.NotNil(capturedBody.MaxTargetNodesPerCycle)
	r.Equal(int32(5), *capturedBody.MaxTargetNodesPerCycle)
	r.NotNil(capturedBody.MinTargetNodesPerCycle)
	r.Equal(int32(1), *capturedBody.MinTargetNodesPerCycle)
	r.NotNil(capturedBody.TargetNodePercentage)
	r.Equal(int32(50), *capturedBody.TargetNodePercentage)
	r.NotNil(capturedBody.PricingAwarenessEnabled)
	r.True(*capturedBody.PricingAwarenessEnabled)
	r.NotNil(capturedBody.Arm64Supported)
	r.True(*capturedBody.Arm64Supported)
	r.NotNil(capturedBody.PricingModel)
	r.Equal("0.05", *capturedBody.PricingModel.BaseCpuCost)
	r.Equal("0.01", *capturedBody.PricingModel.BaseMemCost)
	r.Equal("0.6", *capturedBody.PricingModel.SpotDiscount)
}

func TestResourceEvictor_UpdateContext(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	status := workload_eviction.ConfigStatusEVICTORSTATUSCOMPATIBLE

	mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)
	provider := newEvictorProvider(mockClient)

	var capturedBody workload_eviction.Config
	mockClient.EXPECT().
		EvictorAPIUpdateConfigWithResponse(mock.Anything, clusterId, mock.Anything).
		RunAndReturn(func(_ context.Context, _ string, body workload_eviction.Config, _ ...workload_eviction.RequestEditorFn) (*workload_eviction.EvictorAPIUpdateConfigResponse, error) {
			capturedBody = body
			return &workload_eviction.EvictorAPIUpdateConfigResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}},
				JSON200:      fullEvictorConfig(&status),
			}, nil
		})

	resource := resourceEvictor()
	stateValue := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:                     cty.StringVal(clusterId),
		FieldEvictorEnabled:                cty.BoolVal(false),
		FieldEvictorCycleInterval:          cty.StringVal("30s"),
		FieldEvictorNodeGracePeriodMinutes: cty.NumberIntVal(3),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	data := resource.Data(state)

	diags := resource.UpdateContext(context.Background(), data, provider)

	r := require.New(t)
	r.False(diags.HasError())
	r.NotNil(capturedBody.Enabled)
	r.False(*capturedBody.Enabled)
	r.NotNil(capturedBody.CycleInterval)
	r.Equal("30s", *capturedBody.CycleInterval)
	r.NotNil(capturedBody.NodeGracePeriodMinutes)
	r.Equal(int32(3), *capturedBody.NodeGracePeriodMinutes)
}

func TestResourceEvictor_DeleteContext(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)
	provider := newEvictorProvider(mockClient)

	resource := resourceEvictor()
	stateValue := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	data := resource.Data(state)

	diags := resource.DeleteContext(context.Background(), data, provider)

	r := require.New(t)
	r.False(diags.HasError())
	r.Empty(data.Id())
}

func TestResourceEvictor_Import(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"
	status := workload_eviction.ConfigStatusEVICTORSTATUSCOMPATIBLE

	mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)
	provider := newEvictorProvider(mockClient)

	mockClient.EXPECT().
		EvictorAPIGetConfigWithResponse(mock.Anything, clusterId).
		Return(&workload_eviction.EvictorAPIGetConfigResponse{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}},
			JSON200:      fullEvictorConfig(&status),
		}, nil)

	resource := resourceEvictor()
	stateValue := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	data := resource.Data(state)
	data.SetId(clusterId)

	imported, err := resource.Importer.StateContext(context.Background(), data, provider)

	r := require.New(t)
	r.NoError(err)
	r.Len(imported, 1)
	r.Equal(clusterId, imported[0].Id())
	r.Equal(clusterId, imported[0].Get(FieldClusterId))
}

func TestResourceEvictor_DurationValidation(t *testing.T) {
	t.Parallel()

	resource := resourceEvictor()
	fields := []string{
		FieldEvictorCycleInterval,
		FieldEvictorPodEvictionFailureBackOffInterval,
		FieldEvictorDrainTimeout,
		FieldEvictorDrainRollbackTimeout,
	}

	for _, field := range fields {
		t.Run(field, func(t *testing.T) {
			r := require.New(t)
			schemaField := resource.Schema[field]
			r.NotNil(schemaField.ValidateFunc)
			warnings, errors := schemaField.ValidateFunc("not-a-duration", field)
			r.Empty(warnings)
			r.Len(errors, 1)
		})
	}
}

func TestResourceEvictor_SchemaFieldCount(t *testing.T) {
	t.Parallel()

	resource := resourceEvictor()

	expectedFields := []string{
		FieldClusterId,
		FieldEvictorEnabled,
		FieldEvictorDryRun,
		FieldEvictorAggressiveMode,
		FieldEvictorScopedMode,
		FieldEvictorCycleInterval,
		FieldEvictorNodeGracePeriodMinutes,
		FieldEvictorPodEvictionFailureBackOffInterval,
		FieldEvictorIgnorePodDisruptionBudgets,
		FieldEvictorStatus,
		FieldEvictorSoftTainting,
		FieldEvictorEmitNodeRelatedPodEvents,
		FieldEvictorDrainTimeout,
		FieldEvictorDrainRollbackTimeout,
		FieldEvictorWindows,
		FieldEvictorForceDisableLiveMigration,
		FieldEvictorForceDisableWoop,
		FieldEvictorForceDisablePodMutations,
		FieldEvictorForceDisableKarpenterMode,
		FieldEvictorMaxTargetNodesPerCycle,
		FieldEvictorMinTargetNodesPerCycle,
		FieldEvictorTargetNodePercentage,
		FieldEvictorPricingAwarenessEnabled,
		FieldEvictorPricingModel,
		FieldEvictorArm64Supported,
	}

	r := require.New(t)
	r.Len(resource.Schema, len(expectedFields))
	for _, field := range expectedFields {
		r.Contains(resource.Schema, field)
	}
}

func TestResourceEvictor_ReadContext_Error(t *testing.T) {
	t.Parallel()

	clusterId := "b6bfc074-a267-400f-b8f1-db0850c369b1"

	tests := map[string]struct {
		response *workload_eviction.EvictorAPIGetConfigResponse
	}{
		"non-200 status": {
			response: &workload_eviction.EvictorAPIGetConfigResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusInternalServerError, Header: map[string][]string{"Content-Type": {"application/json"}}},
				JSON200:      nil,
			},
		},
		"nil JSON200": {
			response: &workload_eviction.EvictorAPIGetConfigResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK, Header: map[string][]string{"Content-Type": {"application/json"}}},
				JSON200:      nil,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockClient := mock_workload_eviction.NewMockClientWithResponsesInterface(t)
			provider := newEvictorProvider(mockClient)

			mockClient.EXPECT().
				EvictorAPIGetConfigWithResponse(mock.Anything, clusterId).
				Return(tc.response, nil)

			resource := resourceEvictor()
			stateValue := cty.ObjectVal(map[string]cty.Value{
				FieldClusterId: cty.StringVal(clusterId),
			})
			state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
			data := resource.Data(state)

			diags := resource.ReadContext(context.Background(), data, provider)

			r := require.New(t)
			r.True(diags.HasError())
		})
	}
}

func TestAccResourceEvictor(t *testing.T) {
	t.Skip("requires CO-4292 V2 API to be live")
}

func fullEvictorConfig(status *workload_eviction.ConfigStatus) *workload_eviction.Config {
	return &workload_eviction.Config{
		Enabled:                           lo.ToPtr(true),
		DryRun:                            lo.ToPtr(true),
		AggressiveMode:                    lo.ToPtr(true),
		ScopedMode:                        lo.ToPtr(true),
		CycleInterval:                     lo.ToPtr("2m"),
		NodeGracePeriodMinutes:            lo.ToPtr(int32(10)),
		PodEvictionFailureBackOffInterval: lo.ToPtr("10s"),
		IgnorePodDisruptionBudgets:        lo.ToPtr(true),
		SoftTainting:                      lo.ToPtr(true),
		EmitNodeRelatedPodEvents:          lo.ToPtr(true),
		DrainTimeout:                      lo.ToPtr("15m"),
		DrainRollbackTimeout:              lo.ToPtr("2m"),
		Windows:                           lo.ToPtr(true),
		ForceDisableLiveMigration:         lo.ToPtr(true),
		ForceDisableWoop:                  lo.ToPtr(true),
		ForceDisablePodMutations:          lo.ToPtr(true),
		ForceDisableKarpenterMode:         lo.ToPtr(true),
		MaxTargetNodesPerCycle:            lo.ToPtr(int32(5)),
		MinTargetNodesPerCycle:            lo.ToPtr(int32(1)),
		TargetNodePercentage:              lo.ToPtr(int32(50)),
		PricingAwarenessEnabled:           lo.ToPtr(true),
		Arm64Supported:                    lo.ToPtr(true),
		Status:                            status,
		PricingModel: &workload_eviction.PricingModel{
			BaseCpuCost:  lo.ToPtr("0.05"),
			BaseMemCost:  lo.ToPtr("0.01"),
			SpotDiscount: lo.ToPtr("0.6"),
		},
	}
}
