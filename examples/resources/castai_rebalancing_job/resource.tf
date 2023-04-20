resource "castai_rebalancing_job" "spots" {
	cluster_id = castai_eks_cluster.test.id
	rebalancing_schedule_id = castai_rebalancing_schedule.spots.id
	enabled = true
}
