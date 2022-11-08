# Set node configuration as default one.
resource "castai_node_configuration_default" "test" {
  cluster_id       = castai_eks_cluster.test.id
  configuration_id = castai_node_configuration.test.id
}