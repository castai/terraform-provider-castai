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
)

const (
	FieldOmniClusterOrganizationID = "organization_id"
	FieldOmniClusterID             = "cluster_id"
)

func resourceOmniCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceOmniClusterCreate,
		ReadContext:   resourceOmniClusterRead,
		DeleteContext: resourceOmniClusterDelete,
		Description:   `Omni cluster resource allows registering a cluster with CAST AI Omni provider.`,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldOmniClusterOrganizationID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "CAST AI organization ID",
			},
			FieldOmniClusterID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "CAST AI cluster ID to register",
			},
		},
	}
}

func resourceOmniClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).OmniAPI

	organizationID := d.Get(FieldOmniClusterOrganizationID).(string)
	clusterID := d.Get(FieldOmniClusterID).(string)

	resp, err := client.ClustersAPIRegisterClusterWithResponse(ctx, organizationID, clusterID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("registering omni cluster: %w", err))
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("unexpected status code registering omni cluster: %d, body: %s", resp.StatusCode(), string(resp.Body)))
	}

	d.SetId(clusterID)

	return resourceOmniClusterRead(ctx, d, meta)
}

func resourceOmniClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).OmniAPI

	organizationID := d.Get(FieldOmniClusterOrganizationID).(string)
	clusterID := d.Id()

	resp, err := client.ClustersAPIGetClusterWithResponse(ctx, organizationID, clusterID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("getting omni cluster: %w", err))
	}

	if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Omni cluster %s not found, removing from state", clusterID)
		d.SetId("")
		return nil
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("unexpected status code getting omni cluster: %d, body: %s", resp.StatusCode(), string(resp.Body)))
	}

	if err := d.Set(FieldOmniClusterID, resp.JSON200.Id); err != nil {
		return diag.FromErr(fmt.Errorf("setting cluster_id: %w", err))
	}

	return nil
}

func resourceOmniClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Omni cluster %s deleted (registration cannot be undone)", d.Id())
	return nil
}
