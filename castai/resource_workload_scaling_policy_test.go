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
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestAccResourceWorkloadScalingPolicy(t *testing.T) {
	rName := fmt.Sprintf("%v-policy-%v", ResourcePrefix, acctest.RandString(8))
	resourceName := "castai_workload_scaling_policy.test"
	clusterName := "tf-core-acc-20230723"
	projectID := os.Getenv("GOOGLE_PROJECT_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckScalingPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: scalingPolicyConfig(clusterName, projectID, rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "apply_type", "IMMEDIATE"),
					resource.TestCheckResourceAttr(resourceName, "management_option", "READ_ONLY"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.function", "QUANTILE"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.overhead", "0.05"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.apply_threshold_strategy.0.type", "PERCENTAGE"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.apply_threshold_strategy.0.percentage", "0.6"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.args.0", "0.86"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.look_back_period_seconds", "86401"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.min", "0.1"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.max", "1"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.limit.0.type", "MULTIPLIER"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.limit.0.multiplier", "1.2"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.function", "MAX"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.overhead", "0.25"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.apply_threshold_strategy.0.type", "CUSTOM_ADAPTIVE"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.apply_threshold_strategy.0.numerator", "0.4"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.apply_threshold_strategy.0.denominator", "0.5"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.apply_threshold_strategy.0.exponent", "0.6"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.args.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.min", "100"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.limit.0.type", "MULTIPLIER"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.limit.0.multiplier", "1.8"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.management_option", "READ_ONLY"),
					resource.TestCheckResourceAttr(resourceName, "confidence.0.threshold", "0.4"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.0.namespace.0.names.0", "default"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.0.namespace.0.names.1", "kube-system"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.1.workload.0.gvk.0", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.1.workload.0.gvk.1", "StatefulSet"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.1.workload.0.labels_expressions.0.key", "region"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.1.workload.0.labels_expressions.0.operator", "NotIn"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.1.workload.0.labels_expressions.0.values.0", "eu-west-1"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.1.workload.0.labels_expressions.0.values.1", "eu-west-2"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.1.workload.0.labels_expressions.1.key", "helm.sh/chart"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.1.workload.0.labels_expressions.1.operator", "Exists"),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					clusterID := s.RootModule().Resources["castai_gke_cluster.test"].Primary.ID
					return fmt.Sprintf("%v/%v", clusterID, rName), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: scalingPolicyConfigUpdated(clusterName, projectID, rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-updated"),
					resource.TestCheckResourceAttr(resourceName, "apply_type", "IMMEDIATE"),
					resource.TestCheckResourceAttr(resourceName, "management_option", "MANAGED"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.function", "QUANTILE"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.overhead", "0.15"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.apply_threshold", "0.1"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.args.0", "0.9"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.look_back_period_seconds", "86402"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.min", "0.1"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.limit.0.type", "NO_LIMIT"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.function", "QUANTILE"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.overhead", "0.35"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.apply_threshold_strategy.0.type", "PERCENTAGE"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.apply_threshold_strategy.0.percentage", "0.2"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.args.0", "0.9"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.min", "100"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.max", "512"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.limit.0.type", "NO_LIMIT"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.management_option", "READ_ONLY"),
					resource.TestCheckResourceAttr(resourceName, "startup.0.period_seconds", "123"),
					resource.TestCheckResourceAttr(resourceName, "downscaling.0.apply_type", "DEFERRED"),
					resource.TestCheckResourceAttr(resourceName, "memory_event.0.apply_type", "DEFERRED"),
					resource.TestCheckResourceAttr(resourceName, "confidence.0.threshold", "0.6"),
					resource.TestCheckResourceAttr(resourceName, "anti_affinity.0.consider_anti_affinity", "true"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.0.namespace.0.names.0", "team-a"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.1.workload.0.gvk.0", "DaemonSet"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.1.workload.0.labels_expressions.0.key", "helm.sh/chart"),
					resource.TestCheckResourceAttr(resourceName, "assignment_rules.0.rules.1.workload.0.labels_expressions.0.operator", "DoesNotExist"),
					resource.TestCheckResourceAttr(resourceName, "predictive_scaling.0.cpu.0.enabled", "true"),
					// Requires workload-autoscaler from v0.35.3
					resource.TestCheckResourceAttr(resourceName, "rollout_behavior.0.type", "NO_DISRUPTION"),
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

func clusterConnectConfig(clusterName, projectID, name string) string {
	return testAccGKEClusterConfig(name, clusterName, projectID)
}

// clusterComponentsConfig returns agent, cluster-controller, workload-autoscaler helm release
// installation configs. workload-autoscaler is installed since some features are checking WA version.
func clusterComponentsConfig(clusterName, projectID, name string) string {
	cfg := fmt.Sprintf(`
	resource "helm_release" "castai_agent" {
		name             = "castai-agent"
		repository       = "https://castai.github.io/helm-charts"
		chart            = "castai-agent"
		namespace        = "castai-agent"
		create_namespace = true
		cleanup_on_fail  = true
		wait             = true
	
		set {
			name  = "provider"
			value = "gke"
		}
	
		set_sensitive {
			name  = "apiKey"
			value = %[1]q
		}
	
		set {
			name  = "apiURL"
			value = %[2]q
		}
	
		set {
			name  = "createNamespace"
			value = "false"
		}
	
		depends_on = [castai_gke_cluster.test]
	}
	
	resource "helm_release" "castai_cluster_controller" {
		depends_on       = [helm_release.castai_agent]
		name             = "castai-cluster-controller"
		repository       = "https://castai.github.io/helm-charts"
		chart            = "castai-cluster-controller"
		namespace        = "castai-agent"
		create_namespace = false
		cleanup_on_fail  = true
		wait             = true
	
		set_sensitive {
			name  = "castai.apiKey"
			value = %[1]q
		}
	
		set {
			name  = "castai.apiURL"
			value = %[2]q
		}
	
		set {
			name  = "castai.clusterID"
			value = castai_gke_cluster.test.id
		}
	
		set {
			name  = "autoscaling.enabled"
			value = "false"
		}
	}
	
	resource "helm_release" "castai_workload_autoscaler" {
		depends_on       = [helm_release.castai_agent, helm_release.castai_cluster_controller]
		name             = "castai-workload-autoscaler"
		repository       = "https://castai.github.io/helm-charts"
		chart            = "castai-workload-autoscaler"
		namespace        = "castai-agent"
		create_namespace = false
		cleanup_on_fail  = true
		wait             = true
	
		set_sensitive {
			name  = "castai.apiKey"
			value = %[1]q
		}
	
		set {
			name  = "castai.apiURL"
			value = %[2]q
		}
	
		set {
			name  = "castai.clusterID"
			value = castai_gke_cluster.test.id
		}
	}`, getAPIToken(), getAPIUrl())
	return ConfigCompose(clusterConnectConfig(clusterName, projectID, name), cfg)
}

func scalingPolicyConfig(clusterName, projectID, name string) string {
	cfg := fmt.Sprintf(`
	resource "castai_workload_scaling_policy" "test" {
  		depends_on          = [helm_release.castai_workload_autoscaler]

		name 				= %[1]q
		cluster_id			= castai_gke_cluster.test.id
		apply_type			= "IMMEDIATE"
		management_option	= "READ_ONLY"
		confidence {
			threshold = 0.4
		}
		assignment_rules {
			rules {
				namespace {
					names = ["default", "kube-system"]
				}
			}
			rules {
				workload {
					gvk = ["Deployment", "StatefulSet"]
					labels_expressions {
						key      = "region"
						operator = "NotIn"
						values = ["eu-west-1", "eu-west-2"]
					}
					labels_expressions {
						key      = "helm.sh/chart"
						operator = "Exists"
					}
				}
			}
		}
		cpu {
			function 		= "QUANTILE"
			overhead 		= 0.05
			args 			= ["0.86"]
            min             = 0.1
            max             = 1
			look_back_period_seconds = 86401
			limit {
				type 		    = "MULTIPLIER"
				multiplier 	= 1.2
			}
			apply_threshold_strategy {
				type = "PERCENTAGE"
				percentage = 0.6
			}
		}
		memory {
			function 		= "MAX"
			overhead 		= 0.25
			apply_threshold_strategy {
				type = "CUSTOM_ADAPTIVE"
				numerator = 0.4
				denominator = 0.5
                exponent = 0.6
			}
            min             = 100
			limit {
				type 		    = "MULTIPLIER"
				multiplier 	= 1.8
			}
            management_option	= "READ_ONLY"
		}
	}`, name)

	return ConfigCompose(clusterComponentsConfig(clusterName, projectID, name), cfg)
}

func scalingPolicyConfigUpdated(clusterName, projectID, name string) string {
	updatedName := name + "-updated"
	cfg := fmt.Sprintf(`
	resource "castai_workload_scaling_policy" "test" {
  		depends_on          = [helm_release.castai_workload_autoscaler]

		name 				= %[1]q
		cluster_id			= castai_gke_cluster.test.id
		apply_type			= "IMMEDIATE"
		management_option	= "MANAGED"
		assignment_rules {
			rules {
				namespace {
					names = ["team-a"]
				}
			}
			rules {
				workload {
					gvk = ["DaemonSet"]
					labels_expressions {
						key      = "helm.sh/chart"
						operator = "DoesNotExist"
					}
				}
			}
		}
		predictive_scaling {
			cpu {
				enabled = true
			}
		}
		rollout_behavior {
			type = "NO_DISRUPTION"
		}
		cpu {
			function 		= "QUANTILE"
			overhead 		= 0.15
			apply_threshold = 0.1
			args 			= ["0.9"]
			look_back_period_seconds = 86402
            min             = 0.1
			limit {
				type 		    = "NO_LIMIT"
			}
		}
		memory {
			function 		= "QUANTILE"
			overhead 		= 0.35
			apply_threshold_strategy {
				type = "PERCENTAGE"
				percentage = 0.2
			}
			args 			= ["0.9"]
            min             = 100
            max             = 512
			limit {
				type 		    = "NO_LIMIT"
			}
            management_option = "READ_ONLY"
		}
		startup {
			period_seconds = 123
		}
	    downscaling {
		    apply_type = "DEFERRED"
	    }
		memory_event {
			apply_type = "DEFERRED"
		}
		anti_affinity {
			consider_anti_affinity = true
		}
		confidence {
			threshold = 0.6
		}
	}`, updatedName)

	return ConfigCompose(clusterComponentsConfig(clusterName, projectID, name), cfg)
}

func testAccCheckScalingPolicyDestroy(s *terraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).api
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_workload_scaling_policy" {
			continue
		}

		id := rs.Primary.ID
		clusterID := rs.Primary.Attributes["cluster_id"]
		resp, err := client.WorkloadOptimizationAPIGetWorkloadScalingPolicyWithResponse(ctx, clusterID, id)
		if err != nil {
			return err
		}
		if resp.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("scaling policy %s still exists", rs.Primary.ID)
	}

	return nil
}

func getAPIToken() string {
	return os.Getenv("CASTAI_API_TOKEN")
}

func getAPIUrl() string {
	apiUrl := os.Getenv("CASTAI_API_URL")
	if apiUrl == "" {
		return "https://api.dev-master.cast.ai"
	}
	return apiUrl
}

func Test_validateResourcePolicy(t *testing.T) {
	tests := map[string]struct {
		args   map[string]interface{}
		errMsg string
	}{
		"should not return error when QUANTILE has args provided": {
			args: map[string]interface{}{
				"function": "QUANTILE",
				"args":     []interface{}{"0.5"},
			},
		},
		"should return error when QUANTILE has not args provided": {
			args: map[string]interface{}{
				"function": "QUANTILE",
			},
			errMsg: `field "cpu": QUANTILE function requires args to be provided`,
		},
		"should return error when MAX has args provided": {
			args: map[string]interface{}{
				"function": "MAX",
				"args":     []interface{}{"0.5"},
			},
			errMsg: `field "cpu": MAX function doesn't accept any args`,
		},
		"should return error when no value is specified for the multiplier strategy": {
			args: map[string]interface{}{
				"limit": []interface{}{map[string]interface{}{
					"type": "MULTIPLIER",
				}},
			},
			errMsg: `field "cpu": field "limit": field "multiplier": value must be set`,
		},
		"should return error when a value is specified for the no limit strategy": {
			args: map[string]interface{}{
				"limit": []interface{}{map[string]interface{}{
					"type":       "NO_LIMIT",
					"multiplier": 4.2,
				}},
			},
			errMsg: `field "cpu": field "limit": "NO_LIMIT" limit type doesn't accept multiplier value`,
		},
		"should return error when a percentage is not specified for the apply threshold strategy": {
			args: map[string]interface{}{
				"apply_threshold_strategy": []interface{}{map[string]interface{}{
					"type": "PERCENTAGE",
				}},
			},
			errMsg: `field "cpu": field "apply_threshold_strategy": field "percentage": value must be set`,
		},
		"should return error when unknown type is specified": {
			args: map[string]interface{}{
				"apply_threshold_strategy": []interface{}{map[string]interface{}{
					"type": "xyz",
				}}},
			errMsg: `field "cpu": field "apply_threshold_strategy": field "type": unknown apply threshold strategy type: "xyz"`,
		},
		"should not return error when strategy is valid": {
			args: map[string]interface{}{
				"apply_threshold_strategy": []interface{}{map[string]interface{}{
					"type":       "PERCENTAGE",
					"percentage": 0.5,
				}},
			},
		},
		"should return error when custom adaptive strategy is missing numerator": {
			args: map[string]interface{}{
				"apply_threshold_strategy": []interface{}{map[string]interface{}{
					"type":        "CUSTOM_ADAPTIVE",
					"denominator": "0.3",
					"exponent":    0.5,
				}},
			},
			errMsg: `field "cpu": field "apply_threshold_strategy": field "numerator": value must be set`,
		},
		"should return error when custom adaptive strategy denominator is zero value": {
			args: map[string]interface{}{
				"apply_threshold_strategy": []interface{}{map[string]interface{}{
					"type":        "CUSTOM_ADAPTIVE",
					"numerator":   0.3,
					"denominator": "",
					"exponent":    0.5,
				}},
			},
			errMsg: `field "cpu": field "apply_threshold_strategy": field "denominator": value must be set`,
		},
		"should return error when custom adaptive strategy exponent is missing": {
			args: map[string]interface{}{
				"apply_threshold_strategy": []interface{}{map[string]interface{}{
					"type":        "CUSTOM_ADAPTIVE",
					"numerator":   0.3,
					"denominator": "0.5",
				}},
			},
			errMsg: `field "cpu": field "apply_threshold_strategy": field "exponent": value must be set`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := toWorkloadScalingPolicies("cpu", tt.args)
			if tt.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.errMsg)
			}
		})
	}
}

