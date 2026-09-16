package castai

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni"
	mock_omni "github.com/castai/terraform-provider-castai/castai/sdk/omni/mock"
)

func TestAccCloudAgnostic_ResourceEdgeConfigurationGCP(t *testing.T) {
	rName := fmt.Sprintf("%v-edgecfg-%v", ResourcePrefix, acctest.RandString(8))
	clusterName := fmt.Sprintf("omni-tf-acc-gcp-%v", acctest.RandString(6))
	resourceName := "castai_edge_configuration.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeResourcesDestroy(testAccCheckEdgeConfigurationDestroy),
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeConfigurationGCPConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_id"),
					resource.TestCheckResourceAttrSet(resourceName, "edge_location_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "default", "false"),
					resource.TestCheckResourceAttr(resourceName, "gcp.image_id", "projects/castai/global/images/castai-edge-v1"),
					resource.TestCheckResourceAttr(resourceName, "gcp.labels.key1", "value1"),
					resource.TestCheckResourceAttr(resourceName, "gcp.labels.key2", "value2"),
					resource.TestCheckResourceAttr(resourceName, "gcp.boot_disk_size_gib", "100"),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd.sock"),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					organizationID := testAccGetOrganizationID()
					clusterID := s.RootModule().Resources["castai_omni_cluster.test"].Primary.ID
					edgeLocationID := s.RootModule().Resources["castai_edge_location.test"].Primary.ID
					configID := s.RootModule().Resources["castai_edge_configuration.test"].Primary.ID
					return fmt.Sprintf("%s/%s/%s/%s", organizationID, clusterID, edgeLocationID, configID), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEdgeConfigurationGCPUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-updated"),
					resource.TestCheckResourceAttr(resourceName, "gcp.image_id", "projects/castai/global/images/castai-edge-v2"),
					resource.TestCheckResourceAttr(resourceName, "gcp.labels.key1", "updated-value1"),
					resource.TestCheckResourceAttr(resourceName, "gcp.labels.key2", "updated-value2"),
					resource.TestCheckResourceAttr(resourceName, "gcp.labels.newkey", "newvalue"),
					resource.TestCheckResourceAttr(resourceName, "gcp.boot_disk_size_gib", "200"),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd-updated.sock"),
				),
			},
		},
	})
}

func TestAccCloudAgnostic_ResourceEdgeConfigurationAWS(t *testing.T) {
	rName := fmt.Sprintf("%v-edgecfg-%v", ResourcePrefix, acctest.RandString(8))
	clusterName := fmt.Sprintf("omni-tf-acc-aws-cfg-%v", acctest.RandString(6))
	resourceName := "castai_edge_configuration.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeConfigurationAWSConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_id"),
					resource.TestCheckResourceAttrSet(resourceName, "edge_location_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "default", "false"),
					resource.TestCheckResourceAttr(resourceName, "aws.image_id", "ami-0abcdef1234567890"),
					resource.TestCheckResourceAttr(resourceName, "aws.tags.key1", "value1"),
					resource.TestCheckResourceAttr(resourceName, "aws.tags.key2", "value2"),
					resource.TestCheckResourceAttr(resourceName, "aws.boot_disk_size_gib", "100"),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd.sock"),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					organizationID := testAccGetOrganizationID()
					clusterID := s.RootModule().Resources["castai_omni_cluster.test"].Primary.ID
					edgeLocationID := s.RootModule().Resources["castai_edge_location.test"].Primary.ID
					configID := s.RootModule().Resources["castai_edge_configuration.test"].Primary.ID
					return fmt.Sprintf("%s/%s/%s/%s", organizationID, clusterID, edgeLocationID, configID), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEdgeConfigurationAWSUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-updated"),
					resource.TestCheckResourceAttr(resourceName, "aws.image_id", "ami-0updated1234567890"),
					resource.TestCheckResourceAttr(resourceName, "aws.tags.key1", "updated-value1"),
					resource.TestCheckResourceAttr(resourceName, "aws.tags.key2", "updated-value2"),
					resource.TestCheckResourceAttr(resourceName, "aws.tags.newkey", "newvalue"),
					resource.TestCheckResourceAttr(resourceName, "aws.boot_disk_size_gib", "200"),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd-updated.sock"),
				),
			},
		},
	})
}

