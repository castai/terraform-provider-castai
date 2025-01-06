package castai

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
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
					resource.TestCheckResourceAttr(resourceName, "cpu.0.apply_threshold", "0.06"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.args.0", "0.86"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.look_back_period_seconds", "86401"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.min", "0.1"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.max", "1"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.limit.0.type", "MULTIPLIER"),
					resource.TestCheckResourceAttr(resourceName, "cpu.0.limit.0.multiplier", "1.2"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.function", "MAX"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.overhead", "0.25"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.apply_threshold", "0.1"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.args.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.min", "100"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.limit.0.type", "MULTIPLIER"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.limit.0.multiplier", "1.8"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.management_option", "READ_ONLY"),
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
					resource.TestCheckResourceAttr(resourceName, "memory.0.apply_threshold", "0.2"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.args.0", "0.9"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.min", "100"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.max", "512"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.limit.0.type", "NO_LIMIT"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.management_option", "READ_ONLY"),
					resource.TestCheckResourceAttr(resourceName, "startup.0.period_seconds", "123"),
					resource.TestCheckResourceAttr(resourceName, "downscaling.0.apply_type", "DEFERRED"),
					resource.TestCheckResourceAttr(resourceName, "memory_event.0.apply_type", "DEFERRED"),
					resource.TestCheckResourceAttr(resourceName, "anti_affinity.0.consider_anti_affinity", "true"),
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
		},
	})
}

func scalingPolicyConfig(clusterName, projectID, name string) string {
	cfg := fmt.Sprintf(`
	resource "castai_workload_scaling_policy" "test" {
		name 				= %[1]q
		cluster_id			= castai_gke_cluster.test.id
		apply_type			= "IMMEDIATE"
		management_option	= "READ_ONLY"
		cpu {
			function 		= "QUANTILE"
			overhead 		= 0.05
			apply_threshold = 0.06
			args 			= ["0.86"]
            min             = 0.1
            max             = 1
			look_back_period_seconds = 86401
			limit {
				type 		    = "MULTIPLIER"
				multiplier 	= 1.2
			}
		}
		memory {
			function 		= "MAX"
			overhead 		= 0.25
			apply_threshold = 0.1
            min             = 100
			limit {
				type 		    = "MULTIPLIER"
				multiplier 	= 1.8
			}
            management_option	= "READ_ONLY"
		}
	}`, name)

	return ConfigCompose(testAccGKEClusterConfig(name, clusterName, projectID), cfg)
}

func scalingPolicyConfigUpdated(clusterName, projectID, name string) string {
	updatedName := name + "-updated"
	cfg := fmt.Sprintf(`
	resource "castai_workload_scaling_policy" "test" {
		name 				= %[1]q
		cluster_id			= castai_gke_cluster.test.id
		apply_type			= "IMMEDIATE"
		management_option	= "MANAGED"
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
			apply_threshold = 0.2
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
	}`, updatedName)

	return ConfigCompose(testAccGKEClusterConfig(name, clusterName, projectID), cfg)
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

func Test_validateResourcePolicy(t *testing.T) {
	tests := map[string]struct {
		args   sdk.WorkloadoptimizationV1ResourcePolicies
		errMsg string
	}{
		"should not return error when QUANTILE has args provided": {
			args: sdk.WorkloadoptimizationV1ResourcePolicies{
				Function: "QUANTILE",
				Args:     []string{"0.5"},
			},
		},
		"should return error when QUANTILE has not args provided": {
			args: sdk.WorkloadoptimizationV1ResourcePolicies{
				Function: "QUANTILE",
			},
			errMsg: `field "cpu": QUANTILE function requires args to be provided`,
		},
		"should return error when MAX has args provided": {
			args: sdk.WorkloadoptimizationV1ResourcePolicies{
				Function: "MAX",
				Args:     []string{"0.5"},
			},
			errMsg: `field "cpu": MAX function doesn't accept any args`,
		},
		"should return error when no value is specified for the multiplier strategy": {
			args: sdk.WorkloadoptimizationV1ResourcePolicies{
				Limit: &sdk.WorkloadoptimizationV1ResourceLimitStrategy{
					Type: sdk.MULTIPLIER,
				},
			},
			errMsg: `field "cpu": field "limit": "MULTIPLIER" limit type requires multiplier value to be provided`,
		},
		"should return error when a value is specified for the no limit strategy": {
			args: sdk.WorkloadoptimizationV1ResourcePolicies{
				Limit: &sdk.WorkloadoptimizationV1ResourceLimitStrategy{
					Type:       sdk.NOLIMIT,
					Multiplier: lo.ToPtr(4.2),
				},
			},
			errMsg: `field "cpu": field "limit": "NO_LIMIT" limit type doesn't accept multiplier value`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateResourcePolicy(tt.args, "cpu")
			if tt.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.errMsg)
			}
		})
	}
}
