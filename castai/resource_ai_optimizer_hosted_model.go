package castai

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/ai_optimizer"
)

const (
	fieldAIHostedModelClusterID          = "cluster_id"
	fieldAIHostedModelModelSpecsID       = "model_specs_id"
	fieldAIHostedModelService            = "service"
	fieldAIHostedModelPort               = "port"
	fieldAIHostedModelNodeTemplateName   = "node_template_name"
	fieldAIHostedModelRegion             = "region"
	fieldAIHostedModelStatus             = "status"
	fieldAIHostedModelStatusReason       = "status_reason"
	fieldAIHostedModelCurrentReplicas    = "current_replicas"
	fieldAIHostedModelCloudProvider      = "cloud_provider"
	fieldAIHostedModelNamespace          = "namespace"
	fieldAIHostedModelEdgeLocationIDs    = "edge_location_ids"
	fieldAIHostedModelVllmConfig         = "vllm_config"
	fieldAIHostedModelVllmSecretName     = "secret_name"
	fieldAIHostedModelVllmHFToken        = "hugging_face_token"
	fieldAIHostedModelHorizontalAS       = "horizontal_autoscaling"
	fieldAIHostedModelHASEnabled         = "enabled"
	fieldAIHostedModelHASMinReplicas     = "min_replicas"
	fieldAIHostedModelHASMaxReplicas     = "max_replicas"
	fieldAIHostedModelHASTargetMetric    = "target_metric"
	fieldAIHostedModelHASTargetValue     = "target_value"
	fieldAIHostedModelHibernation        = "hibernation"
	fieldAIHostedModelHibEnabled         = "enabled"
	fieldAIHostedModelHibResumeCondition = "resume_condition"
	fieldAIHostedModelHibernateCondition = "hibernate_condition"
	fieldAIHostedModelConditionDuration  = "duration"
	fieldAIHostedModelConditionReqCount  = "request_count"
	fieldAIHostedModelFallback           = "fallback"
	fieldAIHostedModelFallbackEnabled    = "enabled"
	fieldAIHostedModelFallbackProviderID = "provider_id"
	fieldAIHostedModelFallbackModel      = "model"
)

var hibernationConditionSchema = &schema.Resource{
	Schema: map[string]*schema.Schema{
		fieldAIHostedModelConditionDuration: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Time period for the condition evaluation.",
		},
		fieldAIHostedModelConditionReqCount: {
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "Request count threshold. Value of 0 is treated as not set.",
		},
	},
}

