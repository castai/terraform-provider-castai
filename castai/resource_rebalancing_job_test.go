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
// %q
resource "castai_rebalancing_job" "test" {
	cluster_id = castai_eks_cluster.test.id
	rebalancing_schedule_id = "b6bfc074-a267-400f-b8f1-db0850c369b1"
}
`
	return ConfigCompose(testAccEKSClusterConfig(rName, "cost-terraform"), fmt.Sprintf(template, rName))
}