func TestAccCloudAgnostic_ResourceEdgeConfigurationOCI(t *testing.T) {
	rName := fmt.Sprintf("%v-edgecfg-%v", ResourcePrefix, acctest.RandString(8))
	clusterName := fmt.Sprintf("test-oci-cluster-cfg-%v", acctest.RandString(6))
	resourceName := "castai_edge_configuration.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeConfigurationOCIConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_id"),
					resource.TestCheckResourceAttrSet(resourceName, "edge_location_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "default", "false"),
					resource.TestCheckResourceAttr(resourceName, "oci.image_id", "ocid1.image.oc1.phx.example"),
					resource.TestCheckResourceAttr(resourceName, "oci.tags.key1", "value1"),
					resource.TestCheckResourceAttr(resourceName, "oci.tags.key2", "value2"),
					resource.TestCheckResourceAttr(resourceName, "oci.boot_disk_size_gib", "100"),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd.sock"),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					organizationID := testAccGetOrganizationID()
					clusterID := s.RootModule().Resources["castai_omni_cluster.test"].Primary.ID
					edgeLocationID := s.RootModule().Resources["castai_edge_location.test"].Primary.ID
					configID := s.RootModule().Resources["castai_edge_configuration.test"].Primary.ID
					return fmt.Sprintf("%s/%s/%s/%s", organizationID, clusterID, edgeLocationID, configID), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEdgeConfigurationOCIUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-updated"),
					resource.TestCheckResourceAttr(resourceName, "oci.image_id", "ocid1.image.oc1.phx.updated"),
					resource.TestCheckResourceAttr(resourceName, "oci.tags.key1", "updated-value1"),
					resource.TestCheckResourceAttr(resourceName, "oci.tags.key2", "updated-value2"),
					resource.TestCheckResourceAttr(resourceName, "oci.tags.newkey", "newvalue"),
					resource.TestCheckResourceAttr(resourceName, "oci.boot_disk_size_gib", "200"),
					resource.TestCheckResourceAttr(resourceName, "user_data_base64", "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd-updated.sock"),
				),
			},
		},
	})
}

func TestAccCloudAgnostic_ResourceEdgeConfigurationNebius(t *testing.T) {
	rName := fmt.Sprintf("%v-edgecfg-%v", ResourcePrefix, acctest.RandString(8))
	clusterName := fmt.Sprintf("omni-tf-acc-nebius-cfg-%v", acctest.RandString(6))
	resourceName := "castai_edge_configuration.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEdgeResourcesDestroy(testAccCheckEdgeConfigurationDestroy),
		Steps: []resource.TestStep{
			{
				Config: testAccEdgeConfigurationNebiusConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_id"),
					resource.TestCheckResourceAttrSet(resourceName, "edge_location_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "default", "false"),
					resource.TestCheckResourceAttr(resourceName, "nebius.image_id", "projects/nebius/global/images/nebius-edge-v1"),
					resource.TestCheckResourceAttr(resourceName, "nebius.boot_disk_size_gib", "100"),
					resource.TestCheckResourceAttr(resourceName, "nebius.labels.key1", "value1"),
					resource.TestCheckResourceAttr(resourceName, "nebius.reservation_ids.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "nebius.reservation_ids.0", "res-1"),
					resource.TestCheckResourceAttr(resourceName, "nebius.reservation_ids.1", "res-2"),
					resource.TestCheckResourceAttr(resourceName, "nebius.gpu_cluster", "gpu-cluster-a"),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd.sock"),
				),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					organizationID := testAccGetOrganizationID()
					clusterID := s.RootModule().Resources["castai_omni_cluster.test"].Primary.ID
					edgeLocationID := s.RootModule().Resources["castai_edge_location.test"].Primary.ID
					configID := s.RootModule().Resources["castai_edge_configuration.test"].Primary.ID
					return fmt.Sprintf("%s/%s/%s/%s", organizationID, clusterID, edgeLocationID, configID), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEdgeConfigurationNebiusUpdated(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-updated"),
					resource.TestCheckResourceAttr(resourceName, "nebius.image_id", "projects/nebius/global/images/nebius-edge-v2"),
					resource.TestCheckResourceAttr(resourceName, "nebius.boot_disk_size_gib", "200"),
					resource.TestCheckResourceAttr(resourceName, "nebius.labels.key1", "updated-value1"),
					resource.TestCheckResourceAttr(resourceName, "nebius.labels.newkey", "newvalue"),
					resource.TestCheckResourceAttr(resourceName, "nebius.reservation_ids.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "nebius.reservation_ids.0", "res-updated"),
					resource.TestCheckResourceAttr(resourceName, "nebius.gpu_cluster", "gpu-cluster-updated"),
					resource.TestCheckResourceAttr(resourceName, "cri.socket", "unix:///run/containerd/containerd-updated.sock"),
				),
			},
		},
	})
}

