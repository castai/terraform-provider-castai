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
					resource.TestCheckResourceAttr(resourceName, "memory.0.function", "MAX"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.overhead", "0.25"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.apply_threshold", "0.1"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.args.#", "0"),
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
					resource.TestCheckResourceAttr(resourceName, "memory.0.function", "QUANTILE"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.overhead", "0.35"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.apply_threshold", "0.2"),
					resource.TestCheckResourceAttr(resourceName, "memory.0.args.0", "0.9"),
					resource.TestCheckResourceAttr(resourceName, "startup.0.period_seconds", "123"),
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
			look_back_period_seconds = 86401
		}
		memory {
			function 		= "MAX"
			overhead 		= 0.25
			apply_threshold = 0.1
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
		}
		memory {
			function 		= "QUANTILE"
			overhead 		= 0.35
			apply_threshold = 0.2
			args 			= ["0.9"]
		}
		startup {
			period_seconds = 123
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

func Test_validateArgs(t *testing.T) {
	tests := map[string]struct {
		args    sdk.WorkloadoptimizationV1ResourcePolicies
		wantErr bool
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
			wantErr: true,
		},
		"should return error when MAX has args provided": {
			args: sdk.WorkloadoptimizationV1ResourcePolicies{
				Function: "MAX",
				Args:     []string{"0.5"},
			},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := validateArgs(tt.args, ""); (err != nil) != tt.wantErr {
				t.Errorf("validateArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
