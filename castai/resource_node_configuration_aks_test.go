package castai

import (
	"fmt"
)

func testAccAKSNodeConfigurationConfig(rName, clusterName, resourceGroupName string) string {
	return ConfigCompose(testAccAKSWithFederationIDConfig(clusterName), fmt.Sprintf(`
provider "azurerm" {
	features {}		
}
data "azurerm_subnet" "internal" {
  name                 =  "internal"
  virtual_network_name = "%[2]s-network"
  resource_group_name  = %[2]q 
}

resource "castai_node_configuration" "test" {
  name   		    = %[1]q
  cluster_id        = castai_aks_cluster.test.id
  disk_cpu_ratio    = 35
  min_disk_size     = 122
  subnets   	    = [data.azurerm_subnet.internal.id]
  tags = {
    env = "development"
  }
  aks {
	max_pods_per_node = 31
    aks_image_family = "ubuntu"
  }
}

resource "castai_node_configuration_default" "test" {
  cluster_id       = castai_aks_cluster.test.id
  configuration_id = castai_node_configuration.test.id
}
`, rName, resourceGroupName))
}

func testAccAKSNodeConfigurationUpdated(rName, clusterName, resourceGroupName string) string {
	return ConfigCompose(testAccAKSWithFederationIDConfig(clusterName), fmt.Sprintf(`
provider "azurerm" {
	features {}		
}
data "azurerm_subnet" "internal" {
  name                 =  "internal"
  virtual_network_name = "%[2]s-network"
  resource_group_name  = %[2]q 
}

resource "castai_node_configuration" "test" {
  name   		    = %[1]q
  cluster_id        = castai_aks_cluster.test.id
  disk_cpu_ratio    = 0
  min_disk_size     = 121
  subnets   	    = [data.azurerm_subnet.internal.id]
  tags = {
    env = "development"
  }
  aks {
	max_pods_per_node = 32
    aks_image_family = "azure-linux"
	ephemeral_os_disk {
    	placement = "CacheDisk"
	}
	loadbalancers {
		name = "test-lb"
		ip_based_backend_pools {
			name = "test"
		}
    }
	network_security_group = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/networkSecurityGroups/test-nsg"
	application_security_groups = ["/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/applicationSecurityGroups/test-asg"]
    public_ip {
		public_ip_prefix = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/publicIPAddresses/test-ip"
		tags = {
			FirstPartyUsage = "something"
		}
		idle_timeout_in_minutes = 10
    }
   pod_subnet_id = data.azurerm_subnet.internal.id
  }
}
`, rName, resourceGroupName))
}
