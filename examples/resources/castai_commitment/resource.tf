# AWS Reserved Instances.
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

# AWS Savings Plan.
resource "castai_commitment" "aws_savings_plan" {
  name               = "prod-sp-us-east-1"
  cloud              = "AWS"
  region             = "us-east-1"
  type               = "SAVINGS_PLAN"
  start_time         = "2026-01-01T00:00:00Z"
  end_time           = "2027-01-01T00:00:00Z"
  autoscaling_status = "ACTIVE"
  allowed_usage      = 1.0

  aws_savings_plan_details = {
    id                = "sp-abcdef01234567890"
    offering_id       = "12345678-1234-1234-1234-123456789012"
    type              = "Compute"
    state             = "active"
    region            = "us-east-1"
    commitment_amount = 5.0
    commitment_term   = "COMMITMENT_TERM_UNIT_ONE_YEAR"
    payment_option    = "NO_UPFRONT"
  }
}

# AWS Capacity Block for ML/AI reserved capacity.
resource "castai_commitment" "aws_capacity_block" {
  name       = "prod-capacityblock-us-east-1"
  cloud      = "AWS"
  region     = "us-east-1"
  type       = "CAPACITY_BLOCK"
  start_time = "2026-01-01T00:00:00Z"
  end_time   = "2026-01-08T00:00:00Z"

  aws_capacity_block_details = {
    id                       = "cr-abcdef01234567890"
    availability_zone        = "us-east-1a"
    instance_type            = "p5.48xlarge"
    instance_platform        = "Linux/UNIX"
    total_instance_count     = 2
    available_instance_count = 2
    state                    = "active"
  }
}

# AWS on-demand capacity reservation (ODCR).
resource "castai_commitment" "aws_odcr" {
  name               = "prod-odcr-us-east-1"
  cloud              = "AWS"
  region             = "us-east-1"
  type               = "ON_DEMAND_CAPACITY_RESERVATION"
  start_time         = "2026-01-01T00:00:00Z"
  autoscaling_status = "ACTIVE"
  allowed_usage      = 1.0

  aws_odcr_details = {
    id                       = "cr-abcdef01234567890"
    availability_zone        = "us-east-1a"
    instance_type            = "m5.xlarge"
    instance_platform        = "Linux/UNIX"
    tenancy                  = "default"
    total_instance_count     = 10
    available_instance_count = 10
    state                    = "active"
    end_date_type            = "unlimited"
    instance_match_criteria  = "open"
    interruptible            = false
  }
}

# Azure Reservation.
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

# Azure Savings Plan.
resource "castai_commitment" "azure_savings_plan" {
  name               = "prod-asp-eastus"
  cloud              = "AZURE"
  region             = "eastus"
  type               = "SAVINGS_PLAN"
  start_time         = "2026-01-01T00:00:00Z"
  end_time           = "2029-01-01T00:00:00Z"
  autoscaling_status = "ACTIVE"
  allowed_usage      = 1.0

  azure_savings_plan_details = {
    id                 = "abcdef01-2345-6789-abcd-ef0123456789"
    term               = "THREE_YEARS"
    provisioning_state = "Succeeded"
    scope              = "SHARED"
    commitment_amount  = 5.0
  }
}

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

# GCP Flex CUD (resource-based committed use discount).
resource "castai_commitment" "gcp_flex_cud" {
  name               = "prod-flex-cud-us-central1"
  cloud              = "GCP"
  region             = "us-central1"
  type               = "FLEX_CUD"
  start_time         = "2026-01-01T00:00:00Z"
  end_time           = "2027-01-01T00:00:00Z"
  autoscaling_status = "ACTIVE"
  allowed_usage      = 1.0

  gcp_flex_cud_details = {
    order_name        = "projects/my-project/locations/global/commitments/orders/12345"
    display_name      = "prod-flex-cud"
    line_item_id      = "li-67890"
    offer             = "6C1B7C20-6C2D-4A85-9C39-A1A7B7E5C3F9"
    region            = "us-central1"
    commitment_amount = 2.5
    state             = "ACTIVE"
    plan              = "TWELVE_MONTH"
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
    id                            = "my-reservation"
    self_link                     = "https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/reservations/my-reservation"
    project_id                    = "my-project"
    zone                          = "us-central1-a"
    instance_type                 = "a2-highgpu-1g"
    total_instance_count          = 4
    specific_reservation_required = true
    state                         = "READY"

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
