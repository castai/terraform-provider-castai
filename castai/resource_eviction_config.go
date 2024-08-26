package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"
	"log"
	"time"
)

const (
	FieldEvictorAdvancedConfig    = "evictor_advanced_config"
	FieldEvictionConfig           = "eviction_config"
	FieldNodeSelector             = "node_selector"
	FieldPodSelector              = "pod_selector"
	FieldEvictionSettings         = "settings"
	FieldEvictionOptionDisabled   = "removal_disabled"
	FieldEvictionOptionAggressive = "aggressive"
	FieldEvictionOptionDisposable = "disposable"
	FieldPodSelectorKind          = "kind"
	FieldPodSelectorNamespace     = "namespace"
	FieldMatchLabels              = "match_labels"
	FieldMatchExpressions         = "match_expressions"
	FieldMatchExpressionKey       = "key"
	FieldMatchExpressionOp        = "operator"
	FieldMatchExpressionVal       = "values"
)

func resourceEvictionConfig() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceEvictionConfigRead,
		CreateContext: resourceEvictionConfigCreate,
		UpdateContext: resourceEvictionConfigUpdate,
		DeleteContext: resourceEvictionConfigDelete,
		Description:   "CAST AI eviction config resource to manage evictor properties ",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldClusterId: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "CAST AI cluster id.",
			},
			FieldEvictorAdvancedConfig: {
				Type:        schema.TypeList,
				Description: "evictor advanced configuration to target specific node/pod",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldPodSelector: {
							Type:        schema.TypeList,
							Description: "pod selector",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldPodSelectorNamespace: {
										Type:     schema.TypeString,
										Optional: true,
									},
									FieldPodSelectorKind: {
										Type:     schema.TypeString,
										Optional: true,
									},
									FieldMatchLabels: {
										Type:     schema.TypeMap,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									FieldMatchExpressions: {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldMatchExpressionKey: {Type: schema.TypeString, Required: true},
												FieldMatchExpressionOp:  {Type: schema.TypeString, Required: true},
												FieldMatchExpressionVal: {
													Type:     schema.TypeList,
													Elem:     &schema.Schema{Type: schema.TypeString},
													Optional: true,
												},
											},
										},
									},
								},
							},
						},
						FieldNodeSelector: {
							Type:        schema.TypeList,
							Description: "node selector",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldMatchLabels: {
										Type:     schema.TypeMap,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									FieldMatchExpressions: {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												FieldMatchExpressionKey: {Type: schema.TypeString, Required: true},
												FieldMatchExpressionOp:  {Type: schema.TypeString, Required: true},
												FieldMatchExpressionVal: {
													Type:     schema.TypeList,
													Elem:     &schema.Schema{Type: schema.TypeString},
													Optional: true,
												},
											},
										},
									},
								},
							},
						},
						FieldEvictionOptionDisabled: {
							Type:     schema.TypeBool,
							Optional: true,
							Description: "Mark pods as removal disabled",
						},
						FieldEvictionOptionAggressive: {
							Type:     schema.TypeBool,
							Optional: true,
							Description: "Apply Aggressive mode to Evictor",
						},
						FieldEvictionOptionDisposable: {
							Type:     schema.TypeBool,
							Optional: true,
							Description: "Mark node as disposable",
						},
					},
				},
			},
		},
	}
}

func resourceEvictionConfigRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := readAdvancedEvictorConfig(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func readAdvancedEvictorConfig(ctx context.Context, data *schema.ResourceData, meta interface{}) error {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}
	client := meta.(*ProviderConfig).api

	resp, err := client.EvictorAPIGetAdvancedConfigWithResponse(ctx, clusterId)
	if err != nil {
		log.Printf("[ERROR] Failed to set read evictor advanced config: %v", err)
		return err
	}
	err = data.Set(FieldEvictorAdvancedConfig, flattenEvictionConfig(resp.JSON200.EvictionConfig))
	if err != nil {
		log.Printf("[ERROR] Failed to set field: %v", err)
		return err
	}

	return nil
}

func resourceEvictionConfigCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := upsertEvictionConfigs(ctx, data, meta); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(getClusterId(data))
	return nil
}