func TestEdgeConfigurationResource_toNebiusConfiguration_Conversions(t *testing.T) {
	t.Parallel()

	r := &edgeConfigurationResource{}
	ctx := context.Background()

	labelsMap, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{"env": "prod", "team": "platform"})
	require.False(t, diags.HasError())

	reservationList, diags := types.ListValueFrom(ctx, types.StringType, []string{"res-1", "res-2"})
	require.False(t, diags.HasError())

	tests := map[string]struct {
		plan     *nebiusConfigurationModel
		expected *omni.NebiusConfiguration
		expError string
	}{
		"all fields populated": {
			plan: &nebiusConfigurationModel{
				ImageID:         types.StringValue("nebius-image-123"),
				BootDiskSizeGiB: types.Int64Value(100),
				Labels:          labelsMap,
				ReservationIDs:  reservationList,
				GpuCluster:      types.StringValue("gpu-cluster-a"),
			},
			expected: &omni.NebiusConfiguration{
				ImageId:         lo.ToPtr("nebius-image-123"),
				BootDiskSizeGib: lo.ToPtr(int32(100)),
				Labels:          &map[string]string{"env": "prod", "team": "platform"},
				ReservationIds:  &[]string{"res-1", "res-2"},
				GpuCluster:      lo.ToPtr("gpu-cluster-a"),
			},
		},
		"empty string ImageID is skipped": {
			plan: &nebiusConfigurationModel{
				ImageID:         types.StringValue(""),
				BootDiskSizeGiB: types.Int64Value(50),
			},
			expected: &omni.NebiusConfiguration{
				BootDiskSizeGib: lo.ToPtr(int32(50)),
			},
		},
		"all null/zero values skipped": {
			plan: &nebiusConfigurationModel{
				ImageID:         types.StringNull(),
				BootDiskSizeGiB: types.Int64Null(),
				Labels:          types.MapNull(types.StringType),
				ReservationIDs:  types.ListNull(types.StringType),
				GpuCluster:      types.StringNull(),
			},
			expected: &omni.NebiusConfiguration{},
		},
		"labels with wrong element type produces diagnostics": {
			plan: &nebiusConfigurationModel{
				ImageID: types.StringValue("img"),
				// Build a Map that declares StringType but holds Int64 values,
				// forcing ElementsAs to fail.
				Labels: func() types.Map {
					m, _ := types.MapValueFrom(ctx, types.Int64Type, map[string]int64{"k": 1})
					return m
				}(),
			},
			expected: &omni.NebiusConfiguration{
				ImageId: lo.ToPtr("img"),
			},
			expError: "can't unmarshal tftypes.Number into *string, expected string",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			config, diags := r.toNebiusConfiguration(ctx, tc.plan)
			if tc.expError != "" {
				require.True(t, diags.HasError(), "expected diagnostics on labels conversion failure")
				var found bool
				for _, d := range diags.Errors() {
					msg := d.Summary() + ": " + d.Detail()
					if strings.Contains(msg, tc.expError) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected diagnostic to contain %q, got %v", tc.expError, diags)
			} else {
				assert.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
			}
			assert.Equal(t, tc.expected, config)
		})
	}
}

