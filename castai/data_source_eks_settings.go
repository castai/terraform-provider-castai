package castai

import (
	"context"
	"fmt"
	"strings"

	"github.com/castai/terraform-provider-castai/castai/policies"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	EKSSettingsFieldAccountId             = "account_id"
	EKSSettingsFieldRegion                = "region"
	EKSSettingsFieldVpc                   = "vpc"
	EKSSettingsFieldCluster               = "cluster"
	EKSSettingsFieldIamPolicyJson         = "iam_policy_json"
	EKSSettingsFieldIamUserPolicyJson     = "iam_user_policy_json"
	EKSSettingsFieldIamManagedPolicies    = "iam_managed_policies"
	EKSSettingsFieldAWSSharedVPCAccountId = "aws_shared_vpc_account_id"

	GovCloudPrefix = "us-gov"
)

func dataSourceEKSSettings() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCastaiEKSSettingsRead,
		Description: "Retrieve IAM policy, IAM User Policy and instance profile policies for the specified cluster",
		Schema: map[string]*schema.Schema{
			EKSSettingsFieldAccountId: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			EKSSettingsFieldRegion: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			EKSSettingsFieldVpc: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			EKSSettingsFieldCluster: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			EKSSettingsFieldIamPolicyJson: {
				Type:     schema.TypeString,
				Computed: true,
			},
			EKSSettingsFieldIamUserPolicyJson: {
				Type:     schema.TypeString,
				Computed: true,
			},
			EKSSettingsFieldIamManagedPolicies: {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			EKSSettingsFieldAWSSharedVPCAccountId: {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Description:      "AWS account ID where the VPC and subnets are located, for shared VPC setups. If not provided, defaults to the account_id.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
		},
	}
}

func dataSourceCastaiEKSSettingsRead(ctx context.Context, data *schema.ResourceData, _ interface{}) diag.Diagnostics {
	accountID := data.Get(EKSSettingsFieldAccountId).(string)
	vpc := data.Get(EKSSettingsFieldVpc).(string)
	region := data.Get(EKSSettingsFieldRegion).(string)
	cluster := data.Get(EKSSettingsFieldCluster).(string)
	sharedVPCAccountID := data.Get(EKSSettingsFieldAWSSharedVPCAccountId).(string)

	arn := fmt.Sprintf("%s:%s", region, accountID)
	partition := getPartition(region)

	var sharedVPCArn string
	if sharedVPCAccountID != "" {
		sharedVPCArn = fmt.Sprintf("%s:%s", region, sharedVPCAccountID)
	}

	userPolicy, _ := policies.GetUserInlinePolicy(cluster, arn, vpc, partition, sharedVPCArn)
	iamPolicy, _ := policies.GetIAMPolicy(accountID, partition)
	managedPolicies := policies.GetManagedPolicies(partition)

	data.SetId(fmt.Sprintf("eks-%s-%s-%s-%s", accountID, vpc, region, cluster))
	if err := data.Set(EKSSettingsFieldIamPolicyJson, iamPolicy); err != nil {
		return diag.FromErr(fmt.Errorf("setting iam policy: %w", err))
	}
	if err := data.Set(EKSSettingsFieldIamUserPolicyJson, userPolicy); err != nil {
		return diag.FromErr(fmt.Errorf("setting iam user policy: %w", err))
	}
	if err := data.Set(EKSSettingsFieldIamManagedPolicies, managedPolicies); err != nil {
		return diag.FromErr(fmt.Errorf("setting iam managed policies: %w", err))
	}

	return nil
}

func getPartition(region string) string {
	switch {
	case strings.Contains(region, GovCloudPrefix):
		return "aws-us-gov"
	default:
		return "aws"
	}
}
