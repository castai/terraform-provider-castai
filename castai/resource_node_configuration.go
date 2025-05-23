package castai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	castval "github.com/castai/terraform-provider-castai/castai/validation"
)

const (
	FieldNodeConfigurationName                         = "name"
	FieldNodeConfigurationDiskCpuRatio                 = "disk_cpu_ratio"
	FieldNodeConfigurationMinDiskSize                  = "min_disk_size"
	FieldNodeConfigurationDrainTimeoutSec              = "drain_timeout_sec"
	FieldNodeConfigurationSubnets                      = "subnets"
	FieldNodeConfigurationSSHPublicKey                 = "ssh_public_key"
	FieldNodeConfigurationImage                        = "image"
	FieldNodeConfigurationTags                         = "tags"
	FieldNodeConfigurationInitScript                   = "init_script"
	FieldNodeConfigurationContainerRuntime             = "container_runtime"
	FieldNodeConfigurationDockerConfig                 = "docker_config"
	FieldNodeConfigurationKubeletConfig                = "kubelet_config"
	FieldNodeConfigurationAKS                          = "aks"
	FieldNodeConfigurationEKS                          = "eks"
	FieldNodeConfigurationKOPS                         = "kops"
	FieldNodeConfigurationGKE                          = "gke"
	FieldNodeConfigurationEKSTargetGroup               = "target_group"
	FieldNodeConfigurationAKSImageFamily               = "aks_image_family"
	FieldNodeConfigurationAKSEphemeralOSDisk           = "ephemeral_os_disk"
	FieldNodeConfigurationEKSImageFamily               = "eks_image_family"
	FieldNodeConfigurationLoadbalancers                = "loadbalancers"
	FieldNodeConfigurationAKSLoadbalancerIPPools       = "ip_based_backend_pools"
	FieldNodeConfigurationAKSLoadbalancerNICPools      = "nic_based_backend_pools"
	FieldNodeConfigurationAKSNetworkSecurityGroup      = "network_security_group"
	FieldNodeConfigurationAKSApplicationSecurityGroups = "application_security_groups"
	FieldNodeConfigurationAKSPublicIP                  = "public_ip"
)

const (
	eksImageFamilyAL2          = "al2"
	eksImageFamilyAL2023       = "al2023"
	eksImageFamilyBottlerocket = "bottlerocket"
)

const (
	aksImageFamilyUbuntu                  = "ubuntu"
	aksImageFamilyAzureLinux              = "azure-linux"
	aksImageFamilyWindows2019             = "windows2019"
	aksImageFamilyWindows2022             = "windows2022"
	aksEphemeralDiskPlacementCacheDisk    = "cacheDisk"
	aksEphemeralDiskPlacementResourceDisk = "resourceDisk"
	aksDiskCacheReadOnly                  = "ReadOnly"
	aksDiskCacheReadWrite                 = "ReadWrite"
)

