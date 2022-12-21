package castai

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldEKSClusterName          = "name"
	FieldEKSClusterAccountId     = "account_id"
	FieldEKSClusterRegion        = "region"
	FieldEKSClusterAssumeRoleArn = "assume_role_arn"
)

func resourceEKSCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiEKSClusterCreate,
		ReadContext:   resourceCastaiEKSClusterRead,
		UpdateContext: resourceCastaiEKSClusterUpdate,
		DeleteContext: resourceCastaiPublicCloudClusterDelete,
		Description:   "EKS cluster resource allows connecting an existing EKS cluster to CAST AI.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
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
			FieldEKSClusterAssumeRoleArn: {
				Type:     schema.TypeString,
				Optional: true,
				Description: "AWS IAM role ARN that will be assumed by CAST AI user. " +
					"This role should allow `sts:AssumeRole` action for CAST AI user that can be retrieved using `castai_eks_user_arn` data source",
			},
			FieldClusterToken: {
				Type:        schema.TypeString,
				Description: "computed value to store cluster token",
				Computed:    true,
				Sensitive:   true,
			},
			FieldClusterCredentialsId: {
				Type:        schema.TypeString,
				Computed:    true,
=				Description: "CAST AI internal credentials ID",
			},
			FieldDeleteNodesOnDisconnect: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Should CAST AI remove nodes managed by CAST AI on disconnect",
			},
		},
	}
}

func resourceCastaiEKSClusterCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	req := sdk.ExternalClusterAPIRegisterClusterJSONRequestBody{
		Name: data.Get(FieldEKSClusterName).(string),
	}

	req.Eks = &sdk.ExternalclusterV1EKSClusterParams{
		AccountId:   toPtr(data.Get(FieldEKSClusterAccountId).(string)),
		Region:      toPtr(data.Get(FieldEKSClusterRegion).(string)),
		ClusterName: toPtr(data.Get(FieldEKSClusterName).(string)),
	}

	log.Printf("[INFO] Registering new external cluster: %#v", req)

	resp, err := client.ExternalClusterAPIRegisterClusterWithResponse(ctx, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	clusterID := *resp.JSON200.Id
	tkn, err := createClusterToken(ctx, client, clusterID)
	if err != nil {
		return diag.FromErr(err)
	}
	data.SetId(clusterID)
	data.Set(FieldClusterToken, tkn)

	if err := updateClusterSettings(ctx, data, client); err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Cluster with id %q has been registered, don't forget to install castai-agent helm chart", data.Id())

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
		data.Set(FieldEKSClusterAssumeRoleArn, toString(eks.AssumeRoleArn))
	}
	clusterID := *resp.JSON200.Id

	if _, ok := data.GetOk(FieldClusterToken); !ok {
		tkn, err := createClusterToken(ctx, client, clusterID)
		if err != nil {
			return diag.FromErr(err)
		}
		data.Set(FieldClusterToken, tkn)
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
		FieldEKSClusterAssumeRoleArn,
		FieldClusterCredentialsId,
	) {
		log.Printf("[INFO] Nothing to update in cluster setttings.")
		return nil
	}

	log.Printf("[INFO] Updating cluster settings.")

	req := sdk.ExternalClusterAPIUpdateClusterJSONRequestBody{
		Eks: &sdk.ExternalclusterV1UpdateEKSClusterParams{},
	}

	assumeRoleARN, assumeRoleProvided := data.GetOk(FieldEKSClusterAssumeRoleArn)
	if assumeRoleProvided {
		req.Eks.AssumeRoleArn = toPtr(assumeRoleARN.(string))
	}

	if err := backoff.Retry(func() error {
		response, err := client.ExternalClusterAPIUpdateClusterWithResponse(ctx, data.Id(), req)
		if err != nil {
			return err
		}
		err = sdk.StatusOk(response)
		// In case of malformed user request return error to user right away.
		if response.StatusCode() == 400 {
			return backoff.Permanent(err)
		}
		return err
	}, backoff.NewExponentialBackOff()); err != nil {
		return fmt.Errorf("updating cluster configuration: %w", err)
	}

	return nil
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
