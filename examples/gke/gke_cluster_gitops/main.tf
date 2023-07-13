resource "castai_gke_cluster" "my_castai_cluster" {
  project_id                 = var.project_id
  location                   = var.cluster_region
  name                       = var.cluster_name
  delete_nodes_on_disconnect = var.delete_nodes_on_disconnect

  credentials_json = module.castai-gke-iam.private_key
}

resource "castai_node_configuration" "default" {
  cluster_id     = castai_gke_cluster.my_castai_cluster.id
  name           = "default"
  disk_cpu_ratio = 0
  min_disk_size  = 100
  subnets        = var.subnets

  gke {
    disk_type = "pd-standard"
  }
}

module "castai-gke-iam" {
  source = "castai/gke-iam/castai"

  project_id       = var.project_id
  gke_cluster_name = var.cluster_name
}

resource "castai_node_configuration_default" "this" {
  cluster_id       = castai_gke_cluster.my_castai_cluster.id
  configuration_id = castai_node_configuration.default.id
}