func TestEdgeConfigurationResource_toNebiusConfigurationModel(t *testing.T) {
	t.Parallel()

	r := &edgeConfigurationResource{}
	ctx := context.Background()

	t.Run("nil config returns nil", func(t *testing.T) {
		t.Parallel()
		model, diag := r.toNebiusConfigurationModel(ctx, nil)
		assert.Nil(t, model)
		assert.False(t, diag.HasError())
	})

	expectedLabels, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{"foo": "bar"})
	require.False(t, diags.HasError())

	expectedReservations, diags := types.ListValueFrom(ctx, types.StringType, []string{"r1", "r2", "r3"})
	require.False(t, diags.HasError())

	fullyNullModel := &nebiusConfigurationModel{
		Labels:          types.MapNull(types.StringType),
		ImageID:         types.StringNull(),
		BootDiskSizeGiB: types.Int64Null(),
		ReservationIDs:  types.ListNull(types.StringType),
		GpuCluster:      types.StringNull(),
	}

	tests := map[string]struct {
		config   *omni.NebiusConfiguration
		expected *nebiusConfigurationModel
	}{
		"all fields populated": {
			config: &omni.NebiusConfiguration{
				ImageId:         lo.ToPtr("nebius-image-xyz"),
				BootDiskSizeGib: lo.ToPtr(int32(200)),
				Labels:          &map[string]string{"foo": "bar"},
				ReservationIds:  &[]string{"r1", "r2", "r3"},
				GpuCluster:      lo.ToPtr("gpu-cluster-z"),
			},
			expected: &nebiusConfigurationModel{
				ImageID:         types.StringValue("nebius-image-xyz"),
				BootDiskSizeGiB: types.Int64Value(200),
				Labels:          expectedLabels,
				ReservationIDs:  expectedReservations,
				GpuCluster:      types.StringValue("gpu-cluster-z"),
			},
		},
		"empty string ImageId becomes null": {
			config: &omni.NebiusConfiguration{
				ImageId: lo.ToPtr(""),
			},
			expected: fullyNullModel,
		},
		"nil Labels and ReservationIds produce null values": {
			config: &omni.NebiusConfiguration{
				ImageId: lo.ToPtr("img-only"),
			},
			expected: &nebiusConfigurationModel{
				Labels:          types.MapNull(types.StringType),
				ImageID:         types.StringValue("img-only"),
				BootDiskSizeGiB: types.Int64Null(),
				ReservationIDs:  types.ListNull(types.StringType),
				GpuCluster:      types.StringNull(),
			},
		},
		"empty Labels and ReservationIds produce null values": {
			config: &omni.NebiusConfiguration{
				Labels:         &map[string]string{},
				ReservationIds: &[]string{},
			},
			expected: fullyNullModel,
		},
		"zero BootDiskSizeGib is preserved as non-null zero": {
			config: &omni.NebiusConfiguration{
				BootDiskSizeGib: lo.ToPtr(int32(0)),
			},
			expected: &nebiusConfigurationModel{
				Labels:          types.MapNull(types.StringType),
				ImageID:         types.StringNull(),
				BootDiskSizeGiB: types.Int64Value(0),
				ReservationIDs:  types.ListNull(types.StringType),
				GpuCluster:      types.StringNull(),
			},
		},
		"all fields missing produce fully-null model": {
			config:   &omni.NebiusConfiguration{},
			expected: fullyNullModel,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			model, diag := r.toNebiusConfigurationModel(ctx, tc.config)
			assert.Equal(t, tc.expected, model)
			assert.False(t, diag.HasError())
		})
	}
}

