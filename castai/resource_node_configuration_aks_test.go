package castai

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceNodeConfiguration_aks(t *testing.T) {
<<<<<<< HEAD
	rName := fmt.Sprintf("%v-node-cfg-aks-%v", ResourcePrefix, acctest.RandString(8))
=======
	rName := fmt.Sprintf("%v-aks-%v", ResourcePrefix, acctest.RandString(8))
>>>>>>> 5552411 (test: acceptance tests for AKS node configuration)
	resourceName := "castai_node_configuration.test"
	clusterName := "core-tf-acc"
	resourceGroupName := "core-tf-acc"
	nodeResourceGroupName := "core-tf-acc-ng"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		// Destroy of the cluster is not working properly. Cluster wasn't full onboarded and it's getting destroyed.
		// https://castai.atlassian.net/browse/CORE-2868 should solve the issue
		//CheckDestroy:      testAccCheckAKSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAKSNodeConfigurationConfig(rName, clusterName, resourceGroupName, nodeResourceGroupName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "disk_cpu_ratio", "35"),
					resource.TestCheckResourceAttr(resourceName, "min_disk_size", "122"),
					resource.TestCheckResourceAttr(resourceName, "aks.0.max_pods_per_node", "31"),
					resource.TestCheckResourceAttr(resourceName, "eks.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "kops.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "gke.#", "0"),
				),
			},
			{
				Config: testAccAKSNodeConfigurationUpdated(rName, clusterName, resourceGroupName, nodeResourceGroupName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
<<<<<<< HEAD
					resource.TestCheckResourceAttr(resourceName, "disk_cpu_ratio", "0"),
					resource.TestCheckResourceAttr(resourceName, "min_disk_size", "121"),
					resource.TestCheckResourceAttr(resourceName, "aks.0.max_pods_per_node", "32"),
=======
					resource.TestCheckResourceAttr(resourceName, "disk_cpu_ratio", "35"),
					resource.TestCheckResourceAttr(resourceName, "min_disk_size", "122"),
					resource.TestCheckResourceAttr(resourceName, "aks.0.max_pods_per_node", "31"),
>>>>>>> 5552411 (test: acceptance tests for AKS node configuration)
					resource.TestCheckResourceAttr(resourceName, "eks.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "kops.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "gke.#", "0"),
				),
			},
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"azurerm": {
				Source:            "hashicorp/azurerm",
				VersionConstraint: "~> 3.7.0",
			},
			"azuread": {
				Source:            "hashicorp/azuread",
				VersionConstraint: "~> 2.22.0",
			},
		},
	})
}

func testAccAKSNodeConfigurationConfig(rName, clusterName, rgName, ngName string) string {
	return ConfigCompose(testAccAKSClusterConfig(rName, clusterName, rgName, ngName), fmt.Sprintf(`
<<<<<<< HEAD
resource "castai_node_configuration" "test" {
  name   		    = %[1]q
  cluster_id        = castai_aks_cluster.test.id
  disk_cpu_ratio    = 35
=======


resource "castai_node_configuration" "test" {
  name   		    = %[2]q
  cluster_id        = castai_aks_cluster.test.id
  disk_cpu_ratio    = 0
>>>>>>> 5552411 (test: acceptance tests for AKS node configuration)
  min_disk_size     = 122
  subnets   	    = [data.azurerm_subnet.internal.id]
  tags = {
    env = "development"
  }
  aks {
	max_pods_per_node = 31
  }
}

resource "castai_node_configuration_default" "test" {
  cluster_id       = castai_aks_cluster.test.id
  configuration_id = castai_node_configuration.test.id
}
<<<<<<< HEAD
`, rName))
=======
`, rgName, rName))
>>>>>>> 5552411 (test: acceptance tests for AKS node configuration)
}

func testAccAKSNodeConfigurationUpdated(rName, clusterName, rgName, ngName string) string {
	return ConfigCompose(testAccAKSClusterConfig(rName, clusterName, rgName, ngName), fmt.Sprintf(`
resource "castai_node_configuration" "test" {
  name   		    = %[2]q
  cluster_id        = castai_aks_cluster.test.id
  disk_cpu_ratio    = 0
  min_disk_size     = 121
  subnets   	    = [data.azurerm_subnet.internal.id]
  tags = {
    env = "development"
  }
  aks {
	max_pods_per_node = 32
  }
}
`, rgName, rName))
}
