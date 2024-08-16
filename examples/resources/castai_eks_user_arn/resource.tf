resource "castai_eks_user_arn" "castai_user_arn" {
  cluster_id = castai_eks_clusterid.cluster_id.id
}
