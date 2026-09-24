package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdkterraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func workloadScalingPolicyFixture(id, name string) sdk.WorkloadoptimizationV1WorkloadScalingPolicy {
	return sdk.WorkloadoptimizationV1WorkloadScalingPolicy{
		Id:                                  id,
		Name:                                name,
		ClusterId:                           "b6bfc074-a267-400f-b8f1-db0850c36gf4",
		ApplyType:                           sdk.WorkloadoptimizationV1ApplyType("IMMEDIATE"),
		IsDefault:                           false,
		IsReadonly:                          true,
		IsCastware:                          false,
		IsOpsPilot:                          false,
		HasWorkloadsConfiguredByAnnotations: false,
		RecommendationPolicies: sdk.WorkloadoptimizationV1RecommendationPolicies{
			ManagementOption: sdk.WorkloadoptimizationV1ManagementOption("MANAGED"),
			Cpu: sdk.WorkloadoptimizationV1ResourcePolicies{
				Function: sdk.WorkloadoptimizationV1ResourcePoliciesFunction("QUANTILE"),
				Args:     []string{"0.9"},
				Overhead: 0.1,
			},
			Memory: sdk.WorkloadoptimizationV1ResourcePolicies{
				Function: sdk.WorkloadoptimizationV1ResourcePoliciesFunction("MAX"),
				Overhead: 0.35,
			},
		},
	}
}

func httpJSONResponse(t *testing.T, status int, v any) *http.Response {
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshaling response body: %v", err)
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func dataSourceWorkloadScalingPolicyForTest(t *testing.T, config map[string]cty.Value) (*schema.Resource, *schema.ResourceData) {
	t.Helper()
	ds := dataSourceWorkloadScalingPolicy()
	state := sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(config), 0)
	return ds, ds.Data(state)
}

func TestDataSourceWorkloadScalingPolicyReadByPolicyID(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
	ctx := context.Background()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gf4"
	policyID := "9c9e5f83-03a7-42e1-b7f5-14d0c3a2b111"

	policy := workloadScalingPolicyFixture(policyID, "balanced")
	mockClient.EXPECT().
		WorkloadOptimizationAPIGetWorkloadScalingPolicy(ctx, clusterID, policyID).
		Return(httpJSONResponse(t, http.StatusOK, policy), nil)

	_, data := dataSourceWorkloadScalingPolicyForTest(t, map[string]cty.Value{
		"cluster_id": cty.StringVal(clusterID),
		"policy_id":  cty.StringVal(policyID),
	})

	diags := dataSourceWorkloadScalingPolicyRead(ctx, data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})
	r.Empty(diags)

	r.Equal(policyID, data.Id())
	r.Equal("balanced", data.Get("name"))
	r.Equal(true, data.Get("is_readonly"))
	r.Equal(false, data.Get("is_default"))
	r.Equal(false, data.Get("is_castware"))

	// Typed definition fields mirror the resource schema.
	r.Equal("IMMEDIATE", data.Get("apply_type"))
	r.Equal("MANAGED", data.Get("management_option"))
	r.Equal("QUANTILE", data.Get("cpu.0.function"))
	r.Equal([]any{"0.9"}, data.Get("cpu.0.args"))
	r.Equal(0.1, data.Get("cpu.0.overhead"))
	r.Equal("MAX", data.Get("memory.0.function"))
}

func TestDataSourceWorkloadScalingPolicyReadByName(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
	ctx := context.Background()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gf4"

	listResponse := sdk.WorkloadoptimizationV1ListWorkloadScalingPoliciesResponse{
		Items: []sdk.WorkloadoptimizationV1WorkloadScalingPolicy{
			workloadScalingPolicyFixture("policy-1-id", "readonly"),
			workloadScalingPolicyFixture("policy-2-id", "balanced"),
			workloadScalingPolicyFixture("policy-3-id", "custom"),
		},
	}
	mockClient.EXPECT().
		WorkloadOptimizationAPIListWorkloadScalingPolicies(ctx, clusterID).
		Return(httpJSONResponse(t, http.StatusOK, listResponse), nil)

	_, data := dataSourceWorkloadScalingPolicyForTest(t, map[string]cty.Value{
		"cluster_id": cty.StringVal(clusterID),
		"name":       cty.StringVal("balanced"),
	})

	diags := dataSourceWorkloadScalingPolicyRead(ctx, data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})
	r.Empty(diags)

	r.Equal("policy-2-id", data.Id())
	r.Equal("balanced", data.Get("name"))
	r.Equal("QUANTILE", data.Get("cpu.0.function"))
}

