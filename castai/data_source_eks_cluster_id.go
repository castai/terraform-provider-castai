package castai

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	EKSClusterIDFieldAccountId   = "account_id"
	EKSClusterIDFieldRegion      = "region"
	EKSClusterIDFieldClusterName = "cluster_name"
)

// Deprecated.
func dataSourceEKSClusterID() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: `Use castai_eks_clusterid resource instead`,
		ReadContext: func(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
			return diag.FromErr(errors.New("use castai_eks_clusterid resource instead"))
		},
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
