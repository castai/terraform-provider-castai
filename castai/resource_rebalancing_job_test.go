package castai

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccResourceRebalancingJob_basic(t *testing.T) {
	rName := fmt.Sprintf("%v-rebalancing-job-%v", ResourcePrefix, acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: makeInitialRebalancingJobConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_job.test", "enabled", "true"),
				),
			},
		},
	})
}

func makeInitialRebalancingJobConfig(rName string) string {
	template := `
resource "castai_eks_clusterid" "test" {
  account_id   = "fake"
  region       = "eu-central-1"
  cluster_name = "fake"
}

resource "castai_rebalancing_schedule" "test" {
	name = %q
	schedule {
		cron = "5 4 * * *"
	}
	trigger_conditions {
		savings_percentage = 15.25
	}
	launch_configuration {
	}
}

// %q
resource "castai_rebalancing_job" "test" {
	cluster_id = castai_eks_clusterid.test.id
	rebalancing_schedule_id = castai_rebalancing_schedule.test.id
}
`
	return fmt.Sprintf(template, rName)
}
