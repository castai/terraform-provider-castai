package castai

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni_provisioner"
)

const (
	FieldOmniClusterName              = "name"
	FieldOmniClusterOrganizationID    = "organization_id"
	FieldOmniClusterServiceAccountID  = "service_account_id"
	FieldOmniClusterProviderType      = "provider_type"
	FieldOmniClusterState             = "state"
	FieldOmniClusterStatus            = "status"
	FieldOmniClusterOnboardingScript  = "onboarding_script"
)

func resourceOmniCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceOmniClusterOnboard,
		ReadContext:   resourceOmniClusterRead,
		UpdateContext: resourceOmniClusterUpdate,
		DeleteContext: resourceOmniClusterDelete,

		Schema: map[string]*schema.Schema{
			FieldOmniClusterName: {
				Type:        schema.TypeString,
				Description: "Name of the Omni cluster",
				Required:    true,
			},
			FieldOmniClusterOrganizationID: {
				Type:        schema.TypeString,
				Description: "Organization ID",
				Required:    true,
				ForceNew:    true,
			},
			FieldOmniClusterServiceAccountID: {
				Type:        schema.TypeString,
				Description: "Service account ID for cluster authentication",
				Optional:    true,
			},
			FieldOmniClusterProviderType: {
				Type:        schema.TypeString,
				Description: "Cloud provider type (GKE, EKS, etc.)",
				Computed:    true,
			},
			FieldOmniClusterState: {
				Type:        schema.TypeString,
				Description: "Current state of the cluster",
				Computed:    true,
			},
			FieldOmniClusterStatus: {
				Type:        schema.TypeString,
				Description: "Current status of the cluster",
				Computed:    true,
			},
			FieldOmniClusterOnboardingScript: {
				Type:        schema.TypeString,
				Description: "Script to onboard the cluster",
				Computed:    true,
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOmniClusterOnboard(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniProvisionerClient

	organizationID := data.Get(FieldOmniClusterOrganizationID).(string)
	clusterID := data.Id()

	// First, get the cluster details to ensure it exists
	resp, err := client.ListClustersWithResponse(ctx, organizationID, &omni_provisioner.ListClustersParams{})
	if err != nil {
		return diag.FromErr(fmt.Errorf("listing clusters: %w", err))
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("listing clusters: unexpected status code %d", resp.StatusCode()))
	}

	// If we have an ID, onboard the cluster
	if clusterID != "" {
		onboardResp, err := client.OnboardClusterWithResponse(ctx, organizationID, clusterID)
		if err != nil {
			return diag.FromErr(fmt.Errorf("onboarding cluster: %w", err))
		}

		if onboardResp.StatusCode() != http.StatusOK && onboardResp.StatusCode() != http.StatusCreated {
			return diag.FromErr(fmt.Errorf("onboarding cluster: unexpected status code %d", onboardResp.StatusCode()))
		}
	}

	return resourceOmniClusterRead(ctx, data, meta)
}

func resourceOmniClusterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniProvisionerClient

	organizationID := data.Get(FieldOmniClusterOrganizationID).(string)
	clusterID := data.Id()

	if clusterID == "" {
		return diag.Errorf("cluster ID is required")
	}

	resp, err := client.GetClusterWithResponse(ctx, organizationID, clusterID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("getting cluster: %w", err))
	}

	if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Omni cluster (%s) not found, removing from state", clusterID)
		data.SetId("")
		return nil
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("getting cluster: unexpected status code %d", resp.StatusCode()))
	}

	cluster := resp.JSON200
	if cluster == nil {
		return diag.Errorf("cluster response is nil")
	}

	if err := data.Set(FieldOmniClusterName, cluster.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %w", err))
	}

	if cluster.ProviderType != nil {
		if err := data.Set(FieldOmniClusterProviderType, string(*cluster.ProviderType)); err != nil {
			return diag.FromErr(fmt.Errorf("setting provider_type: %w", err))
		}
	}

	if cluster.State != nil {
		if err := data.Set(FieldOmniClusterState, string(*cluster.State)); err != nil {
			return diag.FromErr(fmt.Errorf("setting state: %w", err))
		}
	}

	if cluster.Status != nil {
		if err := data.Set(FieldOmniClusterStatus, string(*cluster.Status)); err != nil {
			return diag.FromErr(fmt.Errorf("setting status: %w", err))
		}
	}

	if cluster.ServiceAccountId != nil {
		if err := data.Set(FieldOmniClusterServiceAccountID, *cluster.ServiceAccountId); err != nil {
			return diag.FromErr(fmt.Errorf("setting service_account_id: %w", err))
		}
	}

	// Get onboarding script
	scriptResp, err := client.GetOnboardScriptWithResponse(ctx, organizationID, clusterID)
	if err == nil && scriptResp.StatusCode() == http.StatusOK && scriptResp.JSON200 != nil {
		if scriptResp.JSON200.Script != nil {
			if err := data.Set(FieldOmniClusterOnboardingScript, *scriptResp.JSON200.Script); err != nil {
				return diag.FromErr(fmt.Errorf("setting onboarding_script: %w", err))
			}
		}
	}

	return nil
}

func resourceOmniClusterUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Omni clusters don't support updates through the API yet
	// Most changes would require re-onboarding
	return resourceOmniClusterRead(ctx, data, meta)
}

func resourceOmniClusterDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Note: The API doesn't have a delete cluster endpoint
	// Clusters are managed externally and this resource only tracks the onboarding state
	log.Printf("[WARN] Omni cluster deletion is not supported via API. Remove the cluster manually if needed.")
	data.SetId("")
	return nil
}
