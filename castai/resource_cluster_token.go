package castai

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	FieldClusterID    = "cluster_id"
	FieldClusterToken = "cluster_token"
)

// Deprecated.
func resourceClusterToken() *schema.Resource {
	return &schema.Resource{
		CreateContext: func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
			return diag.FromErr(errors.New("use castai_eks_cluster.cluster_token instead"))
		},
		ReadContext: func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
			return diag.FromErr(errors.New("use castai_eks_cluster.cluster_token instead"))
		},
		DeleteContext: func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
			return nil
		},
		DeprecationMessage: `Resource "cluster_token" will be deprecated in the next major release in favour of cluster resource attribute.`,

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
				Deprecated:  `Resource "cluster_token" will be deprecated in the next major release in favour of cluster resource attribute.`,
			},
		},
	}
}
