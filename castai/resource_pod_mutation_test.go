package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	tfterraform "github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/patching_engine"
	mock_patching_engine "github.com/castai/terraform-provider-castai/castai/sdk/patching_engine/mock"
)

const (
	testOrgID      = "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
	testClusterID  = "b6bfc074-a267-400f-b8f1-db0850c369b1"
	testMutationID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
)

func TestPodMutation_ReadContext(t *testing.T) {
	t.Parallel()

	t.Run("when API responds with 404 then remove from state", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_patching_engine.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			patchingEngineClient: &patching_engine.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		body := io.NopCloser(bytes.NewReader([]byte(`{"message":"not found"}`)))
		mockClient.EXPECT().
			PodMutationsAPIGetPodMutation(gomock.Any(), testOrgID, testClusterID, testMutationID).
			Return(&http.Response{StatusCode: http.StatusNotFound, Body: body, Header: map[string][]string{"Content-Type": {"application/json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(testOrgID),
			"cluster_id":      cty.StringVal(testClusterID),
			"name":            cty.StringVal("test-mutation"),
			"enabled":         cty.BoolVal(true),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = testMutationID

		resource := resourcePodMutation()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.Empty(data.Id())
	})

	t.Run("when API responds with 200 then populate state", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_patching_engine.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			patchingEngineClient: &patching_engine.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		mutation := patching_engine.PodMutation{
			Id:             lo.ToPtr(testMutationID),
			Name:           lo.ToPtr("test-mutation"),
			Enabled:        lo.ToPtr(true),
			ClusterId:      lo.ToPtr(testClusterID),
			OrganizationId: lo.ToPtr(testOrgID),
			Labels: &map[string]string{
				"app": "web",
			},
			SpotType:                   lo.ToPtr(patching_engine.PodMutationSpotTypeOPTIONALSPOT),
			SpotDistributionPercentage: lo.ToPtr(int32(80)),
			Source:                     lo.ToPtr(patching_engine.API),
			ObjectFilterV2: &patching_engine.ObjectFilterV2{
				Namespaces: &[]patching_engine.ObjectFilterV2Matcher{
					{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("default")},
				},
				Kinds: &[]patching_engine.ObjectFilterV2Matcher{
					{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("Deployment")},
				},
			},
			Tolerations: &[]patching_engine.Toleration{
				{
					Key:      lo.ToPtr("dedicated"),
					Operator: lo.ToPtr("Equal"),
					Value:    lo.ToPtr("spot"),
					Effect:   lo.ToPtr("NoSchedule"),
				},
			},
		}

		respBody, _ := json.Marshal(mutation)
		mockClient.EXPECT().
			PodMutationsAPIGetPodMutation(gomock.Any(), testOrgID, testClusterID, testMutationID).
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(respBody)),
				Header:     map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(testOrgID),
			"cluster_id":      cty.StringVal(testClusterID),
			"name":            cty.StringVal("test-mutation"),
			"enabled":         cty.BoolVal(true),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = testMutationID

		resource := resourcePodMutation()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.Equal(testMutationID, data.Id())
		r.Equal("test-mutation", data.Get(FieldPodMutationName))
		r.Equal(true, data.Get(FieldPodMutationEnabled))
		spotConfig := data.Get(FieldPodMutationSpotConfig).([]interface{})
		r.Len(spotConfig, 1)
		spotConfigMap := spotConfig[0].(map[string]interface{})
		r.Equal("OPTIONAL_SPOT", spotConfigMap[FieldPodMutationSpotMode])
		r.Equal(80, spotConfigMap[FieldPodMutationSpotDistributionPct])
		r.Equal("API", data.Get(FieldPodMutationSource))

		// Verify filter_v2 is flattened under workload sub-block
		filterV2 := data.Get(FieldPodMutationFilterV2).([]interface{})
		r.Len(filterV2, 1)
		filterMap := filterV2[0].(map[string]interface{})

		workloadList := filterMap[FieldPodMutationFilterWorkload].([]interface{})
		r.Len(workloadList, 1)
		wm := workloadList[0].(map[string]interface{})

		namespaces := wm[FieldPodMutationFilterNamespaces].(*schema.Set).List()
		r.Len(namespaces, 1)
		r.Equal("EXACT", namespaces[0].(map[string]interface{})[FieldPodMutationMatcherType])
		r.Equal("default", namespaces[0].(map[string]interface{})[FieldPodMutationMatcherValue])

		kinds := wm[FieldPodMutationFilterKinds].(*schema.Set).List()
		r.Len(kinds, 1)
		r.Equal("EXACT", kinds[0].(map[string]interface{})[FieldPodMutationMatcherType])
		r.Equal("Deployment", kinds[0].(map[string]interface{})[FieldPodMutationMatcherValue])

		// Verify labels
		r.Equal("web", data.Get("labels.app"))

		// Verify tolerations
		r.Equal("dedicated", data.Get("tolerations.0.key"))
		r.Equal("Equal", data.Get("tolerations.0.operator"))
		r.Equal("spot", data.Get("tolerations.0.value"))
		r.Equal("NoSchedule", data.Get("tolerations.0.effect"))
	})

	t.Run("when API returns 500 then return error", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_patching_engine.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			patchingEngineClient: &patching_engine.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		body := io.NopCloser(bytes.NewReader([]byte(`{"message":"internal error"}`)))
		mockClient.EXPECT().
			PodMutationsAPIGetPodMutation(gomock.Any(), testOrgID, testClusterID, testMutationID).
			Return(&http.Response{StatusCode: http.StatusInternalServerError, Body: body, Header: map[string][]string{"Content-Type": {"application/json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(testOrgID),
			"cluster_id":      cty.StringVal(testClusterID),
			"name":            cty.StringVal("test-mutation"),
			"enabled":         cty.BoolVal(true),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = testMutationID

		resource := resourcePodMutation()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
	})

	t.Run("when API returns network error then return error", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_patching_engine.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			patchingEngineClient: &patching_engine.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		mockClient.EXPECT().
			PodMutationsAPIGetPodMutation(gomock.Any(), testOrgID, testClusterID, testMutationID).
			Return(nil, fmt.Errorf("connection refused"))

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(testOrgID),
			"cluster_id":      cty.StringVal(testClusterID),
			"name":            cty.StringVal("test-mutation"),
			"enabled":         cty.BoolVal(true),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = testMutationID

		resource := resourcePodMutation()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
	})
}

func TestPodMutation_ReadContext_WithDistributionGroups(t *testing.T) {
	t.Parallel()

	t.Run("when API responds with distribution groups then populate state", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_patching_engine.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			patchingEngineClient: &patching_engine.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		mutation := patching_engine.PodMutation{
			Id:             lo.ToPtr(testMutationID),
			Name:           lo.ToPtr("dg-mutation"),
			Enabled:        lo.ToPtr(true),
			ClusterId:      lo.ToPtr(testClusterID),
			OrganizationId: lo.ToPtr(testOrgID),
			Source:         lo.ToPtr(patching_engine.API),
			ObjectFilterV2: &patching_engine.ObjectFilterV2{
				Namespaces: &[]patching_engine.ObjectFilterV2Matcher{
					{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("default")},
				},
			},
			DistributionGroups: &[]patching_engine.DistributionGroup{
				{
					Name:       lo.ToPtr("spot-group"),
					Percentage: lo.ToPtr(int32(70)),
					Config: &patching_engine.DistributionGroupConfig{
						SpotType: lo.ToPtr(patching_engine.DistributionGroupConfigSpotTypePREFERREDSPOT),
						Labels: &map[string]string{
							"group": "spot",
						},
						Tolerations: &[]patching_engine.Toleration{
							{
								Key:      lo.ToPtr("dedicated"),
								Operator: lo.ToPtr("Equal"),
								Value:    lo.ToPtr("spot"),
								Effect:   lo.ToPtr("NoSchedule"),
							},
						},
					},
				},
				{
					Name:       lo.ToPtr("on-demand-group"),
					Percentage: lo.ToPtr(int32(30)),
					Config: &patching_engine.DistributionGroupConfig{
						SpotType: lo.ToPtr(patching_engine.DistributionGroupConfigSpotTypeOPTIONALSPOT),
						Annotations: &map[string]string{
							"cost-center": "default",
						},
					},
				},
			},
		}

		respBody, _ := json.Marshal(mutation)
		mockClient.EXPECT().
			PodMutationsAPIGetPodMutation(gomock.Any(), testOrgID, testClusterID, testMutationID).
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(respBody)),
				Header:     map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(testOrgID),
			"cluster_id":      cty.StringVal(testClusterID),
			"name":            cty.StringVal("dg-mutation"),
			"enabled":         cty.BoolVal(true),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = testMutationID

		resource := resourcePodMutation()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.Equal(testMutationID, data.Id())
		r.Equal("dg-mutation", data.Get(FieldPodMutationName))

		dgs := data.Get(FieldPodMutationDistributionGroups).([]interface{})
		r.Len(dgs, 2)

		dg0 := dgs[0].(map[string]interface{})
		r.Equal("spot-group", dg0[FieldPodMutationDistributionGroupName])
		r.Equal(70, dg0[FieldPodMutationDistributionGroupPct])

		dg0Config := dg0[FieldPodMutationDistributionGroupConfiguration].([]interface{})
		r.Len(dg0Config, 1)
		dg0ConfigMap := dg0Config[0].(map[string]interface{})
		r.Equal(string(patching_engine.DistributionGroupConfigSpotTypePREFERREDSPOT), dg0ConfigMap[FieldPodMutationSpotType])

		dg1 := dgs[1].(map[string]interface{})
		r.Equal("on-demand-group", dg1[FieldPodMutationDistributionGroupName])
		r.Equal(30, dg1[FieldPodMutationDistributionGroupPct])

		dg1Config := dg1[FieldPodMutationDistributionGroupConfiguration].([]interface{})
		r.Len(dg1Config, 1)
		dg1ConfigMap := dg1Config[0].(map[string]interface{})
		r.Equal(string(patching_engine.DistributionGroupConfigSpotTypeOPTIONALSPOT), dg1ConfigMap[FieldPodMutationSpotType])
	})
}

func TestPodMutation_CreateContext(t *testing.T) {
	t.Parallel()

	t.Run("when API responds with 200 then set ID and read back", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_patching_engine.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			patchingEngineClient: &patching_engine.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		createdMutation := patching_engine.PodMutation{
			Id:             lo.ToPtr(testMutationID),
			Name:           lo.ToPtr("new-mutation"),
			Enabled:        lo.ToPtr(true),
			ClusterId:      lo.ToPtr(testClusterID),
			OrganizationId: lo.ToPtr(testOrgID),
			Source:         lo.ToPtr(patching_engine.API),
		}

		createRespBody, _ := json.Marshal(createdMutation)
		mockClient.EXPECT().
			PodMutationsAPICreatePodMutation(gomock.Any(), testOrgID, testClusterID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _, _ string, body patching_engine.PodMutationsAPICreatePodMutationJSONRequestBody, _ ...patching_engine.RequestEditorFn) (*http.Response, error) {
				r.Equal("new-mutation", lo.FromPtr(body.Name))
				r.Equal(true, lo.FromPtr(body.Enabled))
				r.NotNil(body.ObjectFilterV2)
				r.NotNil(body.ObjectFilterV2.Namespaces)
				r.Len(*body.ObjectFilterV2.Namespaces, 1)
				r.Equal("default", lo.FromPtr((*body.ObjectFilterV2.Namespaces)[0].Value))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(createRespBody)),
					Header:     map[string][]string{"Content-Type": {"application/json"}},
				}, nil
			})

		// Expect read-back after create
		getRespBody, _ := json.Marshal(createdMutation)
		mockClient.EXPECT().
			PodMutationsAPIGetPodMutation(gomock.Any(), testOrgID, testClusterID, testMutationID).
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(getRespBody)),
				Header:     map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(testOrgID),
			"cluster_id":      cty.StringVal(testClusterID),
			"name":            cty.StringVal("new-mutation"),
			"enabled":         cty.BoolVal(true),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourcePodMutation()
		data := resource.Data(state)
		r.NoError(data.Set(FieldPodMutationFilterV2, []interface{}{
			map[string]interface{}{
				FieldPodMutationFilterWorkload: []interface{}{
					map[string]interface{}{
						FieldPodMutationFilterNamespaces: []interface{}{
							map[string]interface{}{
								FieldPodMutationMatcherType:  string(patching_engine.EXACT),
								FieldPodMutationMatcherValue: "default",
							},
						},
					},
				},
			},
		}))

		result := resource.CreateContext(ctx, data, provider)

		r.Nil(result)
		r.Equal(testMutationID, data.Id())
	})

	t.Run("when API returns 500 then return error", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_patching_engine.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			patchingEngineClient: &patching_engine.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		body := io.NopCloser(bytes.NewReader([]byte(`{"message":"internal error"}`)))
		mockClient.EXPECT().
			PodMutationsAPICreatePodMutation(gomock.Any(), testOrgID, testClusterID, gomock.Any()).
			Return(&http.Response{StatusCode: http.StatusInternalServerError, Body: body, Header: map[string][]string{"Content-Type": {"application/json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(testOrgID),
			"cluster_id":      cty.StringVal(testClusterID),
			"name":            cty.StringVal("new-mutation"),
			"enabled":         cty.BoolVal(true),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		resource := resourcePodMutation()
		data := resource.Data(state)
		r.NoError(data.Set(FieldPodMutationFilterV2, []interface{}{
			map[string]interface{}{
				FieldPodMutationFilterWorkload: []interface{}{
					map[string]interface{}{
						FieldPodMutationFilterNamespaces: []interface{}{
							map[string]interface{}{
								FieldPodMutationMatcherType:  string(patching_engine.EXACT),
								FieldPodMutationMatcherValue: "default",
							},
						},
					},
				},
			},
		}))

		result := resource.CreateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
	})
}

func TestPodMutation_DeleteContext(t *testing.T) {
	t.Parallel()

	t.Run("when API responds with 200 then clear ID", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_patching_engine.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			patchingEngineClient: &patching_engine.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		body := io.NopCloser(bytes.NewReader([]byte("{}")))
		mockClient.EXPECT().
			PodMutationsAPIDeletePodMutation(gomock.Any(), testOrgID, testClusterID, testMutationID).
			Return(&http.Response{StatusCode: http.StatusOK, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(testOrgID),
			"cluster_id":      cty.StringVal(testClusterID),
			"name":            cty.StringVal("test-mutation"),
			"enabled":         cty.BoolVal(true),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = testMutationID

		resource := resourcePodMutation()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.Nil(result)
		r.Empty(data.Id())
	})

	t.Run("when API returns 500 then return error", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_patching_engine.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			patchingEngineClient: &patching_engine.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		body := io.NopCloser(bytes.NewReader([]byte(`{"message":"internal error"}`)))
		mockClient.EXPECT().
			PodMutationsAPIDeletePodMutation(gomock.Any(), testOrgID, testClusterID, testMutationID).
			Return(&http.Response{StatusCode: http.StatusInternalServerError, Body: body, Header: map[string][]string{"Content-Type": {"application/json"}}}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(testOrgID),
			"cluster_id":      cty.StringVal(testClusterID),
			"name":            cty.StringVal("test-mutation"),
			"enabled":         cty.BoolVal(true),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = testMutationID

		resource := resourcePodMutation()
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
	})
}

func TestPodMutation_UpdateContext(t *testing.T) {
	t.Parallel()

	t.Run("when API responds with 200 then read back updated state", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_patching_engine.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			patchingEngineClient: &patching_engine.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		updatedMutation := patching_engine.PodMutation{
			Id:             lo.ToPtr(testMutationID),
			Name:           lo.ToPtr("updated-mutation"),
			Enabled:        lo.ToPtr(false),
			ClusterId:      lo.ToPtr(testClusterID),
			OrganizationId: lo.ToPtr(testOrgID),
			Source:         lo.ToPtr(patching_engine.API),
		}

		updateRespBody, _ := json.Marshal(updatedMutation)
		mockClient.EXPECT().
			PodMutationsAPIUpdatePodMutation(gomock.Any(), testOrgID, testClusterID, testMutationID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _, _, _ string, body patching_engine.PodMutationsAPIUpdatePodMutationJSONRequestBody, _ ...patching_engine.RequestEditorFn) (*http.Response, error) {
				r.Equal("updated-mutation", lo.FromPtr(body.Name))
				r.Equal(false, lo.FromPtr(body.Enabled))
				r.Equal(testMutationID, lo.FromPtr(body.Id))
				r.NotNil(body.ObjectFilterV2)
				r.NotNil(body.ObjectFilterV2.Namespaces)
				r.Len(*body.ObjectFilterV2.Namespaces, 1)
				r.Equal("default", lo.FromPtr((*body.ObjectFilterV2.Namespaces)[0].Value))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(updateRespBody)),
					Header:     map[string][]string{"Content-Type": {"application/json"}},
				}, nil
			})

		// Expect read-back after update
		getRespBody, _ := json.Marshal(updatedMutation)
		mockClient.EXPECT().
			PodMutationsAPIGetPodMutation(gomock.Any(), testOrgID, testClusterID, testMutationID).
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(getRespBody)),
				Header:     map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			"organization_id": cty.StringVal(testOrgID),
			"cluster_id":      cty.StringVal(testClusterID),
			"name":            cty.StringVal("updated-mutation"),
			"enabled":         cty.BoolVal(false),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = testMutationID

		resource := resourcePodMutation()
		data := resource.Data(state)
		r.NoError(data.Set(FieldPodMutationFilterV2, []interface{}{
			map[string]interface{}{
				FieldPodMutationFilterWorkload: []interface{}{
					map[string]interface{}{
						FieldPodMutationFilterNamespaces: []interface{}{
							map[string]interface{}{
								FieldPodMutationMatcherType:  string(patching_engine.EXACT),
								FieldPodMutationMatcherValue: "default",
							},
						},
					},
				},
			},
		}))

		result := resource.UpdateContext(ctx, data, provider)

		r.Nil(result)
		r.Equal(testMutationID, data.Id())
		r.Equal("updated-mutation", data.Get(FieldPodMutationName))
		r.Equal(false, data.Get(FieldPodMutationEnabled))
	})
}

