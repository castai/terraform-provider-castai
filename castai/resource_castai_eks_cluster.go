package castai

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldEKSClusterName                   = "name"
	FieldEKSClusterAccountId              = "account_id"
	FieldEKSClusterRegion                 = "region"
	FieldEKSClusterAccessKeyId            = "access_key_id"
	FieldEKSClusterSecretAccessKey        = "secret_access_key"
	FieldEKSClusterAssumeRoleArn          = "assume_role_arn"
	FieldEKSClusterInstanceProfileArn     = "instance_profile_arn"
	FieldEKSClusterSecurityGroups         = "security_groups"
	FieldEKSClusterOverrideSecurityGroups = "override_security_groups"
	FieldEKSClusterSubnets                = "subnets"
	FieldEKSClusterTags                   = "tags"
	FieldEKSClusterDNSClusterIP           = "dns_cluster_ip"
)

func resourceCastaiEKSCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiEKSClusterCreate,
		ReadContext:   resourceCastaiEKSClusterRead,
		UpdateContext: resourceCastaiEKSClusterUpdate,
		DeleteContext: resourceCastaiPublicCloudClusterDelete,
		Description:   "EKS cluster resource allows connecting an existing EKS cluster to CAST AI.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(2 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldEKSClusterName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "name of your EKS cluster",
			},
			FieldEKSClusterAccountId: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "ID of AWS account",
			},
			FieldEKSClusterRegion: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "AWS region where the cluster is placed",
			},
			FieldEKSClusterAccessKeyId: {
				Type:        schema.TypeString,
				Sensitive:   true,
				Optional:    true,
				Description: "AWS access key ID of the CAST AI IAM account",
			},
			FieldEKSClusterSecretAccessKey: {
				Type:        schema.TypeString,
				Sensitive:   true,
				Optional:    true,
				Description: "AWS secret access key of the CAST AI IAM account",
			},
			FieldEKSClusterAssumeRoleArn: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "AWS ARN for assume role that should be used instead of IAM account",
			},
			FieldEKSClusterInstanceProfileArn: {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "AWS ARN of the instance profile to be used by CAST AI",
			},
			FieldClusterCredentialsId: {
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "CAST AI internal credentials ID",
			},
			FieldClusterAgentToken: {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "agent_token is deprecated, use cluster_token instead",
				Sensitive:  true,
			},
			FieldEKSClusterOverrideSecurityGroups: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Optional custom security groups for the cluster. If not set security groups from the EKS cluster configuration are used.",
			},
			FieldEKSClusterSecurityGroups: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "IDs of security groups that are used by CAST AI",
			},
			FieldEKSClusterSubnets: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Custom subnets for the cluster. If not set subnets from the EKS cluster configuration are used.",
			},
			FieldEKSClusterDNSClusterIP: {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsIPv4Address),
				Description:      "Overrides the IP address to use for DNS queries within the cluster. Defaults to 10.100.0.10 or 172.20.0.10 based on the IP address of the primary interface",
			},
			FieldClusterSSHPublicKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Accepted values are base64 encoded SSH public key or AWS key pair ID.",
			},
			FieldEKSClusterTags: {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Tags which should be added to CAST AI nodes",
			},
			FieldDeleteNodesOnDisconnect: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Should CAST AI remove nodes managed by CAST AI on disconnect",
			},
		},
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, i interface{}) error {
			_, accessKeyIdProvided := diff.GetOk(FieldEKSClusterAccessKeyId)
			_, secretAccessKeyProvided := diff.GetOk(FieldEKSClusterSecretAccessKey)

			if accessKeyIdProvided != secretAccessKeyProvided {
				return fmt.Errorf("when used `%s` and `%s` must be both specified", FieldEKSClusterAccessKeyId, FieldEKSClusterSecretAccessKey)
			}

			return nil
		},
	}
}

func resourceCastaiEKSClusterCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	req := sdk.ExternalClusterAPIRegisterClusterJSONRequestBody{
		Name: data.Get(FieldEKSClusterName).(string),
	}

	req.Eks = &sdk.ExternalclusterV1EKSClusterParams{
		AccountId:   toStringPtr(data.Get(FieldEKSClusterAccountId).(string)),
		Region:      toStringPtr(data.Get(FieldEKSClusterRegion).(string)),
		ClusterName: toStringPtr(data.Get(FieldEKSClusterName).(string)),
	}

	log.Printf("[INFO] Registering new external cluster: %#v", req)

	resp, err := client.ExternalClusterAPIRegisterClusterWithResponse(ctx, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	clusterID := *resp.JSON200.Id
	data.SetId(clusterID)

	if err := updateClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}

	return resourceCastaiEKSClusterRead(ctx, data, meta)
}

func resourceCastaiEKSClusterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if data.Id() == "" {
		log.Printf("[INFO] id is null not fetching anything.")
		return nil
	}

	log.Printf("[INFO] Getting cluster information.")

	resp, err := fetchClusterData(ctx, client, data.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if resp == nil {
		data.SetId("")
		return nil
	}

	data.Set(FieldClusterCredentialsId, *resp.JSON200.CredentialsId)

	if eks := resp.JSON200.Eks; eks != nil {
		data.Set(FieldEKSClusterAccountId, toString(eks.AccountId))
		data.Set(FieldEKSClusterRegion, toString(eks.Region))
		data.Set(FieldEKSClusterName, toString(eks.ClusterName))
		data.Set(FieldEKSClusterInstanceProfileArn, toString(eks.InstanceProfileArn))
		data.Set(FieldEKSClusterSubnets, toStringSlice(eks.Subnets))
		if v := toString(eks.DnsClusterIp); v != "" {
			data.Set(FieldEKSClusterDNSClusterIP, v)
		}
		if v := toString(resp.JSON200.SshPublicKey); v != "" {
			data.Set(FieldClusterSSHPublicKey, v)
		}
		data.Set(FieldEKSClusterSecurityGroups, toStringSlice(eks.SecurityGroups))
		if eks.Tags != nil {
			data.Set(FieldEKSClusterTags, eks.Tags.AdditionalProperties)
		}
	}

	if _, ok := data.GetOk(FieldClusterAgentToken); !ok {
		tkn, err := retrieveAgentToken(ctx, client)
		if err != nil {
			return diag.FromErr(err)
		}
		data.Set(FieldClusterAgentToken, tkn)
	}

	return nil
}

func resourceCastaiEKSClusterUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if err := updateClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}

	return resourceCastaiEKSClusterRead(ctx, data, meta)
}

