locals {
  resource_name_postfix = var.cluster_name
  account_id            = data.aws_caller_identity.current.account_id

  instance_profile_role_name = "castai-eks-${local.resource_name_postfix}-node-role"
  iam_role_name              = "castai-eks-${local.resource_name_postfix}-cluster-role"
  iam_inline_policy_name     = "CastEKSRestrictedAccess"
}

################################################################################
# Instance profile
################################################################################

# Create instance profile for CAST AI created nodes
resource "aws_iam_instance_profile" "castai_instance_profile" {
  name = local.instance_profile_role_name
  role = aws_iam_role.castai_instance_profile_role.name
}

# Create instance profile role
resource "aws_iam_role" "castai_instance_profile_role" {
  name               = local.instance_profile_role_name
  assume_role_policy = jsonencode({
    Version : "2012-10-17"
    Statement : [
      {
        "Sid"     = ""
        Effect    = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
        Action = ["sts:AssumeRole"]
      }
    ]
  })
}

# Attach policies to instance profile
resource "aws_iam_role_policy_attachment" "instance_profile_policy" {
  for_each = toset([
    "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
    "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
    "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
  ])

  role       = aws_iam_instance_profile.castai_instance_profile.role
  policy_arn = each.value
}

################################################################################
# Assume role
################################################################################

# Create role that will be assumed by CAST AI user
resource "aws_iam_role" "assume_role" {
  name               = local.iam_role_name
  assume_role_policy = jsonencode({
    Version   = "2012-10-17"
    Statement = [
      {
        Action    = "sts:AssumeRole"
        Effect    = "Allow"
        Principal = {
          AWS = data.castai_eks_user_arn.castai_user_arn.arn
        }
        Condition = {
          StringEquals = {
            "sts:ExternalId" = data.castai_eks_clusterid.castai_cluster_id.id
          }
        }
      },
    ]
  })
}

# Attach readonly policies to role
resource "aws_iam_role_policy_attachment" "assume_role_readonly_policy_attachment" {
  for_each = toset([
    "arn:aws:iam::aws:policy/AmazonEC2ReadOnlyAccess"
  ])
  role       = aws_iam_role.assume_role.name
  policy_arn = each.value
}

# Attach inline policy to role
resource "aws_iam_role_policy" "inline_role_policy" {
  name   = local.iam_inline_policy_name
  role   = aws_iam_role.assume_role.name
  policy = jsonencode({
    Version   = "2012-10-17"
    Statement = [
      # Needed to be able launch instance with instance profile
      {
        Sid       = "PassRoleEC2",
        Action    = "iam:PassRole",
        Effect    = "Allow",
        Resource  = "arn:aws:iam::${local.account_id}:role/${aws_iam_role.castai_instance_profile_role.name}",
        Condition = {
          StringEquals = {
            "iam:PassedToService" = "ec2.amazonaws.com"
          }
        }
      },
      # Needed to validate permissions
      {
        "Sid" : "IAMPermissions",
        "Effect" : "Allow",
        "Action" : [
          "iam:GetRole",
          "iam:SimulatePrincipalPolicy",
          "iam:GetInstanceProfile"
        ],
        "Resource" : [
          "arn:aws:iam::${local.account_id}:role/${aws_iam_role.assume_role.name}",
          "arn:aws:iam::${local.account_id}:instance-profile/${aws_iam_instance_profile.castai_instance_profile.name}"
        ]
      },
      # Needed to tag non CAST nodes
      {
        Sid    = "NonResourcePermissions",
        Effect = "Allow",
        Action = [
          "ec2:CreateTags",
        ],
        Resource = "*"
      },
      # Restrict run instance to cluster tag
      {
        Sid : "RunInstancesTagRestriction",
        Effect : "Allow",
        Action : "ec2:RunInstances",
        Resource : "arn:aws:ec2:${var.cluster_region}:${local.account_id}:instance/*",
        Condition : {
          StringEquals : {
            "aws:RequestTag/kubernetes.io/cluster/${var.cluster_name}" : "owned",
            "aws:RequestTag/cast:cluster-id" : data.castai_eks_clusterid.castai_cluster_id.id,
          }
        }
      },
      # Restrict run instance to specific vpc and subnets
      {
        Sid : "RunInstancesVpcRestriction",
        Effect : "Allow",
        Action : "ec2:RunInstances",
        Resource : formatlist("arn:aws:ec2:${var.cluster_region}:${local.account_id}:subnet/%s", module.vpc.private_subnets)
        Condition : {
          StringEquals : {
            "ec2:Vpc" : "arn:aws:ec2:${var.cluster_region}:${local.account_id}:vpc/${module.vpc.vpc_id}"
          }
        }
      },
      # Restrict instance actions to specific cluster tag
      {
        Sid : "InstanceActionsTagRestriction",
        Effect : "Allow",
        Action : [
          "ec2:TerminateInstances",
          "ec2:StartInstances",
          "ec2:StopInstances",
        ],
        Resource : "arn:aws:ec2:${var.cluster_region}:${local.account_id}:instance/*",
        Condition : {
          StringEquals : {
            "ec2:ResourceTag/kubernetes.io/cluster/${var.cluster_name}" : ["owned", "shared"],
            "ec2:ResourceTag/cast:cluster-id" : data.castai_eks_clusterid.castai_cluster_id.id,
          }
        }
      },
      # Needed to be able to run instance with provided resources
      {
        Sid      = "RunInstancesPermissions",
        Effect   = "Allow",
        Action   = "ec2:RunInstances",
        Resource = concat(
          formatlist("arn:aws:ec2:*:${local.account_id}:security-group/%s", [
            module.eks.cluster_security_group_id,
            module.eks.node_security_group_id,
            aws_security_group.additional.id,
          ]),
          [
            "arn:aws:ec2:*:${local.account_id}:network-interface/*",
            "arn:aws:ec2:*:${local.account_id}:volume/*",
            "arn:aws:ec2:*::image/*",
          ]
        ),
      },
      # Restrict scaling down autoscaling groups to specific cluster
      {
        Sid : "AutoscalingActionsTagRestriction",
        Effect : "Allow",
        Action : [
          "autoscaling:UpdateAutoScalingGroup",
          "autoscaling:TerminateInstanceInAutoScalingGroup",
          # Necessary for pause/resume
          "autoscaling:SuspendProcesses",
          "autoscaling:ResumeProcesses",
        ],
        Resource : "arn:aws:autoscaling:${var.cluster_region}:${local.account_id}:autoScalingGroup:*:autoScalingGroupName/*",
        Condition : {
          StringEquals : {
            "autoscaling:ResourceTag/kubernetes.io/cluster/${var.cluster_name}" : [
              "owned",
              "shared"
            ]
          }
        }
      },
      # Allow to read cluster related resources
      {
        Sid : "EKS",
        Effect : "Allow",
        Action : [
          "eks:Describe*",
          "eks:List*"
        ],
        Resource : [
          "arn:aws:eks:${var.cluster_region}:${local.account_id}:cluster/${var.cluster_name}",
          "arn:aws:eks:${var.cluster_region}:${local.account_id}:nodegroup/${var.cluster_name}/*/*"
        ]
      }
    ]
  })
}

