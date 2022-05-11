package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldAutoscalerPoliciesJSON = "autoscaler_policies_json"
	FieldClusterId              = "cluster_id"
	FieldAutoscalerPolicies     = "autoscaler_policies"
)

func resourceCastaiAutoscaler() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceCastaiAutoscalerRead,
		CreateContext: resourceCastaiAutoscalerCreate,
		UpdateContext: resourceCastaiAutoscalerUpdate,
		DeleteContext: resourceCastaiAutoscalerDelete,
		Description:   "CAST AI autoscaler resource to manage autoscaler settings",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldClusterId: {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "CAST AI cluster id",
			},
			FieldAutoscalerPoliciesJSON: {
				Type:        schema.TypeString,
				Description: "autoscaler policies JSON string to override current autoscaler settings",
				Optional:    true,
			},
			FieldAutoscalerPolicies: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "computed value to store full policies configuration",
			},
		},
	}
}

func resourceCastaiAutoscalerDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	err := upsertPolicies(ctx, meta, clusterId, `{"enabled":false}`)
	if err != nil {
		log.Printf("[ERROR] Failed to disable autoscaler policies: %v", err)
		return diag.FromErr(err)
	}

	return nil
}

func resourceCastaiAutoscalerRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := readAutoscalerPolicies(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCastaiAutoscalerCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	err := updateAutoscalerPolicies(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(string(getClusterId(data)))
	return nil
}

func resourceCastaiAutoscalerUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := updateAutoscalerPolicies(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(string(getClusterId(data)))
	return nil
}

func getCurrentPolicies(ctx context.Context, client *sdk.ClientWithResponses, clusterId sdk.ClusterId) ([]byte, error) {
	log.Printf("[INFO] Getting cluster autoscaler information.")

	resp, err := client.PoliciesAPIGetClusterPolicies(ctx, string(clusterId))
	if err != nil {
		return nil, err
	} else if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("cluster %s policies does not exists in CAST.AI", clusterId)
	}

	bytes, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	log.Printf("[DEBUG] Read autoscaler policies for cluster %s:\n%v\n", clusterId, string(bytes))

	return bytes, nil
}

func updateAutoscalerPolicies(ctx context.Context, data *schema.ResourceData, meta interface{}) error {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	err := readAutoscalerPolicies(ctx, data, meta)
	if err != nil {
		return err
	}

	changedPolicies, found := data.GetOk(FieldAutoscalerPolicies)
	if !found {
		log.Printf("[DEBUG] changed policies json not found. Skipping autoscaler policies changes")
		return nil
	}

	changedPoliciesJSON := changedPolicies.(string)
	if changedPoliciesJSON == "" {
		log.Printf("[DEBUG] changed policies json not found. Skipping autoscaler policies changes")
		return nil
	}

	return upsertPolicies(ctx, meta, clusterId, changedPoliciesJSON)
}

func upsertPolicies(ctx context.Context, meta interface{}, clusterId sdk.ClusterId, changedPoliciesJSON string) error {
	client := meta.(*ProviderConfig).api

	result, err := client.PoliciesAPIUpsertClusterPoliciesWithBody(ctx, string(clusterId), "application/json", bytes.NewReader([]byte(changedPoliciesJSON)))
	if err != nil {
		log.Printf("[ERROR] Error upserting policies: %v", err)
		return fmt.Errorf("error updating policies: %v", err)
	}

	if result.StatusCode > 199 && result.StatusCode <= 299 {
		log.Printf("[INFO] Policies updated: \n%v\n", changedPoliciesJSON)
		return nil
	}

	log.Printf("[ERROR] Failed updating policies. Received status code: %v", result.Status)
	return fmt.Errorf("failed updating policies: %v", result.Status)
}

func readAutoscalerPolicies(ctx context.Context, data *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] AUTOSCALER policies get call start")
	defer log.Printf("[INFO] AUTOSCALER policies get call end")

	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	policies, err := getChangedPolicies(ctx, data, meta, clusterId)
	if err != nil {
		return err
	}

	err = data.Set(FieldAutoscalerPolicies, string(policies))
	if err != nil {
		log.Printf("[ERROR] Failed to set field: %v", err)
		return err
	}

	return nil
}

func getChangedPolicies(ctx context.Context, data *schema.ResourceData, meta interface{}, clusterId sdk.ClusterId) ([]byte, error) {
	policyChangesJSON, found := data.GetOk(FieldAutoscalerPoliciesJSON)
	if !found {
		log.Printf("[DEBUG] policies json not provided. Skipping autoscaler policies changes")
		return nil, nil
	}

	policyChanges := []byte(policyChangesJSON.(string))
	if !json.Valid(policyChanges) {
		log.Printf("[WARN] policies JSON invalid: %v", string(policyChanges))
		return nil, fmt.Errorf("policies JSON invalid")
	}

	client := meta.(*ProviderConfig).api

	currentPolicies, err := getCurrentPolicies(ctx, client, clusterId)
	if err != nil {
		log.Printf("[WARN] Getting current policies: %v", err)
		return nil, fmt.Errorf("failed to get policies from API: %v", err)
	}

	policies, err := jsonpatch.MergePatch(currentPolicies, policyChanges)
	if err != nil {
		log.Printf("[WARN] Failed mergin policy changes: %v", err)
		return nil, fmt.Errorf("failed to merge policies: %v", err)
	}

	return policies, nil
}

func getClusterId(data *schema.ResourceData) sdk.ClusterId {
	value, found := data.GetOk(FieldClusterId)
	if !found {
		return ""
	}

	return sdk.ClusterId(value.(string))
}
