resource "castai_hibernation_schedule" "my_schedule" {
  name    = "workday"
  enabled = false

  pause_config {
    enabled = true

    schedule {
      cron_expression = "0 17 * * 1-5"
    }
  }

  resume_config {
    enabled = true

    schedule {
      cron_expression = "0 9 * * 1-5"
    }

    job_config {
      node_config {
        instance_type = "e2-standard-4"
      }
    }
  }

  cluster_assignments {
    assignment {
      cluster_id = "38a49ce8-e900-4a10-be89-48fb2efb1025"
    }
  }
}