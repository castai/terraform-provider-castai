# GCP committed use discount (resource CUD).
resource "castai_commitment" "gcp_cud" {
  name               = "prod-cud-us-central1"
  cloud              = "GCP"
  region             = "us-central1"
  type               = "RESOURCE_CUD"
  start_time         = "2026-01-01T00:00:00Z"
  end_time           = "2027-01-01T00:00:00Z"
  autoscaling_status = "ACTIVE"
  allowed_usage      = 1.0

  gcp_resource_cud_details = {
    cud_id    = "123456789"
    plan      = "TWELVE_MONTH"
    type      = "GENERAL_PURPOSE_E2"
    cpu       = 32
    memory_mb = 131072
    status    = "ACTIVE"
  }
}

# AWS reserved instances.
resource "castai_commitment" "aws_ri" {
  name       = "prod-ri-us-east-1"
  cloud      = "AWS"
  region     = "us-east-1"
  type       = "RESERVED_INSTANCE"
  start_time = "2026-01-01T00:00:00Z"
  end_time   = "2027-01-01T00:00:00Z"

  aws_reserved_instances_details = {
    id             = "abcdef01-2345-6789-abcd-ef0123456789"
    scope          = "Region"
    instance_type  = "m5.xlarge"
    instance_count = 10
    state          = "active"
  }
}

# Azure reservation.
resource "castai_commitment" "azure_reservation" {
  name       = "prod-reservation-eastus"
  cloud      = "AZURE"
  region     = "eastus"
  type       = "RESERVED_INSTANCE"
  start_time = "2026-01-01T00:00:00Z"
  end_time   = "2029-01-01T00:00:00Z"

  azure_reservation_details = {
    id                   = "abcdef01-2345-6789-abcd-ef0123456789"
    plan                 = "THREE_YEAR"
    status               = "Succeeded"
    scope                = "Shared"
    instance_type        = "Standard_D4s_v3"
    count                = 5
    instance_flexibility = "ON"
  }
}

# GCP capacity reservation (on-demand).
resource "castai_commitment" "gcp_capacity_reservation" {
  name       = "gpu-reservation-us-central1"
  cloud      = "GCP"
  region     = "us-central1"
  type       = "ON_DEMAND_CAPACITY_RESERVATION"
  start_time = "2026-01-01T00:00:00Z"

  gcp_capacity_reservation_details = {
    id                   = "my-reservation"
    project_id           = "my-project"
    zone                 = "us-central1-a"
    instance_type        = "a2-highgpu-1g"
    total_instance_count = 4
    state                = "READY"

    accelerators = [
      {
        accelerator_type  = "https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/acceleratorTypes/nvidia-tesla-a100"
        accelerator_count = 1
      }
    ]
  }
}

# Bulk upload from a JSON file using for_each. Terraform parallelises the API
# calls (-parallelism, default 10); raise it for large fleets, e.g.
# terraform apply -parallelism=50
locals {
  cuds = jsondecode(file("${path.module}/cuds.json"))
}

resource "castai_commitment" "bulk_gcp_cuds" {
  for_each = { for cud in local.cuds : cud.cud_id => cud }

  name       = each.value.name
  cloud      = "GCP"
  region     = each.value.region
  type       = "RESOURCE_CUD"
  start_time = each.value.start_time
  end_time   = each.value.end_time

  gcp_resource_cud_details = {
    cud_id    = each.value.cud_id
    plan      = each.value.plan
    type      = each.value.type
    cpu       = each.value.cpu
    memory_mb = each.value.memory_mb
    status    = each.value.status
  }
}
