package castai

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldCacheRuleCacheGroupID         = "cache_group_id"
	FieldCacheRuleCacheConfigurationID = "cache_configuration_id"
	FieldCacheRuleMode                 = "mode"
	FieldCacheRuleManualTTL            = "manual_ttl"
	FieldCacheRuleTable                = "table"
	FieldCacheRuleTemplateHash         = "template_hash"
)

func resourceCacheRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCacheRuleCreate,
		ReadContext:   resourceCacheRuleRead,
		UpdateContext: resourceCacheRuleUpdate,
		DeleteContext: resourceCacheRuleDelete,
		CustomizeDiff: resourceCacheRuleCustomizeDiff,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Manage CAST AI DBO Cache Rule (TTL Configuration). Cache rules define caching behavior and TTL settings for specific queries or tables within a cache configuration.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(2 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldCacheRuleCacheGroupID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the cache group this rule belongs to.",
			},
			FieldCacheRuleCacheConfigurationID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the cache configuration this rule belongs to.",
			},
			FieldCacheRuleMode: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"Auto", "DontCache", "Manual"}, false)),
				Description:      "TTL mode for queries matching this rule. Valid values: Auto, DontCache, Manual.",
			},
			FieldCacheRuleManualTTL: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "TTL in seconds. Required when mode is Manual.",
			},
			FieldCacheRuleTable: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Database table name to apply this rule to. Either table or template_hash must be specified.",
			},
			FieldCacheRuleTemplateHash: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Hash of the query template. Either table or template_hash must be specified.",
			},
		},
	}
}

func resourceCacheRuleCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	mode := d.Get(FieldCacheRuleMode).(string)
	manualTTL := d.Get(FieldCacheRuleManualTTL).(int)

	if mode == "Manual" && manualTTL == 0 {
		return fmt.Errorf("manual_ttl is required when mode is Manual")
	}

	table := d.Get(FieldCacheRuleTable).(string)
	templateHash := d.Get(FieldCacheRuleTemplateHash).(string)

	if table == "" && templateHash == "" {
		return fmt.Errorf("either table or template_hash must be specified")
	}

	if table != "" && templateHash != "" {
		return fmt.Errorf("only one of table or template_hash can be specified")
	}

	return nil
}

func resourceCacheRuleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	cacheGroupID := d.Get(FieldCacheRuleCacheGroupID).(string)
	cacheConfigID := d.Get(FieldCacheRuleCacheConfigurationID).(string)

	req := buildCacheRuleRequest(d)
	resp, err := client.DboAPICreateCacheTTLWithResponse(ctx, cacheGroupID, cacheConfigID, req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	if resp.JSON200 == nil || resp.JSON200.Id == nil {
		return diag.FromErr(fmt.Errorf("cache rule ID not returned from API"))
	}

	d.SetId(*resp.JSON200.Id)
	tflog.Info(ctx, "Cache rule created", map[string]any{
		"id":                     *resp.JSON200.Id,
		"cache_group_id":         cacheGroupID,
		"cache_configuration_id": cacheConfigID,
	})

	return resourceCacheRuleRead(ctx, d, meta)
}

func resourceCacheRuleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	cacheGroupID := d.Get(FieldCacheRuleCacheGroupID).(string)
	cacheConfigID := d.Get(FieldCacheRuleCacheConfigurationID).(string)
	ruleID := d.Id()

	resp, err := client.DboAPIListCacheTTLsWithResponse(ctx, cacheGroupID, cacheConfigID, &sdk.DboAPIListCacheTTLsParams{})
	if err != nil {
		return diag.FromErr(err)
	}

	if !d.IsNewResource() && resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "Cache configuration not found, removing rule from state", map[string]any{
			"id":                     ruleID,
			"cache_group_id":         cacheGroupID,
			"cache_configuration_id": cacheConfigID,
		})
		d.SetId("")
		return nil
	}

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	if resp.JSON200 == nil || resp.JSON200.Items == nil {
		return diag.FromErr(fmt.Errorf("cache rule data not returned from API"))
	}

	var rule *sdk.DboV1TTLConfiguration
	for _, r := range *resp.JSON200.Items {
		if r.Id != nil && *r.Id == ruleID {
			rule = &r
			break
		}
	}

	if rule == nil {
		if !d.IsNewResource() {
			tflog.Warn(ctx, "Cache rule not found, removing from state", map[string]any{
				"id":                     ruleID,
				"cache_group_id":         cacheGroupID,
				"cache_configuration_id": cacheConfigID,
			})
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("cache rule with ID %s not found in cache configuration %s/%s", ruleID, cacheGroupID, cacheConfigID))
	}

	if err := d.Set(FieldCacheRuleCacheGroupID, cacheGroupID); err != nil {
		return diag.FromErr(fmt.Errorf("setting cache_group_id: %w", err))
	}

	if err := d.Set(FieldCacheRuleCacheConfigurationID, cacheConfigID); err != nil {
		return diag.FromErr(fmt.Errorf("setting cache_configuration_id: %w", err))
	}

	if err := d.Set(FieldCacheRuleMode, string(rule.Mode)); err != nil {
		return diag.FromErr(fmt.Errorf("setting mode: %w", err))
	}

	if rule.ManualTtl != nil {
		if err := d.Set(FieldCacheRuleManualTTL, int(*rule.ManualTtl)); err != nil {
			return diag.FromErr(fmt.Errorf("setting manual_ttl: %w", err))
		}
	}

	if rule.Table != nil {
		if err := d.Set(FieldCacheRuleTable, *rule.Table); err != nil {
			return diag.FromErr(fmt.Errorf("setting table: %w", err))
		}
	}

	if rule.TemplateHash != nil {
		if err := d.Set(FieldCacheRuleTemplateHash, *rule.TemplateHash); err != nil {
			return diag.FromErr(fmt.Errorf("setting template_hash: %w", err))
		}
	}

	return nil
}

func resourceCacheRuleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	cacheGroupID := d.Get(FieldCacheRuleCacheGroupID).(string)
	cacheConfigID := d.Get(FieldCacheRuleCacheConfigurationID).(string)
	ruleID := d.Id()

	req := buildCacheRuleRequest(d)
	resp, err := client.DboAPIUpdateCacheTTLWithResponse(ctx, cacheGroupID, cacheConfigID, ruleID, req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Cache rule updated", map[string]any{
		"id":                     ruleID,
		"cache_group_id":         cacheGroupID,
		"cache_configuration_id": cacheConfigID,
	})

	return resourceCacheRuleRead(ctx, d, meta)
}

func resourceCacheRuleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	cacheGroupID := d.Get(FieldCacheRuleCacheGroupID).(string)
	cacheConfigID := d.Get(FieldCacheRuleCacheConfigurationID).(string)
	ruleID := d.Id()

	resp, err := client.DboAPIDeleteCacheTTLWithResponse(ctx, cacheGroupID, cacheConfigID, ruleID)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Cache rule deleted", map[string]any{
		"id":                     ruleID,
		"cache_group_id":         cacheGroupID,
		"cache_configuration_id": cacheConfigID,
	})
	return nil
}

func buildCacheRuleRequest(d *schema.ResourceData) sdk.DboAPICreateCacheTTLJSONRequestBody {
	req := sdk.DboAPICreateCacheTTLJSONRequestBody{
		Mode: sdk.DboV1TTLMode(d.Get(FieldCacheRuleMode).(string)),
	}

	if v, ok := d.GetOk(FieldCacheRuleManualTTL); ok {
		ttl := int64(v.(int))
		req.ManualTtl = &ttl
	}

	if v, ok := d.GetOk(FieldCacheRuleTable); ok {
		table := v.(string)
		req.Table = &table
	}

	if v, ok := d.GetOk(FieldCacheRuleTemplateHash); ok {
		hash := v.(string)
		req.TemplateHash = &hash
	}

	return req
}
