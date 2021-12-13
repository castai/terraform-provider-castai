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
	EksSettingsFieldAccountId                = "account_id"
	EksSettingsFieldRegion                   = "region"
	EksSettingsFieldVpc                      = "vpc"
	EksSettingsFieldCluster                  = "cluster"
	EksSettingsFieldIamPolicyJson            = "iam_policy_json"
	EksSettingsFieldIamUserPolicyJson        = "iam_user_policy_json"
	EksSettingsFieldIamManagedPolicies       = "iam_managed_policies"
	EksSettingsFieldLambdaPolicies          = "lambda_policies"
	EksSettingsFieldInstanceProfilePolicies = "instance_profile_policies"
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
			EksSettingsFieldLambdaPolicies: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{Type:schema.TypeString},
				Computed: true,
			},
			EksSettingsFieldInstanceProfilePolicies: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{Type:schema.TypeString},
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

	arn := fmt.Sprintf("%s:%s", region, accountID)

	userPolicy, _ := policies.GetUserInlinePolicy(cluster, arn, vpc)
	iamPolicy, _ := policies.GetIAMPolicy(accountID)
	lambdaPolicy := policies.GetLambdaPolicy()
	instanceProfilePolicy := policies.GetInstanceProfilePolicy()

	data.SetId(fmt.Sprintf("eks-%s-%s-%s-%s", accountID, vpc, region, cluster))
	data.Set(EksSettingsFieldIamPolicyJson, iamPolicy)
	data.Set(EksSettingsFieldIamUserPolicyJson, userPolicy)
	data.Set(EksSettingsFieldIamManagedPolicies, buildManagedPolicies())
	data.Set(EksSettingsFieldLambdaPolicies, lambdaPolicy)
	data.Set(EksSettingsFieldInstanceProfilePolicies, instanceProfilePolicy)

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