func resourceNodeConfiguration() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNodeConfigurationCreate,
		ReadContext:   resourceNodeConfigurationRead,
		UpdateContext: resourceNodeConfigurationUpdate,
		DeleteContext: resourceNodeConfigurationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: nodeConfigStateImporter,
		},
		Description: "Create node configuration for given cluster. Node configuration [reference](https://docs.cast.ai/docs/node-configuration)",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldClusterID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "CAST AI cluster id",
			},
			FieldNodeConfigurationName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "Name of the node configuration. It must be unique within the cluster. In case of cluster is reonboarded to Terraform, when previously it was onboarded with CAST AI UI, and the name corresponds to previously created node configuration this resource will override existing configuration instead of creating new.",
			},
			FieldNodeConfigurationDiskCpuRatio: {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          0,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(0)),
				Description:      "Disk to CPU ratio. Sets the number of GiBs to be added for every CPU on the node. Defaults to 0",
			},
			FieldNodeConfigurationDrainTimeoutSec: {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          0,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 3600)),
				Description:      "Timeout in seconds for draining the node. Defaults to 0",
			},
			FieldNodeConfigurationMinDiskSize: {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          100,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(30, 65536)),
				Description:      "Minimal disk size in GiB. Defaults to 100, min 30, max 65536",
			},
			FieldNodeConfigurationSubnets: {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Subnet ids to be used for provisioned nodes",
			},
			FieldNodeConfigurationSSHPublicKey: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "SSH public key to be used for provisioned nodes",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsBase64),
			},
			FieldNodeConfigurationImage: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Image to be used while provisioning the node. If nothing is provided will be resolved to latest available image based on Image family, Kubernetes version and node architecture if possible. See Cast.ai documentation for details.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldNodeConfigurationTags: {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Tags to be added on cloud instances for provisioned nodes",
			},
			FieldNodeConfigurationInitScript: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Init script to be run on your instance at launch. Should not contain any sensitive data. Value should be base64 encoded",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsBase64),
			},
			FieldNodeConfigurationContainerRuntime: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Optional container runtime to be used by kubelet. Applicable for EKS only.  Supported values include: `dockerd`, `containerd`",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"dockerd", "containerd"}, true)),
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					return strings.EqualFold(oldValue, newValue)
				},
			},
			FieldNodeConfigurationDockerConfig: {
				Type:     schema.TypeString,
				Optional: true,
				Description: "Optional docker daemon configuration properties in JSON format. Provide only properties that you want to override. Applicable for EKS only. " +
					"[Available values](https://docs.docker.com/engine/reference/commandline/dockerd/#daemon-configuration-file)",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsJSON),
			},
			FieldNodeConfigurationKubeletConfig: {
				Type:     schema.TypeString,
				Optional: true,
				Description: "Optional kubelet configuration properties in JSON format. Provide only properties that you want to override. Applicable for EKS only. " +
					"[Available values](https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/)",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsJSON),
			},
			FieldNodeConfigurationEKS: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"security_groups": {
							Type:     schema.TypeList,
							Required: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "Cluster's security groups configuration for CAST provisioned nodes",
						},
						"node_group_arn": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "Cluster's node group ARN used for CAST provisioned node pools. Required for hibernate/resume functionality",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},

						"dns_cluster_ip": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "IP address to use for DNS queries within the cluster",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IsIPv4Address),
						},
						"instance_profile_arn": {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "Cluster's instance profile ARN used for CAST provisioned nodes",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
						"key_pair_id": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "AWS key pair ID to be used for CAST provisioned nodes. Has priority over ssh_public_key",
							ValidateDiagFunc: castval.ValidKeyPairFormat(),
						},
						"volume_type": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "AWS EBS volume type to be used for CAST provisioned nodes. One of: gp3, gp2, io1, io2",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"gp3", "gp2", "io1", "io2"}, true)),
						},
						"volume_iops": {
							Type:             schema.TypeInt,
							Optional:         true,
							Description:      "AWS EBS volume IOPS to be used for CAST provisioned nodes",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(100, 100000)),
						},
						"volume_throughput": {
							Type:             schema.TypeInt,
							Optional:         true,
							Description:      "AWS EBS volume throughput in MiB/s to be used for CAST provisioned nodes",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(125, 1000)),
						},
						"imds_v1": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
							Description: "When the value is true both IMDSv1 and IMDSv2 are enabled. Setting the value to false disables permanently IMDSv1 and might affect legacy workloads running on the node created with this configuration. The default is true if the flag isn't provided",
						},
						"imds_hop_limit": {
							Type:             schema.TypeInt,
							Optional:         true,
							Default:          2,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(2)),
							Description:      "Allow configure the IMDSv2 hop limit, the default is 2",
						},
						"volume_kms_key_arn": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "AWS KMS key ARN for encrypting EBS volume attached to the node",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile(`arn:aws:kms:.*`), "Must be a valid KMS key ARN")),
						},
						"max_pods_per_node_formula": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Formula to calculate the maximum number of pods that can be run on a node. The following list of variables will be bound to a number before evaluating and can be used in the formula: NUM_MAX_NET_INTERFACES, NUM_IP_PER_INTERFACE, NUM_IP_PER_PREFIX, NUM_CPU, NUM_RAM_GB .",
						},
						"ips_per_prefix": {
							Type:             schema.TypeInt,
							Optional:         true,
							Default:          nil,
							Description:      "Number of IPs per prefix to be used for calculating max pods.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 256)),
							DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
								log.Printf("[DEBUG] changing 'ips_per_prefix' attribute for eks: old=%s, new=%s", oldValue, newValue)
								if oldValue == "1" && newValue == "0" {
									return true
								}
								return oldValue == newValue
							},
						},
						FieldNodeConfigurationEKSImageFamily: {
							Type:     schema.TypeString,
							Optional: true,
							Description: fmt.Sprintf(
								"Image OS Family to use when provisioning node in EKS. "+
									"If both image and family are provided, the system will use provided image and provisioning logic for given family. "+
									"If only image family is provided, the system will attempt to resolve the latest image from that family based on kubernetes version and node architecture. "+
									"If image family is omitted, a default family (based on cloud provider) will be used. "+
									"See Cast.ai documentation for details. Possible values: (%v)", strings.Join([]string{eksImageFamilyAL2, eksImageFamilyAL2023, eksImageFamilyBottlerocket}, ",")),
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{eksImageFamilyAL2, eksImageFamilyAL2023, eksImageFamilyBottlerocket}, true)),
							DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
								return strings.EqualFold(oldValue, newValue)
							},
						},
						FieldNodeConfigurationEKSTargetGroup: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "AWS target groups configuration for CAST provisioned nodes",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"arn": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "AWS target group ARN for CAST provisioned nodes",
									},
									"port": {
										Type:             schema.TypeInt,
										Optional:         true,
										Description:      "Port for AWS target group for CAST provisioned nodes",
										ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 65535)),
									},
								},
							},
						},
					},
				},
			},
			FieldNodeConfigurationAKS: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_pods_per_node": {
							Type:             schema.TypeInt,
							Default:          30,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(10, 250)),
							Description:      "Maximum number of pods that can be run on a node, which affects how many IP addresses you will need for each node. Defaults to 30",
						},
						"os_disk_type": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "Type of managed os disk attached to the node. (See [disk types](https://learn.microsoft.com/en-us/azure/virtual-machines/disks-types)). One of: standard, standard-ssd, premium-ssd (ultra and premium-ssd-v2 are not supported for os disk)",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"standard", "standard-ssd", "premium-ssd"}, false)),
						},
						FieldNodeConfigurationAKSImageFamily: {
							Type:     schema.TypeString,
							Optional: true,
							Description: fmt.Sprintf(
								"Image OS Family to use when provisioning node in AKS. "+
									"If both image and family are provided, the system will use provided image and provisioning logic for given family. "+
									"If only image family is provided, the system will attempt to resolve the latest image from that family based on kubernetes version and node architecture. "+
									"If image family is omitted, a default family (based on cloud provider) will be used. "+
									"See Cast.ai documentation for details. Possible values: (%v)", strings.Join([]string{aksImageFamilyUbuntu, aksImageFamilyAzureLinux, aksImageFamilyWindows2019, aksImageFamilyWindows2022}, ",")),
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{aksImageFamilyUbuntu, aksImageFamilyAzureLinux, aksImageFamilyWindows2019, aksImageFamilyWindows2022}, true)),
							DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
								return strings.EqualFold(oldValue, newValue)
							},
						},
						FieldNodeConfigurationAKSEphemeralOSDisk: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Ephemeral OS disk configuration for CAST provisioned nodes",
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"placement": {
										Type:                  schema.TypeString,
										Required:              true,
										Description:           "Placement of the ephemeral OS disk. One of: cacheDisk, resourceDisk",
										ValidateDiagFunc:      validation.ToDiagFunc(validation.StringInSlice([]string{aksEphemeralDiskPlacementCacheDisk, aksEphemeralDiskPlacementResourceDisk}, true)),
										DiffSuppressOnRefresh: true,
										DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
											return strings.EqualFold(oldValue, newValue)
										},
									},
									"cache": {
										Type:                  schema.TypeString,
										Optional:              true,
										Description:           "Cache type for the ephemeral OS disk. One of: ReadOnly, ReadWrite",
										ValidateDiagFunc:      validation.ToDiagFunc(validation.StringInSlice([]string{aksDiskCacheReadOnly, aksDiskCacheReadWrite}, true)),
										DiffSuppressOnRefresh: true,
										DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
											return strings.EqualFold(oldValue, newValue)
										},
									},
								},
							},
						},
						FieldNodeConfigurationAKSNetworkSecurityGroup: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Network security group to be used for provisioned nodes, if not provided default security group from `castpool` will be used",
						},
						FieldNodeConfigurationAKSApplicationSecurityGroups: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Application security groups to be used for provisioned nodes",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						FieldNodeConfigurationAKSPublicIP: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Public IP configuration for CAST AI provisioned nodes",
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"public_ip_prefix": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Public IP prefix to be used for provisioned nodes",
									},
									"tags": {
										Type:     schema.TypeMap,
										Optional: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
										ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
											allowedKeys := []string{
												"FirstPartyUsage",
												"NetworkDomain",
												"RoutingPreference",
											}

											tags := v.(map[string]interface{})
											for key := range tags {
												if !lo.Contains(allowedKeys, key) {
													errors = append(errors, fmt.Errorf("invalid key %q in %q, allowed keys: %v", key, k, allowedKeys))
												}
											}
											return
										},
									},
									"idle_timeout_in_minutes": {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "Idle timeout in minutes for public IP",
									},
								},
							},
						},
						FieldNodeConfigurationLoadbalancers: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Load balancer configuration for CAST provisioned nodes",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Description: "The full ID of the load balancer in azure.",
										Optional:    true, // Can't make it required as it was added after `name` so it'd be a breaking change.
									},
									"name": {
										Type:        schema.TypeString,
										Description: "Name of load balancer",
										Optional:    true,
										Deprecated:  "name field is deprecated, use ID instead. Will be removed in future versions.",
									},
									FieldNodeConfigurationAKSLoadbalancerIPPools: {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "IP based backend pools configuration for CAST provisioned nodes",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"name": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Name of the ip based backend pool",
												},
											},
										},
									},
									FieldNodeConfigurationAKSLoadbalancerNICPools: {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "NIC based backend pools configuration for CAST provisioned nodes.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"name": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Name of the NIC based backend pool",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			FieldNodeConfigurationKOPS: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key_pair_id": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "AWS key pair ID to be used for provisioned nodes. Has priority over sshPublicKey",
							ValidateDiagFunc: castval.ValidKeyPairFormat(),
						},
					},
				},
			},
			FieldNodeConfigurationGKE: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_pods_per_node": {
							Type:             schema.TypeInt,
							Default:          110,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(10, 256)),
							Description:      "Maximum number of pods that can be run on a node, which affects how many IP addresses you will need for each node. Defaults to 110",
						},
						"network_tags": {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							MaxItems:    64,
							Optional:    true,
							Description: "Network tags to be added on a VM. (See [network tags](https://cloud.google.com/vpc/docs/add-remove-network-tags))",
						},
						"disk_type": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "Type of boot disk attached to the node. (See [disk types](https://cloud.google.com/compute/docs/disks#pdspecs)). One of: pd-standard, pd-balanced, pd-ssd, pd-extreme ",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"pd-standard", "pd-balanced", "pd-ssd", "pd-extreme"}, false)),
						},
						"zones": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "List of preferred availability zones to choose from when provisioning new nodes.",
							Deprecated:  "The argument will be moved into node template.",
						},
						"use_ephemeral_storage_local_ssd": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     nil,
							Description: "Use ephemeral storage local SSD. Defaults to false",
						},
						"secondary_ip_range": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Secondary IP range configuration for pods in GKE nodes",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"range_name": {
										Type:             schema.TypeString,
										Required:         true,
										Description:      "Name of the secondary IP range",
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 63)),
									},
								},
							},
						},
						FieldNodeConfigurationLoadbalancers: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Loadboalancer configuration for CAST provisioned nodes",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"target_backend_pools": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Target backend pools configuration for CAST provisioned nodes",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"name": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Name of the target group",
												},
											},
										},
									},
									"unmanaged_instance_groups": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Unmanaged instance groups configuration for CAST provisioned nodes",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"name": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Name of the instance group",
												},
												"zone": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Zone of the instance group",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, i interface{}) error {
			return nil
		},
	}
}

func resourceNodeConfigurationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := d.Get(FieldClusterID).(string)
	req := sdk.NodeConfigurationAPICreateConfigurationJSONRequestBody{
		Name:            d.Get(FieldNodeConfigurationName).(string),
		DiskCpuRatio:    toPtr(int32(d.Get(FieldNodeConfigurationDiskCpuRatio).(int))),
		DrainTimeoutSec: toPtr(int32(d.Get(FieldNodeConfigurationDrainTimeoutSec).(int))),
		MinDiskSize:     toPtr(int32(d.Get(FieldNodeConfigurationMinDiskSize).(int))),
	}

	if v, ok := d.GetOk(FieldNodeConfigurationSubnets); ok {
		req.Subnets = toPtr(toStringList(v.([]interface{})))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationImage); ok {
		req.Image = toPtr(v.(string))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationSSHPublicKey); ok {
		req.SshPublicKey = toPtr(v.(string))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationInitScript); ok {
		req.InitScript = toPtr(v.(string))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationContainerRuntime); ok {
		req.ContainerRuntime = toPtr(sdk.NodeconfigV1ContainerRuntime(v.(string)))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationDockerConfig); ok {
		m, err := stringToMap(v.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		req.DockerConfig = toPtr(m)
	}
	if v, ok := d.GetOk(FieldNodeConfigurationKubeletConfig); ok {
		m, err := stringToMap(v.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		req.KubeletConfig = toPtr(m)
	}
	if v := d.Get(FieldNodeConfigurationTags).(map[string]interface{}); len(v) > 0 {
		req.Tags = lo.ToPtr(toStringMap(v))
	}

	// Map provider specific configurations.
	if v, ok := d.GetOk(FieldNodeConfigurationEKS); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		req.Eks = toEKSConfig(v.([]interface{})[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationKOPS); ok && len(v.([]interface{})) > 0 {
		req.Kops = toKOPSConfig(v.([]interface{})[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationAKS); ok && len(v.([]interface{})) > 0 {
		req.Aks = toAKSSConfig(v.([]interface{})[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationGKE); ok && len(v.([]interface{})) > 0 {
		req.Gke = toGKEConfig(v.([]interface{})[0].(map[string]interface{}))
	}

	resp, err := client.NodeConfigurationAPICreateConfigurationWithResponse(ctx, clusterID, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	d.SetId(*resp.JSON200.Id)

	return resourceNodeConfigurationRead(ctx, d, meta)
}

func resourceNodeConfigurationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := d.Get(FieldClusterID).(string)
	resp, err := client.NodeConfigurationAPIGetConfigurationWithResponse(ctx, clusterID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if !d.IsNewResource() && resp.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Node configuration (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	nodeConfig := resp.JSON200

	if err := d.Set(FieldNodeConfigurationName, nodeConfig.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %w", err))
	}
	if err := d.Set(FieldNodeConfigurationDiskCpuRatio, nodeConfig.DiskCpuRatio); err != nil {
		return diag.FromErr(fmt.Errorf("setting disk cpu ratio: %w", err))
	}
	if err := d.Set(FieldNodeConfigurationDrainTimeoutSec, nodeConfig.DrainTimeoutSec); err != nil {
		return diag.FromErr(fmt.Errorf("setting drain timeout: %w", err))
	}
	if err := d.Set(FieldNodeConfigurationMinDiskSize, nodeConfig.MinDiskSize); err != nil {
		return diag.FromErr(fmt.Errorf("setting min disk size: %w", err))
	}
	if err := d.Set(FieldNodeConfigurationSubnets, nodeConfig.Subnets); err != nil {
		return diag.FromErr(fmt.Errorf("setting subnets: %w", err))
	}
	if err := d.Set(FieldNodeConfigurationSSHPublicKey, nodeConfig.SshPublicKey); err != nil {
		return diag.FromErr(fmt.Errorf("setting ssh public key: %w", err))
	}
	if err := d.Set(FieldNodeConfigurationImage, nodeConfig.Image); err != nil {
		return diag.FromErr(fmt.Errorf("setting image: %w", err))
	}
	if err := d.Set(FieldNodeConfigurationInitScript, nodeConfig.InitScript); err != nil {
		return diag.FromErr(fmt.Errorf("setting init script: %w", err))
	}
	if err := d.Set(FieldNodeConfigurationContainerRuntime, nodeConfig.ContainerRuntime); err != nil {
		return diag.FromErr(fmt.Errorf("setting container runtime: %w", err))
	}
	if err := d.Set(FieldNodeConfigurationTags, nodeConfig.Tags); err != nil {
		return diag.FromErr(fmt.Errorf("setting tags: %w", err))
	}

	if cfg := nodeConfig.DockerConfig; cfg != nil {
		b, err := json.Marshal(nodeConfig.DockerConfig)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(FieldNodeConfigurationDockerConfig, string(b)); err != nil {
			return diag.FromErr(fmt.Errorf("setting docker config: %w", err))
		}
	}
	if cfg := nodeConfig.KubeletConfig; cfg != nil {
		b, err := json.Marshal(nodeConfig.KubeletConfig)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(FieldNodeConfigurationKubeletConfig, string(b)); err != nil {
			return diag.FromErr(fmt.Errorf("setting kubelet config: %w", err))
		}
	}

	if err := d.Set(FieldNodeConfigurationEKS, flattenEKSConfig(nodeConfig.Eks)); err != nil {
		return diag.Errorf("error setting eks config: %v", err)
	}
	if err := d.Set(FieldNodeConfigurationKOPS, flattenKOPSConfig(nodeConfig.Kops)); err != nil {
		return diag.Errorf("error setting kops config: %v", err)
	}
	if err := d.Set(FieldNodeConfigurationAKS, flattenAKSConfig(nodeConfig.Aks)); err != nil {
		return diag.Errorf("error setting aks config: %v", err)
	}
	if err := d.Set(FieldNodeConfigurationGKE, flattenGKEConfig(nodeConfig.Gke)); err != nil {
		return diag.Errorf("error setting gke config: %v", err)
	}

	return nil
}

func resourceNodeConfigurationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !d.HasChanges(
		FieldNodeConfigurationDiskCpuRatio,
		FieldNodeConfigurationDrainTimeoutSec,
		FieldNodeConfigurationMinDiskSize,
		FieldNodeConfigurationSubnets,
		FieldNodeConfigurationSSHPublicKey,
		FieldNodeConfigurationImage,
		FieldNodeConfigurationInitScript,
		FieldNodeConfigurationContainerRuntime,
		FieldNodeConfigurationDockerConfig,
		FieldNodeConfigurationKubeletConfig,
		FieldNodeConfigurationTags,
		FieldNodeConfigurationAKS,
		FieldNodeConfigurationEKS,
		FieldNodeConfigurationKOPS,
		FieldNodeConfigurationGKE,
	) {
		log.Printf("[INFO] Nothing to update in node configuration")
		return nil
	}

	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)
	req := sdk.NodeConfigurationAPIUpdateConfigurationJSONRequestBody{
		DiskCpuRatio:    toPtr(int32(d.Get(FieldNodeConfigurationDiskCpuRatio).(int))),
		DrainTimeoutSec: toPtr(int32(d.Get(FieldNodeConfigurationDrainTimeoutSec).(int))),
		MinDiskSize:     toPtr(int32(d.Get(FieldNodeConfigurationMinDiskSize).(int))),
	}

	if v, ok := d.GetOk(FieldNodeConfigurationSubnets); ok {
		req.Subnets = toPtr(toStringList(v.([]interface{})))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationImage); ok {
		req.Image = toPtr(v.(string))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationSSHPublicKey); ok {
		req.SshPublicKey = toPtr(v.(string))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationInitScript); ok {
		req.InitScript = toPtr(v.(string))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationContainerRuntime); ok {
		req.ContainerRuntime = toPtr(sdk.NodeconfigV1ContainerRuntime(v.(string)))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationDockerConfig); ok {
		m, err := stringToMap(v.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		req.DockerConfig = toPtr(m)
	}
	if v, ok := d.GetOk(FieldNodeConfigurationKubeletConfig); ok {
		m, err := stringToMap(v.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		req.KubeletConfig = toPtr(m)
	}
	if v := d.Get(FieldNodeConfigurationTags).(map[string]interface{}); len(v) > 0 {
		req.Tags = lo.ToPtr(toStringMap(v))
	}

	// Map provider specific configurations.
	if v, ok := d.GetOk(FieldNodeConfigurationEKS); ok && len(v.([]interface{})) > 0 {
		req.Eks = toEKSConfig(v.([]interface{})[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationKOPS); ok && len(v.([]interface{})) > 0 {
		req.Kops = toKOPSConfig(v.([]interface{})[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationAKS); ok && len(v.([]interface{})) > 0 {
		req.Aks = toAKSSConfig(v.([]interface{})[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationGKE); ok && len(v.([]interface{})) > 0 {
		req.Gke = toGKEConfig(v.([]interface{})[0].(map[string]interface{}))
	}

	resp, err := client.NodeConfigurationAPIUpdateConfigurationWithResponse(ctx, clusterID, d.Id(), req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	return resourceNodeConfigurationRead(ctx, d, meta)
}

func resourceNodeConfigurationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)

	resp, err := client.NodeConfigurationAPIGetConfigurationWithResponse(ctx, clusterID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[DEBUG] Node configuration (%s) not found, skipping delete", d.Id())
		return nil
	}

	if err := sdk.StatusOk(resp); err != nil {
		return diag.FromErr(err)
	}

	if *resp.JSON200.Default {
		log.Printf("[WARN] Default node configuration (%s) can't be deleted, removing from state", d.Id())
		return nil
	}

	del, err := client.NodeConfigurationAPIDeleteConfigurationWithResponse(ctx, clusterID, d.Id())
	if err := sdk.CheckOKResponse(del, err); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func toEKSConfig(obj map[string]interface{}) *sdk.NodeconfigV1EKSConfig {
	if obj == nil {
		return nil
	}

	out := &sdk.NodeconfigV1EKSConfig{}
	if v, ok := obj["dns_cluster_ip"].(string); ok && v != "" {
		out.DnsClusterIp = toPtr(v)
	}
	if v, ok := obj["instance_profile_arn"].(string); ok {
		out.InstanceProfileArn = v
	}
	if v, ok := obj["node_group_arn"].(string); ok && v != "" {
		out.NodeGroupArn = toPtr(v)
	}
	if v, ok := obj["key_pair_id"].(string); ok && v != "" {
		out.KeyPairId = toPtr(v)
	}
	if v, ok := obj["security_groups"].([]interface{}); ok && len(v) > 0 {
		out.SecurityGroups = toPtr(toStringList(v))
	}
	if v, ok := obj["volume_type"].(string); ok && v != "" {
		out.VolumeType = toPtr(v)
	}
	if v, ok := obj["volume_iops"].(int); ok && v != 0 {
		out.VolumeIops = toPtr(int32(v))
	}
	if v, ok := obj["volume_throughput"].(int); ok && v != 0 {
		out.VolumeThroughput = toPtr(int32(v))
	}
	if v, ok := obj["imds_v1"].(bool); ok {
		out.ImdsV1 = toPtr(v)
	}
	if v, ok := obj["imds_hop_limit"].(int); ok {
		out.ImdsHopLimit = toPtr(int32(v))
	}

	if v, ok := obj["volume_kms_key_arn"].(string); ok && v != "" {
		out.VolumeKmsKeyArn = toPtr(v)
	}

	if v, ok := obj["max_pods_per_node_formula"].(string); ok && v != "" {
		out.MaxPodsPerNodeFormula = toPtr(v)
	}

	if v, ok := obj["ips_per_prefix"].(int); ok && v != 0 {
		out.IpsPerPrefix = toPtr(int32(v))
	}

	if v, ok := obj[FieldNodeConfigurationEKSTargetGroup].([]any); ok && len(v) > 0 {
		resultTGs := make([]sdk.NodeconfigV1TargetGroup, 0, len(v))
		for _, tgRaw := range v {
			if tg, ok := tgRaw.(map[string]any); ok {
				sdkTG := sdk.NodeconfigV1TargetGroup{}
				if arn, ok := tg["arn"].(string); ok && arn != "" {
					sdkTG.Arn = toPtr(arn)
				}
				if port, ok := tg["port"].(int); ok && port > 0 && port < 65536 {
					sdkTG.Port = toPtr(int32(port))
				}
				resultTGs = append(resultTGs, sdkTG)
			}
		}
		out.TargetGroups = &resultTGs
	}

	if v, ok := obj[FieldNodeConfigurationEKSImageFamily].(string); ok {
		out.ImageFamily = toEKSImageFamily(v)
	}

	return out
}

func toEKSImageFamily(v string) *sdk.NodeconfigV1EKSConfigImageFamily {
	if v == "" {
		return nil
	}

	switch strings.ToLower(v) {
	case eksImageFamilyAL2:
		return lo.ToPtr(sdk.NodeconfigV1EKSConfigImageFamilyFamilyAl2)
	case eksImageFamilyAL2023:
		return lo.ToPtr(sdk.NodeconfigV1EKSConfigImageFamilyFAMILYAL2023)
	case eksImageFamilyBottlerocket:
		return lo.ToPtr(sdk.NodeconfigV1EKSConfigImageFamilyFAMILYBOTTLEROCKET)
	default:
		return nil
	}
}

func flattenEKSConfig(config *sdk.NodeconfigV1EKSConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}

	m := map[string]interface{}{
		"instance_profile_arn": config.InstanceProfileArn,
	}
	if v := config.KeyPairId; v != nil {
		m["key_pair_id"] = toString(v)
	}
	if v := config.NodeGroupArn; v != nil {
		m["node_group_arn"] = toString(v)
	}
	if v := config.DnsClusterIp; v != nil {
		m["dns_cluster_ip"] = toString(v)
	}
	if v := config.SecurityGroups; v != nil {
		m["security_groups"] = *config.SecurityGroups
	}
	if v := config.VolumeType; v != nil {
		m["volume_type"] = toString(v)
	}
	if v := config.VolumeIops; v != nil {
		m["volume_iops"] = *config.VolumeIops
	}
	if v := config.VolumeThroughput; v != nil {
		m["volume_throughput"] = *config.VolumeThroughput
	}
	if v := config.ImdsV1; v != nil {
		m["imds_v1"] = *config.ImdsV1
	}
	if v := config.ImdsHopLimit; v != nil {
		m["imds_hop_limit"] = *config.ImdsHopLimit
	}

	if v := config.VolumeKmsKeyArn; v != nil {
		m["volume_kms_key_arn"] = toString(config.VolumeKmsKeyArn)
	}

	if v := config.MaxPodsPerNodeFormula; v != nil {
		m["max_pods_per_node_formula"] = toString(config.MaxPodsPerNodeFormula)
	}

	if v := config.IpsPerPrefix; v != nil {
		m["ips_per_prefix"] = *config.IpsPerPrefix
	}

	if v := config.TargetGroups; v != nil && len(*v) > 0 {
		tgs := make([]any, 0, len(*v))
		for _, tg := range *v {
			if tg.Arn == nil {
				// Empty arn is invalid, ignore the entry
				continue
			}

			val := map[string]any{
				"arn": *tg.Arn,
			}
			if tg.Port != nil {
				val["port"] = *tg.Port
			}
			tgs = append(tgs, val)
		}

		m[FieldNodeConfigurationEKSTargetGroup] = tgs
	}

	if v := config.ImageFamily; v != nil {
		m[FieldNodeConfigurationEKSImageFamily] = fromEKSImageFamily(*v)
	}

	return []map[string]interface{}{m}
}

func fromEKSImageFamily(family sdk.NodeconfigV1EKSConfigImageFamily) string {
	switch family {
	case sdk.NodeconfigV1EKSConfigImageFamilyFAMILYBOTTLEROCKET, sdk.NodeconfigV1EKSConfigImageFamilyFamilyBottlerocket:
		return eksImageFamilyBottlerocket
	case sdk.NodeconfigV1EKSConfigImageFamilyFAMILYAL2, sdk.NodeconfigV1EKSConfigImageFamilyFamilyAl2:
		return eksImageFamilyAL2
	case sdk.NodeconfigV1EKSConfigImageFamilyFAMILYAL2023, sdk.NodeconfigV1EKSConfigImageFamilyFamilyAl2023:
		return eksImageFamilyAL2023
	default:
		return ""
	}
}

func toKOPSConfig(obj map[string]interface{}) *sdk.NodeconfigV1KOPSConfig {
	if obj == nil {
		return nil
	}

	out := &sdk.NodeconfigV1KOPSConfig{}
	if v, ok := obj["key_pair_id"].(string); ok && v != "" {
		out.KeyPairId = toPtr(v)
	}

	return out
}

func flattenKOPSConfig(config *sdk.NodeconfigV1KOPSConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}
	m := map[string]interface{}{}
	if v := config.KeyPairId; v != nil {
		m["key_pair_id"] = toString(v)
	}

	return []map[string]interface{}{m}
}

func toAKSSConfig(obj map[string]interface{}) *sdk.NodeconfigV1AKSConfig {
	if obj == nil {
		return nil
	}

	out := &sdk.NodeconfigV1AKSConfig{}
	if v, ok := obj["max_pods_per_node"].(int); ok {
		out.MaxPodsPerNode = toPtr(int32(v))
	}

	if v, ok := obj["os_disk_type"].(string); ok && v != "" {
		out.OsDiskType = toAKSOSDiskType(v)
	}

	if v, ok := obj[FieldNodeConfigurationAKSEphemeralOSDisk].([]any); ok && len(v) > 0 {
		out.OsDiskEphemeral = toAKSEphemeralOSDisk(v[0])
	}

	if v, ok := obj[FieldNodeConfigurationAKSImageFamily].(string); ok {
		out.ImageFamily = toAKSImageFamily(v)
	}

	if v, ok := obj[FieldNodeConfigurationLoadbalancers].([]interface{}); ok && len(v) > 0 {
		out.LoadBalancers = toAksLoadBalancers(v)
	}

	if v, ok := obj[FieldNodeConfigurationAKSPublicIP].([]interface{}); ok && len(v) > 0 {
		out.PublicIp = toAKSNodePublicIP(v[0])
	}

	if v, ok := obj[FieldNodeConfigurationAKSNetworkSecurityGroup].(string); ok && v != "" {
		out.NetworkSecurityGroupId = toPtr(v)
	}

	if v, ok := obj[FieldNodeConfigurationAKSApplicationSecurityGroups].([]interface{}); ok && len(v) > 0 {
		out.ApplicationSecurityGroupIds = toPtr(toStringList(v))
	}

	return out
}

func toAKSNodePublicIP(obj any) *sdk.NodeconfigV1AKSConfigPublicIP {
	if obj == nil {
		return nil
	}

	publicIP := &sdk.NodeconfigV1AKSConfigPublicIP{}

	if v, ok := obj.(map[string]any)["public_ip_prefix"].(string); ok && v != "" {
		publicIP.IpPrefix = lo.ToPtr(v)
	}

	if v, ok := obj.(map[string]any)["tags"].(map[string]any); ok && len(v) > 0 {
		tagList := []sdk.NodeconfigV1AKSConfigPublicIPAKSPublicIPTags{}

		for k, vv := range v {
			tagList = append(tagList, sdk.NodeconfigV1AKSConfigPublicIPAKSPublicIPTags{
				TagValue:  lo.ToPtr(vv.(string)),
				IpTagType: lo.ToPtr(k),
			})
		}
		publicIP.Tags = &tagList
	}

	if v, ok := obj.(map[string]any)["idle_timeout_in_minutes"].(int); ok && v > 0 {
		publicIP.IdleTimeoutInMinutes = lo.ToPtr(int32(v))
	}

	return publicIP

}

func toAKSEphemeralOSDisk(obj any) *sdk.NodeconfigV1AKSConfigOsDiskEphemeral {
	if obj == nil {
		return nil
	}

	osDisk := &sdk.NodeconfigV1AKSConfigOsDiskEphemeral{}

	if v, ok := obj.(map[string]any)["placement"].(string); ok && v != "" {
		switch strings.ToLower(v) {
		case strings.ToLower(aksEphemeralDiskPlacementResourceDisk):
			osDisk.Placement = lo.ToPtr(sdk.NodeconfigV1AKSConfigOsDiskEphemeralPlacementPLACEMENTRESOURCEDISK)
		case strings.ToLower(aksEphemeralDiskPlacementCacheDisk):
			osDisk.Placement = lo.ToPtr(sdk.NodeconfigV1AKSConfigOsDiskEphemeralPlacementPLACEMENTCACHEDISK)
		}
	}

	if v, ok := obj.(map[string]any)["cache"].(string); ok && v != "" {
		switch strings.ToLower(v) {
		case strings.ToLower(aksDiskCacheReadWrite):
			osDisk.CacheType = lo.ToPtr(sdk.NodeconfigV1AKSConfigOsDiskEphemeralCacheTypeREADWRITE)
		case strings.ToLower(aksDiskCacheReadOnly):
			osDisk.CacheType = lo.ToPtr(sdk.NodeconfigV1AKSConfigOsDiskEphemeralCacheTypeREADONLY)
		}
	}

	return osDisk
}

func toAksLoadBalancers(obj []interface{}) *[]sdk.NodeconfigV1AKSConfigLoadBalancers {
	if obj == nil {
		return nil
	}

	out := make([]sdk.NodeconfigV1AKSConfigLoadBalancers, 0, len(obj))
	for _, lbRaw := range obj {
		if lb, ok := lbRaw.(map[string]interface{}); ok {
			sdkLB := sdk.NodeconfigV1AKSConfigLoadBalancers{}
			if id, ok := lb["id"].(string); ok && id != "" {
				sdkLB.Id = lo.ToPtr(id)
			}
			if name, ok := lb["name"].(string); ok && name != "" {
				//nolint:staticcheck //We have to do this until we drop the field in TF major provider version.
				sdkLB.Name = lo.ToPtr(name)
			}
			if ipBasedBackendPools, ok := lb[FieldNodeConfigurationAKSLoadbalancerIPPools].([]interface{}); ok && len(ipBasedBackendPools) > 0 {
				sdkLB.IpBasedBackendPools = toAksIpBasedBackendPools(ipBasedBackendPools)
			}
			if nicBasedBackendPools, ok := lb[FieldNodeConfigurationAKSLoadbalancerNICPools].([]interface{}); ok && len(nicBasedBackendPools) > 0 {
				sdkLB.NicBasedBackendPools = toAksNICBasedBackendPools(nicBasedBackendPools)
			}
			out = append(out, sdkLB)
		}
	}

	return &out
}

func toAksIpBasedBackendPools(obj []interface{}) *[]sdk.NodeconfigV1AKSConfigLoadBalancersIPBasedBackendPool {
	if obj == nil {
		return nil
	}

	pools := lo.Map(extractAksBackendPoolNames(obj), func(name string, _ int) sdk.NodeconfigV1AKSConfigLoadBalancersIPBasedBackendPool {
		return sdk.NodeconfigV1AKSConfigLoadBalancersIPBasedBackendPool{Name: lo.ToPtr(name)}
	})
	return &pools
}

func toAksNICBasedBackendPools(obj []any) *[]sdk.NodeconfigV1AKSConfigLoadBalancersNICBasedBackendPool {
	if obj == nil {
		return nil
	}

	pools := lo.Map(extractAksBackendPoolNames(obj), func(name string, _ int) sdk.NodeconfigV1AKSConfigLoadBalancersNICBasedBackendPool {
		return sdk.NodeconfigV1AKSConfigLoadBalancersNICBasedBackendPool{Name: lo.ToPtr(name)}
	})
	return &pools
}

func extractAksBackendPoolNames(pools []any) []string {
	return lo.Reduce(pools, func(names []string, poolRaw any, _ int) []string {
		if pool, ok := poolRaw.(map[string]interface{}); ok {
			if name, ok := pool["name"].(string); ok && name != "" {
				names = append(names, name)
			}
		}

		return names
	}, make([]string, 0))
}

func toAKSOSDiskType(v string) *sdk.NodeconfigV1AKSConfigOsDiskType {
	if v == "" {
		return nil
	}

	switch v {
	case "standard":
		return toPtr(sdk.OSDISKTYPESTANDARD)
	case "standard-ssd":
		return toPtr(sdk.OSDISKTYPESTANDARDSSD)
	case "premium-ssd":
		return toPtr(sdk.OSDISKTYPEPREMIUMSSD)
	default:
		return nil
	}
}

func toAKSImageFamily(v string) *sdk.NodeconfigV1AKSConfigImageFamily {
	if v == "" {
		return nil
	}

	switch strings.ToLower(v) {
	case aksImageFamilyUbuntu:
		return lo.ToPtr(sdk.NodeconfigV1AKSConfigImageFamilyFAMILYUBUNTU)
	case aksImageFamilyAzureLinux:
		return lo.ToPtr(sdk.NodeconfigV1AKSConfigImageFamilyFAMILYAZURELINUX)
	case aksImageFamilyWindows2019:
		return lo.ToPtr(sdk.NodeconfigV1AKSConfigImageFamilyFAMILYWINDOWS2019)
	case aksImageFamilyWindows2022:
		return lo.ToPtr(sdk.NodeconfigV1AKSConfigImageFamilyFAMILYWINDOWS2022)
	default:
		return nil
	}
}

func flattenAKSConfig(config *sdk.NodeconfigV1AKSConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}
	m := map[string]interface{}{}
	if v := config.MaxPodsPerNode; v != nil {
		m["max_pods_per_node"] = *config.MaxPodsPerNode
	}

	if v := config.OsDiskType; v != nil {
		m["os_disk_type"] = fromAKSDiskType(config.OsDiskType)
	}

	if v := config.ImageFamily; v != nil {
		m[FieldNodeConfigurationAKSImageFamily] = fromAKSImageFamily(*v)
	}

	if v := config.LoadBalancers; v != nil && len(*v) > 0 {
		m[FieldNodeConfigurationLoadbalancers] = fromAksLoadBalancers(*v)
	}

	if v := config.OsDiskEphemeral; v != nil {
		m[FieldNodeConfigurationAKSEphemeralOSDisk] = fromAKSEphemeralOSDisk(v)
	}

	if v := config.PublicIp; v != nil {
		m[FieldNodeConfigurationAKSPublicIP] = fromAKSNodePublicIP(v)
	}

	if v := config.NetworkSecurityGroupId; v != nil {
		m[FieldNodeConfigurationAKSNetworkSecurityGroup] = *config.NetworkSecurityGroupId
	}

	if v := config.ApplicationSecurityGroupIds; v != nil {
		m[FieldNodeConfigurationAKSApplicationSecurityGroups] = *config.ApplicationSecurityGroupIds
	}

	return []map[string]interface{}{m}
}

func fromAKSNodePublicIP(sdkPublicIp *sdk.NodeconfigV1AKSConfigPublicIP) []map[string]any {
	if sdkPublicIp == nil {
		return nil
	}

	m := map[string]interface{}{}
	if sdkPublicIp.IpPrefix != nil {
		m["public_ip_prefix"] = *sdkPublicIp.IpPrefix
	}

	if sdkPublicIp.Tags != nil {
		tags := make(map[string]interface{})
		for _, tag := range *sdkPublicIp.Tags {
			tags[lo.FromPtr(tag.IpTagType)] = lo.FromPtr(tag.TagValue)
		}
		m["tags"] = tags
	}

	if sdkPublicIp.IdleTimeoutInMinutes != nil {
		m["idle_timeout_in_minutes"] = *sdkPublicIp.IdleTimeoutInMinutes
	}

	return []map[string]any{m}

}

func fromAKSEphemeralOSDisk(sdkEph *sdk.NodeconfigV1AKSConfigOsDiskEphemeral) []map[string]interface{} {
	if sdkEph == nil {
		return nil
	}

	m := map[string]interface{}{}
	if sdkEph.Placement != nil {
		switch *sdkEph.Placement {
		case sdk.NodeconfigV1AKSConfigOsDiskEphemeralPlacementPLACEMENTRESOURCEDISK:
			m["placement"] = aksEphemeralDiskPlacementResourceDisk
		case sdk.NodeconfigV1AKSConfigOsDiskEphemeralPlacementPLACEMENTCACHEDISK:
			m["placement"] = aksEphemeralDiskPlacementCacheDisk
		}
	}

	if sdkEph.CacheType != nil {
		switch *sdkEph.CacheType {
		case sdk.NodeconfigV1AKSConfigOsDiskEphemeralCacheTypeREADWRITE:
			m["cache"] = aksDiskCacheReadWrite
		case sdk.NodeconfigV1AKSConfigOsDiskEphemeralCacheTypeREADONLY:
			m["cache"] = aksDiskCacheReadOnly
		}
	}

	return []map[string]interface{}{m}
}

func fromAksLoadBalancers(lbs []sdk.NodeconfigV1AKSConfigLoadBalancers) []map[string]interface{} {
	if lbs == nil {
		return nil
	}

	out := make([]map[string]interface{}, 0, len(lbs))
	for _, lb := range lbs {
		m := map[string]interface{}{}
		if lb.Id != nil {
			m["id"] = *lb.Id
		}
		//nolint:staticcheck //We have to do this until we drop the field in TF major provider version.
		if lb.Name != nil {
			//nolint:staticcheck //We have to do this until we drop the field in TF major provider version.
			m["name"] = *lb.Name
		}
		if lb.IpBasedBackendPools != nil && len(*lb.IpBasedBackendPools) > 0 {
			m[FieldNodeConfigurationAKSLoadbalancerIPPools] = fromAksIpBasedBackendPoolNames(lo.FilterMap(*lb.IpBasedBackendPools, func(pool sdk.NodeconfigV1AKSConfigLoadBalancersIPBasedBackendPool, _ int) (string, bool) {
				if pool.Name != nil {
					return *pool.Name, true
				}
				return "", false
			}))
		}
		if lb.NicBasedBackendPools != nil && len(*lb.NicBasedBackendPools) > 0 {
			m[FieldNodeConfigurationAKSLoadbalancerNICPools] = fromAksIpBasedBackendPoolNames(lo.FilterMap(*lb.NicBasedBackendPools, func(pool sdk.NodeconfigV1AKSConfigLoadBalancersNICBasedBackendPool, _ int) (string, bool) {
				if pool.Name != nil {
					return *pool.Name, true
				}
				return "", false
			}))
		}
		out = append(out, m)
	}

	return out
}

func fromAksIpBasedBackendPoolNames(names []string) []map[string]interface{} {
	if names == nil {
		return nil
	}

	out := make([]map[string]interface{}, 0, len(names))
	for _, name := range names {
		m := map[string]interface{}{}
		m["name"] = name
		out = append(out, m)
	}

	return out
}

func fromAKSDiskType(osDiskType *sdk.NodeconfigV1AKSConfigOsDiskType) string {
	if osDiskType == nil {
		return ""
	}
	switch *osDiskType {
	case sdk.OSDISKTYPESTANDARD:
		return "standard"
	case sdk.OSDISKTYPESTANDARDSSD:
		return "standard-ssd"
	case sdk.OSDISKTYPEPREMIUMSSD:
		return "premium-ssd"
	default:
		return ""
	}
}

func fromAKSImageFamily(family sdk.NodeconfigV1AKSConfigImageFamily) string {
	switch family {
	case sdk.NodeconfigV1AKSConfigImageFamilyFAMILYAZURELINUX, sdk.NodeconfigV1AKSConfigImageFamilyFamilyAzureLinux:
		return aksImageFamilyAzureLinux
	case sdk.NodeconfigV1AKSConfigImageFamilyFAMILYUBUNTU, sdk.NodeconfigV1AKSConfigImageFamilyFamilyUbuntu:
		return aksImageFamilyUbuntu
	case sdk.NodeconfigV1AKSConfigImageFamilyFAMILYWINDOWS2019, sdk.NodeconfigV1AKSConfigImageFamilyFamilyWindows2019:
		return aksImageFamilyWindows2019
	case sdk.NodeconfigV1AKSConfigImageFamilyFamilyWindows2022, sdk.NodeconfigV1AKSConfigImageFamilyFAMILYWINDOWS2022:
		return aksImageFamilyWindows2022
	default:
		return ""
	}
}

func toGKEConfig(obj map[string]interface{}) *sdk.NodeconfigV1GKEConfig {
	if obj == nil {
		return nil
	}

	out := &sdk.NodeconfigV1GKEConfig{}
	if v, ok := obj["max_pods_per_node"].(int); ok {
		out.MaxPodsPerNode = toPtr(int32(v))
	}
	if v, ok := obj["network_tags"].([]interface{}); ok {
		out.NetworkTags = toPtr(toStringList(v))
	}
	if v, ok := obj["disk_type"].(string); ok && v != "" {
		out.DiskType = toPtr(v)
	}
	if v, ok := obj["zones"].([]interface{}); ok {
		out.Zones = toPtr(toStringList(v))
	}

	if v, ok := obj["use_ephemeral_storage_local_ssd"].(bool); ok {
		out.UseEphemeralStorageLocalSsd = toPtr(v)
	}

	if v, ok := obj["secondary_ip_range"].([]interface{}); ok && len(v) > 0 && v[0] != nil {
		secondary := v[0].(map[string]interface{})
		if rangeName, ok := secondary["range_name"].(string); ok {
			out.SecondaryIpRange = &sdk.NodeconfigV1SecondaryIPRange{RangeName: &rangeName}
		}
	}

	if v, ok := obj[FieldNodeConfigurationLoadbalancers].([]interface{}); ok && len(v) > 0 {
		out.LoadBalancers = toGkeLoadBalancers(v)
	}

	return out
}

func toGkeLoadBalancers(obj []interface{}) *[]sdk.NodeconfigV1GKEConfigLoadBalancers {
	if obj == nil {
		return nil
	}

	out := make([]sdk.NodeconfigV1GKEConfigLoadBalancers, 0, len(obj))
	for _, lbRaw := range obj {
		if lb, ok := lbRaw.(map[string]interface{}); ok {
			sdkLB := sdk.NodeconfigV1GKEConfigLoadBalancers{}
			if targetBackendPools, ok := lb["target_backend_pools"].([]interface{}); ok && len(targetBackendPools) > 0 {
				sdkLB.TargetBackendPools = toGkeTargetBackendPools(targetBackendPools)
			}
			if unmanagedInstanceGroups, ok := lb["unmanaged_instance_groups"].([]interface{}); ok && len(unmanagedInstanceGroups) > 0 {
				sdkLB.UnmanagedInstanceGroups = toGkeUnmanagedInstanceGroups(unmanagedInstanceGroups)
			}

			if sdkLB.UnmanagedInstanceGroups != nil || sdkLB.TargetBackendPools != nil {
				out = append(out, sdkLB)
			}
		}
	}

	return &out
}

func toGkeTargetBackendPools(obj []interface{}) *[]sdk.NodeconfigV1GKEConfigLoadBalancersTargetBackendPools {
	if obj == nil {
		return nil
	}

	out := make([]sdk.NodeconfigV1GKEConfigLoadBalancersTargetBackendPools, 0, len(obj))
	for _, poolRaw := range obj {
		if pool, ok := poolRaw.(map[string]interface{}); ok {
			sdkPool := sdk.NodeconfigV1GKEConfigLoadBalancersTargetBackendPools{}
			if name, ok := pool["name"].(string); ok && name != "" {
				sdkPool.Name = lo.ToPtr(name)
			}
			out = append(out, sdkPool)
		}
	}

	return &out
}

func toGkeUnmanagedInstanceGroups(obj []interface{}) *[]sdk.NodeconfigV1GKEConfigLoadBalancersUnmanagedInstanceGroups {
	if obj == nil {
		return nil
	}

	out := make([]sdk.NodeconfigV1GKEConfigLoadBalancersUnmanagedInstanceGroups, 0, len(obj))
	for _, groupRaw := range obj {
		if group, ok := groupRaw.(map[string]interface{}); ok {
			sdkGroup := sdk.NodeconfigV1GKEConfigLoadBalancersUnmanagedInstanceGroups{}
			if name, ok := group["name"].(string); ok && name != "" {
				sdkGroup.Name = lo.ToPtr(name)
			}
			if zone, ok := group["zone"].(string); ok && zone != "" {
				sdkGroup.Zone = lo.ToPtr(zone)
			}
			out = append(out, sdkGroup)
		}
	}

	return &out

}

func flattenGKEConfig(config *sdk.NodeconfigV1GKEConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}
	m := map[string]interface{}{}
	if v := config.MaxPodsPerNode; v != nil {
		m["max_pods_per_node"] = *config.MaxPodsPerNode
	}
	if v := config.NetworkTags; v != nil {
		m["network_tags"] = *v
	}
	if v := config.DiskType; v != nil {
		m["disk_type"] = *v
	}
	if v := config.Zones; v != nil {
		m["zones"] = *v
	}

	if v := config.UseEphemeralStorageLocalSsd; v != nil {
		m["use_ephemeral_storage_local_ssd"] = *v
	}

	if v := config.SecondaryIpRange; v != nil {
		m["secondary_ip_range"] = []map[string]interface{}{
			{"range_name": v.RangeName},
		}
	}

	if v := config.LoadBalancers; v != nil && len(*v) > 0 {
		m[FieldNodeConfigurationLoadbalancers] = fromGkeLoadBalancers(*v)
	}

	return []map[string]interface{}{m}
}

func fromGkeLoadBalancers(objs []sdk.NodeconfigV1GKEConfigLoadBalancers) []map[string]interface{} {
	var results []map[string]interface{}
	for _, obj := range objs {
		result := make(map[string]interface{})
		if obj.TargetBackendPools != nil && len(*obj.TargetBackendPools) > 0 {
			tbp := []map[string]interface{}{}
			for _, pool := range *obj.TargetBackendPools {
				tbp = append(tbp, map[string]interface{}{
					"name": *pool.Name,
				})
			}
			result["target_backend_pools"] = tbp
		}

		if obj.UnmanagedInstanceGroups != nil && len(*obj.UnmanagedInstanceGroups) > 0 {
			uig := []map[string]interface{}{}
			for _, group := range *obj.UnmanagedInstanceGroups {
				uig = append(uig, map[string]interface{}{
					"name": *group.Name,
					"zone": *group.Zone,
				})
			}
			result["unmanaged_instance_groups"] = uig
		}
		if len(result) > 0 {
			results = append(results, result)
		}
	}

	return results
}

func nodeConfigStateImporter(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	ids := strings.Split(d.Id(), "/")
	if len(ids) != 2 || ids[0] == "" || ids[1] == "" {
		return nil, fmt.Errorf("expected import id with format: <cluster_id>/<node_configuration name or id>, got: %q", d.Id())
	}

	clusterID, id := ids[0], ids[1]
	if err := d.Set(FieldClusterID, clusterID); err != nil {
		return nil, fmt.Errorf("setting cluster id: %w", err)
	}
	d.SetId(id)

	// Return if node config ID provided.
	if _, err := uuid.Parse(id); err == nil {
		return []*schema.ResourceData{d}, nil
	}

	// Find node configuration ID based on provided name.
	client := meta.(*ProviderConfig).api
	resp, err := client.NodeConfigurationAPIListConfigurationsWithResponse(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	for _, cfg := range *resp.JSON200.Items {
		if lo.FromPtr(cfg.Name) == id {
			d.SetId(toString(cfg.Id))
			return []*schema.ResourceData{d}, nil
		}
	}

	return nil, fmt.Errorf("failed to find node configuration with the following name: %v", id)
}
