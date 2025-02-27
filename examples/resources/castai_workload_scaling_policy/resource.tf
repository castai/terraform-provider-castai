resource "castai_workload_scaling_policy" "services" {
  name              = "services"
  cluster_id        = castai_gke_cluster.dev.id
  apply_type        = "IMMEDIATE"
  management_option = "MANAGED"
  cpu {
    function = "QUANTILE"
    overhead = 0.15
    apply_threshold_strategy {
      type       = "PERCENTAGE"
      percentage = 0.1
    }
    args                     = ["0.9"]
    look_back_period_seconds = 172800
    min                      = 0.1
    max                      = 1
  }
  memory {
    function = "MAX"
    overhead = 0.35
    apply_threshold_strategy {
      type = "DEFAULT_ADAPTIVE"
    }
    limit {
      type       = "MULTIPLIER"
      multiplier = 1.5
    }
    management_option = "READ_ONLY"
  }
  startup {
    period_seconds = 240
  }
  downscaling {
    apply_type = "DEFERRED"
  }
  memory_event {
    apply_type = "IMMEDIATE"
  }
  anti_affinity {
    consider_anti_affinity = false
  }
  confidence {
    threshold = 0.9
  }
}
