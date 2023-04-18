package castai

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccResourceRebalancingSchedule_basic(t *testing.T) {
	rName := fmt.Sprintf("%v-rebalancing-schedule-%v", ResourcePrefix, acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: makeInitialRebalancingScheduleConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "name", rName),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "schedule.0.cron", "5 4 * * *"),
				),
			},
			{
				// import by ID
				ImportState:       true,
				ResourceName:      "castai_rebalancing_schedule.test",
				ImportStateVerify: true,
			},
			{
				// import by name
				ImportState:       true,
				ResourceName:      "castai_rebalancing_schedule.test",
				ImportStateId:     rName,
				ImportStateVerify: true,
			},
			{
				// test edits
				Config: makeUpdatedRebalancingScheduleConfig(rName + " renamed"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "name", rName+" renamed"),
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "schedule.0.cron", "1 4 * * *"),
				),
			},
		},
	})
}

func makeInitialRebalancingScheduleConfig(rName string) string {
	template := `
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
`
	return fmt.Sprintf(template, rName)
}

func makeUpdatedRebalancingScheduleConfig(rName string) string {
	template := `
resource "castai_rebalancing_schedule" "test" {
	name = %q
	schedule {
		cron = "1 4 * * *"
	}
	trigger_conditions {
		savings_percentage = 1.23456
	}
	launch_configuration {
		node_ttl_seconds = 10
		num_targeted_nodes = 3
		rebalancing_min_nodes = 2
		selector = jsonencode({
			nodeSelectorTerms = [{
				matchExpressions = [
					{
						key =  "thing"
						operator = "in"
						values = ["a", "b", "c"]
					}
				]
			}]
		})
	}
}
`
	return fmt.Sprintf(template, rName)
}