func TestPodMutation_StateImporter(t *testing.T) {
	t.Parallel()

	t.Run("three part ID sets organization_id, cluster_id, and mutation_id", func(t *testing.T) {
		r := require.New(t)
		res := resourcePodMutation()
		d := res.Data(nil)
		orgID := "a0000000-0000-0000-0000-000000000001"
		clusterID := "a0000000-0000-0000-0000-000000000002"
		mutationID := "a0000000-0000-0000-0000-000000000003"
		d.SetId(orgID + "/" + clusterID + "/" + mutationID)

		result, err := res.Importer.StateContext(t.Context(), d, nil)

		r.NoError(err)
		r.Len(result, 1)
		r.Equal(mutationID, d.Id())
		r.Equal(orgID, d.Get(FieldPodMutationOrganizationID))
		r.Equal(clusterID, d.Get(FieldPodMutationClusterID))
	})

	for _, id := range []string{"mutation-789", "org-123/mutation-789"} {
		t.Run("invalid import ID "+id, func(t *testing.T) {
			r := require.New(t)
			res := resourcePodMutation()
			d := res.Data(nil)
			d.SetId(id)

			_, err := res.Importer.StateContext(t.Context(), d, nil)

			r.Error(err)
			r.Contains(err.Error(), "invalid import ID")
		})
	}

	t.Run("invalid import ID a/b/c/d", func(t *testing.T) {
		r := require.New(t)
		res := resourcePodMutation()
		d := res.Data(nil)
		d.SetId("a/b/c/d")

		_, err := res.Importer.StateContext(t.Context(), d, nil)

		r.Error(err)
		r.Contains(err.Error(), "invalid import ID")
	})

	t.Run("non-UUID parts return UUID validation error", func(t *testing.T) {
		r := require.New(t)
		res := resourcePodMutation()
		d := res.Data(nil)
		d.SetId("org-123/cluster-456/mutation-789")

		_, err := res.Importer.StateContext(t.Context(), d, nil)

		r.Error(err)
		r.Contains(err.Error(), "invalid organization_id")
	})
}