func TestEdgeConfigurationResource_Update_Nebius(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	labels, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"key1": "value1",
	})
	require.False(t, diags.HasError())

	reservations, diags := types.ListValueFrom(ctx, types.StringType, []string{"res-1", "res-2"})
	require.False(t, diags.HasError())

	tests := map[string]struct {
		name                   string
		planModel              edgeConfigurationModel
		expectedOrganizationID string
		expectedClusterID      string
		expectedEdgeLocationID string
		expectedID             string
		expectedUpdateReq      omni.EdgeConfigurationUpdate
		expectedUpdateParams   omni.EdgeConfigurationsAPIUpdateEdgeConfigurationParams
		apiResponse            *omni.EdgeConfiguration
	}{
		"regular update": {
			expectedOrganizationID: "org-1",
			expectedClusterID:      "cluster-1",
			expectedEdgeLocationID: "edge-loc-1",
			expectedID:             "cfg-initial",
			planModel: edgeConfigurationModel{
				ID:             types.StringValue("cfg-initial"),
				OrganizationID: types.StringValue("org-1"),
				ClusterID:      types.StringValue("cluster-1"),
				EdgeLocationID: types.StringValue("edge-loc-1"),
				Name:           types.StringValue("test-nebius-initial"),
				UserDataBase64: types.StringValue("I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="),
				CRI: &criConfigurationModel{
					Socket: types.StringValue("unix:///run/containerd/containerd.sock"),
				},
				Nebius: &nebiusConfigurationModel{
					ImageID:         types.StringValue("projects/nebius/global/images/nebius-edge-v1"),
					BootDiskSizeGiB: types.Int64Value(100),
					Labels:          labels,
					ReservationIDs:  reservations,
					GpuCluster:      types.StringValue("gpu-cluster-a"),
				},
			},
			expectedUpdateReq: omni.EdgeConfigurationUpdate{
				Name:           lo.ToPtr("test-nebius-initial"),
				UserDataBase64: lo.ToPtr("I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="),
				Nebius: &omni.NebiusConfiguration{
					ImageId:         lo.ToPtr("projects/nebius/global/images/nebius-edge-v1"),
					BootDiskSizeGib: lo.ToPtr(int32(100)),
					Labels:          &map[string]string{"key1": "value1"},
					ReservationIds:  &[]string{"res-1", "res-2"},
					GpuCluster:      lo.ToPtr("gpu-cluster-a"),
				},
				Cri: &omni.EdgeConfigurationCRIConfiguration{
					Socket: lo.ToPtr("unix:///run/containerd/containerd.sock"),
				},
			},
			expectedUpdateParams: omni.EdgeConfigurationsAPIUpdateEdgeConfigurationParams{
				UpdateMask: lo.ToPtr("nebius.gpu_cluster,nebius.reservation_ids"),
			},
			apiResponse: &omni.EdgeConfiguration{
				Id:             lo.ToPtr("cfg-initial"),
				EdgeLocationId: lo.ToPtr("edge-loc-1"),
				Name:           "test-nebius-initial",
				Default:        lo.ToPtr(false),
				UserDataBase64: lo.ToPtr("I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="),
				Nebius: &omni.NebiusConfiguration{
					ImageId:         lo.ToPtr("projects/nebius/global/images/nebius-edge-v1"),
					BootDiskSizeGib: lo.ToPtr(int32(100)),
					Labels:          &map[string]string{"key1": "value1"},
					ReservationIds:  &[]string{"res-1", "res-2"},
					GpuCluster:      lo.ToPtr("gpu-cluster-a"),
				},
				Cri: &omni.EdgeConfigurationCRIConfiguration{
					Socket: lo.ToPtr("unix:///run/containerd/containerd.sock"),
				},
			},
		},
		"update with deletion of fields": {
			expectedOrganizationID: "org-1",
			expectedClusterID:      "cluster-1",
			expectedEdgeLocationID: "edge-loc-1",
			expectedID:             "cfg-initial",
			planModel: edgeConfigurationModel{
				ID:             types.StringValue("cfg-initial"),
				OrganizationID: types.StringValue("org-1"),
				ClusterID:      types.StringValue("cluster-1"),
				EdgeLocationID: types.StringValue("edge-loc-1"),
				Name:           types.StringValue("test-nebius-initial"),
				UserDataBase64: types.StringValue("I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="),
				CRI: &criConfigurationModel{
					Socket: types.StringValue("unix:///run/containerd/containerd.sock"),
				},
				Nebius: &nebiusConfigurationModel{
					ImageID:         types.StringValue("projects/nebius/global/images/nebius-edge-v1"),
					BootDiskSizeGiB: types.Int64Value(100),
					Labels:          labels,
					ReservationIDs:  reservations,
				},
			},
			expectedUpdateReq: omni.EdgeConfigurationUpdate{
				Name:           lo.ToPtr("test-nebius-initial"),
				UserDataBase64: lo.ToPtr("I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="),
				Nebius: &omni.NebiusConfiguration{
					ImageId:         lo.ToPtr("projects/nebius/global/images/nebius-edge-v1"),
					BootDiskSizeGib: lo.ToPtr(int32(100)),
					Labels:          &map[string]string{"key1": "value1"},
					ReservationIds:  &[]string{"res-1", "res-2"},
				},
				Cri: &omni.EdgeConfigurationCRIConfiguration{
					Socket: lo.ToPtr("unix:///run/containerd/containerd.sock"),
				},
			},
			expectedUpdateParams: omni.EdgeConfigurationsAPIUpdateEdgeConfigurationParams{
				UpdateMask: lo.ToPtr("nebius.gpu_cluster,nebius.reservation_ids"),
			},
			apiResponse: &omni.EdgeConfiguration{
				Id:             lo.ToPtr("cfg-initial"),
				EdgeLocationId: lo.ToPtr("edge-loc-1"),
				Name:           "test-nebius-initial",
				Default:        lo.ToPtr(false),
				UserDataBase64: lo.ToPtr("I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="),
				Nebius: &omni.NebiusConfiguration{
					ImageId:         lo.ToPtr("projects/nebius/global/images/nebius-edge-v1"),
					BootDiskSizeGib: lo.ToPtr(int32(100)),
					Labels:          &map[string]string{"key1": "value1"},
					ReservationIds:  &[]string{"res-1", "res-2"},
					GpuCluster:      nil,
				},
				Cri: &omni.EdgeConfigurationCRIConfiguration{
					Socket: lo.ToPtr("unix:///run/containerd/containerd.sock"),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockOmni := mock_omni.NewMockClientWithResponsesInterface(ctrl)

			req := buildTFEdgeConfigurationUpdateRequest(t, ctx, tc.planModel)

			r := &edgeConfigurationResource{}
			schemaResp := fwresource.SchemaResponse{}
			r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

			resp := &fwresource.UpdateResponse{
				State: tfsdk.State{
					Schema: schemaResp.Schema,
				},
			}

			var capturedReq omni.EdgeConfigurationsAPIUpdateEdgeConfigurationJSONRequestBody

			mockOmni.EXPECT().
				EdgeConfigurationsAPIUpdateEdgeConfigurationWithResponse(
					gomock.Any(),
					tc.planModel.OrganizationID.ValueString(),
					tc.planModel.ClusterID.ValueString(),
					tc.planModel.EdgeLocationID.ValueString(),
					tc.planModel.ID.ValueString(),
					gomock.Any(),
					gomock.Any(),
				).
				DoAndReturn(func(_ context.Context, orgID, clusterID, edgeLocID, id string, params *omni.EdgeConfigurationsAPIUpdateEdgeConfigurationParams, body omni.EdgeConfigurationsAPIUpdateEdgeConfigurationJSONRequestBody) (*omni.EdgeConfigurationsAPIUpdateEdgeConfigurationResponse, error) {
					assert.Equal(t, tc.expectedOrganizationID, orgID)
					assert.Equal(t, tc.expectedClusterID, clusterID)
					assert.Equal(t, tc.expectedEdgeLocationID, edgeLocID)
					assert.Equal(t, tc.expectedID, id)

					require.NotNil(t, params, "expected req params")
					require.NotNil(t, params.UpdateMask, "expected req params update mask")
					require.NotNil(t, params.UpdateMask, "expected req params update mask")
					require.NotNil(t, tc.expectedUpdateParams.UpdateMask, "expected update mask")
					assert.Equal(t, *tc.expectedUpdateParams.UpdateMask, *params.UpdateMask)

					capturedReq = body
					return &omni.EdgeConfigurationsAPIUpdateEdgeConfigurationResponse{
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						JSON200:      tc.apiResponse,
					}, nil
				})

			r = &edgeConfigurationResource{client: &ProviderConfig{omniAPI: mockOmni}}
			r.Update(ctx, req, resp)

			assert.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
			assert.Equal(t, tc.expectedUpdateReq, capturedReq)
		})
	}
}

