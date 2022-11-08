package castai

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceClusterToken() *schema.Resource {
	return &schema.Resource{
		CreateContext:      resourceCastaiClusterTokenCreate,
		ReadContext:        resourceCastaiClusterTokenRead,
		UpdateContext:      nil,
		DeleteContext:      resourceCastaiClusterTokenDelete,
		DeprecationMessage: `Resource "cluster_token" is deprecated in favour of cluster resource attribute.`,

		Schema: map[string]*schema.Schema{
			FieldClusterID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "CAST AI cluster id",
			},
			FieldClusterToken: {
				Type:        schema.TypeString,
				Description: "computed value to store cluster token",
				Computed:    true,
				Sensitive:   true,
				Deprecated:  `Resource "cluster_token" is deprecated in favour of cluster resource attribute.`,
			},
		},
	}
}

func resourceCastaiClusterTokenRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return diag.Errorf(`Resource "cluster_token" is deprecated in favour of cluster resource attribute.`)
}

func resourceCastaiClusterTokenCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return diag.Errorf(`Resource "cluster_token" is deprecated in favour of cluster resource attribute.`)
}

func resourceCastaiClusterTokenDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	data.SetId("")
	return nil
}
