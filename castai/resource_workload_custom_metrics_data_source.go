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

func resourceWorkloadCustomMetricsDataSource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceWorkloadCustomMetricsDataSourceCreate,
		ReadContext:   resourceWorkloadCustomMetricsDataSourceRead,
		UpdateContext: resourceWorkloadCustomMetricsDataSourceUpdate,
		DeleteContext: resourceWorkloadCustomMetricsDataSourceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: workloadCustomMetricsDataSourceImporter,
		},
		Description: "Manages a CAST AI workload custom metrics data source. " +
			"Custom metrics data sources allow CAST AI to collect and use non-standard metrics for workload optimization.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldClusterID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "CAST AI cluster ID.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Name of the custom metrics data source (1-63 characters).",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 63)),
			},
			"prometheus": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Prometheus data source configuration.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"url": {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "URL of the Prometheus server.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IsURLWithHTTPorHTTPS),
						},
						"timeout": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Timeout for Prometheus queries (e.g. \"30s\").",
						},
						"presets": {
							Type:     schema.TypeList,
							Optional: true,
							Description: "List of metric presets managed by CAST AI. Presets provide curated metric definitions " +
								"that are kept up to date automatically. This is the recommended approach for most users. " +
								"Currently available: \"jvm\".",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"metric": {
							Type:     schema.TypeList,
							Optional: true,
							Description: "Manually defined metrics. Use this for advanced use cases where presets " +
								"don't cover your needs. Each entry defines a single metric name and PromQL query. " +
								"To specify multiple queries for the same metric, use multiple entries with the same name.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:             schema.TypeString,
										Required:         true,
										Description:      "Name of the metric.",
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
									},
									"query": {
										Type:             schema.TypeString,
										Required:         true,
										Description:      "PromQL query for this metric.",
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
									},
								},
							},
						},
					},
				},
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Synchronization status of the data source (CONNECTING, CONNECTED, SYNCING, FAILED).",
			},
			"kube_resource_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the corresponding Kubernetes resource.",
			},
			"managed_by_cast": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the data source is managed by CAST AI.",
			},
		},
	}
}

func resourceWorkloadCustomMetricsDataSourceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)

	req, err := expandWorkloadCustomMetricsDataSourceCreate(d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "creating workload custom metrics data source", map[string]interface{}{
		"cluster_id": clusterID,
		"name":       req.Name,
	})

	resp, err := client.WorkloadOptimizationAPICreateCustomMetricsDataSourceWithResponse(ctx, clusterID, req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("creating workload custom metrics data source: %v", err)
	}

	d.SetId(resp.JSON200.Id)

	tflog.Info(ctx, "created workload custom metrics data source", map[string]interface{}{
		"cluster_id":  clusterID,
		"resource_id": resp.JSON200.Id,
	})

	return resourceWorkloadCustomMetricsDataSourceRead(ctx, d, meta)
}

func resourceWorkloadCustomMetricsDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)

	if d.Id() == "" {
		return diag.Errorf("workload custom metrics data source ID is not set")
	}

	tflog.Info(ctx, "reading workload custom metrics data source", map[string]interface{}{
		"cluster_id":  clusterID,
		"resource_id": d.Id(),
	})

	resp, err := client.WorkloadOptimizationAPIListCustomMetricsDataSourcesWithResponse(ctx, clusterID)
	if err != nil {
		return diag.Errorf("listing workload custom metrics data sources: %v", err)
	}

	if !d.IsNewResource() && resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "cluster not found, removing custom metrics data source from state", map[string]interface{}{
			"cluster_id":  clusterID,
			"resource_id": d.Id(),
		})
		d.SetId("")
		return nil
	}

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("listing workload custom metrics data sources: %v", err)
	}

	var found *sdk.WorkloadoptimizationV1CustomMetricsDataSource
	for i, item := range resp.JSON200.Items {
		if item.Id == d.Id() {
			found = &resp.JSON200.Items[i]
			break
		}
	}

	if found == nil {
		if !d.IsNewResource() {
			tflog.Warn(ctx, "workload custom metrics data source not found, removing from state", map[string]interface{}{
				"cluster_id":  clusterID,
				"resource_id": d.Id(),
			})
			d.SetId("")
			return nil
		}
		return diag.Errorf("workload custom metrics data source %s not found in cluster %s", d.Id(), clusterID)
	}

	return flattenWorkloadCustomMetricsDataSource(d, found)
}

func resourceWorkloadCustomMetricsDataSourceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)

	req, err := expandWorkloadCustomMetricsDataSourceUpdate(d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "updating workload custom metrics data source", map[string]interface{}{
		"cluster_id":  clusterID,
		"resource_id": d.Id(),
	})

	resp, err := client.WorkloadOptimizationAPIUpdateCustomMetricsDataSourceWithResponse(ctx, clusterID, d.Id(), req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("updating workload custom metrics data source: %v", err)
	}

	tflog.Info(ctx, "updated workload custom metrics data source", map[string]interface{}{
		"cluster_id":  clusterID,
		"resource_id": d.Id(),
	})

	return resourceWorkloadCustomMetricsDataSourceRead(ctx, d, meta)
}

func resourceWorkloadCustomMetricsDataSourceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)

	tflog.Info(ctx, "deleting workload custom metrics data source", map[string]interface{}{
		"cluster_id":  clusterID,
		"resource_id": d.Id(),
	})

	resp, err := client.WorkloadOptimizationAPIDeleteCustomMetricsDataSourceWithResponse(ctx, clusterID, d.Id())
	if err != nil {
		return diag.Errorf("deleting workload custom metrics data source: %v", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		tflog.Debug(ctx, "workload custom metrics data source already deleted", map[string]interface{}{
			"cluster_id":  clusterID,
			"resource_id": d.Id(),
		})
		return nil
	}

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("deleting workload custom metrics data source: %v", err)
	}

	tflog.Info(ctx, "deleted workload custom metrics data source", map[string]interface{}{
		"cluster_id":  clusterID,
		"resource_id": d.Id(),
	})

	return nil
}

func workloadCustomMetricsDataSourceImporter(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	clusterID, id, found := strings.Cut(d.Id(), "/")
	if !found {
		return nil, fmt.Errorf("expected import id with format: <cluster_id>/<data_source_id>, got: %q", d.Id())
	}

	if err := d.Set(FieldClusterID, clusterID); err != nil {
		return nil, fmt.Errorf("setting cluster_id: %w", err)
	}

	d.SetId(id)

	return []*schema.ResourceData{d}, nil
}

func expandWorkloadCustomMetricsDataSourceCreate(d *schema.ResourceData) (sdk.WorkloadOptimizationAPICreateCustomMetricsDataSourceJSONRequestBody, error) {
	name := d.Get("name").(string)
	prometheusInput, err := expandPrometheusInputConfig(d)
	if err != nil {
		return sdk.WorkloadOptimizationAPICreateCustomMetricsDataSourceJSONRequestBody{}, err
	}

	return sdk.WorkloadOptimizationAPICreateCustomMetricsDataSourceJSONRequestBody{
		Name: name,
		Type: sdk.PROMETHEUS,
		Data: sdk.WorkloadoptimizationV1CustomMetricsDataSourceInput{
			Prometheus: prometheusInput,
		},
	}, nil
}

func expandWorkloadCustomMetricsDataSourceUpdate(d *schema.ResourceData) (sdk.WorkloadOptimizationAPIUpdateCustomMetricsDataSourceJSONRequestBody, error) {
	name := d.Get("name").(string)
	prometheusInput, err := expandPrometheusInputConfig(d)
	if err != nil {
		return sdk.WorkloadOptimizationAPIUpdateCustomMetricsDataSourceJSONRequestBody{}, err
	}

	return sdk.WorkloadOptimizationAPIUpdateCustomMetricsDataSourceJSONRequestBody{
		DataSource: &sdk.WorkloadoptimizationV1UpdateCustomMetricsDataSource{
			Name: &name,
			Data: &sdk.WorkloadoptimizationV1CustomMetricsDataSourceInput{
				Prometheus: prometheusInput,
			},
		},
		UpdateMask: "name,data",
	}, nil
}