// buildTFEdgeConfigurationUpdateRequest constructs resource.UpdateRequest with State, Plan and Config
// derived from the provided edgeConfigurationModel, using the resource schema.
func buildTFEdgeConfigurationUpdateRequest(t *testing.T, ctx context.Context, model edgeConfigurationModel) fwresource.UpdateRequest {
	t.Helper()
	r := &edgeConfigurationResource{}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	objType := schemaResp.Schema.Type().(basetypes.ObjectType)
	attrTypes := objType.AttrTypes

	objVal, diags := types.ObjectValueFrom(ctx, attrTypes, &model)
	require.Falsef(t, diags.HasError(), "failed to build ObjectValueFrom from model: %+v", diags)

	rawVal, err := objVal.ToTerraformValue(ctx)
	require.NoError(t, err)

	return fwresource.UpdateRequest{
		State: tfsdk.State{
			Raw:    rawVal,
			Schema: schemaResp.Schema,
		},
		Plan: tfsdk.Plan{
			Raw:    rawVal,
			Schema: schemaResp.Schema,
		},
		Config: tfsdk.Config{
			Raw:    rawVal,
			Schema: schemaResp.Schema,
		},
	}
}

func testAccEdgeConfigurationGCPConfig(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationGCPImpersonationConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = %[2]q
  user_data_base64 = "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="

  cri = {
    socket = "unix:///run/containerd/containerd.sock"
  }

  gcp = {
    image_id          = "projects/castai/global/images/castai-edge-v1"
    boot_disk_size_gib = 100
    labels = {
      key1 = "value1"
      key2 = "value2"
    }
  }
}
`, organizationID, rName),
	)
}

func testAccEdgeConfigurationGCPUpdated(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationGCPImpersonationConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = "%[2]s-updated"
  user_data_base64 = "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="

  cri = {
    socket = "unix:///run/containerd/containerd-updated.sock"
  }

  gcp = {
    image_id          = "projects/castai/global/images/castai-edge-v2"
    boot_disk_size_gib = 200
    labels = {
      key1   = "updated-value1"
      key2   = "updated-value2"
      newkey = "newvalue"
    }
  }
}
`, organizationID, rName),
	)
}

