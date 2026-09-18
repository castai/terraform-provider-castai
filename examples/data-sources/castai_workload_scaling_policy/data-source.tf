# Use a system policy as a template, fetched by name.
data "castai_workload_scaling_policy" "template" {
  cluster_id = castai_gke_cluster.cluster.id
  name       = "balanced"
}

# Clone the template into a managed policy, customizing selected fields.
# Only the fields that differ from the template are set explicitly.
resource "castai_workload_scaling_policy" "custom" {
  cluster_id        = data.castai_workload_scaling_policy.template.cluster_id
  name              = "services-custom"
  apply_type        = data.castai_workload_scaling_policy.template.apply_type
  management_option = data.castai_workload_scaling_policy.template.management_option

  cpu {
    function = data.castai_workload_scaling_policy.template.cpu[0].function
    args     = data.castai_workload_scaling_policy.template.cpu[0].args
    overhead = 0.2 # the only field we want to change
  }

  memory {
    function = data.castai_workload_scaling_policy.template.memory[0].function
    overhead = data.castai_workload_scaling_policy.template.memory[0].overhead
  }
}

# Multiple templates can be fetched with for_each.
data "castai_workload_scaling_policy" "all" {
  for_each   = toset(["balanced", "performance"])
  cluster_id = castai_gke_cluster.cluster.id
  name       = each.value
}
