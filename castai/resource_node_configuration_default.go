package castai

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/v7/castai/sdk"
)

func resourceNodeConfigurationDefault() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNodeConfigurationDefaultCreate,
		ReadContext:   resourceNodeConfigurationDefaultRead,
		DeleteContext: resourceNodeConfigurationDefaultDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldClusterID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "CAST AI cluster id",
			},
			"configuration_id": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "Id of the node configuration",
			},
		},
	}
}

func resourceNodeConfigurationDefaultCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)
	id := d.Get("configuration_id").(string)

	resp, err := client.NodeConfigurationAPISetDefaultWithResponse(ctx, clusterID, id)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	d.SetId(*resp.JSON200.Id)

	return resourceNodeConfigurationDefaultRead(ctx, d, meta)
}

func resourceNodeConfigurationDefaultRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)
	id := d.Get("configuration_id").(string)

	resp, err := client.NodeConfigurationAPIGetConfigurationWithResponse(ctx, clusterID, id)
	if err != nil {
		return diag.FromErr(err)
	}

	if !d.IsNewResource() && resp.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Node configuration (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	configID := resp.JSON200.Id
	if !*resp.JSON200.Default {
		// If configuration is no longer default, we should trigger state change.
		configID = nil
	}
	if err := d.Set("configuration_id", configID); err != nil {
		return diag.FromErr(fmt.Errorf("setting configuration id: %w", err))
	}

	return nil
}

func resourceNodeConfigurationDefaultDelete(ctx context.Context, d *schema.ResourceData, i interface{}) diag.Diagnostics {
	// Nothing to do.
	return nil
}