func testAccEdgeConfigurationAWSConfig(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationAWSImpersonationConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = %[2]q
  user_data_base64 = "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="

  cri = {
    socket = "unix:///run/containerd/containerd.sock"
  }

  aws = {
    image_id          = "ami-0abcdef1234567890"
    boot_disk_size_gib = 100
    tags = {
      key1 = "value1"
      key2 = "value2"
    }
  }
}
`, organizationID, rName),
	)
}

func testAccEdgeConfigurationAWSUpdated(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationAWSImpersonationConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = "%[2]s-updated"
  user_data_base64 = "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="

  cri = {
    socket = "unix:///run/containerd/containerd-updated.sock"
  }

  aws = {
    image_id          = "ami-0updated1234567890"
    boot_disk_size_gib = 200
    tags = {
      key1   = "updated-value1"
      key2   = "updated-value2"
      newkey = "newvalue"
    }
  }
}
`, organizationID, rName),
	)
}

func testAccEdgeConfigurationOCIConfig(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationOCIConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = %[2]q
  user_data_base64 = "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="

  cri = {
    socket = "unix:///run/containerd/containerd.sock"
  }

  oci = {
    image_id          = "ocid1.image.oc1.phx.example"
    boot_disk_size_gib = 100
    tags = {
      key1 = "value1"
      key2 = "value2"
    }
  }
}
`, organizationID, rName),
	)
}

func testAccEdgeConfigurationOCIUpdated(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationOCIConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = "%[2]s-updated"
  user_data_base64 = "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="

  cri = {
    socket = "unix:///run/containerd/containerd-updated.sock"
  }

  oci = {
    image_id          = "ocid1.image.oc1.phx.updated"
    boot_disk_size_gib = 200
    tags = {
      key1   = "updated-value1"
      key2   = "updated-value2"
      newkey = "newvalue"
    }
  }
}
`, organizationID, rName),
	)
}