func getEvictorAdvancedConfigAsJson(data *schema.ResourceData) ([]byte, error) {
	eac, ok := data.GetOk(FieldEvictorAdvancedConfig)
	if !ok {
		return nil, fmt.Errorf("failed to extract evictor advanced config [%v], [%+v]", eac, data.GetRawState())
	}

	evictionConfigs, err := toEvictionConfig(eac)
	if err != nil {
		return nil, err
	}
	ccd := sdk.CastaiEvictorV1AdvancedConfig{EvictionConfig: evictionConfigs}
	return json.Marshal(ccd)
}

func upsertEvictionConfigs(ctx context.Context, data *schema.ResourceData, meta interface{}) error {
	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}
	evictorAdvancedConfigJson, err := getEvictorAdvancedConfigAsJson(data)
	if err != nil {
		log.Printf("[ERROR] Failed to extract evictor advanced config: %v", err)
		return err
	}
	client := meta.(*ProviderConfig).api
	resp, err := client.EvictorAPIUpsertAdvancedConfigWithBodyWithResponse(
		ctx,
		clusterId,
		"application/json",
		bytes.NewReader(evictorAdvancedConfigJson),
	)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		log.Printf("[ERROR] Failed to upsert evictor advanced config: %v", err)
		return checkErr
	}
	err = data.Set(FieldEvictorAdvancedConfig, flattenEvictionConfig(resp.JSON200.EvictionConfig))
	if err != nil {
		log.Printf("[ERROR] Failed to set field: %v", err)
		return err
	}

	return nil
}

func resourceEvictionConfigUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := upsertEvictionConfigs(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(getClusterId(data))
	return nil
}

func resourceEvictionConfigDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	err := deleteEvictionConfigs(ctx, data, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(getClusterId(data))
	return nil
}

func deleteEvictionConfigs(ctx context.Context, data *schema.ResourceData, meta interface{}) error {

	clusterId := getClusterId(data)
	if clusterId == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}
	client := meta.(*ProviderConfig).api
	resp, err := client.EvictorAPIUpsertAdvancedConfigWithBodyWithResponse(
		ctx,
		clusterId,
		"application/json",
		bytes.NewReader([]byte("{}")),
	)
	if err != nil || resp.JSON200 == nil {
		log.Printf("[ERROR] Failed to upsert evictor advanced config: %v", err)
		return err
	}
	err = data.Set(FieldEvictorAdvancedConfig, flattenEvictionConfig(resp.JSON200.EvictionConfig))
	if err != nil {
		log.Printf("[ERROR] Failed to set field: %v", err)
		return err
	}

	return nil
}

func toEvictionConfig(ii interface{}) ([]sdk.CastaiEvictorV1EvictionConfig, error) {
	in, ok := ii.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expecting []interface, got %T", ii)
	}
	if len(in) < 1 {
		return nil, nil
	}
	out := make([]sdk.CastaiEvictorV1EvictionConfig, len(in))
	var err error
	for i, c := range in {

		ec := sdk.CastaiEvictorV1EvictionConfig{}
		cc, ok := c.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("mapping evictionConfig expecting map[string]interface, got %T, %+v", c, c)
		}

		for k, v := range cc {

			switch k {
			case FieldPodSelector:
				ec.PodSelector, err = toPodSelector(v)
				if err != nil {
					return nil, err
				}
			case FieldNodeSelector:
				ec.NodeSelector, err = toNodeSelector(v)
				if err != nil {
					return nil, err
				}
			case FieldEvictionOptionAggressive:
				enabled, ok := v.(bool)
				if !ok {
					return nil, fmt.Errorf("mapping eviction aggressive expecing bool, got %T, %+v", v, v)
				}
				if enabled {
					ec.Settings.Aggressive = &sdk.CastaiEvictorV1EvictionSettingsSettingEnabled{Enabled: enabled}

				}
			case FieldEvictionOptionDisabled:
				enabled, ok := v.(bool)
				if !ok {
					return nil, fmt.Errorf("mapping eviction disabled expecing bool, got %T, %+v", v, v)
				}
				if enabled {
					ec.Settings.RemovalDisabled = &sdk.CastaiEvictorV1EvictionSettingsSettingEnabled{Enabled: enabled}
				}
			case FieldEvictionOptionDisposable:
				enabled, ok := v.(bool)
				if !ok {
					return nil, fmt.Errorf("mapping eviction aggressive expecing bool, got %T, %+v", v, v)
				}
				if enabled {
					ec.Settings.Disposable = &sdk.CastaiEvictorV1EvictionSettingsSettingEnabled{Enabled: enabled}
				}
			default:
				return nil, fmt.Errorf("unexpected field %s, %T, %+v", k, v, v)
			}
		}
		out[i] = ec
	}
	return out, nil
}
func flattenEvictionConfig(ecs []sdk.CastaiEvictorV1EvictionConfig) []map[string]any {
	if ecs == nil {
		return nil
	}
	res := make([]map[string]any, len(ecs))
	for i, c := range ecs {
		out := map[string]any{}
		if c.PodSelector != nil {
			out[FieldPodSelector] = flattenPodSelector(c.PodSelector)
		}
		if c.NodeSelector != nil {
			out[FieldNodeSelector] = flattenNodeSelector(c.NodeSelector)
		}
		if c.Settings.Aggressive != nil {
			out[FieldEvictionOptionAggressive] = c.Settings.Aggressive.Enabled
		}

		if c.Settings.Disposable != nil {
			out[FieldEvictionOptionDisposable] = c.Settings.Disposable.Enabled
		}

		if c.Settings.RemovalDisabled != nil {
			out[FieldEvictionOptionDisabled] = c.Settings.RemovalDisabled.Enabled
		}
		res[i] = out
	}

	return res
}

