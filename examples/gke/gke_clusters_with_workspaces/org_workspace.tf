resource "castai_rebalancing_schedule" "org_rebalancing_schedule" {
  count = terraform.workspace == var.org_workspace ? 1 : 0 # Create only in the organization workspace
  name  = "org rebalancing schedule"
  schedule {
    cron = "5 * * * * *"
  }
  trigger_conditions {
    savings_percentage = 15
  }
  launch_configuration {
    node_ttl_seconds      = 350
    num_targeted_nodes    = 20
    rebalancing_min_nodes = 2
    execution_conditions {
      achieved_savings_percentage = 15
      enabled                     = true
    }
  }
}
