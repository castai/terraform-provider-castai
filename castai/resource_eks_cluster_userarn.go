package castai

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func resourceEKSClusterUserARN() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceEKSUserARNRead,
		CreateContext: resourceEKSUserARNCreate,
		DeleteContext: resourceEKSUserARNDelete,
		Description: "Retrieve EKS Cluster User ARN",
		Schema: map[string]*schema.Schema{
			EKSClusterUserARNFieldClusterID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			EKSClusterUserARNFieldARN: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceEKSUserARNRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := data.Get(EKSClusterUserARNFieldClusterID).(string)

	resp, err := client.ExternalClusterAPIGetAssumeRolePrincipalWithResponse(ctx, clusterID)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	arn := *resp.JSON200.Arn

	data.SetId(arn)
	if err := data.Set(EKSClusterUserARNFieldARN, arn); err != nil {
		return diag.FromErr(fmt.Errorf("setting user arn: %w", err))
	}

	return nil
}

func resourceEKSUserARNCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := resourceEKSUserARNRead(ctx, data, meta); err != nil {
		return err
	}

	arn, ok := data.Get(EKSClusterUserARNFieldARN).(string)
	if ok && arn != "" {
		log.Println("Using created arn for cross role user")
		return nil
	}

	client := meta.(*ProviderConfig).api

	clusterID := data.Get(EKSClusterUserARNFieldClusterID).(string)

	resp, err := client.ExternalClusterAPICreateAssumeRolePrincipalWithResponse(ctx, clusterID)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	arn = *resp.JSON200.Arn

	data.SetId(arn)
	if err := data.Set(EKSClusterUserARNFieldARN, arn); err != nil {
		return diag.FromErr(fmt.Errorf("setting user arn: %w", err))
	}

	return nil
}

func resourceEKSUserARNDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := data.Get(EKSClusterUserARNFieldClusterID).(string)

	resp, err := client.ExternalClusterAPIDeleteAssumeRolePrincipalWithResponse(ctx, clusterID)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	data.SetId("")

	return nil
}