func resourceAIHostedModel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAIHostedModelCreate,
		ReadContext:   resourceAIHostedModelRead,
		UpdateContext: resourceAIHostedModelUpdate,
		DeleteContext: resourceAIHostedModelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceAIHostedModelImporter,
		},
		Schema: map[string]*schema.Schema{
			fieldAIHostedModelClusterID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "CAST AI cluster ID where the model will be deployed.",
			},
			fieldAIHostedModelModelSpecsID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the model specs. Can reference a castai_ai_optimizer_model_specs resource or a pre-existing model specs ID for predefined (CastAI-managed) models.",
			},
			fieldAIHostedModelService: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Kubernetes service name for the deployed model.",
			},
			fieldAIHostedModelPort: {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Port on which the model will be exposed.",
			},
			fieldAIHostedModelNodeTemplateName: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Node template name for model deployment.",
			},
			fieldAIHostedModelRegion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Region the model is deployed in.",
			},
			fieldAIHostedModelEdgeLocationIDs: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of edge location IDs where the model can be deployed.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			fieldAIHostedModelVllmConfig: {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "vLLM configuration for HuggingFace models.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						fieldAIHostedModelVllmSecretName: {
							Type:          schema.TypeString,
							Optional:      true,
							Description:   "Kubernetes secret name containing the HuggingFace token.",
							ConflictsWith: []string{fieldAIHostedModelVllmConfig + ".0." + fieldAIHostedModelVllmHFToken},
						},
						fieldAIHostedModelVllmHFToken: {
							Type:          schema.TypeString,
							Optional:      true,
							Sensitive:     true,
							Description:   "HuggingFace token. Mutually exclusive with secret_name.",
							ConflictsWith: []string{fieldAIHostedModelVllmConfig + ".0." + fieldAIHostedModelVllmSecretName},
						},
					},
				},
			},
			fieldAIHostedModelHorizontalAS: {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Horizontal autoscaling settings.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						fieldAIHostedModelHASEnabled: {
							Type:     schema.TypeBool,
							Optional: true,
						},
						fieldAIHostedModelHASMinReplicas: {
							Type:     schema.TypeInt,
							Required: true,
						},
						fieldAIHostedModelHASMaxReplicas: {
							Type:     schema.TypeInt,
							Required: true,
						},
						fieldAIHostedModelHASTargetMetric: {
							Type:     schema.TypeString,
							Required: true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
								string(ai_optimizer.HorizontalAutoscalingTargetMetricGPUCACHEUSAGEPERCENTAGE),
								string(ai_optimizer.HorizontalAutoscalingTargetMetricNUMBEROFREQUESTSWAITING),
							}, false)),
						},
						fieldAIHostedModelHASTargetValue: {
							Type:     schema.TypeFloat,
							Required: true,
						},
					},
				},
			},
			fieldAIHostedModelHibernation: {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Automatic hibernation settings.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						fieldAIHostedModelHibEnabled: {
							Type:     schema.TypeBool,
							Optional: true,
						},
						fieldAIHostedModelHibResumeCondition: {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem:     hibernationConditionSchema,
						},
						fieldAIHostedModelHibernateCondition: {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem:     hibernationConditionSchema,
						},
					},
				},
			},
			fieldAIHostedModelFallback: {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Fallback model settings.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						fieldAIHostedModelFallbackEnabled: {
							Type:     schema.TypeBool,
							Optional: true,
						},
						fieldAIHostedModelFallbackProviderID: {
							Type:     schema.TypeString,
							Optional: true,
						},
						fieldAIHostedModelFallbackModel: {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			fieldAIHostedModelStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Hosted model status.",
			},
			fieldAIHostedModelStatusReason: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Reason for the current status.",
			},
			fieldAIHostedModelCurrentReplicas: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Current number of replicas.",
			},
			fieldAIHostedModelCloudProvider: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cloud provider.",
			},
			fieldAIHostedModelNamespace: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kubernetes namespace.",
			},
		},
	}
}

func resourceAIHostedModelCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).aiOptimizerClient

	orgID, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching organization ID: %w", err))
	}

	clusterID := d.Get(fieldAIHostedModelClusterID).(string)
	port := int32(d.Get(fieldAIHostedModelPort).(int))

	body := ai_optimizer.HostedModel{
		ClusterId:    clusterID,
		ModelSpecsId: d.Get(fieldAIHostedModelModelSpecsID).(string),
		Service:      d.Get(fieldAIHostedModelService).(string),
		Port:         port,
	}

	if v, ok := d.GetOk(fieldAIHostedModelNodeTemplateName); ok {
		s := v.(string)
		body.NodeTemplateName = &s
	}
	if v, ok := d.GetOk(fieldAIHostedModelEdgeLocationIDs); ok {
		body.EdgeLocations = expandEdgeLocations(v.([]interface{}))
	}
	if v, ok := d.GetOk(fieldAIHostedModelVllmConfig); ok {
		body.VllmConfig = expandVllmConfig(v.([]interface{}))
	}
	if v, ok := d.GetOk(fieldAIHostedModelHorizontalAS); ok {
		body.HorizontalAutoscaling = expandHorizontalAutoscaling(v.([]interface{}))
	}
	if v, ok := d.GetOk(fieldAIHostedModelHibernation); ok {
		body.Hibernation = expandHibernation(v.([]interface{}))
	}
	if v, ok := d.GetOk(fieldAIHostedModelFallback); ok {
		body.Fallback = expandFallback(v.([]interface{}))
	}

	tflog.Debug(ctx, "Creating AI hosted model", map[string]any{"cluster_id": clusterID})

	resp, err := client.HostedModelsAPICreateHostedModelWithResponse(ctx, orgID, clusterID, body)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("creating hosted model: %w", err))
	}

	if resp.JSON200 == nil || resp.JSON200.Id == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from create hosted model"))
	}

	d.SetId(*resp.JSON200.Id)

	return resourceAIHostedModelRead(ctx, d, meta)
}

func resourceAIHostedModelRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).aiOptimizerClient

	orgID, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching organization ID: %w", err))
	}

	clusterID := d.Get(fieldAIHostedModelClusterID).(string)
	if clusterID == "" {
		return diag.Errorf("cluster_id is required but was empty in state")
	}
	modelID := d.Id()

	model, err := findHostedModelByID(ctx, client, orgID, clusterID, modelID)
	if err != nil {
		if !d.IsNewResource() && isNotFoundError(err) {
			tflog.Warn(ctx, "AI hosted model not found, removing from state", map[string]any{"id": modelID})
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("reading hosted model: %w", err))
	}

	if model == nil {
		if d.IsNewResource() {
			return diag.FromErr(fmt.Errorf("hosted model %q not found after create", modelID))
		}
		tflog.Warn(ctx, "AI hosted model not found, removing from state", map[string]any{"id": modelID})
		d.SetId("")
		return nil
	}

	return setAIHostedModelData(d, model)
}

func findHostedModelByID(ctx context.Context, client ai_optimizer.ClientWithResponsesInterface, orgID, clusterID, modelID string) (*ai_optimizer.HostedModel, error) {
	var cursor *string
	for {
		params := &ai_optimizer.HostedModelsAPIListHostedModelsParams{
			PageCursor: cursor,
		}
		resp, err := client.HostedModelsAPIListHostedModelsWithResponse(ctx, orgID, clusterID, params)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode() == http.StatusNotFound {
			return nil, &notFoundError{msg: fmt.Sprintf("cluster %q not found", clusterID)}
		}
		if err := sdk.CheckOKResponse(resp, nil); err != nil {
			return nil, err
		}
		if resp.JSON200 == nil {
			return nil, nil
		}
		for i := range resp.JSON200.Items {
			item := &resp.JSON200.Items[i]
			if item.Id != nil && *item.Id == modelID {
				return item, nil
			}
		}
		if resp.JSON200.NextPageCursor == nil || *resp.JSON200.NextPageCursor == "" {
			break
		}
		cursor = resp.JSON200.NextPageCursor
	}
	return nil, nil
}

type notFoundError struct {
	msg string
}

func (e *notFoundError) Error() string {
	return e.msg
}

func isNotFoundError(err error) bool {
	_, ok := err.(*notFoundError)
	return ok
}

func resourceAIHostedModelUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).aiOptimizerClient

	orgID, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching organization ID: %w", err))
	}

	clusterID := d.Get(fieldAIHostedModelClusterID).(string)
	if clusterID == "" {
		return diag.Errorf("cluster_id is required but was empty in state")
	}

	body := ai_optimizer.HostedModelUpdate{
		VllmConfig:            expandVllmConfig(d.Get(fieldAIHostedModelVllmConfig).([]interface{})),
		HorizontalAutoscaling: expandHorizontalAutoscaling(d.Get(fieldAIHostedModelHorizontalAS).([]interface{})),
		Hibernation:           expandHibernation(d.Get(fieldAIHostedModelHibernation).([]interface{})),
		Fallback:              expandFallback(d.Get(fieldAIHostedModelFallback).([]interface{})),
		EdgeLocations:         expandEdgeLocations(d.Get(fieldAIHostedModelEdgeLocationIDs).([]interface{})),
	}

	if v, ok := d.GetOk(fieldAIHostedModelNodeTemplateName); ok {
		s := v.(string)
		body.NodeTemplateName = &s
	}

	tflog.Debug(ctx, "Updating AI hosted model", map[string]any{"id": d.Id()})

	resp, err := client.HostedModelsAPIUpdateHostedModelWithResponse(ctx, orgID, clusterID, d.Id(), body)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("updating hosted model: %w", err))
	}

	return resourceAIHostedModelRead(ctx, d, meta)
}

func resourceAIHostedModelDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).aiOptimizerClient

	orgID, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching organization ID: %w", err))
	}

	clusterID := d.Get(fieldAIHostedModelClusterID).(string)
	if clusterID == "" {
		return diag.Errorf("cluster_id is required but was empty in state")
	}

	tflog.Debug(ctx, "Deleting AI hosted model", map[string]any{"id": d.Id()})

	resp, err := client.HostedModelsAPIDeleteHostedModelWithResponse(ctx, orgID, clusterID, d.Id())
	if resp != nil && resp.StatusCode() == http.StatusNotFound {
		return nil
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("deleting hosted model: %w", err))
	}

	return nil
}

func resourceAIHostedModelImporter(_ context.Context, d *schema.ResourceData, _ any) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("import ID must be in format {cluster_id}/{hosted_model_id}, got: %q", d.Id())
	}

	if err := d.Set(fieldAIHostedModelClusterID, parts[0]); err != nil {
		return nil, fmt.Errorf("setting cluster_id: %w", err)
	}
	d.SetId(parts[1])

	return []*schema.ResourceData{d}, nil
}

func setAIHostedModelData(d *schema.ResourceData, m *ai_optimizer.HostedModel) diag.Diagnostics {
	if err := d.Set(fieldAIHostedModelClusterID, m.ClusterId); err != nil {
		return diag.FromErr(fmt.Errorf("setting cluster_id: %w", err))
	}
	if err := d.Set(fieldAIHostedModelModelSpecsID, m.ModelSpecsId); err != nil {
		return diag.FromErr(fmt.Errorf("setting model_specs_id: %w", err))
	}
	if err := d.Set(fieldAIHostedModelService, m.Service); err != nil {
		return diag.FromErr(fmt.Errorf("setting service: %w", err))
	}
	if err := d.Set(fieldAIHostedModelPort, int(m.Port)); err != nil {
		return diag.FromErr(fmt.Errorf("setting port: %w", err))
	}
	if m.NodeTemplateName != nil {
		if err := d.Set(fieldAIHostedModelNodeTemplateName, *m.NodeTemplateName); err != nil {
			return diag.FromErr(fmt.Errorf("setting node_template_name: %w", err))
		}
	}
	if m.Region != nil {
		if err := d.Set(fieldAIHostedModelRegion, *m.Region); err != nil {
			return diag.FromErr(fmt.Errorf("setting region: %w", err))
		}
	}
	if m.Status != nil {
		if err := d.Set(fieldAIHostedModelStatus, string(*m.Status)); err != nil {
			return diag.FromErr(fmt.Errorf("setting status: %w", err))
		}
	}
	if m.StatusReason != nil {
		if err := d.Set(fieldAIHostedModelStatusReason, *m.StatusReason); err != nil {
			return diag.FromErr(fmt.Errorf("setting status_reason: %w", err))
		}
	}
	if m.CurrentReplicas != nil {
		if err := d.Set(fieldAIHostedModelCurrentReplicas, int(*m.CurrentReplicas)); err != nil {
			return diag.FromErr(fmt.Errorf("setting current_replicas: %w", err))
		}
	}
	if m.CloudProvider != nil {
		if err := d.Set(fieldAIHostedModelCloudProvider, *m.CloudProvider); err != nil {
			return diag.FromErr(fmt.Errorf("setting cloud_provider: %w", err))
		}
	}
	if m.Namespace != nil {
		if err := d.Set(fieldAIHostedModelNamespace, *m.Namespace); err != nil {
			return diag.FromErr(fmt.Errorf("setting namespace: %w", err))
		}
	}
	if m.EdgeLocations != nil {
		if err := d.Set(fieldAIHostedModelEdgeLocationIDs, flattenEdgeLocations(m.EdgeLocations)); err != nil {
			return diag.FromErr(fmt.Errorf("setting edge_location_ids: %w", err))
		}
	}
	if m.VllmConfig != nil {
		if err := d.Set(fieldAIHostedModelVllmConfig, flattenVllmConfig(m.VllmConfig)); err != nil {
			return diag.FromErr(fmt.Errorf("setting vllm_config: %w", err))
		}
	}
	if m.HorizontalAutoscaling != nil {
		if err := d.Set(fieldAIHostedModelHorizontalAS, flattenHorizontalAutoscaling(m.HorizontalAutoscaling)); err != nil {
			return diag.FromErr(fmt.Errorf("setting horizontal_autoscaling: %w", err))
		}
	}
	if m.Hibernation != nil {
		if err := d.Set(fieldAIHostedModelHibernation, flattenHibernation(m.Hibernation)); err != nil {
			return diag.FromErr(fmt.Errorf("setting hibernation: %w", err))
		}
	}
	if m.Fallback != nil {
		if err := d.Set(fieldAIHostedModelFallback, flattenFallback(m.Fallback)); err != nil {
			return diag.FromErr(fmt.Errorf("setting fallback: %w", err))
		}
	}
	return nil
}

