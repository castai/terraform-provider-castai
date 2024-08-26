package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func resourceEKSClusterID() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEKSClusterIDCreate,
		ReadContext:   resourceEKSClusterIDRead,
		DeleteContext: resourceEKSClusterIDDelete,
		Description:   "Retrieve CAST AI clusterid",
		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"region": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"cluster_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
		},
	}
}

func resourceEKSClusterIDCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	req := sdk.ExternalClusterAPIRegisterClusterJSONRequestBody{
		Name: data.Get("cluster_name").(string),
	}

	req.Eks = &sdk.ExternalclusterV1EKSClusterParams{
		AccountId:   toPtr(data.Get("account_id").(string)),
		Region:      toPtr(data.Get("region").(string)),
		ClusterName: toPtr(data.Get("cluster_name").(string)),
	}

	resp, err := client.ExternalClusterAPIRegisterClusterWithResponse(ctx, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	clusterID := *resp.JSON200.Id
	data.SetId(clusterID)

	return nil
}

func resourceEKSClusterIDRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	if data.Id() == "" {
		return nil
	}

	resp, err := fetchClusterData(ctx, client, data.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if resp == nil {
		data.SetId("")
		return nil
	}

	if eks := resp.JSON200.Eks; eks != nil {
		if err := data.Set("account_id", toString(eks.AccountId)); err != nil {
			return diag.FromErr(fmt.Errorf("setting account id: %w", err))
		}
		if err := data.Set("region", toString(eks.Region)); err != nil {
			return diag.FromErr(fmt.Errorf("setting region: %w", err))
		}
		if err := data.Set("cluster_name", toString(eks.ClusterName)); err != nil {
			return diag.FromErr(fmt.Errorf("setting cluster name: %w", err))
		}
	}

	return nil
}

func resourceEKSClusterIDDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceCastaiClusterDelete(ctx, data, meta)
}
