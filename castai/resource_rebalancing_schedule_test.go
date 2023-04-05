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
				Config: testAccRebalancingScheduleConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_rebalancing_schedule.test", "name", rName),
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
		},
	})
}

func testAccRebalancingScheduleConfig(rName string) string {
	template := `
resource "castai_rebalancing_schedule" "test" {
	name = %q
	schedule {
		cron = "5 4 * * *"
	}
}
`
	return fmt.Sprintf(template, rName)
}
