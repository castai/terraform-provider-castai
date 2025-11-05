package castai

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni_provisioner"
)

const (
	FieldOmniEdgeOrganizationID  = "organization_id"
	FieldOmniEdgeClusterID       = "cluster_id"
	FieldOmniEdgeLocationID      = "edge_location_id"
	FieldOmniEdgeName            = "name"
	FieldOmniEdgeInstanceType    = "instance_type"
	FieldOmniEdgeSchedulingType  = "scheduling_type"
	FieldOmniEdgeZone            = "zone"
	FieldOmniEdgeArchitecture    = "node_architecture"
	FieldOmniEdgePhase           = "phase"
	FieldOmniEdgeProviderID      = "provider_id"
	FieldOmniEdgeKubernetesName  = "kubernetes_name"
	FieldOmniEdgeBootDiskGib     = "boot_disk_gib"
	FieldOmniEdgeImageID         = "image_id"
	FieldOmniEdgeInstanceLabels  = "instance_labels"
	FieldOmniEdgeKubernetesLabels = "kubernetes_labels"
	FieldOmniEdgeKubernetesTaints = "kubernetes_taints"
	FieldOmniEdgeGPUConfig       = "gpu_config"
	FieldOmniEdgeConfigurationID = "configuration_id"

	// GPU config fields
	FieldGPUCount          = "count"
	FieldGPUType           = "type"
	FieldGPUMIG            = "mig"
	FieldGPUTimeSharing    = "time_sharing"
	FieldMIGMemoryGB       = "memory_gb"
	FieldMIGPartitionSizes = "partition_sizes"
	FieldTimeSharingReplicas = "replicas"
)

