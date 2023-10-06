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
	FieldAutoscalerPolicies     = "autoscaler_policies"
)

func resourceAutoscaler() *schema.Resource {
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
				Type:             schema.TypeString,
				Description:      "autoscaler policies JSON string to override current autoscaler settings",
				Optional:         true,
				ValidateDiagFunc: validateAutoscalerPolicyJSON(),
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

	data.SetId(getClusterId(data))
	return nil
}

func resourceCastaiAutoscalerUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := updateAutoscalerPolicies(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(getClusterId(data))
	return nil
}

func getCurrentPolicies(ctx context.Context, client sdk.ClientWithResponsesInterface, clusterId string) ([]byte, error) {
	log.Printf("[INFO] Getting cluster autoscaler information.")

	resp, err := client.PoliciesAPIGetClusterPoliciesWithResponse(ctx, clusterId)
	if err != nil {
		return nil, err
	} else if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("cluster %s policies do not exist at CAST AI", clusterId)
	}

	bytes, err := io.ReadAll(resp.HTTPResponse.Body)
	defer resp.HTTPResponse.Body.Close()
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

func upsertPolicies(ctx context.Context, meta interface{}, clusterId string, changedPoliciesJSON string) error {
	client := meta.(*ProviderConfig).api

	resp, err := client.PoliciesAPIUpsertClusterPoliciesWithBodyWithResponse(ctx, clusterId, "application/json", bytes.NewReader([]byte(changedPoliciesJSON)))
	if checkErr := sdk.CheckOKResponse(resp.HTTPResponse, err); checkErr != nil {
		return checkErr
	}

	return nil
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

func getChangedPolicies(ctx context.Context, data *schema.ResourceData, meta interface{}, clusterId string) ([]byte, error) {
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
		log.Printf("[WARN] Failed to merge policy changes: %v", err)
		return nil, fmt.Errorf("failed to merge policies: %v", err)
	}

	return policies, nil
}

func getClusterId(data *schema.ResourceData) string {
	value, found := data.GetOk(FieldClusterId)
	if !found {
		return ""
	}

	return value.(string)
}

func validateAutoscalerPolicyJSON() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(func(i interface{}, k string) ([]string, []error) {
		v, ok := i.(string)
		if !ok {
			return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
		}
		policyMap := make(map[string]interface{})
		err := json.Unmarshal([]byte(v), &policyMap)
		if err != nil {
			return nil, []error{fmt.Errorf("failed to deserialize JSON: %v", err)}
		}
		errors := make([]error, 0)
		if _, found := policyMap["spotInstances"]; found {
			errors = append(errors, createValidationError("spotInstances", v))
		}
		if unschedulablePods, found := policyMap["unschedulablePods"]; found {
			if unschedulablePodsMap, ok := unschedulablePods.(map[string]interface{}); ok {
				if _, found := unschedulablePodsMap["customInstancesEnabled"]; found {
					errors = append(errors, createValidationError("customInstancesEnabled", v))
				}
				if _, found := unschedulablePodsMap["nodeConstraints"]; found {
					errors = append(errors, createValidationError("nodeConstraints", v))
				}
			}
		}

		return nil, errors
	})
}

func createValidationError(field, value string) error {
	return fmt.Errorf("'%s' field was removed from policies JSON in 5.0.0. "+
		"The configuration was migrated to default node template.\n\n"+
		"See: https://github.com/castai/terraform-provider-castai#migrating-from-4xx-to-5xx\n\n"+
		"Policy:\n%v", field, value)
}
