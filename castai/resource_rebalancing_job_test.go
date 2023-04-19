package castai

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"testing"
)

func TestAccResourceRebalancingJob_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: makeInitialRebalancingJobConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_job.test", "enabled", "true"),
				),
			},
			{
				ResourceName: "castai_rebalancing_job.test",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					clusterID := s.RootModule().Resources["castai_eks_clusterid.test"].Primary.ID
					rebalancingScheduleName := "test"
					return fmt.Sprintf("%v/%v", clusterID, rebalancingScheduleName), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: makeUpdatedRebalancingJobConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_job.test", "enabled", "false"),
				),
			},
		},
	})
}

func makeRebalancingJobConfig(config string) string {
	template := `
resource "castai_eks_clusterid" "test" {
  account_id   = "fake"
  region       = "eu-central-1"
  cluster_name = "fake"
}

resource "castai_rebalancing_schedule" "test" {
	name = "test"
	schedule {
		cron = "5 4 * * *"
	}
	trigger_conditions {
		savings_percentage = 15.25
	}
	launch_configuration {
	}
}

resource "castai_rebalancing_job" "test" {
	cluster_id = castai_eks_clusterid.test.id
	rebalancing_schedule_id = castai_rebalancing_schedule.test.id
	%s
}
`
	return fmt.Sprintf(template, config)
}

func makeInitialRebalancingJobConfig() string {
	return makeRebalancingJobConfig("")
}
func makeUpdatedRebalancingJobConfig() string {
	return makeRebalancingJobConfig("enabled=false")
}
