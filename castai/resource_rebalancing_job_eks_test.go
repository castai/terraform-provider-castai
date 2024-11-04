package castai

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceRebalancingJob_eks(t *testing.T) {
	rName := fmt.Sprintf("%v-rebalancing-job-%v", ResourcePrefix, acctest.RandString(8))
	clusterName := "core-tf-acc"
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: makeInitialRebalancingJobConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_job.test", "enabled", "true"),
				),
			},
			{
				ResourceName: "castai_rebalancing_job.test",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					clusterID := s.RootModule().Resources["castai_eks_cluster.test"].Primary.ID
					return fmt.Sprintf("%v/%v", clusterID, rName), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: makeUpdatedRebalancingJobConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_job.test", "enabled", "false"),
				),
			},
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "~> 4.0",
			},
		},
	})
}

func makeRebalancingJobConfig(rName, config string) string {
	template := `
resource "castai_rebalancing_schedule" "test" {
	name = %[1]q
	schedule {
		cron = "5 4 * * *"
	}
	trigger_conditions {
		savings_percentage = 15.25
	}
	launch_configuration {
		execution_conditions {
			enabled = false
			achieved_savings_percentage = 0
		}
	}
}

resource "castai_rebalancing_job" "test" {
	cluster_id = castai_eks_cluster.test.id
	rebalancing_schedule_id = castai_rebalancing_schedule.test.id
	%[2]s
}
`
	return fmt.Sprintf(template, rName, config)
}

func makeInitialRebalancingJobConfig(rName, clusterName string) string {
	return ConfigCompose(testAccEKSClusterConfig(rName, clusterName), makeRebalancingJobConfig(rName, ""))
}

func makeUpdatedRebalancingJobConfig(rName, clusterName string) string {
	return ConfigCompose(testAccEKSClusterConfig(rName, clusterName), makeRebalancingJobConfig(rName, "enabled=false"))
}

func TestAccResourceRebalancingJobWithDataSource_eks(t *testing.T) {
	rName := fmt.Sprintf("%v-rebalancing-job-with-data-source-%v", ResourcePrefix, acctest.RandString(8))
	clusterName := "core-tf-acc"
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: makeRebalancingScheduleConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test-with-data-source", "name", rName),
				),
			},
			{
				Config: makeInitialRebalancingJobWithDataSourceConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.castai_rebalancing_schedule.data-source-for-rebalancing-schedule", "name", rName),
				),
			},
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "~> 4.0",
			},
		},
	})
}

func makeRebalancingScheduleConfig(rName string) string {
	template := `
resource "castai_rebalancing_schedule" "test-with-data-source" {
	name = %[1]q
	schedule {
		cron = "5 4 * * *"
	}
	trigger_conditions {
		savings_percentage = 15.25
	}
	launch_configuration {
		execution_conditions {
			enabled = false
			achieved_savings_percentage = 0
		}
	}
}
`
	return fmt.Sprintf(template, rName)
}

func makeRebalancingJobWithDataSourceConfig(rName string) string {
	template := `
resource "castai_rebalancing_schedule" "test-with-data-source" {
	name = %[1]q
	schedule {
		cron = "5 4 * * *"
	}
	trigger_conditions {
		savings_percentage = 15.25
	}
	launch_configuration {
		execution_conditions {
			enabled = false
			achieved_savings_percentage = 0
		}
	}
}

data "castai_rebalancing_schedule" "data-source-for-rebalancing-schedule" {
  name = %[1]q
}

resource "castai_rebalancing_job" "test-with-data-source" {
	cluster_id = castai_eks_cluster.test.id
	rebalancing_schedule_id = data.castai_rebalancing_schedule.data-source-for-rebalancing-schedule.id
}
`
	return fmt.Sprintf(template, rName)
}

func makeInitialRebalancingJobWithDataSourceConfig(rName, clusterName string) string {
	return ConfigCompose(testAccEKSClusterConfig(rName, clusterName), makeRebalancingJobWithDataSourceConfig(rName))
}