func TestFlattenDistributionGroups(t *testing.T) {
	t.Parallel()

	t.Run("flattens groups with full config", func(t *testing.T) {
		r := require.New(t)

		groups := []patching_engine.DistributionGroup{
			{
				Name:       lo.ToPtr("group-a"),
				Percentage: lo.ToPtr(int32(60)),
				Config: &patching_engine.DistributionGroupConfig{
					SpotType: lo.ToPtr(patching_engine.DistributionGroupConfigSpotTypePREFERREDSPOT),
					Labels: &map[string]string{
						"env": "prod",
					},
					Annotations: &map[string]string{
						"note": "important",
					},
					Tolerations: &[]patching_engine.Toleration{
						{
							Key:      lo.ToPtr("key1"),
							Operator: lo.ToPtr("Equal"),
							Value:    lo.ToPtr("val1"),
							Effect:   lo.ToPtr("NoSchedule"),
						},
					},
					NodeTemplatesToConsolidate: &[]string{"template-1"},
				},
			},
			{
				Name:       lo.ToPtr("group-b"),
				Percentage: lo.ToPtr(int32(40)),
				Config:     &patching_engine.DistributionGroupConfig{},
			},
		}

		result, err := flattenDistributionGroups(groups)
		r.NoError(err)
		r.Len(result, 2)

		r.Equal("group-a", result[0][FieldPodMutationDistributionGroupName])
		r.Equal(60, result[0][FieldPodMutationDistributionGroupPct])
		configList := result[0][FieldPodMutationDistributionGroupConfiguration].([]map[string]interface{})
		r.Len(configList, 1)
		r.Equal(string(patching_engine.DistributionGroupConfigSpotTypePREFERREDSPOT), configList[0][FieldPodMutationSpotType])
		r.Equal(map[string]string{"env": "prod"}, configList[0][FieldPodMutationLabels])
		r.Equal(map[string]string{"note": "important"}, configList[0][FieldPodMutationAnnotations])
		r.Equal([]string{"template-1"}, configList[0][FieldPodMutationNodeTemplates])

		r.Equal("group-b", result[1][FieldPodMutationDistributionGroupName])
		r.Equal(40, result[1][FieldPodMutationDistributionGroupPct])
	})

	t.Run("flattens empty groups", func(t *testing.T) {
		r := require.New(t)

		result, err := flattenDistributionGroups([]patching_engine.DistributionGroup{})
		r.NoError(err)
		r.Empty(result)
	})

	t.Run("skips SPOT_TYPE_UNSPECIFIED", func(t *testing.T) {
		r := require.New(t)

		groups := []patching_engine.DistributionGroup{
			{
				Name:       lo.ToPtr("group"),
				Percentage: lo.ToPtr(int32(100)),
				Config: &patching_engine.DistributionGroupConfig{
					SpotType: lo.ToPtr(patching_engine.DistributionGroupConfigSpotTypeSPOTTYPEUNSPECIFIED),
				},
			},
		}

		result, err := flattenDistributionGroups(groups)
		r.NoError(err)
		r.Len(result, 1)
		configList := result[0][FieldPodMutationDistributionGroupConfiguration].([]map[string]interface{})
		_, hasSpotType := configList[0][FieldPodMutationSpotType]
		r.False(hasSpotType)
	})
}

