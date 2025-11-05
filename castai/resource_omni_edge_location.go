package castai

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni_provisioner"
)

const (
	FieldOmniEdgeLocationOrganizationID = "organization_id"
	FieldOmniEdgeLocationClusterID      = "cluster_id"
	FieldOmniEdgeLocationName           = "name"
	FieldOmniEdgeLocationRegion         = "region"
	FieldOmniEdgeLocationZones          = "zones"
	FieldOmniEdgeLocationDescription    = "description"
	FieldOmniEdgeLocationState          = "state"
	FieldOmniEdgeLocationTotalEdgeCount = "total_edge_count"
	FieldOmniEdgeLocationAWS            = "aws"
	FieldOmniEdgeLocationGCP            = "gcp"
	FieldOmniEdgeLocationOCI            = "oci"

	// AWS fields
	FieldAWSAccountID        = "account_id"
	FieldAWSAccessKeyID      = "access_key_id"
	FieldAWSSecretAccessKey  = "secret_access_key"
	FieldAWSVpcID            = "vpc_id"
	FieldAWSSubnetIDs        = "subnet_ids"
	FieldAWSSecurityGroupID  = "security_group_id"

	// GCP fields
	FieldGCPProjectID               = "project_id"
	FieldGCPServiceAccountJSON      = "service_account_json_base64"
	FieldGCPNetworkName             = "network_name"
	FieldGCPSubnetName              = "subnet_name"
	FieldGCPTags                    = "tags"

	// OCI fields
	FieldOCITenancyID        = "tenancy_id"
	FieldOCICompartmentID    = "compartment_id"
	FieldOCIUserID           = "user_id"
	FieldOCIFingerprint      = "fingerprint"
	FieldOCIPrivateKeyBase64 = "private_key_base64"
	FieldOCIVcnID            = "vcn_id"
	FieldOCISubnetID         = "subnet_id"
)