func expandPrometheusInputConfig(d *schema.ResourceData) (*sdk.WorkloadoptimizationV1CustomMetricsDataSourceInputPrometheus, error) {
	promList := d.Get("prometheus").([]interface{})
	if len(promList) == 0 {
		return nil, fmt.Errorf("prometheus configuration is required")
	}

	promMap := promList[0].(map[string]interface{})
	url := promMap["url"].(string)

	dataSource := sdk.WorkloadoptimizationV1CustomMetricsDataSourceInputPrometheusDataSource{
		Url: url,
	}

	if v, ok := promMap["timeout"].(string); ok && v != "" {
		dataSource.Timeout = &v
	}

	result := &sdk.WorkloadoptimizationV1CustomMetricsDataSourceInputPrometheus{
		DataSource: dataSource,
	}

	var metrics *sdk.WorkloadoptimizationV1CustomMetricsDataSourceInputPrometheusMetrics

	// Expand presets.
	if v, ok := promMap["presets"].([]interface{}); ok && len(v) > 0 {
		presets := make([]string, len(v))
		for i, p := range v {
			presets[i] = p.(string)
		}
		metrics = &sdk.WorkloadoptimizationV1CustomMetricsDataSourceInputPrometheusMetrics{}
		metrics.Presets = &presets
	}

	if v, ok := promMap["metric"].([]interface{}); ok && len(v) > 0 {
		customMetrics := make([]sdk.WorkloadoptimizationV1CustomMetricsDataSourceInputPrometheusMetric, 0, len(v))
		for _, m := range v {
			metricMap := m.(map[string]interface{})
			customMetrics = append(customMetrics, sdk.WorkloadoptimizationV1CustomMetricsDataSourceInputPrometheusMetric{
				Name:  metricMap["name"].(string),
				Query: metricMap["query"].(string),
			})
		}
		if metrics == nil {
			metrics = &sdk.WorkloadoptimizationV1CustomMetricsDataSourceInputPrometheusMetrics{}
		}
		metrics.Manual = &customMetrics
	}

	result.Metrics = metrics

	return result, nil
}

func flattenWorkloadCustomMetricsDataSource(d *schema.ResourceData, ds *sdk.WorkloadoptimizationV1CustomMetricsDataSource) diag.Diagnostics {
	if err := d.Set("name", ds.Name); err != nil {
		return diag.Errorf("setting name: %v", err)
	}
	if err := d.Set("status", string(ds.Status)); err != nil {
		return diag.Errorf("setting status: %v", err)
	}
	if err := d.Set("kube_resource_name", ds.KubeResourceName); err != nil {
		return diag.Errorf("setting kube_resource_name: %v", err)
	}
	if err := d.Set("managed_by_cast", ds.ManagedByCast); err != nil {
		return diag.Errorf("setting managed_by_cast: %v", err)
	}

	if ds.Data.Prometheus != nil {
		prom := flattenPrometheusConfig(ds.Data.Prometheus)
		if err := d.Set("prometheus", prom); err != nil {
			return diag.Errorf("setting prometheus: %v", err)
		}
	}

	return nil
}

func flattenPrometheusConfig(prom *sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheus) []interface{} {
	promMap := map[string]interface{}{
		"url":     prom.DataSource.Url,
		"timeout": "",
	}

	if prom.DataSource.Timeout != nil {
		promMap["timeout"] = *prom.DataSource.Timeout
	}

	if prom.Metrics != nil && prom.Metrics.Presets != nil {
		promMap["presets"] = *prom.Metrics.Presets
	} else {
		promMap["presets"] = []string{}
	}

	if prom.Metrics != nil && prom.Metrics.Resolved != nil {
		var metrics []interface{}
		for _, rm := range *prom.Metrics.Resolved {
			for _, q := range rm.Queries {
				if q.Origin != sdk.WorkloadoptimizationV1CustomMetricsDataSourceDataPrometheusMetricsResolvedMetricQueryOriginMANUAL {
					continue
				}
				metrics = append(metrics, map[string]interface{}{
					"name":  rm.Name,
					"query": q.Value,
				})
			}
		}
		promMap["metric"] = metrics
	} else {
		promMap["metric"] = []interface{}{}
	}

	return []interface{}{promMap}
}