func toPodSelector(in interface{}) (*sdk.CastaiEvictorV1PodSelector, error) {
	iii, ok := in.([]interface{})
	if !ok {
		return nil, fmt.Errorf("mapping podselector expecting []interface, got %T, %+v", in, in)
	}
	if len(iii) < 1 {
		return nil, nil
	}
	ii := iii[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("mapping podselector expecting map[string]interface, got %T, %+v", in, in)
	}
	out := sdk.CastaiEvictorV1PodSelector{}
	for k, v := range ii {
		switch k {
		case FieldPodSelectorKind:
			if kind, ok := v.(string); ok {
				out.Kind = lo.ToPtr(kind)
			} else {
				return nil, fmt.Errorf("expecting bool, got %T", v)
			}
		case FieldPodSelectorNamespace:
			if namespace, ok := v.(string); ok {
				if len(namespace) == 0 {
					continue
				}
				out.Namespace = lo.ToPtr(namespace)
			} else {
				return nil, fmt.Errorf("expecting bool, got %T", v)
			}
		case FieldMatchExpressions:
			if mes, ok := v.([]interface{}); ok {
				me, err := toMatchExpressions(mes)
				if err != nil {
					return nil, err
				}
				if len(me) < 1 {
					continue
				}

				if out.LabelSelector == nil {
					out.LabelSelector = &sdk.CastaiEvictorV1LabelSelector{}
				}
				out.LabelSelector.MatchExpressions = &me
			} else {
				return nil, fmt.Errorf("mapping match_expressions expecting map[string]interface, got %T, %+v", v, v)
			}
		case FieldMatchLabels:
			mls, err := toMatchLabels(v)
			if err != nil {
				return nil, err
			}

			if mls == nil || len(mls.AdditionalProperties) == 0 {
				continue
			}

			if out.LabelSelector == nil {
				out.LabelSelector = &sdk.CastaiEvictorV1LabelSelector{}
			}
			out.LabelSelector.MatchLabels = mls
		}
	}
	return &out, nil
}

func toNodeSelector(in interface{}) (*sdk.CastaiEvictorV1NodeSelector, error) {
	iii, ok := in.([]interface{})
	if !ok {
		return nil, fmt.Errorf("mapping nodeselector expecting []interface, got %T, %+v", in, in)
	}
	if len(iii) < 1 {
		return nil, nil
	}
	ii := iii[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("mapping podselector expecting map[string]interface, got %T, %+v", in, in)
	}
	out := sdk.CastaiEvictorV1NodeSelector{}
	for k, v := range ii {
		switch k {

		case FieldMatchExpressions:
			if mes, ok := v.([]interface{}); ok {
				me, err := toMatchExpressions(mes)
				if err != nil {
					return nil, err
				}
				out.LabelSelector.MatchExpressions = &me
			} else {
				return nil, fmt.Errorf("mapping match_expressions expecting map[string]interface, got %T, %+v", v, v)
			}
		case FieldMatchLabels:
			mls, err := toMatchLabels(v)
			if err != nil {
				return nil, err
			}
			out.LabelSelector.MatchLabels = mls
		}
	}
	return &out, nil
}

