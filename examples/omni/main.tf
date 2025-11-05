terraform {
  required_providers {
    castai = {
      source  = "castai/castai"
      version = ">= 0.0.1"
    }
  }
}

provider "castai" {
  api_url   = var.castai_api_url
  api_token = var.castai_api_token
}

variable "castai_api_url" {
  type        = string
  description = "CAST AI API URL"
  default     = "https://api.cast.ai"
}

variable "castai_api_token" {
  type        = string
  description = "CAST AI API token"
  sensitive   = true
}

variable "organization_id" {
  type        = string
  description = "CAST AI Organization ID"
}

variable "cluster_id" {
  type        = string
  description = "Omni Cluster ID"
}

# Example: Onboard an Omni cluster
resource "castai_omni_cluster" "example" {
  name               = "my-omni-cluster"
  organization_id    = var.organization_id
  service_account_id = "service-account-uuid"
}

# Example: Create an AWS edge location
resource "castai_omni_edge_location" "aws_us_east" {
  organization_id = var.organization_id
  cluster_id      = var.cluster_id
  name            = "aws-us-east-1"
  region          = "us-east-1"
  zones           = ["us-east-1a", "us-east-1b", "us-east-1c"]
  description     = "AWS US East 1 edge location"

  aws {
    account_id        = "123456789012"
    access_key_id     = var.aws_access_key_id
    secret_access_key = var.aws_secret_access_key
    vpc_id            = "vpc-12345678"
    subnet_ids        = ["subnet-1234", "subnet-5678", "subnet-9012"]
    security_group_id = "sg-12345678"
  }
}

# Example: Create a GCP edge location
resource "castai_omni_edge_location" "gcp_us_central" {
  organization_id = var.organization_id
  cluster_id      = var.cluster_id
  name            = "gcp-us-central1"
  region          = "us-central1"
  zones           = ["us-central1-a", "us-central1-b", "us-central1-c"]
  description     = "GCP US Central 1 edge location"

  gcp {
    project_id                   = "my-gcp-project"
    service_account_json_base64  = var.gcp_service_account_json_base64
    network_name                 = "default"
    subnet_name                  = "default"
    tags                         = ["castai", "omni"]
  }
}

# Example: Get edge configuration
data "castai_omni_edge_configuration" "default" {
  organization_id  = var.organization_id
  cluster_id       = var.cluster_id
  edge_location_id = castai_omni_edge_location.aws_us_east.id
  name             = "default"
}

# Example: Create an edge (compute node) with on-demand scheduling
resource "castai_omni_edge" "worker_1" {
  organization_id  = var.organization_id
  cluster_id       = var.cluster_id
  edge_location_id = castai_omni_edge_location.aws_us_east.id

  name              = "worker-1"
  instance_type     = "m5.xlarge"
  scheduling_type   = "ON_DEMAND"
  zone              = "us-east-1a"
  node_architecture = "X86_64"
  boot_disk_gib     = 100

  kubernetes_labels = {
    "environment" = "production"
    "workload"    = "general"
  }

  kubernetes_taints = [
    {
      key    = "dedicated"
      value  = "gpu-workloads"
      effect = "NoSchedule"
    }
  ]

  configuration_id = data.castai_omni_edge_configuration.default.id
}

# Example: Create a spot instance edge
resource "castai_omni_edge" "spot_worker" {
  organization_id  = var.organization_id
  cluster_id       = var.cluster_id
  edge_location_id = castai_omni_edge_location.aws_us_east.id

  name              = "spot-worker-1"
  instance_type     = "m5.2xlarge"
  scheduling_type   = "SPOT"
  zone              = "us-east-1b"
  node_architecture = "X86_64"
  boot_disk_gib     = 100

  kubernetes_labels = {
    "environment" = "production"
    "workload"    = "batch-processing"
    "spot"        = "true"
  }
}

# Example: Create an edge with GPU configuration
resource "castai_omni_edge" "gpu_worker" {
  organization_id  = var.organization_id
  cluster_id       = var.cluster_id
  edge_location_id = castai_omni_edge_location.aws_us_east.id

  name              = "gpu-worker-1"
  instance_type     = "p3.2xlarge"
  scheduling_type   = "ON_DEMAND"
  zone              = "us-east-1a"
  node_architecture = "X86_64"
  boot_disk_gib     = 200

  gpu_config {
    count = 1
    type  = "nvidia-tesla-v100"

    time_sharing {
      replicas = 4
    }
  }

  kubernetes_labels = {
    "environment" = "production"
    "workload"    = "ml-training"
    "gpu"         = "true"
  }
}

# Outputs
output "cluster_onboarding_script" {
  value       = castai_omni_cluster.example.onboarding_script
  description = "Script to onboard the cluster"
  sensitive   = true
}

output "edge_location_id" {
  value       = castai_omni_edge_location.aws_us_east.id
  description = "AWS edge location ID"
}

output "edge_location_state" {
  value       = castai_omni_edge_location.aws_us_east.state
  description = "Edge location state"
}

output "worker_kubernetes_name" {
  value       = castai_omni_edge.worker_1.kubernetes_name
  description = "Kubernetes node name for worker 1"
}

output "worker_provider_id" {
  value       = castai_omni_edge.worker_1.provider_id
  description = "Cloud provider instance ID for worker 1"
}
