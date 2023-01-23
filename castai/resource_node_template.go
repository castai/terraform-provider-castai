package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"io"
	"log"
	"net/http"
)

const (
	FieldNodeTemplatesJSON = "node_templates_json"
	FieldNodeTemplates     = "node_templates"
)

func resourceNodeTemplate() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceNodeTemplateRead,
		UpdateContext: resourceNodeTemplateUpdate,
		Description:   "CAST AI node template resource to manage autoscaler node templates",
	}
}

func resourceNodeTemplateRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := readNodeTemplate(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
func resourceNodeTemplateUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := updateNodeTemplate(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(getClusterId(data))
	return nil
}

func updateNodeTemplate(ctx context.Context, data *schema.ResourceData, meta interface{}) error {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	err := readNodeTemplate(ctx, data, meta)
	if err != nil {
		return err
	}

	changedNodeTemplates, found := data.GetOk(FieldNodeTemplates)
	if !found {
		log.Printf("[DEBUG] changed node templates json not found. Skipping node templates changes")
		return nil
	}

	changedNodeTemplatesJSON := changedNodeTemplates.(string)
	if changedNodeTemplatesJSON == "" {
		log.Printf("[DEBUG] changed policies json not found. Skipping autoscaler policies changes")
		return nil
	}

	return updateNodeTemplates(ctx, meta, clusterId, changedNodeTemplatesJSON)
}

func updateNodeTemplates(ctx context.Context, meta interface{}, clusterId sdk.ClusterId, changedNodeTemplatesJSON string) error {
	client := meta.(*ProviderConfig).api

	resp, err := client.NodeTemplatesAPIUpdateNodeTemplateWithBodyWithResponse(ctx, string(clusterId), "name", "application/json", bytes.NewReader([]byte(changedNodeTemplatesJSON)))
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return checkErr
	}

	return nil
}

func readNodeTemplate(ctx context.Context, data *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] List Node Templates get call start")
	defer log.Printf("[INFO] List Node Templates get call end")

	clusterID := getClusterId(data)
	if clusterID == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	nodeTemplates, err := getChangedNodeTemplate(ctx, data, meta, clusterID)
	if err != nil {
		return err
	}

	err = data.Set(FieldNodeTemplates, string(nodeTemplates))
	if err != nil {
		log.Printf("[ERROR] Faield to set field: %v", err)
		return err
	}

	return nil
}

func getChangedNodeTemplate(ctx context.Context, data *schema.ResourceData, meta interface{}, clusterId sdk.ClusterId) ([]byte, error) {
	nodeTemplatesJSON, found := data.GetOk(FieldNodeTemplatesJSON)
	if !found {
		log.Printf("[DEBUG] node template JSON is not provided. Skipping node template changes")
		return nil, nil
	}

	nodeTemplatesChanges := []byte(nodeTemplatesJSON.(string))
	if !json.Valid(nodeTemplatesChanges) {
		log.Printf("[WARN] node template JSON is invalid: %v", string(nodeTemplatesChanges))
		return nil, fmt.Errorf("node template JSON is invalid")
	}

	client := meta.(*ProviderConfig).api
	currentNodeTemplates, err := getCurrentNodeTemplate(ctx, client, clusterId)
	if err != nil {
		log.Printf("[WARN] Getting current node templates: %v", err)
		return nil, fmt.Errorf("failed to get current node templates from API: %v", err)
	}

	nodeTemplates, err := jsonpatch.MergePatch(currentNodeTemplates, nodeTemplatesChanges)
	if err != nil {
		log.Printf("[WARN] Failed merging node template changes: %v", err)
		return nil, fmt.Errorf("failed to merge node template changes: %v", err)
	}

	return nodeTemplates, nil
}

func getCurrentNodeTemplate(ctx context.Context, client *sdk.ClientWithResponses, clusterID sdk.ClusterId) ([]byte, error) {
	log.Printf("[INFO] Getting current node templates")
	resp, err := client.NodeTemplatesAPIListNodeTemplates(ctx, clusterID)
	if err != nil {
		return nil, err
	} else if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("cluster %s node templates not found at CAST AI", clusterID)
	}

	bytes, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	log.Printf("[DEBUG] Read node templates for cluster %s:\n%v\n", clusterID, string(bytes))
	return bytes, nil
}