func Test_toPredictiveScaling(t *testing.T) {
	tests := map[string]struct {
		args map[string]any
		exp  *sdk.WorkloadoptimizationV1PredictiveScalingSettings
	}{
		"should return predictive scaling settings": {
			args: map[string]any{
				FieldCPU: []any{
					map[string]any{
						FieldEnabled: true,
					},
				},
			},
			exp: &sdk.WorkloadoptimizationV1PredictiveScalingSettings{
				Cpu: &sdk.WorkloadoptimizationV1PredictiveScaling{
					Enabled: true,
				},
			},
		},
		"should return nil on empty map": {
			args: map[string]any{},
			exp:  nil,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			got := toPredictiveScaling(tt.args)
			r.Equal(tt.exp, got)
		})
	}
}

func Test_toRolloutBehavior(t *testing.T) {
	tests := map[string]struct {
		args map[string]any
		exp  *sdk.WorkloadoptimizationV1RolloutBehaviorSettings
	}{
		"should return rollout behavior settings": {
			args: map[string]any{
				FieldRolloutBehaviorType: FieldRolloutBehaviorNoDisruptionType,
			},
			exp: &sdk.WorkloadoptimizationV1RolloutBehaviorSettings{
				Type: lo.ToPtr(sdk.NODISRUPTION),
			},
		},
		"should return nil on empty map": {
			args: map[string]any{},
			exp:  nil,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			got := toRolloutBehavior(tt.args)
			r.Equal(tt.exp, got)
		})
	}
}