func TestStateToDistributionGroups(t *testing.T) {
	t.Parallel()

	t.Run("converts state items to distribution groups", func(t *testing.T) {
		r := require.New(t)

		items := []interface{}{
			map[string]interface{}{
				FieldPodMutationDistributionGroupName: "spot-group",
				FieldPodMutationDistributionGroupPct:  70,
				FieldPodMutationDistributionGroupConfiguration: []interface{}{
					map[string]interface{}{
						FieldPodMutationSpotType: "PREFERRED_SPOT",
						FieldPodMutationLabels: map[string]interface{}{
							"tier": "spot",
						},
					},
				},
			},
			map[string]interface{}{
				FieldPodMutationDistributionGroupName: "on-demand-group",
				FieldPodMutationDistributionGroupPct:  30,
				FieldPodMutationDistributionGroupConfiguration: []interface{}{
					map[string]interface{}{
						FieldPodMutationSpotType: string(patching_engine.DistributionGroupConfigSpotTypeOPTIONALSPOT),
					},
				},
			},
		}

		groups := stateToDistributionGroups(items)
		r.Len(groups, 2)

		r.Equal("spot-group", lo.FromPtr(groups[0].Name))
		r.Equal(int32(70), lo.FromPtr(groups[0].Percentage))
		r.NotNil(groups[0].Config)
		r.Equal(patching_engine.DistributionGroupConfigSpotTypePREFERREDSPOT, lo.FromPtr(groups[0].Config.SpotType))
		r.Equal(map[string]string{"tier": "spot"}, lo.FromPtr(groups[0].Config.Labels))

		r.Equal("on-demand-group", lo.FromPtr(groups[1].Name))
		r.Equal(int32(30), lo.FromPtr(groups[1].Percentage))
		r.NotNil(groups[1].Config)
		r.Equal(patching_engine.DistributionGroupConfigSpotTypeOPTIONALSPOT, lo.FromPtr(groups[1].Config.SpotType))
	})

	t.Run("skips nil items", func(t *testing.T) {
		r := require.New(t)

		items := []interface{}{nil}
		groups := stateToDistributionGroups(items)
		r.Empty(groups)
	})

	t.Run("handles group without config", func(t *testing.T) {
		r := require.New(t)

		items := []interface{}{
			map[string]interface{}{
				FieldPodMutationDistributionGroupName:          "bare-group",
				FieldPodMutationDistributionGroupPct:           100,
				FieldPodMutationDistributionGroupConfiguration: []interface{}{},
			},
		}

		groups := stateToDistributionGroups(items)
		r.Len(groups, 1)
		r.Equal("bare-group", lo.FromPtr(groups[0].Name))
		r.Equal(int32(100), lo.FromPtr(groups[0].Percentage))
		r.Nil(groups[0].Config)
	})
}

func TestFlattenObjectFilterV2(t *testing.T) {
	t.Parallel()

	t.Run("workload only", func(t *testing.T) {
		r := require.New(t)

		filter := &patching_engine.ObjectFilterV2{
			Namespaces: &[]patching_engine.ObjectFilterV2Matcher{
				{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("default")},
			},
			ExcludeKinds: &[]patching_engine.ObjectFilterV2Matcher{
				{Type: lo.ToPtr(patching_engine.REGEX), Value: lo.ToPtr("^Job$")},
			},
		}

		result := flattenObjectFilterV2(filter)
		r.Len(result, 1)
		m := result[0]

		wl := m[FieldPodMutationFilterWorkload].([]map[string]interface{})
		r.Len(wl, 1)
		r.Len(wl[0][FieldPodMutationFilterNamespaces].([]map[string]interface{}), 1)
		r.Len(wl[0][FieldPodMutationFilterExcludeKinds].([]map[string]interface{}), 1)

		_, hasPod := m[FieldPodMutationFilterPod]
		r.False(hasPod)
	})

	t.Run("pod only", func(t *testing.T) {
		r := require.New(t)

		op := patching_engine.AND
		filter := &patching_engine.ObjectFilterV2{
			Labels: &patching_engine.ObjectFilterV2LabelsFilter{
				Operator: &op,
				Matchers: &[]patching_engine.ObjectFilterV2LabelMatcher{
					{
						Key:   &patching_engine.ObjectFilterV2Matcher{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("app")},
						Value: &patching_engine.ObjectFilterV2Matcher{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("web")},
					},
				},
			},
		}

		result := flattenObjectFilterV2(filter)
		r.Len(result, 1)
		m := result[0]

		_, hasWorkload := m[FieldPodMutationFilterWorkload]
		r.False(hasWorkload)

		pod := m[FieldPodMutationFilterPod].([]map[string]interface{})
		r.Len(pod, 1)
		lf := pod[0][FieldPodMutationFilterLabelsFilter].([]map[string]interface{})
		r.Len(lf, 1)
		r.Equal("AND", lf[0][FieldPodMutationLabelsFilterOperator])
	})

	t.Run("workload and pod combined", func(t *testing.T) {
		r := require.New(t)

		op := patching_engine.OR
		filter := &patching_engine.ObjectFilterV2{
			Names: &[]patching_engine.ObjectFilterV2Matcher{
				{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("my-deploy")},
			},
			ExcludeLabels: &patching_engine.ObjectFilterV2LabelsFilter{
				Operator: &op,
				Matchers: &[]patching_engine.ObjectFilterV2LabelMatcher{
					{
						Key: &patching_engine.ObjectFilterV2Matcher{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("env")},
					},
				},
			},
		}

		result := flattenObjectFilterV2(filter)
		r.Len(result, 1)
		m := result[0]

		wl := m[FieldPodMutationFilterWorkload].([]map[string]interface{})
		r.Len(wl, 1)
		r.Len(wl[0][FieldPodMutationFilterNames].([]map[string]interface{}), 1)

		pod := m[FieldPodMutationFilterPod].([]map[string]interface{})
		r.Len(pod, 1)
		el := pod[0][FieldPodMutationFilterExcludeLabels].([]map[string]interface{})
		r.Len(el, 1)
		r.Equal("OR", el[0][FieldPodMutationLabelsFilterOperator])
	})

	t.Run("empty filter", func(t *testing.T) {
		r := require.New(t)

		result := flattenObjectFilterV2(&patching_engine.ObjectFilterV2{})
		r.Len(result, 1)
		m := result[0]
		_, hasWorkload := m[FieldPodMutationFilterWorkload]
		r.False(hasWorkload)
		_, hasPod := m[FieldPodMutationFilterPod]
		r.False(hasPod)
	})
}