func testAccEdgeConfigurationNebiusConfig(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationNebiusWIFConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = %[2]q
  user_data_base64 = "I2Nsb3VkLWNvbmZpZwojIFVzZXIgZGF0YQ=="

  cri = {
    socket = "unix:///run/containerd/containerd.sock"
  }

  nebius = {
    image_id           = "projects/nebius/global/images/nebius-edge-v1"
    boot_disk_size_gib = 100
    labels = {
      key1 = "value1"
    }
    reservation_ids = ["res-1", "res-2"]
    gpu_cluster     = "gpu-cluster-a"
  }
}
`, organizationID, rName),
	)
}

func testAccEdgeConfigurationNebiusUpdated(rName, clusterName string) string {
	organizationID := testAccGetOrganizationID()

	return ConfigCompose(
		testAccEdgeLocationNebiusWIFConfig(rName, clusterName),
		fmt.Sprintf(`
resource "castai_edge_configuration" "test" {
  organization_id  = %[1]q
  cluster_id       = castai_omni_cluster.test.id
  edge_location_id = castai_edge_location.test.id
  name             = "%[2]s-updated"
  user_data_base64 = "I2Nsb3VkLWNvbmZpZy11cGRhdGVkCg=="

  cri = {
    socket = "unix:///run/containerd/containerd-updated.sock"
  }

  nebius = {
    image_id           = "projects/nebius/global/images/nebius-edge-v2"
    boot_disk_size_gib = 200
    labels = {
      key1   = "updated-value1"
      newkey = "newvalue"
    }
    reservation_ids = ["res-updated"]
    gpu_cluster     = "gpu-cluster-updated"
  }
}
`, organizationID, rName),
	)
}

func testAccCheckEdgeConfigurationDestroy(s *terraform.State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := testAccProvider.Meta().(*ProviderConfig).omniAPI
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "castai_edge_configuration" {
			continue
		}

		organizationID := rs.Primary.Attributes["organization_id"]
		clusterID := rs.Primary.Attributes["cluster_id"]
		edgeLocationID := rs.Primary.Attributes["edge_location_id"]
		configID := rs.Primary.ID

		response, err := client.EdgeConfigurationsAPIGetEdgeConfigurationWithResponse(ctx, organizationID, clusterID, edgeLocationID, configID, nil)
		if err != nil {
			return err
		}
		if response.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("edge configuration %s still exists", rs.Primary.ID)
	}

	return nil
}
