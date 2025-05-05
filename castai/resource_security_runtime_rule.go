package castai

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldRuntimeRuleName             = "name"
	FieldRuntimeRuleType             = "type"
	FieldRuntimeRuleSeverity         = "severity"
	FieldRuntimeRuleEnabled          = "enabled"
	FieldRuntimeRuleRuleText         = "rule_text"
	FieldRuntimeRuleRuleEngineType   = "rule_engine_type"
	FieldRuntimeRuleResourceSelector = "resource_selector"
	FieldRuntimeRuleCategory         = "category"
	FieldRuntimeRuleLabels           = "labels"

	// COMPUTED fields (for better UX, terraform show will show if rule is built in and similar metadata).
	FieldRuntimeRuleAnomaliesCount  = "anomalies_count"
	FieldRuntimeRuleIsBuiltIn       = "is_built_in"
	FieldRuntimeRuleUsedCustomLists = "used_custom_lists"
)

var supportedSeverities = []string{
	string(sdk.RuntimeV1SeveritySEVERITYCRITICAL),
	string(sdk.RuntimeV1SeveritySEVERITYHIGH),
	string(sdk.RuntimeV1SeveritySEVERITYMEDIUM),
	string(sdk.RuntimeV1SeveritySEVERITYLOW),
	string(sdk.RuntimeV1SeveritySEVERITYNONE),
}

var supportedRuleEngineTypes = []string{
	string(sdk.RULEENGINETYPECEL),
}

var rulesPageLimit = "50"

func resourceSecurityRuntimeRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSecurityRuntimeRuleCreate,
		ReadContext:   resourceSecurityRuntimeRuleRead,
		UpdateContext: resourceSecurityRuntimeRuleUpdate,
		DeleteContext: resourceSecurityRuntimeRuleDelete,

		// non-default importer, because we use name for identifier, backend uses UUID (ID field).
		// TF state stores ID as identifier for performance reasons, but provider users use name for identification.
		Importer: &schema.ResourceImporter{
			StateContext: resourceSecurityRuntimeRuleImporter,
		},

		Description: "Manages a CAST AI security runtime rule.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(3 * time.Minute),
			Read:   schema.DefaultTimeout(3 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldRuntimeRuleName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Unique name of the runtime security rule. Name is used as resource identifier in Terraform.",
				ForceNew:    true, // update is not supported
			},
			FieldRuntimeRuleType: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Type of the rule (internal value).",
			},
			FieldRuntimeRuleCategory: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Category of the rule.",
				Default:     "event",
				ForceNew:    true, // update is not supported
			},
			FieldRuntimeRuleSeverity: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Severity of the rule. One of SEVERITY_CRITICAL, SEVERITY_HIGH, SEVERITY_MEDIUM, SEVERITY_LOW, SEVERITY_NONE.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(supportedSeverities, true)),
			},
			FieldRuntimeRuleEnabled: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the rule is enabled.",
				Default:     false,
			},
			FieldRuntimeRuleRuleText: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "CEL rule expression text.",
			},
			FieldRuntimeRuleRuleEngineType: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The engine type used to evaluate the rule. Only RULE_ENGINE_TYPE_CEL is currently supported.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(supportedRuleEngineTypes, true)),
				Default:          sdk.RULEENGINETYPECEL,
				ForceNew:         true, // update is not supported
			},
			FieldRuntimeRuleResourceSelector: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional CEL expression for resource selection.",
			},
			FieldRuntimeRuleLabels: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Key-value labels attached to the rule.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			// COMPUTED fields (for better UX, terraform show will show if rule is built in and similar metadata).
			FieldRuntimeRuleAnomaliesCount: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of anomalies detected using this rule.",
			},
			FieldRuntimeRuleIsBuiltIn: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates whether the rule is a built-in rule.",
			},
			FieldRuntimeRuleUsedCustomLists: {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Custom lists used in this rule, if any.",
			},
		},
	}
}

func resourceSecurityRuntimeRuleImporter(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*ProviderConfig).api
	name := d.Id() // customer provides name as ID

	// Search for rule by name
	rule, err := findRuntimeRuleByName(ctx, client, name)
	if err != nil {
		return nil, fmt.Errorf("import: finding rule by name: %w", err)
	}
	if rule == nil || rule.Id == nil {
		return nil, fmt.Errorf("import: runtime rule with name %q not found", name)
	}

	// save UUID (api: ID) as ID in tf state
	d.SetId(*rule.Id)

	return []*schema.ResourceData{d}, nil
}

func resourceSecurityRuntimeRuleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	req := sdk.RuntimeV1CreateRuleRequest{
		Name:           d.Get(FieldRuntimeRuleName).(string),
		Category:       d.Get(FieldRuntimeRuleCategory).(string),
		Severity:       sdk.RuntimeV1Severity(d.Get(FieldRuntimeRuleSeverity).(string)),
		RuleText:       d.Get(FieldRuntimeRuleRuleText).(string),
		RuleEngineType: sdk.RuntimeV1RuleEngineType(d.Get(FieldRuntimeRuleRuleEngineType).(string)),
	}

	if v, ok := d.GetOk(FieldRuntimeRuleEnabled); ok {
		val := v.(bool)
		req.Enabled = &val
	}
	if v, ok := d.GetOk(FieldRuntimeRuleResourceSelector); ok {
		val := v.(string)
		req.ResourceSelector = &val
	}
	if v, ok := d.GetOk(FieldRuntimeRuleLabels); ok {
		raw := v.(map[string]interface{})
		labels := make(map[string]string, len(raw))
		for k, v := range raw {
			labels[k] = v.(string)
		}
		req.Labels = &labels
	}

	resp, err := client.RuntimeSecurityAPICreateRuleWithResponse(ctx, req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("creating security runtime rule: %v", err)
	}

	// After create, read Rule again to get ID
	createdRule, err := findRuntimeRuleByName(ctx, client, req.Name)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching created rule: %w", err))
	}
	if createdRule == nil || createdRule.Id == nil {
		return diag.Errorf("created rule not found after creation")
	}

	// Save UUID, not name
	d.SetId(*createdRule.Id)

	return resourceSecurityRuntimeRuleRead(ctx, d, meta)
}

func resourceSecurityRuntimeRuleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	ruleID := d.Id()

	req := sdk.RuntimeSecurityAPIEditRuleRequest{
		Enabled:  false,
		Severity: sdk.RuntimeV1Severity(d.Get(FieldRuntimeRuleSeverity).(string)),
	}

	req.Enabled = d.Get(FieldRuntimeRuleEnabled).(bool)

	if v, ok := d.GetOk(FieldRuntimeRuleResourceSelector); ok {
		str := v.(string)
		req.ResourceSelector = &str
	}
	if v, ok := d.GetOk(FieldRuntimeRuleRuleText); ok {
		str := v.(string)
		req.RuleText = &str
	}
	if v, ok := d.GetOk(FieldRuntimeRuleLabels); ok {
		raw := v.(map[string]interface{})
		labels := make(map[string]string, len(raw))
		for k, v := range raw {
			labels[k] = v.(string)
		}
		req.Labels = &labels
	}

	resp, err := client.RuntimeSecurityAPIEditRuleWithResponse(ctx, ruleID, req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("updating security runtime rule: %v", err)
	}

	return resourceSecurityRuntimeRuleRead(ctx, d, meta)
}

func resourceSecurityRuntimeRuleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	ruleID := d.Id()

	resp, err := client.RuntimeSecurityAPIGetRuleWithResponse(ctx, ruleID)
	if resp != nil && resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "runtime rule not found", map[string]interface{}{"id": ruleID})
		d.SetId("")
		return nil
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("getting runtime rule: %v", err)
	}
	if resp == nil || resp.JSON200 == nil || resp.JSON200.Rule == nil {
		tflog.Warn(ctx, "runtime rule not found", map[string]interface{}{"id": ruleID})
		d.SetId("")
		return nil
	}

	rule := resp.JSON200.Rule

	if err := d.Set(FieldRuntimeRuleName, rule.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting %s: %w", FieldRuntimeRuleName, err))
	}
	if err := d.Set(FieldRuntimeRuleCategory, rule.Category); err != nil {
		return diag.FromErr(fmt.Errorf("setting %s: %w", FieldRuntimeRuleCategory, err))
	}
	if err := d.Set(FieldRuntimeRuleSeverity, rule.Severity); err != nil {
		return diag.FromErr(fmt.Errorf("setting %s: %w", FieldRuntimeRuleSeverity, err))
	}
	normalizedRuleText := normalizeRuleText(rule.RuleText)
	if err := d.Set(FieldRuntimeRuleRuleText, normalizedRuleText); err != nil {
		return diag.FromErr(fmt.Errorf("setting %s: %w", FieldRuntimeRuleRuleText, err))
	}
	if err := d.Set(FieldRuntimeRuleRuleEngineType, rule.RuleEngineType); err != nil {
		return diag.FromErr(fmt.Errorf("setting %s: %w", FieldRuntimeRuleRuleEngineType, err))
	}
	if err := d.Set(FieldRuntimeRuleResourceSelector, rule.ResourceSelector); err != nil {
		return diag.FromErr(fmt.Errorf("setting %s: %w", FieldRuntimeRuleResourceSelector, err))
	}
	if err := d.Set(FieldRuntimeRuleEnabled, rule.Enabled); err != nil {
		return diag.FromErr(fmt.Errorf("setting %s: %w", FieldRuntimeRuleEnabled, err))
	}
	if err := d.Set(FieldRuntimeRuleType, rule.Type); err != nil {
		return diag.FromErr(fmt.Errorf("setting %s: %w", FieldRuntimeRuleType, err))
	}

	if rule.Labels != nil {
		if err := d.Set(FieldRuntimeRuleLabels, flattenLabels(rule.Labels)); err != nil {
			return diag.FromErr(fmt.Errorf("setting %s: %w", FieldRuntimeRuleLabels, err))
		}
	}
	if rule.AnomaliesCount != nil {
		if err := d.Set(FieldRuntimeRuleAnomaliesCount, *rule.AnomaliesCount); err != nil {
			return diag.FromErr(fmt.Errorf("setting %s: %w", FieldRuntimeRuleAnomaliesCount, err))
		}
	}
	if rule.IsBuiltIn != nil {
		if err := d.Set(FieldRuntimeRuleIsBuiltIn, *rule.IsBuiltIn); err != nil {
			return diag.FromErr(fmt.Errorf("setting %s: %w", FieldRuntimeRuleIsBuiltIn, err))
		}
	}
	if rule.UsedCustomLists != nil {
		if err := d.Set(FieldRuntimeRuleUsedCustomLists, flattenCustomLists(rule.UsedCustomLists)); err != nil {
			return diag.FromErr(fmt.Errorf("setting %s: %w", FieldRuntimeRuleUsedCustomLists, err))
		}
	}

	return nil
}