func TestStateToObjectFilterV2(t *testing.T) {
	t.Parallel()

	t.Run("workload fields", func(t *testing.T) {
		r := require.New(t)

		state := map[string]interface{}{
			FieldPodMutationFilterWorkload: []interface{}{
				map[string]interface{}{
					FieldPodMutationFilterNamespaces: matcherSet(
						map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "prod"},
					),
					FieldPodMutationFilterKinds: matcherSet(
						map[string]interface{}{FieldPodMutationMatcherType: "REGEX", FieldPodMutationMatcherValue: "^Deploy.*"},
					),
					FieldPodMutationFilterNames:             matcherSet(),
					FieldPodMutationFilterExcludeNames:      matcherSet(),
					FieldPodMutationFilterExcludeNamespaces: matcherSet(),
					FieldPodMutationFilterExcludeKinds:      matcherSet(),
				},
			},
			FieldPodMutationFilterPod: []interface{}{},
		}

		filter := stateToObjectFilterV2(state)

		r.NotNil(filter.Namespaces)
		r.Len(*filter.Namespaces, 1)
		r.Equal("prod", lo.FromPtr((*filter.Namespaces)[0].Value))

		r.NotNil(filter.Kinds)
		r.Len(*filter.Kinds, 1)

		r.Nil(filter.Names)
		r.Nil(filter.Labels)
		r.Nil(filter.ExcludeLabels)
	})

	t.Run("pod fields", func(t *testing.T) {
		r := require.New(t)

		state := map[string]interface{}{
			FieldPodMutationFilterWorkload: []interface{}{},
			FieldPodMutationFilterPod: []interface{}{
				map[string]interface{}{
					FieldPodMutationFilterLabelsFilter: []interface{}{
						map[string]interface{}{
							FieldPodMutationLabelsFilterOperator: "AND",
							FieldPodMutationLabelsFilterMatchers: labelMatcherSet(
								map[string]interface{}{
									FieldPodMutationLabelMatcherKey: []interface{}{
										map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "app"},
									},
									FieldPodMutationLabelMatcherValue: []interface{}{
										map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "web"},
									},
								},
							),
						},
					},
					FieldPodMutationFilterExcludeLabels: []interface{}{},
				},
			},
		}

		filter := stateToObjectFilterV2(state)

		r.Nil(filter.Namespaces)
		r.Nil(filter.Kinds)
		r.NotNil(filter.Labels)
		r.Equal(patching_engine.AND, lo.FromPtr(filter.Labels.Operator))
		r.Len(*filter.Labels.Matchers, 1)
		r.Equal("app", lo.FromPtr((*filter.Labels.Matchers)[0].Key.Value))
		r.Equal("web", lo.FromPtr((*filter.Labels.Matchers)[0].Value.Value))
		r.Nil(filter.ExcludeLabels)
	})

	t.Run("workload exclusion fields", func(t *testing.T) {
		r := require.New(t)

		state := map[string]interface{}{
			FieldPodMutationFilterWorkload: []interface{}{
				map[string]interface{}{
					FieldPodMutationFilterNamespaces: matcherSet(),
					FieldPodMutationFilterKinds:      matcherSet(),
					FieldPodMutationFilterNames:      matcherSet(),
					FieldPodMutationFilterExcludeNamespaces: matcherSet(
						map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "kube-system"},
					),
					FieldPodMutationFilterExcludeKinds: matcherSet(
						map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "DaemonSet"},
					),
					FieldPodMutationFilterExcludeNames: matcherSet(
						map[string]interface{}{FieldPodMutationMatcherType: "REGEX", FieldPodMutationMatcherValue: "^skip-.*"},
					),
				},
			},
			FieldPodMutationFilterPod: []interface{}{},
		}

		filter := stateToObjectFilterV2(state)

		r.Nil(filter.Namespaces)
		r.Nil(filter.Kinds)
		r.Nil(filter.Names)

		r.NotNil(filter.ExcludeNamespaces)
		r.Len(*filter.ExcludeNamespaces, 1)
		r.Equal("kube-system", lo.FromPtr((*filter.ExcludeNamespaces)[0].Value))

		r.NotNil(filter.ExcludeKinds)
		r.Len(*filter.ExcludeKinds, 1)
		r.Equal("DaemonSet", lo.FromPtr((*filter.ExcludeKinds)[0].Value))

		r.NotNil(filter.ExcludeNames)
		r.Len(*filter.ExcludeNames, 1)
		r.Equal("^skip-.*", lo.FromPtr((*filter.ExcludeNames)[0].Value))
		r.Equal(patching_engine.REGEX, lo.FromPtr((*filter.ExcludeNames)[0].Type))
	})

	t.Run("pod exclude_labels", func(t *testing.T) {
		r := require.New(t)

		state := map[string]interface{}{
			FieldPodMutationFilterWorkload: []interface{}{},
			FieldPodMutationFilterPod: []interface{}{
				map[string]interface{}{
					FieldPodMutationFilterLabelsFilter: []interface{}{},
					FieldPodMutationFilterExcludeLabels: []interface{}{
						map[string]interface{}{
							FieldPodMutationLabelsFilterOperator: "OR",
							FieldPodMutationLabelsFilterMatchers: labelMatcherSet(
								map[string]interface{}{
									FieldPodMutationLabelMatcherKey: []interface{}{
										map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "env"},
									},
									FieldPodMutationLabelMatcherValue: []interface{}{
										map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "dev"},
									},
								},
							),
						},
					},
				},
			},
		}

		filter := stateToObjectFilterV2(state)

		r.Nil(filter.Labels)
		r.NotNil(filter.ExcludeLabels)
		r.Equal(patching_engine.OR, lo.FromPtr(filter.ExcludeLabels.Operator))
		r.Len(*filter.ExcludeLabels.Matchers, 1)
		r.Equal("env", lo.FromPtr((*filter.ExcludeLabels.Matchers)[0].Key.Value))
		r.Equal("dev", lo.FromPtr((*filter.ExcludeLabels.Matchers)[0].Value.Value))
	})
}

func TestFlattenObjectFilterV2_MatcherValues(t *testing.T) {
	t.Parallel()

	t.Run("labels_filter matcher key and value are preserved", func(t *testing.T) {
		r := require.New(t)

		op := patching_engine.AND
		filter := &patching_engine.ObjectFilterV2{
			Labels: &patching_engine.ObjectFilterV2LabelsFilter{
				Operator: &op,
				Matchers: &[]patching_engine.ObjectFilterV2LabelMatcher{
					{
						Key:   &patching_engine.ObjectFilterV2Matcher{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("app")},
						Value: &patching_engine.ObjectFilterV2Matcher{Type: lo.ToPtr(patching_engine.REGEX), Value: lo.ToPtr("^web-.*")},
					},
				},
			},
		}

		result := flattenObjectFilterV2(filter)
		pod := result[0][FieldPodMutationFilterPod].([]map[string]interface{})
		lf := pod[0][FieldPodMutationFilterLabelsFilter].([]map[string]interface{})
		matchers := lf[0][FieldPodMutationLabelsFilterMatchers].([]map[string]interface{})
		r.Len(matchers, 1)

		key := matchers[0][FieldPodMutationLabelMatcherKey].([]map[string]interface{})
		r.Equal("EXACT", key[0][FieldPodMutationMatcherType])
		r.Equal("app", key[0][FieldPodMutationMatcherValue])

		val := matchers[0][FieldPodMutationLabelMatcherValue].([]map[string]interface{})
		r.Equal("REGEX", val[0][FieldPodMutationMatcherType])
		r.Equal("^web-.*", val[0][FieldPodMutationMatcherValue])
	})

	t.Run("exclude_labels matcher content is preserved", func(t *testing.T) {
		r := require.New(t)

		op := patching_engine.OR
		filter := &patching_engine.ObjectFilterV2{
			ExcludeLabels: &patching_engine.ObjectFilterV2LabelsFilter{
				Operator: &op,
				Matchers: &[]patching_engine.ObjectFilterV2LabelMatcher{
					{
						Key:   &patching_engine.ObjectFilterV2Matcher{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("tier")},
						Value: &patching_engine.ObjectFilterV2Matcher{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("frontend")},
					},
				},
			},
		}

		result := flattenObjectFilterV2(filter)
		pod := result[0][FieldPodMutationFilterPod].([]map[string]interface{})
		el := pod[0][FieldPodMutationFilterExcludeLabels].([]map[string]interface{})
		r.Equal("OR", el[0][FieldPodMutationLabelsFilterOperator])

		matchers := el[0][FieldPodMutationLabelsFilterMatchers].([]map[string]interface{})
		r.Len(matchers, 1)

		key := matchers[0][FieldPodMutationLabelMatcherKey].([]map[string]interface{})
		r.Equal("tier", key[0][FieldPodMutationMatcherValue])

		val := matchers[0][FieldPodMutationLabelMatcherValue].([]map[string]interface{})
		r.Equal("frontend", val[0][FieldPodMutationMatcherValue])
	})

	t.Run("workload exclusion fields are preserved", func(t *testing.T) {
		r := require.New(t)

		filter := &patching_engine.ObjectFilterV2{
			ExcludeNamespaces: &[]patching_engine.ObjectFilterV2Matcher{
				{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("kube-system")},
			},
			ExcludeKinds: &[]patching_engine.ObjectFilterV2Matcher{
				{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("DaemonSet")},
			},
			ExcludeNames: &[]patching_engine.ObjectFilterV2Matcher{
				{Type: lo.ToPtr(patching_engine.REGEX), Value: lo.ToPtr("^skip-.*")},
			},
		}

		result := flattenObjectFilterV2(filter)
		wl := result[0][FieldPodMutationFilterWorkload].([]map[string]interface{})

		exNs := wl[0][FieldPodMutationFilterExcludeNamespaces].([]map[string]interface{})
		r.Len(exNs, 1)
		r.Equal("kube-system", exNs[0][FieldPodMutationMatcherValue])

		exKinds := wl[0][FieldPodMutationFilterExcludeKinds].([]map[string]interface{})
		r.Len(exKinds, 1)
		r.Equal("DaemonSet", exKinds[0][FieldPodMutationMatcherValue])

		exNames := wl[0][FieldPodMutationFilterExcludeNames].([]map[string]interface{})
		r.Len(exNames, 1)
		r.Equal("^skip-.*", exNames[0][FieldPodMutationMatcherValue])
		r.Equal("REGEX", exNames[0][FieldPodMutationMatcherType])
	})
}

func TestNodeSelectorRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("state to API and back", func(t *testing.T) {
		r := require.New(t)

		state := map[string]interface{}{
			FieldPodMutationNodeSelectorAdd: map[string]interface{}{
				"node.kubernetes.io/gpu": "true",
				"tier":                   "compute",
			},
			FieldPodMutationNodeSelectorRemove: map[string]interface{}{
				"old-label": "remove-me",
			},
		}

		ns := stateToNodeSelector(state)
		r.NotNil(ns.Add)
		r.Equal("true", (*ns.Add)["node.kubernetes.io/gpu"])
		r.Equal("compute", (*ns.Add)["tier"])
		r.NotNil(ns.Remove)
		r.Equal("remove-me", (*ns.Remove)["old-label"])

		flat := flattenPodMutationNodeSelector(ns)
		r.Len(flat, 1)
		r.Equal(map[string]string{"node.kubernetes.io/gpu": "true", "tier": "compute"}, flat[0][FieldPodMutationNodeSelectorAdd])
		r.Equal(map[string]string{"old-label": "remove-me"}, flat[0][FieldPodMutationNodeSelectorRemove])
	})

	t.Run("add only", func(t *testing.T) {
		r := require.New(t)

		state := map[string]interface{}{
			FieldPodMutationNodeSelectorAdd:    map[string]interface{}{"key": "val"},
			FieldPodMutationNodeSelectorRemove: map[string]interface{}{},
		}

		ns := stateToNodeSelector(state)
		r.NotNil(ns.Add)
		r.Nil(ns.Remove)
	})

	t.Run("empty maps", func(t *testing.T) {
		r := require.New(t)

		state := map[string]interface{}{
			FieldPodMutationNodeSelectorAdd:    map[string]interface{}{},
			FieldPodMutationNodeSelectorRemove: map[string]interface{}{},
		}

		ns := stateToNodeSelector(state)
		r.Nil(ns.Add)
		r.Nil(ns.Remove)
	})
}

func TestAffinityRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("state to API and back", func(t *testing.T) {
		r := require.New(t)

		state := map[string]interface{}{
			FieldPodMutationNodeAffinity: []interface{}{
				map[string]interface{}{
					FieldPodMutationPreferred: []interface{}{
						map[string]interface{}{
							FieldPodMutationWeight: 100,
							FieldPodMutationPreference: []interface{}{
								map[string]interface{}{
									FieldPodMutationMatchExpressions: []interface{}{
										map[string]interface{}{
											FieldPodMutationMatchExpressionsKey:      "kubernetes.io/arch",
											FieldPodMutationMatchExpressionsOperator: "In",
											FieldPodMutationValues:                   []interface{}{"amd64", "arm64"},
										},
										map[string]interface{}{
											FieldPodMutationMatchExpressionsKey:      "node-type",
											FieldPodMutationMatchExpressionsOperator: "Exists",
											FieldPodMutationValues:                   []interface{}{},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		affinity := stateToAffinity(state)
		r.NotNil(affinity.NodeAffinity)
		r.NotNil(affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
		terms := *affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
		r.Len(terms, 1)
		r.Equal(int32(100), lo.FromPtr(terms[0].Weight))
		r.NotNil(terms[0].Preference)
		reqs := *terms[0].Preference.MatchExpressions
		r.Len(reqs, 2)
		r.Equal("kubernetes.io/arch", lo.FromPtr(reqs[0].Key))
		r.Equal("In", lo.FromPtr(reqs[0].Operator))
		r.Equal([]string{"amd64", "arm64"}, *reqs[0].Values)
		r.Equal("node-type", lo.FromPtr(reqs[1].Key))
		r.Equal("Exists", lo.FromPtr(reqs[1].Operator))
		r.Nil(reqs[1].Values)

		// Flatten back
		flat := flattenAffinity(affinity)
		r.Len(flat, 1)
		na := flat[0][FieldPodMutationNodeAffinity].([]map[string]interface{})
		r.Len(na, 1)
		prefTerms := na[0][FieldPodMutationPreferred].([]map[string]interface{})
		r.Len(prefTerms, 1)
		r.Equal(100, prefTerms[0][FieldPodMutationWeight])
		pref := prefTerms[0][FieldPodMutationPreference].([]map[string]interface{})
		r.Len(pref, 1)
		exprs := pref[0][FieldPodMutationMatchExpressions].([]map[string]interface{})
		r.Len(exprs, 2)
		r.Equal("kubernetes.io/arch", exprs[0][FieldPodMutationMatchExpressionsKey])
		r.Equal("In", exprs[0][FieldPodMutationMatchExpressionsOperator])
		r.Equal([]string{"amd64", "arm64"}, exprs[0][FieldPodMutationValues])
	})

	t.Run("nil node affinity returns nil", func(t *testing.T) {
		r := require.New(t)

		result := flattenAffinity(&patching_engine.Affinity{})
		r.Nil(result)
	})
}

func TestTolerationsRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("state to API and back", func(t *testing.T) {
		r := require.New(t)

		items := []interface{}{
			map[string]interface{}{
				FieldPodMutationTolerationKey:      "dedicated",
				FieldPodMutationTolerationOperator: "Equal",
				FieldPodMutationTolerationValue:    "spot",
				FieldPodMutationTolerationEffect:   "NoSchedule",
				FieldPodMutationTolerationSeconds:  300,
			},
			map[string]interface{}{
				FieldPodMutationTolerationKey:      "node.kubernetes.io/not-ready",
				FieldPodMutationTolerationOperator: "Exists",
				FieldPodMutationTolerationValue:    "",
				FieldPodMutationTolerationEffect:   "NoExecute",
				FieldPodMutationTolerationSeconds:  0,
			},
		}

		tols := stateToTolerations(items)
		r.Len(tols, 2)
		r.Equal("dedicated", lo.FromPtr(tols[0].Key))
		r.Equal("Equal", lo.FromPtr(tols[0].Operator))
		r.Equal("spot", lo.FromPtr(tols[0].Value))
		r.Equal("NoSchedule", lo.FromPtr(tols[0].Effect))
		r.Equal("300", lo.FromPtr(tols[0].TolerationSeconds))

		r.Equal("node.kubernetes.io/not-ready", lo.FromPtr(tols[1].Key))
		r.Equal("Exists", lo.FromPtr(tols[1].Operator))
		r.Nil(tols[1].Value)
		r.Equal("NoExecute", lo.FromPtr(tols[1].Effect))
		r.Nil(tols[1].TolerationSeconds)

		// Flatten back
		flat, err := flattenTolerations(tols)
		r.NoError(err)
		r.Len(flat, 2)
		r.Equal("dedicated", flat[0][FieldPodMutationTolerationKey])
		r.Equal("Equal", flat[0][FieldPodMutationTolerationOperator])
		r.Equal("spot", flat[0][FieldPodMutationTolerationValue])
		r.Equal("NoSchedule", flat[0][FieldPodMutationTolerationEffect])
		r.Equal(300, flat[0][FieldPodMutationTolerationSeconds])

		r.Equal("node.kubernetes.io/not-ready", flat[1][FieldPodMutationTolerationKey])
		r.Equal("Exists", flat[1][FieldPodMutationTolerationOperator])
	})

	t.Run("skips nil items", func(t *testing.T) {
		r := require.New(t)

		tols := stateToTolerations([]interface{}{nil})
		r.Empty(tols)
	})

	t.Run("tolerationSecondsToInt with invalid string returns error", func(t *testing.T) {
		r := require.New(t)

		s := "not-a-number"
		_, err := tolerationSecondsToInt(&s)
		r.Error(err)
	})

	t.Run("tolerationSecondsToInt with nil returns zero", func(t *testing.T) {
		r := require.New(t)

		v, err := tolerationSecondsToInt(nil)
		r.NoError(err)
		r.Equal(0, v)
	})
}

func TestPatchRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("parseMutationConfigFromMap parses patch JSON", func(t *testing.T) {
		r := require.New(t)

		m := map[string]interface{}{
			FieldPodMutationPatch: `[{"op":"add","path":"/metadata/labels/foo","value":"bar"},{"op":"remove","path":"/metadata/labels/old"}]`,
		}

		result := parseMutationConfigFromMap(m)
		r.NotNil(result.Patch)
		r.Len(*result.Patch, 2)
		r.Equal("add", (*result.Patch)[0]["op"])
		r.Equal("/metadata/labels/foo", (*result.Patch)[0]["path"])
		r.Equal("bar", (*result.Patch)[0]["value"])
		r.Equal("remove", (*result.Patch)[1]["op"])
	})

	t.Run("empty patch string produces nil", func(t *testing.T) {
		r := require.New(t)

		m := map[string]interface{}{
			FieldPodMutationPatch: "",
		}

		result := parseMutationConfigFromMap(m)
		r.Nil(result.Patch)
	})

	t.Run("flatten then parse round-trips", func(t *testing.T) {
		r := require.New(t)

		apiPatch := []map[string]interface{}{
			{"op": "add", "path": "/spec/containers/0/env/-", "value": map[string]interface{}{"name": "ENV_VAR", "value": "test"}},
		}

		// Flatten (API → state): marshal to JSON string
		patchJSON, err := json.Marshal(apiPatch)
		r.NoError(err)
		patchStr := string(patchJSON)

		// Parse back (state → API): unmarshal from JSON string
		m := map[string]interface{}{
			FieldPodMutationPatch: patchStr,
		}
		result := parseMutationConfigFromMap(m)
		r.NotNil(result.Patch)
		r.Len(*result.Patch, 1)
		r.Equal("add", (*result.Patch)[0]["op"])
		r.Equal("/spec/containers/0/env/-", (*result.Patch)[0]["path"])
	})
}

func TestPodMutationCustomizeDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		filterV2  []interface{}
		expectErr string
	}{
		{
			name: "valid workload filter passes",
			filterV2: []interface{}{
				map[string]interface{}{
					FieldPodMutationFilterWorkload: []interface{}{
						map[string]interface{}{
							FieldPodMutationFilterNamespaces: matcherSet(
								map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "default"},
							),
						},
					},
					FieldPodMutationFilterPod: []interface{}{},
				},
			},
		},
		{
			name: "valid pod filter passes",
			filterV2: []interface{}{
				map[string]interface{}{
					FieldPodMutationFilterWorkload: []interface{}{},
					FieldPodMutationFilterPod: []interface{}{
						map[string]interface{}{
							FieldPodMutationFilterLabelsFilter: []interface{}{
								map[string]interface{}{
									FieldPodMutationLabelsFilterOperator: "AND",
									FieldPodMutationLabelsFilterMatchers: labelMatcherSet(
										map[string]interface{}{
											FieldPodMutationLabelMatcherKey: []interface{}{
												map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "app"},
											},
										},
									),
								},
							},
							FieldPodMutationFilterExcludeLabels: []interface{}{},
						},
					},
				},
			},
		},
		{
			name: "empty workload and pod fails",
			filterV2: []interface{}{
				map[string]interface{}{
					FieldPodMutationFilterWorkload: []interface{}{},
					FieldPodMutationFilterPod:      []interface{}{},
				},
			},
			expectErr: "filter_v2 must specify at least one filter in workload or pod",
		},
		{
			name: "workload present but all sub-fields empty fails",
			filterV2: []interface{}{
				map[string]interface{}{
					FieldPodMutationFilterWorkload: []interface{}{
						map[string]interface{}{
							FieldPodMutationFilterNames:             matcherSet(),
							FieldPodMutationFilterNamespaces:        matcherSet(),
							FieldPodMutationFilterKinds:             matcherSet(),
							FieldPodMutationFilterExcludeNames:      matcherSet(),
							FieldPodMutationFilterExcludeNamespaces: matcherSet(),
							FieldPodMutationFilterExcludeKinds:      matcherSet(),
						},
					},
					FieldPodMutationFilterPod: []interface{}{},
				},
			},
			expectErr: "filter_v2 must specify at least one filter in workload or pod",
		},
		{
			name:      "nil filter_v2 fails",
			filterV2:  []interface{}{nil},
			expectErr: "filter_v2 must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)

			err := validatePodMutationFilter(tt.filterV2)
			if tt.expectErr != "" {
				r.EqualError(err, tt.expectErr)
			} else {
				r.NoError(err)
			}
		})
	}
}

func TestStateToObjectFilterV2_UnorderedSet(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	state := map[string]interface{}{
		FieldPodMutationFilterWorkload: []interface{}{
			map[string]interface{}{
				FieldPodMutationFilterNamespaces: matcherSet(
					map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "a"},
					map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "b"},
					map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "c"},
				),
			},
		},
	}

	got := stateToObjectFilterV2(state)
	r.NotNil(got.Namespaces)
	r.Len(*got.Namespaces, 3)

	values := map[string]bool{}
	for _, m := range *got.Namespaces {
		r.NotNil(m.Value)
		values[*m.Value] = true
	}
	r.Equal(map[string]bool{"a": true, "b": true, "c": true}, values)
}

