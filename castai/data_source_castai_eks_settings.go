package castai

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	EksSettingsFieldAccountId          = "account_id"
	EksSettingsFieldRegion             = "region"
	EksSettingsFieldVpc                = "vpc"
	EksSettingsFieldCluster            = "cluster"
	EksSettingsFieldIamPolicyJson      = "iam_policy_json"
	EksSettingsFieldIamUserPolicyJson  = "iam_user_policy_json"
	EksSettingsFieldIamManagedPolicies = "iam_managed_policies"
)

func dataSourceCastaiEksSettings() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCastaiEksSettingsRead,

		Schema: map[string]*schema.Schema{
			EksSettingsFieldAccountId: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			EksSettingsFieldRegion: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			EksSettingsFieldVpc: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			EksSettingsFieldCluster: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			EksSettingsFieldIamPolicyJson: {
				Type:     schema.TypeString,
				Computed: true,
			},
			EksSettingsFieldIamUserPolicyJson: {
				Type:     schema.TypeString,
				Computed: true,
			},
			EksSettingsFieldIamManagedPolicies: {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func dataSourceCastaiEksSettingsRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	accountID := data.Get(EksSettingsFieldAccountId).(string)
	vpc := data.Get(EksSettingsFieldVpc).(string)
	region := data.Get(EksSettingsFieldRegion).(string)
	cluster := data.Get(EksSettingsFieldCluster).(string)

	data.SetId(fmt.Sprintf("eks-%s-%s-%s-%s", accountID, vpc, region, cluster))
	data.Set(EksSettingsFieldIamPolicyJson, buildPolicyJSON(accountID))
	data.Set(EksSettingsFieldIamUserPolicyJson, buildUserPolicyJSON(accountID, vpc, region, cluster))
	data.Set(EksSettingsFieldIamManagedPolicies, buildManagedPolicies())
	return nil
}

func buildManagedPolicies() []string {
	return []string{
		"arn:aws:iam::aws:policy/AmazonEC2ReadOnlyAccess",
		"arn:aws:iam::aws:policy/AmazonEventBridgeReadOnlyAccess",
		"arn:aws:iam::aws:policy/IAMReadOnlyAccess",
		"arn:aws:iam::aws:policy/AWSLambda_ReadOnlyAccess",
	}
}

func buildPolicyJSON(accountID string) string {
	r := strings.NewReplacer(
		"${account_id}", accountID,
	)

	return fmt.Sprintf(r.Replace(
		`{
   "Version": "2012-10-17",
   "Statement": [
      {
         "Sid": "PassRoleEC2",
         "Action": "iam:PassRole",
         "Effect": "Allow",
         "Resource": "arn:aws:iam::*:role/*",
         "Condition": {
            "StringEquals": {
               "iam:PassedToService": "ec2.amazonaws.com"
            }
         }
      },
      {
         "Sid": "PassRoleLambda",
         "Action": "iam:PassRole",
         "Effect": "Allow",
         "Resource": "arn:aws:iam::*:role/*",
         "Condition": {
            "StringEquals": {
               "iam:PassedToService": "lambda.amazonaws.com"
            }
         }
      },
      {
         "Sid": "NonResourcePermissions",
         "Effect": "Allow",
         "Action": [
            "iam:CreateInstanceProfile",
            "iam:DeleteInstanceProfile",
            "iam:CreateRole",
            "iam:DeleteRole",
            "iam:AttachRolePolicy",
            "iam:DetachRolePolicy",
            "iam:AddRoleToInstanceProfile",
            "iam:RemoveRoleFromInstanceProfile",
            "iam:CreateServiceLinkedRole",
            "iam:DeleteServiceLinkedRole",
            "ec2:CreateSecurityGroup",
            "ec2:CreateKeyPair",
            "ec2:DeleteKeyPair",
            "ec2:CreateTags"
         ],
         "Resource": "*"
      },
      {
         "Sid": "TagOnLaunching",
         "Effect": "Allow",
         "Action": "ec2:CreateTags",
         "Resource": "arn:aws:ec2:*:${account_id}:instance/*",
         "Condition": {
            "StringEquals": {
               "ec2:CreateAction": "RunInstances"
            }
         }
      },
      {
         "Sid": "TagSecurityGroups",
         "Effect": "Allow",
         "Action": "ec2:CreateTags",
         "Resource": "arn:aws:ec2:*:${account_id}:security-group/*",
         "Condition": {
            "StringEquals": {
               "ec2:CreateAction": "CreateSecurityGroup"
            }
         }
      },
      {
         "Sid": "RunInstancesPermissions",
         "Effect": "Allow",
         "Action": "ec2:RunInstances",
         "Resource": [
            "arn:aws:ec2:*:${account_id}:network-interface/*",
            "arn:aws:ec2:*:${account_id}:security-group/*",
            "arn:aws:ec2:*:${account_id}:volume/*",
            "arn:aws:ec2:*:${account_id}:key-pair/*",
            "arn:aws:ec2:*::image/*"
         ]
      },
      {
         "Sid": "CreateLambdaFunctionRestriction",
         "Effect": "Allow",
         "Action": [
            "lambda:CreateFunction",
            "lambda:UpdateFunctionCode",
            "lambda:AddPermission",
            "lambda:DeleteFunction",
            "events:PutRule",
            "events:PutTargets",
            "events:DeleteRule",
            "events:RemoveTargets"
         ],
         "Resource": "*"
      }
   ]
}`))
}

func buildUserPolicyJSON(accountID, vpc, region, cluster string) string {
	r := strings.NewReplacer(
		"${account_id}", accountID,
		"${vpc}", vpc,
		"${region}", region,
		"${cluster}", cluster,
	)

	return fmt.Sprintf(r.Replace(
		`{
   "Version": "2012-10-17",
   "Statement": [
      {
         "Sid": "RunInstancesTagRestriction",
         "Effect": "Allow",
         "Action": "ec2:RunInstances",
         "Resource": "arn:aws:ec2:${region}:${account_id}:instance/*",
         "Condition": {
            "StringEquals": {
               "aws:RequestTag/kubernetes.io/cluster/${cluster}": "owned"
            }
         }
      },
      {
         "Sid": "RunInstancesVpcRestriction",
         "Effect": "Allow",
         "Action": "ec2:RunInstances",
         "Resource": "arn:aws:ec2:${region}:${account_id}:subnet/*",
         "Condition": {
            "StringEquals": {
               "ec2:Vpc": "arn:aws:ec2:${region}:${account_id}:vpc/${vpc}"
            }
         }
      },
      {
         "Sid": "InstanceActionsTagRestriction",
         "Effect": "Allow",
         "Action": [
            "ec2:TerminateInstances",
            "ec2:StartInstances",
            "ec2:StopInstances",
            "ec2:CreateTags"
         ],
         "Resource": "arn:aws:ec2:${region}:${account_id}:instance/*",
         "Condition": {
            "StringEquals": {
               "ec2:ResourceTag/kubernetes.io/cluster/${cluster}": [
                  "owned",
                  "shared"
               ]
            }
         }
      },
      {
         "Sid": "VpcRestrictedActions",
         "Effect": "Allow",
         "Action": [
            "ec2:RevokeSecurityGroupIngress",
            "ec2:RevokeSecurityGroupEgress",
            "ec2:AuthorizeSecurityGroupEgress",
            "ec2:AuthorizeSecurityGroupIngress",
            "ec2:DeleteSecurityGroup"
         ],
         "Resource": "*",
         "Condition": {
            "StringEquals": {
               "ec2:Vpc": "arn:aws:ec2:${region}:${account_id}:vpc/${cluster}"
            }
         }
      },
      {
         "Sid": "AutoscalingActionsTagRestriction",
         "Effect": "Allow",
         "Action": [
            "autoscaling:UpdateAutoScalingGroup",
            "autoscaling:DeleteAutoScalingGroup",
            "autoscaling:SuspendProcesses",
            "autoscaling:ResumeProcesses",
            "autoscaling:TerminateInstanceInAutoScalingGroup"
         ],
         "Resource": "arn:aws:autoscaling:${region}:${account_id}:autoScalingGroup:*:autoScalingGroupName/*",
         "Condition": {
            "StringEquals": {
               "autoscaling:ResourceTag/kubernetes.io/cluster/${cluster}": [
                  "owned",
                  "shared"
               ]
            }
         }
      },
      {
         "Sid": "EKS",
         "Effect": "Allow",
         "Action": [
            "eks:Describe*",
            "eks:List*",
            "eks:DeleteNodegroup",
            "eks:UpdateNodegroupConfig"
         ],
         "Resource": [
            "arn:aws:eks:${region}:${account_id}:cluster/${cluster}",
            "arn:aws:eks:${region}:${account_id}:nodegroup/${cluster}/*/*"
         ]
      }
   ]
}
`))
}