// normalizeRuleText trims spaces and normalizes newlines to prevent inconsistent TF value comparisons.
func normalizeRuleText(s *string) string {
	if s == nil {
		return ""
	}
	// Trim leading/trailing spaces on each line
	lines := strings.Split(*s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t") // Remove trailing spaces and tabs
	}
	return strings.Join(lines, "\n")
}

func resourceSecurityRuntimeRuleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	ruleID := d.Id()

	isBuiltInRaw, ok := d.GetOk(FieldRuntimeRuleIsBuiltIn)
	isBuiltIn := ok && isBuiltInRaw.(bool)

	if isBuiltIn {
		// Built-in rule: disable instead of deleting, we can't delete built in rules
		return disableRule(ctx, d, ruleID, client)
	}

	// not Build-in rule: delete it
	delReq := sdk.RuntimeSecurityAPIDeleteRulesJSONRequestBody{
		Ids: []string{ruleID},
	}
	resp, err := client.RuntimeSecurityAPIDeleteRulesWithResponse(ctx, delReq)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("deleting security runtime rule: %v", err)
	}

	d.SetId("")
	return nil
}

func disableRule(ctx context.Context, d *schema.ResourceData, ruleID string, client sdk.ClientWithResponsesInterface) diag.Diagnostics {
	req := sdk.RuntimeV1ToggleRulesRequest{
		Enabled: false,
		Ids:     []string{ruleID},
	}
	resp, err := client.RuntimeSecurityAPIToggleRulesWithResponse(ctx, req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("disabling built-in runtime rule (instead of deleting, we can't delete built in rules): %v", err)
	}

	d.SetId("")
	return nil
}

// findRuntimeRuleByName pages through API results to find a rule by name.
func findRuntimeRuleByName(ctx context.Context, client sdk.ClientWithResponsesInterface, name string) (*sdk.RuntimeV1Rule, error) {
	var cursor *string

	for {
		params := &sdk.RuntimeSecurityAPIGetRulesParams{
			PageLimit:  &rulesPageLimit,
			PageCursor: cursor,
		}

		resp, err := client.RuntimeSecurityAPIGetRulesWithResponse(ctx, params)
		if err := sdk.CheckOKResponse(resp, err); err != nil {
			return nil, fmt.Errorf("listing runtime rules: %w", err)
		}
		if resp.JSON200 == nil || resp.JSON200.Rules == nil {
			return nil, nil
		}

		if len(*resp.JSON200.Rules) == 0 {
			break
		}

		for _, rule := range *resp.JSON200.Rules {
			if rule.Name != nil && *rule.Name == name {
				return &rule, nil
			}
		}

		if resp.JSON200.NextCursor == nil || *resp.JSON200.NextCursor == "" {
			break
		}
		cursor = resp.JSON200.NextCursor
	}

	return nil, nil
}

// Helpers
func flattenLabels(m *map[string]string) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{}, len(*m))
	for k, v := range *m {
		out[k] = v
	}
	return out
}

func flattenCustomLists(v *[]sdk.RuntimeV1ListHeader) []interface{} {
	if v == nil {
		return nil
	}
	out := make([]interface{}, 0, len(*v))
	for _, item := range *v {
		if item.Name != nil {
			out = append(out, *item.Name)
		}
	}
	return out
}