func TestPodMutation_ReadContext_RoundTrip_UnorderedNamespaces(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	mockClient := mock_patching_engine.NewMockClientInterface(gomock.NewController(t))
	ctx := context.Background()
	provider := &ProviderConfig{
		patchingEngineClient: &patching_engine.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	// Server returns namespaces in a different order than the user's config.
	mutation := patching_engine.PodMutation{
		Id:             lo.ToPtr(testMutationID),
		Name:           lo.ToPtr("test-mutation"),
		Enabled:        lo.ToPtr(true),
		ClusterId:      lo.ToPtr(testClusterID),
		OrganizationId: lo.ToPtr(testOrgID),
		SpotType:       lo.ToPtr(patching_engine.PodMutationSpotTypeOPTIONALSPOT),
		Source:         lo.ToPtr(patching_engine.API),
		ObjectFilterV2: &patching_engine.ObjectFilterV2{
			Namespaces: &[]patching_engine.ObjectFilterV2Matcher{
				{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("c")},
				{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("a")},
				{Type: lo.ToPtr(patching_engine.EXACT), Value: lo.ToPtr("b")},
			},
		},
	}
	respBody, _ := json.Marshal(mutation)
	mockClient.EXPECT().
		PodMutationsAPIGetPodMutation(gomock.Any(), testOrgID, testClusterID, testMutationID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(respBody)),
			Header:     map[string][]string{"Content-Type": {"application/json"}},
		}, nil)

	stateValue := cty.ObjectVal(map[string]cty.Value{
		"organization_id": cty.StringVal(testOrgID),
		"cluster_id":      cty.StringVal(testClusterID),
		"name":            cty.StringVal("test-mutation"),
		"enabled":         cty.BoolVal(true),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	state.ID = testMutationID

	resource := resourcePodMutation()
	data := resource.Data(state)

	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)

	filterV2 := data.Get(FieldPodMutationFilterV2).([]interface{})
	r.Len(filterV2, 1)
	workloadList := filterV2[0].(map[string]interface{})[FieldPodMutationFilterWorkload].([]interface{})
	r.Len(workloadList, 1)
	wm := workloadList[0].(map[string]interface{})

	namespaces := wm[FieldPodMutationFilterNamespaces].(*schema.Set).List()
	r.Len(namespaces, 3)

	values := map[string]bool{}
	for _, n := range namespaces {
		values[n.(map[string]interface{})[FieldPodMutationMatcherValue].(string)] = true
	}
	r.Equal(map[string]bool{"a": true, "b": true, "c": true}, values)
}

func TestAccCloudAgnostic_ResourcePodMutation(t *testing.T) {
	rName := fmt.Sprintf("%v-pod-mutation-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_pod_mutation.test"
	clusterName := fmt.Sprintf("pod-mut-tf-acc-%v", acctest.RandString(8))
	organizationID := testAccGetOrganizationID()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckPodMutationDestroy,
		Steps: []resource.TestStep{
			{
				// Test creation
				Config: testAccPodMutationConfig(rName, clusterName, organizationID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "spot_config.0.spot_mode", "PREFERRED_SPOT"),
					resource.TestCheckResourceAttr(resourceName, "spot_config.0.distribution_percentage", "80"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "filter_v2.0.workload.0.namespaces.*", map[string]string{
						"type":  "EXACT",
						"value": "default",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "filter_v2.0.workload.0.kinds.*", map[string]string{
						"type":  "EXACT",
						"value": "Deployment",
					}),
					resource.TestCheckResourceAttr(resourceName, "tolerations.0.key", "scheduling.cast.ai/spot"),
					resource.TestCheckResourceAttr(resourceName, "tolerations.0.operator", "Exists"),
					resource.TestCheckResourceAttr(resourceName, "tolerations.0.effect", "NoSchedule"),
				),
			},
			{
				// Test update
				Config: testAccPodMutationConfigUpdated(rName, clusterName, organizationID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "spot_config.0.spot_mode", "OPTIONAL_SPOT"),
					resource.TestCheckResourceAttr(resourceName, "spot_config.0.distribution_percentage", "50"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "filter_v2.0.workload.0.namespaces.*", map[string]string{
						"type":  "REGEX",
						"value": "^prod-.*$",
					}),
				),
			},
			{
				// Test import with organization_id/cluster_id/mutation_id
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *tfterraform.State) (string, error) {
					rs := s.RootModule().Resources[resourceName]
					clusterID := rs.Primary.Attributes["cluster_id"]
					return fmt.Sprintf("%s/%s/%s", organizationID, clusterID, rs.Primary.ID), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Test update to zero values - verifies no drift on next plan
				Config: testAccPodMutationConfigZeroValues(rName, clusterName, organizationID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "spot_config.0.spot_mode", "OPTIONAL_SPOT"),
					resource.TestCheckResourceAttr(resourceName, "spot_config.0.distribution_percentage", "0"),
				),
			},
		},
	})
}

func testAccCheckPodMutationDestroy(s *tfterraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).patchingEngineClient
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_pod_mutation" {
			continue
		}

		organizationID := rs.Primary.Attributes["organization_id"]
		clusterID := rs.Primary.Attributes["cluster_id"]

		response, err := client.PodMutationsAPIGetPodMutationWithResponse(ctx, organizationID, clusterID, rs.Primary.ID)
		if err != nil {
			return err
		}

		if response.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("pod mutation %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccPodMutationConfig(rName, clusterName, organizationID string) string {
	return fmt.Sprintf(`
resource "castai_gke_cluster" "test" {
  project_id = "test-project-123456"
  location   = "us-central1-c"
  name       = %[2]q
}

resource "castai_pod_mutation" "test" {
  cluster_id      = castai_gke_cluster.test.id
  organization_id = %[3]q
  name            = %[1]q
  enabled         = true

  filter_v2 {
    workload {
      namespaces {
        type  = "EXACT"
        value = "default"
      }
      kinds {
        type  = "EXACT"
        value = "Deployment"
      }
    }
  }

  spot_config {
    spot_mode               = "PREFERRED_SPOT"
    distribution_percentage = 80
  }

  tolerations {
    key      = "scheduling.cast.ai/spot"
    operator = "Exists"
    effect   = "NoSchedule"
  }
}
`, rName, clusterName, organizationID)
}

func testAccPodMutationConfigUpdated(rName, clusterName, organizationID string) string {
	return fmt.Sprintf(`
resource "castai_gke_cluster" "test" {
  project_id = "test-project-123456"
  location   = "us-central1-c"
  name       = %[2]q
}

resource "castai_pod_mutation" "test" {
  cluster_id      = castai_gke_cluster.test.id
  organization_id = %[3]q
  name            = %[1]q
  enabled         = false

  filter_v2 {
    workload {
      namespaces {
        type  = "REGEX"
        value = "^prod-.*$"
      }
    }
  }

  spot_config {
    spot_mode               = "OPTIONAL_SPOT"
    distribution_percentage = 50
  }
}
`, rName, clusterName, organizationID)
}

func testAccPodMutationConfigZeroValues(rName, clusterName, organizationID string) string {
	return fmt.Sprintf(`
resource "castai_gke_cluster" "test" {
  project_id = "test-project-123456"
  location   = "us-central1-c"
  name       = %[2]q
}

resource "castai_pod_mutation" "test" {
  cluster_id      = castai_gke_cluster.test.id
  organization_id = %[3]q
  name            = %[1]q
  enabled         = true

  filter_v2 {
    workload {
      namespaces {
        type  = "REGEX"
        value = "^prod-.*$"
      }
    }
  }

  spot_config {
    spot_mode               = "OPTIONAL_SPOT"
    distribution_percentage = 0
  }
}
`, rName, clusterName, organizationID)
}

func matcherSet(items ...map[string]interface{}) *schema.Set {
	raw := make([]interface{}, 0, len(items))
	for _, it := range items {
		raw = append(raw, it)
	}
	return schema.NewSet(schema.HashResource(matcherSchema), raw)
}

func labelMatcherSet(items ...map[string]interface{}) *schema.Set {
	raw := make([]interface{}, 0, len(items))
	for _, it := range items {
		raw = append(raw, it)
	}
	return schema.NewSet(schema.HashResource(labelMatcherElemSchema), raw)
}