func updateClusterSettings(ctx context.Context, data *schema.ResourceData, client *sdk.ClientWithResponses) error {
	if !data.HasChanges(
		FieldEKSClusterAccessKeyId,
		FieldEKSClusterSecretAccessKey,
		FieldEKSClusterAssumeRoleArn,
		FieldEKSClusterInstanceProfileArn,
		FieldEKSClusterSubnets,
		FieldEKSClusterDNSClusterIP,
		FieldEKSClusterOverrideSecurityGroups,
		FieldEKSClusterTags,
		FieldClusterCredentialsId,
		FieldClusterSSHPublicKey,
	) {
		log.Printf("[INFO] Nothing to update in cluster setttings.")
		return nil
	}

	log.Printf("[INFO] Updating cluster settings.")

	req := sdk.ExternalClusterAPIUpdateClusterJSONRequestBody{
		Eks: &sdk.ExternalclusterV1UpdateEKSClusterParams{},
	}

	accessKeyId, accessKeyIdProvided := data.GetOk(FieldEKSClusterAccessKeyId)
	secretAccessKey, secretAccessKeyProvided := data.GetOk(FieldEKSClusterSecretAccessKey)
	assumeRoleARN, assumeRoleProvided := data.GetOk(FieldEKSClusterAssumeRoleArn)

	if accessKeyIdProvided && secretAccessKeyProvided && assumeRoleProvided {
		return fmt.Errorf("specify either the access key ID and secret access key pair or AssumeRole ARN")
	}

	if accessKeyIdProvided && secretAccessKeyProvided {
		credentials, err := sdk.ToCloudCredentialsAWS(accessKeyId.(string), secretAccessKey.(string))
		if err != nil {
			return fmt.Errorf("marshaling credentials for cluster access: %w", err)
		}

		req.Credentials = &credentials
	}

	if assumeRoleProvided {
		req.Eks.AssumeRoleArn = toStringPtr(assumeRoleARN.(string))
	}

	if arn, ok := data.GetOk(FieldEKSClusterInstanceProfileArn); ok {
		req.Eks.InstanceProfileArn = toStringPtr(arn.(string))
	}

	if s, ok := data.GetOk(FieldEKSClusterOverrideSecurityGroups); ok {
		sgsRaw, ok := s.([]interface{})
		if ok {
			securityGroups := make([]string, len(sgsRaw))
			for idx, group := range sgsRaw {
				securityGroups[idx] = group.(string)
			}
			req.Eks.SecurityGroups = &securityGroups
		}
	}

	if s, ok := data.GetOk(FieldEKSClusterSubnets); ok {
		subnetsRaw := s.([]interface{})
		subnetsString := make([]string, len(subnetsRaw))

		for idx, subnet := range subnetsRaw {
			subnetsString[idx] = subnet.(string)
		}
		req.Eks.Subnets = &subnetsString
	}

	if s, ok := data.GetOk(FieldEKSClusterDNSClusterIP); ok {
		req.Eks.DnsClusterIp = toStringPtr(s.(string))
	}

	if s, ok := data.GetOk(FieldClusterSSHPublicKey); ok {
		req.SshPublicKey = toStringPtr(s.(string))
	}

	if tags, ok := data.GetOk(FieldEKSClusterTags); ok {
		tagsRaw := tags.(map[string]interface{})
		tagsString := make(map[string]string, len(tagsRaw))

		for k, v := range tagsRaw {
			tagsString[k] = v.(string)
		}
		req.Eks.Tags = &sdk.ExternalclusterV1UpdateEKSClusterParams_Tags{
			AdditionalProperties: tagsString,
		}
	}

	if err := backoff.Retry(func() error {
		response, err := client.ExternalClusterAPIUpdateClusterWithResponse(ctx, data.Id(), req)
		return sdk.CheckOKResponse(response, err)
	}, backoff.NewExponentialBackOff()); err != nil {
		return fmt.Errorf("updating cluster configuration: %w", err)
	}

	return nil
}

// Deprecated. Remove with agent_token.
func retrieveAgentToken(ctx context.Context, client *sdk.ClientWithResponses) (string, error) {
	response, err := client.GetAgentInstallScriptWithResponse(ctx, &sdk.GetAgentInstallScriptParams{})
	if err != nil {
		return "", fmt.Errorf("retrieving agent install script: %w", err)
	}

	// at the moment, agent registration token only appears in `curl agent manifests` snippet and is extracted from there.
	return strings.Split(strings.TrimPrefix(string(response.Body), `curl -H "Authorization: Token `), `"`)[0], nil
}

func createClusterToken(ctx context.Context, client *sdk.ClientWithResponses, clusterID string) (string, error) {
	resp, err := client.ExternalClusterAPICreateClusterTokenWithResponse(ctx, clusterID)
	if err != nil {
		return "", fmt.Errorf("creating cluster token: %w", err)
	}

	return *resp.JSON200.Token, nil
}

func getOptionalBool(data *schema.ResourceData, field string, defaultValue bool) *bool {
	del, ok := data.GetOk(field)
	if ok {
		deleteNodes := del.(bool)
		return &deleteNodes
	}
	return &defaultValue
}
