# Recommended: Use presets for standard metrics (managed by CAST AI).
# Presets provide curated metric definitions that are kept up to date automatically.
# Currently available: "jvm". More presets may be added in the future.
resource "castai_workload_custom_metrics_data_source" "prometheus" {
  cluster_id = castai_eks_cluster.this.id

  name = "my-prometheus"

  prometheus {
    url     = "http://prometheus-server.monitoring.svc.cluster.local:9090"
    timeout = "30s"

    presets = ["jvm"]
  }
}

# Advanced: Define custom metrics manually with PromQL queries.
# This can be combined with presets if needed.
resource "castai_workload_custom_metrics_data_source" "prometheus_custom" {
  cluster_id = castai_eks_cluster.this.id

  name = "my-prometheus-custom"

  prometheus {
    url = "http://prometheus-server.monitoring.svc.cluster.local:9090"

    metric {
      name  = "http_requests_total"
      query = "sum(rate(http_requests_total[5m])) by (pod)"
    }

    metric {
      name  = "custom_queue_depth"
      query = "avg(queue_depth) by (pod)"
    }
  }
}
