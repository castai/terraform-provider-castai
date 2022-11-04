package castai

import (
	"context"
	"fmt"

	"github.com/castai/terraform-provider-castai/castai/policies"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	EKSSettingsFieldAccountId               = "account_id"
	EKSSettingsFieldRegion                  = "region"
	EKSSettingsFieldVpc                     = "vpc"
	EKSSettingsFieldCluster                 = "cluster"
	EKSSettingsFieldIamPolicyJson           = "iam_policy_json"
	EKSSettingsFieldIamUserPolicyJson       = "iam_user_policy_json"
	EKSSettingsFieldIamManagedPolicies      = "iam_managed_policies"
	EKSSettingsFieldInstanceProfilePolicies = "instance_profile_policies"
)

func dataSourceEKSSettings() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCastaiEKSSettingsRead,

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
			EKSSettingsFieldInstanceProfilePolicies: {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func dataSourceCastaiEKSSettingsRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	accountID := data.Get(EKSSettingsFieldAccountId).(string)
	vpc := data.Get(EKSSettingsFieldVpc).(string)
	region := data.Get(EKSSettingsFieldRegion).(string)
	cluster := data.Get(EKSSettingsFieldCluster).(string)

	arn := fmt.Sprintf("%s:%s", region, accountID)

	userPolicy, _ := policies.GetUserInlinePolicy(cluster, arn, vpc)
	iamPolicy, _ := policies.GetIAMPolicy(accountID)
	instanceProfilePolicy := policies.GetInstanceProfilePolicy()

	data.SetId(fmt.Sprintf("eks-%s-%s-%s-%s", accountID, vpc, region, cluster))
	data.Set(EKSSettingsFieldIamPolicyJson, iamPolicy)
	data.Set(EKSSettingsFieldIamUserPolicyJson, userPolicy)
	data.Set(EKSSettingsFieldIamManagedPolicies, buildManagedPolicies())
	data.Set(EKSSettingsFieldInstanceProfilePolicies, instanceProfilePolicy)

	return nil
}

func buildManagedPolicies() []string {
	return []string{
		"arn:aws:iam::aws:policy/AmazonEC2ReadOnlyAccess",
		"arn:aws:iam::aws:policy/IAMReadOnlyAccess",
	}
}