func resourceOmniEdgeLocation() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceOmniEdgeLocationCreate,
		ReadContext:   resourceOmniEdgeLocationRead,
		UpdateContext: resourceOmniEdgeLocationUpdate,
		DeleteContext: resourceOmniEdgeLocationDelete,

		Schema: map[string]*schema.Schema{
			FieldOmniEdgeLocationOrganizationID: {
				Type:        schema.TypeString,
				Description: "Organization ID",
				Required:    true,
				ForceNew:    true,
			},
			FieldOmniEdgeLocationClusterID: {
				Type:        schema.TypeString,
				Description: "Omni cluster ID",
				Required:    true,
				ForceNew:    true,
			},
			FieldOmniEdgeLocationName: {
				Type:        schema.TypeString,
				Description: "Name of the edge location",
				Required:    true,
			},
			FieldOmniEdgeLocationRegion: {
				Type:        schema.TypeString,
				Description: "Cloud provider region",
				Required:    true,
			},
			FieldOmniEdgeLocationZones: {
				Type:        schema.TypeList,
				Description: "Availability zones",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			FieldOmniEdgeLocationDescription: {
				Type:        schema.TypeString,
				Description: "Description of the edge location",
				Optional:    true,
			},
			FieldOmniEdgeLocationState: {
				Type:        schema.TypeString,
				Description: "Current state of the edge location",
				Computed:    true,
			},
			FieldOmniEdgeLocationTotalEdgeCount: {
				Type:        schema.TypeInt,
				Description: "Total number of edges in this location",
				Computed:    true,
			},
			FieldOmniEdgeLocationAWS: {
				Type:        schema.TypeList,
				Description: "AWS configuration",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldAWSAccountID: {
							Type:        schema.TypeString,
							Description: "AWS account ID",
							Required:    true,
						},
						FieldAWSAccessKeyID: {
							Type:        schema.TypeString,
							Description: "AWS access key ID",
							Required:    true,
							Sensitive:   true,
						},
						FieldAWSSecretAccessKey: {
							Type:        schema.TypeString,
							Description: "AWS secret access key",
							Required:    true,
							Sensitive:   true,
						},
						FieldAWSVpcID: {
							Type:        schema.TypeString,
							Description: "VPC ID",
							Required:    true,
						},
						FieldAWSSubnetIDs: {
							Type:        schema.TypeList,
							Description: "Subnet IDs",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						FieldAWSSecurityGroupID: {
							Type:        schema.TypeString,
							Description: "Security group ID",
							Optional:    true,
						},
					},
				},
			},
			FieldOmniEdgeLocationGCP: {
				Type:        schema.TypeList,
				Description: "GCP configuration",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldGCPProjectID: {
							Type:        schema.TypeString,
							Description: "GCP project ID",
							Required:    true,
						},
						FieldGCPServiceAccountJSON: {
							Type:        schema.TypeString,
							Description: "Base64-encoded service account JSON",
							Required:    true,
							Sensitive:   true,
						},
						FieldGCPNetworkName: {
							Type:        schema.TypeString,
							Description: "VPC network name",
							Required:    true,
						},
						FieldGCPSubnetName: {
							Type:        schema.TypeString,
							Description: "Subnet name",
							Required:    true,
						},
						FieldGCPTags: {
							Type:        schema.TypeList,
							Description: "Network tags",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			FieldOmniEdgeLocationOCI: {
				Type:        schema.TypeList,
				Description: "OCI configuration",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldOCITenancyID: {
							Type:        schema.TypeString,
							Description: "OCI tenancy ID",
							Required:    true,
						},
						FieldOCICompartmentID: {
							Type:        schema.TypeString,
							Description: "OCI compartment ID",
							Required:    true,
						},
						FieldOCIUserID: {
							Type:        schema.TypeString,
							Description: "OCI user ID",
							Required:    true,
						},
						FieldOCIFingerprint: {
							Type:        schema.TypeString,
							Description: "API key fingerprint",
							Required:    true,
						},
						FieldOCIPrivateKeyBase64: {
							Type:        schema.TypeString,
							Description: "Base64-encoded private key",
							Required:    true,
							Sensitive:   true,
						},
						FieldOCIVcnID: {
							Type:        schema.TypeString,
							Description: "VCN ID",
							Required:    true,
						},
						FieldOCISubnetID: {
							Type:        schema.TypeString,
							Description: "Subnet ID",
							Required:    true,
						},
					},
				},
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceOmniEdgeLocationCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniProvisionerClient

	organizationID := data.Get(FieldOmniEdgeLocationOrganizationID).(string)
	clusterID := data.Get(FieldOmniEdgeLocationClusterID).(string)

	req := omni_provisioner.CreateEdgeLocationJSONRequestBody{
		Name:   data.Get(FieldOmniEdgeLocationName).(string),
		Region: data.Get(FieldOmniEdgeLocationRegion).(string),
	}

	if v, ok := data.GetOk(FieldOmniEdgeLocationZones); ok {
		zones := make([]string, 0)
		for _, zone := range v.([]interface{}) {
			zones = append(zones, zone.(string))
		}
		req.Zones = &zones
	}

	if v, ok := data.GetOk(FieldOmniEdgeLocationDescription); ok {
		desc := v.(string)
		req.Description = &desc
	}

	// Set AWS parameters
	if v, ok := data.GetOk(FieldOmniEdgeLocationAWS); ok && len(v.([]interface{})) > 0 {
		awsConfig := v.([]interface{})[0].(map[string]interface{})

		accountID := awsConfig[FieldAWSAccountID].(string)
		accessKeyID := awsConfig[FieldAWSAccessKeyID].(string)
		secretAccessKey := awsConfig[FieldAWSSecretAccessKey].(string)
		vpcID := awsConfig[FieldAWSVpcID].(string)

		subnetIDs := make([]string, 0)
		for _, sid := range awsConfig[FieldAWSSubnetIDs].([]interface{}) {
			subnetIDs = append(subnetIDs, sid.(string))
		}

		req.Aws = &omni_provisioner.CastaiOmniProvisionerV1beta1AwsParameters{
			AccountId: &accountID,
			Credentials: &omni_provisioner.CastaiOmniProvisionerV1beta1AwsCredentials{
				AccessKeyId:     &accessKeyID,
				SecretAccessKey: &secretAccessKey,
			},
			Networking: &omni_provisioner.CastaiOmniProvisionerV1beta1AwsNetworking{
				VpcId:     &vpcID,
				SubnetIds: &subnetIDs,
			},
		}

		if secGroupID, ok := awsConfig[FieldAWSSecurityGroupID].(string); ok && secGroupID != "" {
			req.Aws.Networking.SecurityGroupId = &secGroupID
		}
	}

	// Set GCP parameters
	if v, ok := data.GetOk(FieldOmniEdgeLocationGCP); ok && len(v.([]interface{})) > 0 {
		gcpConfig := v.([]interface{})[0].(map[string]interface{})

		projectID := gcpConfig[FieldGCPProjectID].(string)
		serviceAccountJSON := gcpConfig[FieldGCPServiceAccountJSON].(string)
		networkName := gcpConfig[FieldGCPNetworkName].(string)
		subnetName := gcpConfig[FieldGCPSubnetName].(string)

		req.Gcp = &omni_provisioner.CastaiOmniProvisionerV1beta1GcpParameters{
			ProjectId: &projectID,
			Credentials: &omni_provisioner.CastaiOmniProvisionerV1beta1GcpCredentials{
				ClientServiceAccountJsonBase64: &serviceAccountJSON,
			},
			Networking: &omni_provisioner.CastaiOmniProvisionerV1beta1GcpNetworking{
				NetworkName: &networkName,
				SubnetName:  &subnetName,
			},
		}

		if tags, ok := gcpConfig[FieldGCPTags]; ok {
			tagsList := make([]string, 0)
			for _, tag := range tags.([]interface{}) {
				tagsList = append(tagsList, tag.(string))
			}
			req.Gcp.Networking.Tags = &tagsList
		}
	}

	// Set OCI parameters
	if v, ok := data.GetOk(FieldOmniEdgeLocationOCI); ok && len(v.([]interface{})) > 0 {
		ociConfig := v.([]interface{})[0].(map[string]interface{})

		tenancyID := ociConfig[FieldOCITenancyID].(string)
		compartmentID := ociConfig[FieldOCICompartmentID].(string)
		userID := ociConfig[FieldOCIUserID].(string)
		fingerprint := ociConfig[FieldOCIFingerprint].(string)
		privateKey := ociConfig[FieldOCIPrivateKeyBase64].(string)
		vcnID := ociConfig[FieldOCIVcnID].(string)
		subnetID := ociConfig[FieldOCISubnetID].(string)

		req.Oci = &omni_provisioner.CastaiOmniProvisionerV1beta1OciParameters{
			TenancyId:     &tenancyID,
			CompartmentId: &compartmentID,
			Credentials: &omni_provisioner.CastaiOmniProvisionerV1beta1OciCredentials{
				UserId:           &userID,
				Fingerprint:      &fingerprint,
				PrivateKeyBase64: &privateKey,
			},
			Networking: &omni_provisioner.CastaiOmniProvisionerV1beta1OciNetworking{
				VcnId:    &vcnID,
				SubnetId: &subnetID,
			},
		}
	}

	resp, err := client.CreateEdgeLocationWithResponse(ctx, organizationID, clusterID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("creating edge location: %w", err))
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return diag.FromErr(fmt.Errorf("creating edge location: unexpected status code %d", resp.StatusCode()))
	}

	if resp.JSON200 == nil || resp.JSON200.Id == nil {
		return diag.Errorf("edge location response is nil or missing ID")
	}

	data.SetId(*resp.JSON200.Id)

	// Onboard the edge location
	onboardResp, err := client.OnboardEdgeLocationWithResponse(ctx, organizationID, clusterID, *resp.JSON200.Id)
	if err != nil {
		return diag.FromErr(fmt.Errorf("onboarding edge location: %w", err))
	}

	if onboardResp.StatusCode() != http.StatusOK && onboardResp.StatusCode() != http.StatusCreated {
		return diag.FromErr(fmt.Errorf("onboarding edge location: unexpected status code %d", onboardResp.StatusCode()))
	}

	return resourceOmniEdgeLocationRead(ctx, data, meta)
}

func resourceOmniEdgeLocationRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniProvisionerClient

	organizationID := data.Get(FieldOmniEdgeLocationOrganizationID).(string)
	clusterID := data.Get(FieldOmniEdgeLocationClusterID).(string)
	edgeLocationID := data.Id()

	resp, err := client.GetEdgeLocationWithResponse(ctx, organizationID, clusterID, edgeLocationID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("getting edge location: %w", err))
	}

	if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Edge location (%s) not found, removing from state", edgeLocationID)
		data.SetId("")
		return nil
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("getting edge location: unexpected status code %d", resp.StatusCode()))
	}

	location := resp.JSON200
	if location == nil {
		return diag.Errorf("edge location response is nil")
	}

	if err := data.Set(FieldOmniEdgeLocationName, location.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %w", err))
	}

	if location.Region != nil {
		if err := data.Set(FieldOmniEdgeLocationRegion, *location.Region); err != nil {
			return diag.FromErr(fmt.Errorf("setting region: %w", err))
		}
	}

	if location.Zones != nil {
		if err := data.Set(FieldOmniEdgeLocationZones, *location.Zones); err != nil {
			return diag.FromErr(fmt.Errorf("setting zones: %w", err))
		}
	}

	if location.Description != nil {
		if err := data.Set(FieldOmniEdgeLocationDescription, *location.Description); err != nil {
			return diag.FromErr(fmt.Errorf("setting description: %w", err))
		}
	}

	if location.State != nil {
		if err := data.Set(FieldOmniEdgeLocationState, string(*location.State)); err != nil {
			return diag.FromErr(fmt.Errorf("setting state: %w", err))
		}
	}

	if location.TotalEdgeCount != nil {
		if err := data.Set(FieldOmniEdgeLocationTotalEdgeCount, int(*location.TotalEdgeCount)); err != nil {
			return diag.FromErr(fmt.Errorf("setting total_edge_count: %w", err))
		}
	}

	return nil
}

func resourceOmniEdgeLocationUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniProvisionerClient

	organizationID := data.Get(FieldOmniEdgeLocationOrganizationID).(string)
	clusterID := data.Get(FieldOmniEdgeLocationClusterID).(string)
	edgeLocationID := data.Id()

	req := omni_provisioner.UpdateEdgeLocationJSONRequestBody{}
	updated := false

	if data.HasChange(FieldOmniEdgeLocationName) {
		name := data.Get(FieldOmniEdgeLocationName).(string)
		req.Name = &name
		updated = true
	}

	if data.HasChange(FieldOmniEdgeLocationDescription) {
		desc := data.Get(FieldOmniEdgeLocationDescription).(string)
		req.Description = &desc
		updated = true
	}

	if updated {
		resp, err := client.UpdateEdgeLocationWithResponse(ctx, organizationID, clusterID, edgeLocationID, req)
		if err != nil {
			return diag.FromErr(fmt.Errorf("updating edge location: %w", err))
		}

		if resp.StatusCode() != http.StatusOK {
			return diag.FromErr(fmt.Errorf("updating edge location: unexpected status code %d", resp.StatusCode()))
		}
	}

	return resourceOmniEdgeLocationRead(ctx, data, meta)
}

func resourceOmniEdgeLocationDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniProvisionerClient

	organizationID := data.Get(FieldOmniEdgeLocationOrganizationID).(string)
	clusterID := data.Get(FieldOmniEdgeLocationClusterID).(string)
	edgeLocationID := data.Id()

	// First offboard
	offboardResp, err := client.OffboardEdgeLocationWithResponse(ctx, organizationID, clusterID, edgeLocationID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("offboarding edge location: %w", err))
	}

	if offboardResp.StatusCode() != http.StatusOK && offboardResp.StatusCode() != http.StatusNoContent {
		return diag.FromErr(fmt.Errorf("offboarding edge location: unexpected status code %d", offboardResp.StatusCode()))
	}

	// Then delete
	deleteResp, err := client.DeleteEdgeLocationWithResponse(ctx, organizationID, clusterID, edgeLocationID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("deleting edge location: %w", err))
	}

	if deleteResp.StatusCode() != http.StatusOK && deleteResp.StatusCode() != http.StatusNoContent && deleteResp.StatusCode() != http.StatusNotFound {
		return diag.FromErr(fmt.Errorf("deleting edge location: unexpected status code %d", deleteResp.StatusCode()))
	}

	return nil
}
