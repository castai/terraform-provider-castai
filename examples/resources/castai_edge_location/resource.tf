# AWS Edge Location
resource "castai_edge_location" "aws_example" {
  organization_id = "your-org-id"
  cluster_id      = castai_omni_cluster.example.id
  name            = "aws-edge-us-east"
  description     = "AWS edge location in us-east-1"
  region          = "us-east-1"

  zones = [
    {
      id   = "us-east-1a"
      name = "us-east-1a"
    },
    {
      id   = "us-east-1b"
      name = "us-east-1b"
    }
  ]

  aws = {
    account_id           = "123456789012"
    access_key_id_wo     = "AKIAIOSFODNN7EXAMPLE"
    secret_access_key_wo = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
    vpc_id               = "vpc-12345678"
    security_group_id    = "sg-12345678"
    subnet_ids = {
      "us-east-1a" = "subnet-12345678"
      "us-east-1b" = "subnet-87654321"
    }
    name_tag = "castai-edge-location"
  }
}
