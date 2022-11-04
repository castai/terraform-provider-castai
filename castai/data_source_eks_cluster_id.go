package castai

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	EKSClusterIDFieldAccountId   = "account_id"
	EKSClusterIDFieldRegion      = "region"
	EKSClusterIDFieldClusterName = "cluster_name"
)

func dataSourceEKSClusterID() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCastaiEKSClusterIDRead,
		Schema: map[string]*schema.Schema{
			EKSClusterIDFieldAccountId: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			EKSClusterIDFieldRegion: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			EKSClusterIDFieldClusterName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
		},
	}
}

// like in Agent startup - we do RegisterCluster
func dataSourceCastaiEKSClusterIDRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Terraform SDK Data source has no access to state
	client := meta.(*ProviderConfig).api

	req := sdk.ExternalClusterAPIRegisterClusterJSONRequestBody{
		Name: data.Get(EKSClusterIDFieldClusterName).(string),
	}

	req.Eks = &sdk.ExternalclusterV1EKSClusterParams{
		AccountId:   toStringPtr(data.Get(EKSClusterIDFieldAccountId).(string)),
		Region:      toStringPtr(data.Get(EKSClusterIDFieldRegion).(string)),
		ClusterName: toStringPtr(data.Get(EKSClusterIDFieldClusterName).(string)),
	}

	resp, err := client.ExternalClusterAPIRegisterClusterWithResponse(ctx, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	clusterID := *resp.JSON200.Id
	data.SetId(clusterID)

	return nil
}