func TestDataSourceWorkloadScalingPolicyReadByNameNotFound(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
	ctx := context.Background()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gf4"

	listResponse := sdk.WorkloadoptimizationV1ListWorkloadScalingPoliciesResponse{
		Items: []sdk.WorkloadoptimizationV1WorkloadScalingPolicy{
			workloadScalingPolicyFixture("policy-1-id", "readonly"),
		},
	}
	mockClient.EXPECT().
		WorkloadOptimizationAPIListWorkloadScalingPolicies(ctx, clusterID).
		Return(httpJSONResponse(t, http.StatusOK, listResponse), nil)

	_, data := dataSourceWorkloadScalingPolicyForTest(t, map[string]cty.Value{
		"cluster_id": cty.StringVal(clusterID),
		"name":       cty.StringVal("missing"),
	})

	diags := dataSourceWorkloadScalingPolicyRead(ctx, data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})
	r.True(diags.HasError())
	r.Contains(diags[0].Summary, `scaling policy "missing" not found`)
}

func TestDataSourceWorkloadScalingPolicyReadByNameAmbiguous(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
	ctx := context.Background()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gf4"

	listResponse := sdk.WorkloadoptimizationV1ListWorkloadScalingPoliciesResponse{
		Items: []sdk.WorkloadoptimizationV1WorkloadScalingPolicy{
			workloadScalingPolicyFixture("policy-1-id", "balanced"),
			workloadScalingPolicyFixture("policy-2-id", "balanced"),
		},
	}
	mockClient.EXPECT().
		WorkloadOptimizationAPIListWorkloadScalingPolicies(ctx, clusterID).
		Return(httpJSONResponse(t, http.StatusOK, listResponse), nil)

	_, data := dataSourceWorkloadScalingPolicyForTest(t, map[string]cty.Value{
		"cluster_id": cty.StringVal(clusterID),
		"name":       cty.StringVal("balanced"),
	})

	diags := dataSourceWorkloadScalingPolicyRead(ctx, data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})
	r.True(diags.HasError())
	r.Contains(diags[0].Summary, "use policy_id")
}

func TestDataSourceWorkloadScalingPolicyReadAPIError(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
	ctx := context.Background()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gf4"
	policyID := "9c9e5f83-03a7-42e1-b7f5-14d0c3a2b111"

	mockClient.EXPECT().
		WorkloadOptimizationAPIGetWorkloadScalingPolicy(ctx, clusterID, policyID).
		Return(httpJSONResponse(t, http.StatusInternalServerError, map[string]string{"message": "internal error"}), nil)

	_, data := dataSourceWorkloadScalingPolicyForTest(t, map[string]cty.Value{
		"cluster_id": cty.StringVal(clusterID),
		"policy_id":  cty.StringVal(policyID),
	})

	diags := dataSourceWorkloadScalingPolicyRead(ctx, data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})
	r.True(diags.HasError())
}

func TestDataSourceWorkloadScalingPolicyReadEmptyBody(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
	ctx := context.Background()

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gf4"
	policyID := "9c9e5f83-03a7-42e1-b7f5-14d0c3a2b111"

	// A 2xx response with a null/empty body leaves resp.JSON200 nil; the read
	// must return an error, not panic.
	mockClient.EXPECT().
		WorkloadOptimizationAPIGetWorkloadScalingPolicy(ctx, clusterID, policyID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`null`))),
		}, nil)

	_, data := dataSourceWorkloadScalingPolicyForTest(t, map[string]cty.Value{
		"cluster_id": cty.StringVal(clusterID),
		"policy_id":  cty.StringVal(policyID),
	})

	diags := dataSourceWorkloadScalingPolicyRead(ctx, data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})
	r.True(diags.HasError())
	r.Contains(diags[0].Summary, "scaling policy not found")
}

func TestDataSourceWorkloadScalingPolicyDefinitionSchemaParity(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	resourceSchema := resourceWorkloadScalingPolicy().Schema
	dsSchema := dataSourceWorkloadScalingPolicy().Schema

	// Every definition field of the resource is exposed by the data source.
	for k := range resourceSchema {
		if k == FieldClusterID || k == "name" {
			continue
		}
		r.Contains(dsSchema, k, "data source is missing field %q present in the resource schema", k)
	}

	// Everything except the data source inputs is computed-only.
	inputs := map[string]bool{
		FieldClusterID:  true,
		FieldPolicyID:   true,
		"name":          true,
		FieldIsDefault:  true,
		FieldIsReadonly: true,
		FieldIsCastware: true,
	}
	for k, s := range dsSchema {
		if inputs[k] {
			continue
		}
		r.True(s.Computed, "%q must be computed", k)
		r.False(s.Required, "%q must not be required", k)
		r.False(s.Optional, "%q must not be optional", k)
	}

	// Nested blocks are flipped to computed as well, and defaults are dropped.
	cpu, ok := dsSchema["cpu"].Elem.(*schema.Resource)
	r.True(ok)
	r.True(cpu.Schema["function"].Computed)
	r.False(cpu.Schema["function"].Required)
	r.Nil(cpu.Schema["constraints"].Default)

	confidence, ok := dsSchema[FieldConfidence].Elem.(*schema.Resource)
	r.True(ok)
	r.Nil(confidence.Schema[FieldConfidenceThreshold].Default)
}

