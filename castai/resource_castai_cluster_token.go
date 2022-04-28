package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	FieldClusterID = "cluster_id"
	FieldClusterToken = "cluster_token"
)

func resourceCastaiClusterToken() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiClusterTokenCreate,
		ReadContext: resourceCastaiClusterTokenRead,
		UpdateContext: nil,
		DeleteContext: resourceCastaiClusterTokenDelete,
		Schema: map[string]*schema.Schema{
			FieldClusterID: {
				Type: schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			FieldClusterToken: {
				Type: schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func resourceCastaiClusterTokenRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if _, ok := data.GetOk(FieldClusterToken); !ok {
		return diag.Errorf("Cluster token is not created")
	}

	return nil
}

func resourceCastaiClusterTokenCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := data.Get(FieldClusterID).(string)
	tkn, err := createClusterToken(ctx, client, clusterID)
	if err != nil {
		return diag.FromErr(err)
	}

	data.Set(FieldClusterToken, tkn)
	data.SetId(fmt.Sprintf("%s-cluster-token", clusterID))
	return nil
}

func resourceCastaiClusterTokenDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	data.SetId("")
	return nil
}