func resourceOmniEdge() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceOmniEdgeCreate,
		ReadContext:   resourceOmniEdgeRead,
		UpdateContext: resourceOmniEdgeUpdate,
		DeleteContext: resourceOmniEdgeDelete,

		Schema: map[string]*schema.Schema{
			FieldOmniEdgeOrganizationID: {
				Type:        schema.TypeString,
				Description: "Organization ID",
				Required:    true,
				ForceNew:    true,
			},
			FieldOmniEdgeClusterID: {
				Type:        schema.TypeString,
				Description: "Omni cluster ID",
				Required:    true,
				ForceNew:    true,
			},
			FieldOmniEdgeLocationID: {
				Type:        schema.TypeString,
				Description: "Edge location ID",
				Required:    true,
				ForceNew:    true,
			},
			FieldOmniEdgeName: {
				Type:        schema.TypeString,
				Description: "Name of the edge",
				Optional:    true,
				ForceNew:    true,
			},
			FieldOmniEdgeInstanceType: {
				Type:        schema.TypeString,
				Description: "Instance type (e.g., m5.xlarge, n1-standard-4)",
				Required:    true,
				ForceNew:    true,
			},
			FieldOmniEdgeSchedulingType: {
				Type:         schema.TypeString,
				Description:  "Scheduling type (ON_DEMAND or SPOT)",
				Optional:     true,
				ForceNew:     true,
				Default:      "ON_DEMAND",
				ValidateFunc: validation.StringInSlice([]string{"ON_DEMAND", "SPOT"}, false),
			},
			FieldOmniEdgeZone: {
				Type:        schema.TypeString,
				Description: "Availability zone",
				Optional:    true,
				ForceNew:    true,
			},
			FieldOmniEdgeArchitecture: {
				Type:         schema.TypeString,
				Description:  "Node architecture",
				Optional:     true,
				ForceNew:     true,
				Default:      "X86_64",
				ValidateFunc: validation.StringInSlice([]string{"X86_64", "ARM64"}, false),
			},
			FieldOmniEdgeBootDiskGib: {
				Type:        schema.TypeInt,
				Description: "Boot disk size in GiB",
				Optional:    true,
				ForceNew:    true,
			},
			FieldOmniEdgeImageID: {
				Type:        schema.TypeString,
				Description: "Custom image ID",
				Optional:    true,
				ForceNew:    true,
			},
			FieldOmniEdgeInstanceLabels: {
				Type:        schema.TypeMap,
				Description: "Labels to apply to the cloud instance",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			FieldOmniEdgeKubernetesLabels: {
				Type:        schema.TypeMap,
				Description: "Labels to apply to the Kubernetes node",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			FieldOmniEdgeKubernetesTaints: {
				Type:        schema.TypeList,
				Description: "Taints to apply to the Kubernetes node",
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Description: "Taint key",
							Required:    true,
						},
						"value": {
							Type:        schema.TypeString,
							Description: "Taint value",
							Optional:    true,
						},
						"effect": {
							Type:         schema.TypeString,
							Description:  "Taint effect (NoSchedule, PreferNoSchedule, NoExecute)",
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"NoSchedule", "PreferNoSchedule", "NoExecute"}, false),
						},
					},
				},
			},
			FieldOmniEdgeGPUConfig: {
				Type:        schema.TypeList,
				Description: "GPU configuration",
				Optional:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldGPUCount: {
							Type:        schema.TypeInt,
							Description: "Number of GPUs",
							Required:    true,
						},
						FieldGPUType: {
							Type:        schema.TypeString,
							Description: "GPU type",
							Optional:    true,
						},
						FieldGPUMIG: {
							Type:        schema.TypeList,
							Description: "MIG (Multi-Instance GPU) configuration",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldMIGMemoryGB: {
										Type:        schema.TypeInt,
										Description: "Memory per MIG partition in GB",
										Optional:    true,
									},
									FieldMIGPartitionSizes: {
										Type:        schema.TypeList,
										Description: "MIG partition sizes",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						FieldGPUTimeSharing: {
							Type:        schema.TypeList,
							Description: "Time-sharing configuration",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldTimeSharingReplicas: {
										Type:        schema.TypeInt,
										Description: "Number of time-sharing replicas",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
			FieldOmniEdgeConfigurationID: {
				Type:        schema.TypeString,
				Description: "Edge configuration ID to use",
				Optional:    true,
				ForceNew:    true,
			},
			FieldOmniEdgePhase: {
				Type:        schema.TypeString,
				Description: "Current phase of the edge",
				Computed:    true,
			},
			FieldOmniEdgeProviderID: {
				Type:        schema.TypeString,
				Description: "Cloud provider instance ID",
				Computed:    true,
			},
			FieldOmniEdgeKubernetesName: {
				Type:        schema.TypeString,
				Description: "Kubernetes node name",
				Computed:    true,
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOmniEdgeCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniProvisionerClient

	organizationID := data.Get(FieldOmniEdgeOrganizationID).(string)
	clusterID := data.Get(FieldOmniEdgeClusterID).(string)
	edgeLocationID := data.Get(FieldOmniEdgeLocationID).(string)

	instanceType := data.Get(FieldOmniEdgeInstanceType).(string)
	schedulingType := omni_provisioner.CastaiOmniProvisionerV2beta1EdgeSchedulingType(data.Get(FieldOmniEdgeSchedulingType).(string))
	architecture := omni_provisioner.CastaiOmniProvisionerV2beta1EdgeNodeArchitecture(data.Get(FieldOmniEdgeArchitecture).(string))

	req := omni_provisioner.CreateEdgeJSONRequestBody{
		InstanceType:     &instanceType,
		SchedulingType:   &schedulingType,
		NodeArchitecture: &architecture,
	}

	if v, ok := data.GetOk(FieldOmniEdgeName); ok {
		name := v.(string)
		req.Name = &name
	}

	if v, ok := data.GetOk(FieldOmniEdgeZone); ok {
		zone := v.(string)
		req.Zone = &zone
	}

	if v, ok := data.GetOk(FieldOmniEdgeBootDiskGib); ok {
		bootDiskGib := int32(v.(int))
		req.BootDiskGib = &bootDiskGib
	}

	if v, ok := data.GetOk(FieldOmniEdgeImageID); ok {
		imageID := v.(string)
		req.ImageId = &imageID
	}

	if v, ok := data.GetOk(FieldOmniEdgeInstanceLabels); ok {
		labels := make(map[string]string)
		for k, val := range v.(map[string]interface{}) {
			labels[k] = val.(string)
		}
		req.InstanceLabels = &labels
	}

	if v, ok := data.GetOk(FieldOmniEdgeKubernetesLabels); ok {
		labels := make(map[string]string)
		for k, val := range v.(map[string]interface{}) {
			labels[k] = val.(string)
		}
		req.KubernetesLabels = &labels
	}

	if v, ok := data.GetOk(FieldOmniEdgeKubernetesTaints); ok {
		taints := make([]omni_provisioner.CastaiInventoryV1beta1Taint, 0)
		for _, taint := range v.([]interface{}) {
			taintMap := taint.(map[string]interface{})
			key := taintMap["key"].(string)
			effect := taintMap["effect"].(string)

			t := omni_provisioner.CastaiInventoryV1beta1Taint{
				Key:    &key,
				Effect: &effect,
			}

			if val, ok := taintMap["value"].(string); ok && val != "" {
				t.Value = &val
			}

			taints = append(taints, t)
		}
		req.KubernetesTaints = &taints
	}

	if v, ok := data.GetOk(FieldOmniEdgeGPUConfig); ok && len(v.([]interface{})) > 0 {
		gpuConfigMap := v.([]interface{})[0].(map[string]interface{})

		count := int32(gpuConfigMap[FieldGPUCount].(int))
		gpuConfig := omni_provisioner.CastaiOmniProvisionerV2beta1GpuConfig{
			Count: &count,
		}

		if gpuType, ok := gpuConfigMap[FieldGPUType].(string); ok && gpuType != "" {
			gpuConfig.Type = &gpuType
		}

		if migList, ok := gpuConfigMap[FieldGPUMIG].([]interface{}); ok && len(migList) > 0 {
			migMap := migList[0].(map[string]interface{})
			mig := omni_provisioner.CastaiOmniProvisionerV2beta1MigConfig{}

			if memGB, ok := migMap[FieldMIGMemoryGB].(int); ok && memGB > 0 {
				mem := int32(memGB)
				mig.MemoryGb = &mem
			}

			if partitions, ok := migMap[FieldMIGPartitionSizes].([]interface{}); ok && len(partitions) > 0 {
				sizes := make([]string, 0)
				for _, p := range partitions {
					sizes = append(sizes, p.(string))
				}
				mig.PartitionSizes = &sizes
			}

			gpuConfig.Mig = &mig
		}

		if tsList, ok := gpuConfigMap[FieldGPUTimeSharing].([]interface{}); ok && len(tsList) > 0 {
			tsMap := tsList[0].(map[string]interface{})
			replicas := int32(tsMap[FieldTimeSharingReplicas].(int))

			gpuConfig.TimeSharing = &omni_provisioner.CastaiOmniProvisionerV2beta1TimeSharingConfig{
				Replicas: &replicas,
			}
		}

		req.GpuConfig = &gpuConfig
	}

	if v, ok := data.GetOk(FieldOmniEdgeConfigurationID); ok {
		configID := v.(string)
		req.ConfigurationId = &configID
	}

	resp, err := client.CreateEdgeWithResponse(ctx, organizationID, clusterID, edgeLocationID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("creating edge: %w", err))
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return diag.FromErr(fmt.Errorf("creating edge: unexpected status code %d", resp.StatusCode()))
	}

	if resp.JSON200 == nil || resp.JSON200.Id == nil {
		return diag.Errorf("edge response is nil or missing ID")
	}

	data.SetId(*resp.JSON200.Id)

	return resourceOmniEdgeRead(ctx, data, meta)
}

func resourceOmniEdgeRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniProvisionerClient

	organizationID := data.Get(FieldOmniEdgeOrganizationID).(string)
	clusterID := data.Get(FieldOmniEdgeClusterID).(string)
	edgeLocationID := data.Get(FieldOmniEdgeLocationID).(string)
	edgeID := data.Id()

	resp, err := client.GetEdgeWithResponse(ctx, organizationID, clusterID, edgeLocationID, edgeID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("getting edge: %w", err))
	}

	if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Edge (%s) not found, removing from state", edgeID)
		data.SetId("")
		return nil
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("getting edge: unexpected status code %d", resp.StatusCode()))
	}

	edge := resp.JSON200
	if edge == nil {
		return diag.Errorf("edge response is nil")
	}

	if edge.Name != nil {
		if err := data.Set(FieldOmniEdgeName, *edge.Name); err != nil {
			return diag.FromErr(fmt.Errorf("setting name: %w", err))
		}
	}

	if edge.InstanceType != nil {
		if err := data.Set(FieldOmniEdgeInstanceType, *edge.InstanceType); err != nil {
			return diag.FromErr(fmt.Errorf("setting instance_type: %w", err))
		}
	}

	if edge.SchedulingType != nil {
		if err := data.Set(FieldOmniEdgeSchedulingType, string(*edge.SchedulingType)); err != nil {
			return diag.FromErr(fmt.Errorf("setting scheduling_type: %w", err))
		}
	}

	if edge.Zone != nil {
		if err := data.Set(FieldOmniEdgeZone, *edge.Zone); err != nil {
			return diag.FromErr(fmt.Errorf("setting zone: %w", err))
		}
	}

	if edge.NodeArchitecture != nil {
		if err := data.Set(FieldOmniEdgeArchitecture, string(*edge.NodeArchitecture)); err != nil {
			return diag.FromErr(fmt.Errorf("setting node_architecture: %w", err))
		}
	}

	if edge.Phase != nil {
		if err := data.Set(FieldOmniEdgePhase, string(*edge.Phase)); err != nil {
			return diag.FromErr(fmt.Errorf("setting phase: %w", err))
		}
	}

	if edge.ProviderId != nil {
		if err := data.Set(FieldOmniEdgeProviderID, *edge.ProviderId); err != nil {
			return diag.FromErr(fmt.Errorf("setting provider_id: %w", err))
		}
	}

	if edge.KubernetesName != nil {
		if err := data.Set(FieldOmniEdgeKubernetesName, *edge.KubernetesName); err != nil {
			return diag.FromErr(fmt.Errorf("setting kubernetes_name: %w", err))
		}
	}

	if edge.BootDiskGib != nil {
		if err := data.Set(FieldOmniEdgeBootDiskGib, int(*edge.BootDiskGib)); err != nil {
			return diag.FromErr(fmt.Errorf("setting boot_disk_gib: %w", err))
		}
	}

	if edge.ImageId != nil {
		if err := data.Set(FieldOmniEdgeImageID, *edge.ImageId); err != nil {
			return diag.FromErr(fmt.Errorf("setting image_id: %w", err))
		}
	}

	if edge.ConfigurationId != nil {
		if err := data.Set(FieldOmniEdgeConfigurationID, *edge.ConfigurationId); err != nil {
			return diag.FromErr(fmt.Errorf("setting configuration_id: %w", err))
		}
	}

	if edge.InstanceLabels != nil {
		if err := data.Set(FieldOmniEdgeInstanceLabels, *edge.InstanceLabels); err != nil {
			return diag.FromErr(fmt.Errorf("setting instance_labels: %w", err))
		}
	}

	if edge.KubernetesLabels != nil {
		if err := data.Set(FieldOmniEdgeKubernetesLabels, *edge.KubernetesLabels); err != nil {
			return diag.FromErr(fmt.Errorf("setting kubernetes_labels: %w", err))
		}
	}

	return nil
}

func resourceOmniEdgeUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Edges are immutable - any changes require recreation
	return resourceOmniEdgeRead(ctx, data, meta)
}

func resourceOmniEdgeDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniProvisionerClient

	organizationID := data.Get(FieldOmniEdgeOrganizationID).(string)
	clusterID := data.Get(FieldOmniEdgeClusterID).(string)
	edgeLocationID := data.Get(FieldOmniEdgeLocationID).(string)
	edgeID := data.Id()

	resp, err := client.DeleteEdgeWithResponse(ctx, organizationID, clusterID, edgeLocationID, edgeID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("deleting edge: %w", err))
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent && resp.StatusCode() != http.StatusNotFound {
		return diag.FromErr(fmt.Errorf("deleting edge: unexpected status code %d", resp.StatusCode()))
	}

	return nil
}