func TestAccGKE_DataSourceWorkloadScalingPolicy(t *testing.T) {
	rName := fmt.Sprintf("%v-ds-%v", ResourcePrefix, acctest.RandString(8))
	policyResource := "castai_workload_scaling_policy.test"
	byName := "data.castai_workload_scaling_policy.by_name"
	byID := "data.castai_workload_scaling_policy.by_id"
	clusterName := "tf-core-acc-20230723"
	projectID := os.Getenv("GOOGLE_PROJECT_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			cleanupLeftoverHelmReleases(t)
		},
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckScalingPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: dataSourceWorkloadScalingPolicyConfig(clusterName, projectID, rName),
				Check: resource.ComposeTestCheckFunc(
					// Lookup by name returns the created policy with its full definition.
					resource.TestCheckResourceAttrPair(byName, "id", policyResource, "id"),
					resource.TestCheckResourceAttrPair(byName, "name", policyResource, "name"),
					resource.TestCheckResourceAttrPair(byName, "apply_type", policyResource, "apply_type"),
					resource.TestCheckResourceAttrPair(byName, "management_option", policyResource, "management_option"),
					resource.TestCheckResourceAttrPair(byName, "cpu.0.function", policyResource, "cpu.0.function"),
					resource.TestCheckResourceAttrPair(byName, "cpu.0.args.0", policyResource, "cpu.0.args.0"),
					resource.TestCheckResourceAttrPair(byName, "cpu.0.overhead", policyResource, "cpu.0.overhead"),
					resource.TestCheckResourceAttrPair(byName, "cpu.0.constraints.0.min.0.constant", policyResource, "cpu.0.constraints.0.min.0.constant"),
					resource.TestCheckResourceAttrPair(byName, "memory.0.function", policyResource, "memory.0.function"),
					resource.TestCheckResourceAttrPair(byName, "memory.0.overhead", policyResource, "memory.0.overhead"),
					resource.TestCheckResourceAttrPair(byName, "excluded_containers.0", policyResource, "excluded_containers.0"),
					resource.TestCheckResourceAttrPair(byName, "confidence.0.threshold", policyResource, "confidence.0.threshold"),
					resource.TestCheckResourceAttr(byName, "is_readonly", "false"),
					resource.TestCheckResourceAttr(byName, "is_default", "false"),
					// Lookup by id returns the same policy.
					resource.TestCheckResourceAttrPair(byID, "id", policyResource, "id"),
					resource.TestCheckResourceAttrPair(byID, "name", policyResource, "name"),
					resource.TestCheckResourceAttrPair(byID, "cpu.0.overhead", policyResource, "cpu.0.overhead"),
				),
			},
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"google": {
				Source:            "hashicorp/google",
				VersionConstraint: "> 4.75.0",
			},
			"google-beta": {
				Source:            "hashicorp/google-beta",
				VersionConstraint: "> 4.75.0",
			},
			"helm": {
				Source:            "hashicorp/helm",
				VersionConstraint: "~> 2.17.0",
			},
		},
	})
}

func dataSourceWorkloadScalingPolicyConfig(clusterName, projectID, name string) string {
	cfg := fmt.Sprintf(`
resource "castai_workload_scaling_policy" "test" {
	depends_on = [helm_release.castai_workload_autoscaler]

	name                = %[1]q
	cluster_id          = castai_gke_cluster.test.id
	apply_type          = "IMMEDIATE"
	management_option   = "READ_ONLY"
	excluded_containers = ["acc-test"]
	confidence {
		threshold = 0.4
	}
	cpu {
		function = "QUANTILE"
		overhead = 0.07
		args     = ["0.9"]
		constraints {
			min { constant = 0.05 }
		}
	}
	memory {
		function = "MAX"
		overhead = 0.2
	}
}

data "castai_workload_scaling_policy" "by_name" {
	cluster_id = castai_gke_cluster.test.id
	name       = castai_workload_scaling_policy.test.name
}

data "castai_workload_scaling_policy" "by_id" {
	cluster_id = castai_gke_cluster.test.id
	policy_id  = castai_workload_scaling_policy.test.id
}
`, name)
	return ConfigCompose(clusterComponentsConfig(clusterName, projectID, name), cfg)
}
