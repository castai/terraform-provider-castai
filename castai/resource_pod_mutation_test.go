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
		r.Equal("OPTIONAL_SPOT", data.Get(FieldPodMutationSpotType))
		r.Equal(80, data.Get(FieldPodMutationSpotDistributionPct))
		r.Equal("API", data.Get(FieldPodMutationSource))

		// Verify filter_v2 is flattened under workload sub-block
		filterV2 := data.Get(FieldPodMutationFilterV2).([]interface{})
		r.Len(filterV2, 1)
		filterMap := filterV2[0].(map[string]interface{})

		workloadList := filterMap[FieldPodMutationFilterWorkload].([]interface{})
		r.Len(workloadList, 1)
		wm := workloadList[0].(map[string]interface{})

		namespaces := wm[FieldPodMutationFilterNamespaces].([]interface{})
		r.Len(namespaces, 1)
		r.Equal("EXACT", namespaces[0].(map[string]interface{})[FieldPodMutationMatcherType])
		r.Equal("default", namespaces[0].(map[string]interface{})[FieldPodMutationMatcherValue])

		kinds := wm[FieldPodMutationFilterKinds].([]interface{})
		r.Len(kinds, 1)
		r.Equal("EXACT", kinds[0].(map[string]interface{})[FieldPodMutationMatcherType])
		r.Equal("Deployment", kinds[0].(map[string]interface{})[FieldPodMutationMatcherValue])
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

		dg0Config := dg0[FieldPodMutationDistributionGroupConfig].([]interface{})
		r.Len(dg0Config, 1)
		dg0ConfigMap := dg0Config[0].(map[string]interface{})
		r.Equal(string(patching_engine.DistributionGroupConfigSpotTypePREFERREDSPOT), dg0ConfigMap[FieldPodMutationSpotType])

		dg1 := dgs[1].(map[string]interface{})
		r.Equal("on-demand-group", dg1[FieldPodMutationDistributionGroupName])
		r.Equal(30, dg1[FieldPodMutationDistributionGroupPct])

		dg1Config := dg1[FieldPodMutationDistributionGroupConfig].([]interface{})
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
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(createRespBody)),
				Header:     map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

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

		result := resource.CreateContext(ctx, data, provider)

		r.Nil(result)
		r.Equal(testMutationID, data.Id())
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
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(updateRespBody)),
				Header:     map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

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
		d.SetId("org-123/cluster-456/mutation-789")

		result, err := res.Importer.StateContext(t.Context(), d, nil)

		r.NoError(err)
		r.Len(result, 1)
		r.Equal("mutation-789", d.Id())
		r.Equal("org-123", d.Get(FieldPodMutationOrganizationID))
		r.Equal("cluster-456", d.Get(FieldPodMutationClusterID))
	})

	for _, id := range []string{"mutation-789", "org-123/mutation-789", "a/b/c/d"} {
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
		configList := result[0][FieldPodMutationDistributionGroupConfig].([]map[string]interface{})
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
		configList := result[0][FieldPodMutationDistributionGroupConfig].([]map[string]interface{})
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
				FieldPodMutationDistributionGroupConfig: []interface{}{
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
				FieldPodMutationDistributionGroupConfig: []interface{}{
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
				FieldPodMutationDistributionGroupName:   "bare-group",
				FieldPodMutationDistributionGroupPct:    100,
				FieldPodMutationDistributionGroupConfig: []interface{}{},
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
					FieldPodMutationFilterNamespaces: []interface{}{
						map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "prod"},
					},
					FieldPodMutationFilterKinds: []interface{}{
						map[string]interface{}{FieldPodMutationMatcherType: "REGEX", FieldPodMutationMatcherValue: "^Deploy.*"},
					},
					FieldPodMutationFilterNames:             []interface{}{},
					FieldPodMutationFilterExcludeNames:      []interface{}{},
					FieldPodMutationFilterExcludeNamespaces: []interface{}{},
					FieldPodMutationFilterExcludeKinds:      []interface{}{},
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
							FieldPodMutationLabelsFilterMatchers: []interface{}{
								map[string]interface{}{
									FieldPodMutationLabelMatcherKey: []interface{}{
										map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "app"},
									},
									FieldPodMutationLabelMatcherValue: []interface{}{
										map[string]interface{}{FieldPodMutationMatcherType: "EXACT", FieldPodMutationMatcherValue: "web"},
									},
								},
							},
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
					resource.TestCheckResourceAttr(resourceName, "spot_type", "PREFERRED_SPOT"),
					resource.TestCheckResourceAttr(resourceName, "spot_distribution_percentage", "80"),
					resource.TestCheckResourceAttr(resourceName, "filter_v2.0.workload.0.namespaces.0.type", "EXACT"),
					resource.TestCheckResourceAttr(resourceName, "filter_v2.0.workload.0.namespaces.0.value", "default"),
					resource.TestCheckResourceAttr(resourceName, "filter_v2.0.workload.0.kinds.0.type", "EXACT"),
					resource.TestCheckResourceAttr(resourceName, "filter_v2.0.workload.0.kinds.0.value", "Deployment"),
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
					resource.TestCheckResourceAttr(resourceName, "spot_type", "OPTIONAL_SPOT"),
					resource.TestCheckResourceAttr(resourceName, "spot_distribution_percentage", "50"),
					resource.TestCheckResourceAttr(resourceName, "filter_v2.0.workload.0.namespaces.0.type", "REGEX"),
					resource.TestCheckResourceAttr(resourceName, "filter_v2.0.workload.0.namespaces.0.value", "^prod-.*$"),
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
					resource.TestCheckResourceAttr(resourceName, "spot_type", "OPTIONAL_SPOT"),
					resource.TestCheckResourceAttr(resourceName, "spot_distribution_percentage", "0"),
					resource.TestCheckResourceAttr(resourceName, "restart_matching_workloads", "false"),
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

  spot_type                    = "PREFERRED_SPOT"
  spot_distribution_percentage = 80

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

  spot_type                    = "OPTIONAL_SPOT"
  spot_distribution_percentage = 50
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

  spot_type                    = "OPTIONAL_SPOT"
  spot_distribution_percentage = 0
  restart_matching_workloads   = false
}
`, rName, clusterName, organizationID)
}