func expandEdgeLocations(raw []interface{}) *ai_optimizer.EdgeLocations {
	if len(raw) == 0 {
		return nil
	}
	ids := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && s != "" {
			ids = append(ids, s)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return &ai_optimizer.EdgeLocations{Ids: &ids}
}

func flattenEdgeLocations(el *ai_optimizer.EdgeLocations) []string {
	if el == nil || el.Ids == nil {
		return nil
	}
	return *el.Ids
}

func expandVllmConfig(raw []interface{}) *ai_optimizer.VLLMConfig {
	if len(raw) == 0 {
		return nil
	}
	cfg := raw[0].(map[string]interface{})
	out := &ai_optimizer.VLLMConfig{}
	if v, ok := cfg[fieldAIHostedModelVllmSecretName].(string); ok && v != "" {
		out.SecretName = &v
	}
	if v, ok := cfg[fieldAIHostedModelVllmHFToken].(string); ok && v != "" {
		out.HuggingFaceToken = &v
	}
	return out
}

func flattenVllmConfig(cfg *ai_optimizer.VLLMConfig) []map[string]interface{} {
	if cfg == nil {
		return nil
	}
	m := map[string]interface{}{}
	if cfg.SecretName != nil {
		m[fieldAIHostedModelVllmSecretName] = *cfg.SecretName
	}
	if cfg.HuggingFaceToken != nil {
		m[fieldAIHostedModelVllmHFToken] = *cfg.HuggingFaceToken
	}
	return []map[string]interface{}{m}
}

func expandHorizontalAutoscaling(raw []interface{}) *ai_optimizer.HorizontalAutoscaling {
	if len(raw) == 0 {
		return nil
	}
	cfg := raw[0].(map[string]interface{})
	out := &ai_optimizer.HorizontalAutoscaling{
		MinReplicas:  int32(cfg[fieldAIHostedModelHASMinReplicas].(int)),
		MaxReplicas:  int32(cfg[fieldAIHostedModelHASMaxReplicas].(int)),
		TargetMetric: ai_optimizer.HorizontalAutoscalingTargetMetric(cfg[fieldAIHostedModelHASTargetMetric].(string)),
		TargetValue:  float32(cfg[fieldAIHostedModelHASTargetValue].(float64)),
	}
	if v, ok := cfg[fieldAIHostedModelHASEnabled].(bool); ok {
		out.Enabled = &v
	}
	return out
}

func flattenHorizontalAutoscaling(has *ai_optimizer.HorizontalAutoscaling) []map[string]interface{} {
	if has == nil {
		return nil
	}
	m := map[string]interface{}{
		fieldAIHostedModelHASMinReplicas:  int(has.MinReplicas),
		fieldAIHostedModelHASMaxReplicas:  int(has.MaxReplicas),
		fieldAIHostedModelHASTargetMetric: string(has.TargetMetric),
		fieldAIHostedModelHASTargetValue:  roundFloat64(float64(has.TargetValue), 6),
	}
	if has.Enabled != nil {
		m[fieldAIHostedModelHASEnabled] = *has.Enabled
	}
	return []map[string]interface{}{m}
}

func roundFloat64(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func expandHibernationCondition(raw []interface{}) ai_optimizer.HibernationCondition {
	if len(raw) == 0 {
		return ai_optimizer.HibernationCondition{}
	}
	cfg := raw[0].(map[string]interface{})
	out := ai_optimizer.HibernationCondition{
		Duration: cfg[fieldAIHostedModelConditionDuration].(string),
	}
	if v, ok := cfg[fieldAIHostedModelConditionReqCount].(int); ok && v != 0 {
		rc := uint32(v)
		out.RequestCount = &rc
	}
	return out
}

func flattenHibernationCondition(c ai_optimizer.HibernationCondition) []interface{} {
	m := map[string]interface{}{
		fieldAIHostedModelConditionDuration: c.Duration,
	}
	if c.RequestCount != nil {
		m[fieldAIHostedModelConditionReqCount] = int(*c.RequestCount)
	}
	return []interface{}{m}
}

func expandHibernation(raw []interface{}) *ai_optimizer.Hibernation {
	if len(raw) == 0 {
		return nil
	}
	cfg := raw[0].(map[string]interface{})
	out := &ai_optimizer.Hibernation{
		ResumeCondition:    expandHibernationCondition(cfg[fieldAIHostedModelHibResumeCondition].([]interface{})),
		HibernateCondition: expandHibernationCondition(cfg[fieldAIHostedModelHibernateCondition].([]interface{})),
	}
	if v, ok := cfg[fieldAIHostedModelHibEnabled].(bool); ok {
		out.Enabled = &v
	}
	return out
}

func flattenHibernation(h *ai_optimizer.Hibernation) []map[string]interface{} {
	if h == nil {
		return nil
	}
	m := map[string]interface{}{
		fieldAIHostedModelHibResumeCondition: flattenHibernationCondition(h.ResumeCondition),
		fieldAIHostedModelHibernateCondition: flattenHibernationCondition(h.HibernateCondition),
	}
	if h.Enabled != nil {
		m[fieldAIHostedModelHibEnabled] = *h.Enabled
	}
	return []map[string]interface{}{m}
}

func expandFallback(raw []interface{}) *ai_optimizer.Fallback {
	if len(raw) == 0 {
		return nil
	}
	cfg := raw[0].(map[string]interface{})
	out := &ai_optimizer.Fallback{}
	if v, ok := cfg[fieldAIHostedModelFallbackEnabled].(bool); ok {
		out.Enabled = &v
	}
	if v, ok := cfg[fieldAIHostedModelFallbackProviderID].(string); ok && v != "" {
		out.ProviderId = &v
	}
	if v, ok := cfg[fieldAIHostedModelFallbackModel].(string); ok && v != "" {
		out.Model = &v
	}
	return out
}

func flattenFallback(f *ai_optimizer.Fallback) []map[string]interface{} {
	if f == nil {
		return nil
	}
	m := map[string]interface{}{}
	if f.Enabled != nil {
		m[fieldAIHostedModelFallbackEnabled] = *f.Enabled
	}
	if f.ProviderId != nil {
		m[fieldAIHostedModelFallbackProviderID] = *f.ProviderId
	}
	if f.Model != nil {
		m[fieldAIHostedModelFallbackModel] = *f.Model
	}
	return []map[string]interface{}{m}
}
