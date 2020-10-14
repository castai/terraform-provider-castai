package castai

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/terraform-providers/terraform-provider-castai/castai/sdk"
)

func dataSourceCastaiCluster() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCastaiClusterRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: toDiagFunc(validation.IsUUID),
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"region": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"credentials": {
				Type:     schema.TypeSet,
				Set:      schema.HashString,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"kubeconfig": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func dataSourceCastaiClusterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	id := data.Get("id").(string)

	response, err := client.GetClusterWithResponse(ctx, sdk.ClusterId(id))
	if checkErr := sdk.CheckGetResponse(response, err); checkErr != nil {
		return diag.Errorf("fetching cluster by id=%s: %v", id, checkErr)
	}

	log.Printf("[INFO] found cluster: %v", response.JSON200)

	data.SetId(response.JSON200.Id)
	data.Set("name", response.JSON200.Name)
	data.Set("status", response.JSON200.Status)
	data.Set("region", response.JSON200.Region.Name)
	data.Set("credentials", response.JSON200.CloudCredentialsIDs)

	kubeconfigResponse, err := client.GetClusterKubeconfigWithResponse(ctx, sdk.ClusterId(id))
	if checkErr := sdk.CheckGetResponse(kubeconfigResponse, err); checkErr != nil {
		return diag.Errorf("fetching kubeconfig for cluster %q: %v", id, checkErr)
	}

	log.Printf("[INFO] found cluster kubeconfig, id=%s", id)

	data.Set("kubeconfig", string(kubeconfigResponse.Body))

	return nil
}