func Test_resourceWorkloadScalingPolicyCreate(t *testing.T) {
	organizationId := "63d2af53-9a42-4968-be1e-39316ebfd8d4"
	clusterId := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
	name := "test-sp"
	policyId := "98173807-6568-4e2b-9fe1-bcece3301649"

	tests := map[string]struct {
		state         map[string]cty.Value
		schemaVersion int
		setup         func(r *require.Assertions, mockClient *mock_sdk.MockClientInterface)
		expDiag       diag.Diagnostics
	}{
		"should create scaling policy": {
			schemaVersion: 0,
			state: map[string]cty.Value{
				"cluster_id": cty.StringVal(clusterId),
				"name":       cty.StringVal(name),
				"apply_type": cty.StringVal("IMMEDIATE"),
			},
			setup: func(r *require.Assertions, mockClient *mock_sdk.MockClientInterface) {
				applyType := sdk.WorkloadoptimizationV1ApplyType("IMMEDIATE")
				policy := &sdk.WorkloadoptimizationV1WorkloadScalingPolicy{
					OrganizationId: organizationId,
					ClusterId:      clusterId,
					Id:             policyId,
					Name:           name,
					ApplyType:      applyType,
				}

				mockClient.EXPECT().
					WorkloadOptimizationAPICreateWorkloadScalingPolicy(gomock.Any(), clusterId, gomock.Any()).
					DoAndReturn(func(_ context.Context, cID string, req sdk.WorkloadOptimizationAPICreateWorkloadScalingPolicyJSONRequestBody) (*http.Response, error) {
						r.Equal(clusterId, cID)
						return toResponse(r, policy, http.StatusOK)
					})
				mockClient.EXPECT().
					WorkloadOptimizationAPIGetWorkloadScalingPolicy(gomock.Any(), clusterId, policyId).
					DoAndReturn(func(_ context.Context, cID string, pID string) (*http.Response, error) {
						r.Equal(clusterId, cID)
						r.Equal(policyId, pID)
						return toResponse(r, policy, http.StatusOK)
					})
			},
		},
		"should create scaling policy with retries": {
			schemaVersion: 0,
			state: map[string]cty.Value{
				"cluster_id": cty.StringVal(clusterId),
				"name":       cty.StringVal(name),
				"apply_type": cty.StringVal("IMMEDIATE"),
			},
			setup: func(r *require.Assertions, mockClient *mock_sdk.MockClientInterface) {
				applyType := sdk.WorkloadoptimizationV1ApplyType("IMMEDIATE")
				policy := &sdk.WorkloadoptimizationV1WorkloadScalingPolicy{
					OrganizationId: organizationId,
					ClusterId:      clusterId,
					Id:             policyId,
					Name:           name,
					ApplyType:      applyType,
				}

				cnt := 0
				mockClient.EXPECT().
					WorkloadOptimizationAPICreateWorkloadScalingPolicy(gomock.Any(), clusterId, gomock.Any()).
					DoAndReturn(func(_ context.Context, cID string, req sdk.WorkloadOptimizationAPICreateWorkloadScalingPolicyJSONRequestBody) (*http.Response, error) {
						r.Equal(clusterId, cID)
						cnt++
						if cnt < 2 {
							return toResponse(r, nil, http.StatusServiceUnavailable)
						}
						return toResponse(r, policy, http.StatusOK)
					}).Times(2)
				mockClient.EXPECT().
					WorkloadOptimizationAPIGetWorkloadScalingPolicy(gomock.Any(), clusterId, policyId).
					DoAndReturn(func(_ context.Context, cID string, pID string) (*http.Response, error) {
						r.Equal(clusterId, cID)
						r.Equal(policyId, pID)
						return toResponse(r, policy, http.StatusOK)
					})
			},
		},
		"should not retry 400 status code": {
			schemaVersion: 0,
			state: map[string]cty.Value{
				"cluster_id": cty.StringVal(clusterId),
				"name":       cty.StringVal(name),
				"apply_type": cty.StringVal("IMMEDIATE"),
			},
			setup: func(r *require.Assertions, mockClient *mock_sdk.MockClientInterface) {
				mockClient.EXPECT().
					WorkloadOptimizationAPICreateWorkloadScalingPolicy(gomock.Any(), clusterId, gomock.Any()).
					DoAndReturn(func(_ context.Context, cID string, req sdk.WorkloadOptimizationAPICreateWorkloadScalingPolicyJSONRequestBody) (*http.Response, error) {
						r.Equal(clusterId, cID)
						return toResponse(r, nil, http.StatusBadRequest)
					})
			},
			expDiag: diag.Diagnostics{
				diag.Diagnostic{
					Summary: "expected status code 200, received: status=400 body=null\n",
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

			provider := &ProviderConfig{
				api: &sdk.ClientWithResponses{
					ClientInterface: mockClient,
				},
			}

			scalingPolicyResource := resourceWorkloadScalingPolicy()
			stateValue := cty.ObjectVal(tt.state)

			state := terraform.NewInstanceStateShimmedFromValue(stateValue, tt.schemaVersion)
			data := scalingPolicyResource.Data(state)

			tt.setup(r, mockClient)

			result := scalingPolicyResource.CreateContext(t.Context(), data, provider)

			if tt.expDiag != nil {
				r.Equal(tt.expDiag, result)
			} else {
				r.Nil(result)
				r.False(result.HasError())
				r.Equal(policyId, data.Id())
			}
		})
	}
}

func toJSON(r *require.Assertions, v any) *bytes.Buffer {
	body := bytes.NewBuffer([]byte(""))
	err := json.NewEncoder(body).Encode(v)
	r.NoError(err)
	return body
}

func toResponse(r *require.Assertions, v any, statusCode int) (*http.Response, error) {
	return &http.Response{StatusCode: statusCode, Body: io.NopCloser(toJSON(r, v)), Header: map[string][]string{"Content-Type": {"json"}}}, nil
}