func toMatchLabels(in interface{}) (*sdk.CastaiEvictorV1LabelSelector_MatchLabels, error) {
	mls, ok := in.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("mapping match_labels expecting map[string]interface, got %T %+v", in, in)
	}
	if len(mls) == 0 {
		return nil, nil
	}
	out := sdk.CastaiEvictorV1LabelSelector_MatchLabels{AdditionalProperties: map[string]string{}}
	for k, v := range mls {
		value, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("mapping match_labels expecting string, got %T %+v", v, v)
		}
		out.AdditionalProperties[k] = value
	}

	return &out, nil
}

func flattenPodSelector(ps *sdk.CastaiEvictorV1PodSelector) []map[string]any {
	if ps == nil {
		return nil
	}
	out := map[string]any{}
	if ps.Kind != nil {
		out[FieldPodSelectorKind] = *ps.Kind
	}
	if ps.Namespace != nil {
		out[FieldPodSelectorNamespace] = *ps.Namespace
	}
	if ps.LabelSelector != nil {
		if ps.LabelSelector.MatchLabels != nil {
			out[FieldMatchLabels] = ps.LabelSelector.MatchLabels.AdditionalProperties
		}
		if ps.LabelSelector.MatchExpressions != nil {
			out[FieldMatchExpressions] = flattenMatchExpressions(*ps.LabelSelector.MatchExpressions)
		}
	}
	return []map[string]any{out}
}

func flattenNodeSelector(ns *sdk.CastaiEvictorV1NodeSelector) []map[string]any {
	if ns == nil {
		return nil
	}
	out := map[string]any{}
	if ns.LabelSelector.MatchLabels != nil {
		out[FieldMatchLabels] = ns.LabelSelector.MatchLabels.AdditionalProperties
	}
	if ns.LabelSelector.MatchExpressions != nil {
		out[FieldMatchExpressions] = flattenMatchExpressions(*ns.LabelSelector.MatchExpressions)
	}

	return []map[string]any{out}
}

func flattenMatchExpressions(mes []sdk.CastaiEvictorV1LabelSelectorExpression) []map[string]any {
	if mes == nil {
		return nil
	}

	out := make([]map[string]any, len(mes))
	for i, me := range mes {
		out[i] = map[string]any{
			FieldMatchExpressionKey: me.Key,
			FieldMatchExpressionOp:  string(me.Operator),
		}
		if me.Values != nil && len(*me.Values) > 0 {
			out[i][FieldMatchExpressionVal] = *me.Values
		}
	}

	return out
}

func toMatchExpressions(in []interface{}) ([]sdk.CastaiEvictorV1LabelSelectorExpression, error) {
	out := make([]sdk.CastaiEvictorV1LabelSelectorExpression, len(in))
	for i, mei := range in {
		if me, ok := mei.(map[string]interface{}); ok {
			out[i] = sdk.CastaiEvictorV1LabelSelectorExpression{}
			for k, v := range me {
				switch k {
				case FieldMatchExpressionKey:
					if key, ok := v.(string); ok {
						out[i].Key = key
					} else {
						return nil, fmt.Errorf("mapping match_expression key expecting string, got %T %+v", v, v)
					}
				case FieldMatchExpressionOp:
					if op, ok := v.(string); ok {
						out[i].Operator = sdk.CastaiEvictorV1LabelSelectorExpressionOperator(op)
					} else {
						return nil, fmt.Errorf("mapping match_expression operator expecting string, got %T %+v", v, v)
					}
				case FieldMatchExpressionVal:
					if vals, ok := v.([]interface{}); ok {
						outVals := make([]string, len(vals))
						for vi, vv := range vals {
							outVals[vi], ok = vv.(string)
							if !ok {
								return nil, fmt.Errorf("mapping match_expression values expecting string, got %T %+v", vv, vv)
							}
						}
						out[i].Values = &outVals
					} else {
						return nil, fmt.Errorf("mapping match_expression values expecting []interface{}, got %T %+v", v, v)
					}

				}

			}
		} else {
			return nil, fmt.Errorf("mapping match_expressions expecting map[string]interface, got %T, %+v", mei, mei)
		}

	}
	return out, nil
}
