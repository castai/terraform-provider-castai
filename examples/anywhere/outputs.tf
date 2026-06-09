output "castai_agent_status" {
  value = helm_release.castai_agent.status
}

output "castai_cluster_controller_status" {
  value = helm_release.castai_cluster_controller.status
}

output "castai_evictor_status" {
  value = helm_release.castai_evictor.status
}

output "castai_pod_mutator_status" {
  value = helm_release.castai_pod_mutator.status
}

output "castai_workload_autoscaler_status" {
  value = helm_release.castai_workload_autoscaler.status
}
