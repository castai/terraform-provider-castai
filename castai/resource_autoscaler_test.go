package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
	"github.com/castai/terraform-provider-castai/castai/types"
)

func TestAutoscalerResource_PoliciesUpdateAction(t *testing.T) {
	currentPolicies := `
		{
		    "enabled": true,
		    "isScopedMode": false,
		    "unschedulablePods": {
		        "enabled": true,
		        "headroom": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "headroomSpot": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "nodeConstraints": {
		            "minCpuCores": 2,
		            "maxCpuCores": 32,
		            "minRamMib": 4096,
		            "maxRamMib": 262144,
		            "enabled": false
		        },
		        "diskGibToCpuRatio": 25
		    },
		    "clusterLimits": {
		        "enabled": false,
		        "cpu": {
		            "minCores": 1,
		            "maxCores": 20
		        }
		    },
		    "nodeDownscaler": {
		        "emptyNodes": {
		            "enabled": false,
		            "delaySeconds": 0
		        }
		    }
		}`

	// 1. enable scope mode
	// 2. enable node constraints and change max CPU
	// 3. enable spot backups
	// 4. change spot cloud to aws - just to test if we can do change on arrays
	// 5. enable the spot interruption predictions
	policyChanges := `{
		"isScopedMode":true,
		"unschedulablePods": {
			"nodeConstraints": {
				"enabled": true,
				"maxCpuCores": 96
			},
			"podPinner": {
				"enabled": false
			}
		}
	}`

	updatedPolicies := `
		{
		    "enabled": true,
		    "isScopedMode": true,
		    "unschedulablePods": {
		        "enabled": true,
		        "headroom": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "headroomSpot": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "nodeConstraints": {
		            "minCpuCores": 2,
		            "maxCpuCores": 96,
		            "minRamMib": 4096,
		            "maxRamMib": 262144,
		            "enabled": true
		        },
		        "diskGibToCpuRatio": 25,
				"podPinner": {
					"enabled": false
				}
		    },
		    "clusterLimits": {
		        "enabled": false,
		        "cpu": {
		            "minCores": 1,
		            "maxCores": 20
		        }
		    },
		    "nodeDownscaler": {
		        "emptyNodes": {
		            "enabled": false,
		            "delaySeconds": 0
		        }
		    }
		}`

	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceAutoscaler()

	clusterId := "cluster_id"
	val := cty.ObjectVal(map[string]cty.Value{
		FieldAutoscalerPoliciesJSON: cty.StringVal(policyChanges),
		FieldClusterId:              cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	body := io.NopCloser(bytes.NewReader([]byte(currentPolicies)))
	response := &http.Response{StatusCode: 200, Body: body}

	policiesUpdated := false

	mockClient.EXPECT().PoliciesAPIGetClusterPolicies(gomock.Any(), clusterId, gomock.Any()).Return(response, nil).Times(1)
	mockClient.EXPECT().PoliciesAPIUpsertClusterPoliciesWithBody(gomock.Any(), clusterId, "application/json", gomock.Any()).
		DoAndReturn(func(ctx context.Context, clusterId string, contentType string, body io.Reader) (*http.Response, error) {
			got, _ := io.ReadAll(body)
			expected := []byte(updatedPolicies)

			eq, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(eq, fmt.Sprintf("got:      %v\n"+
				"expected: %v\n", string(got), string(expected)))

			policiesUpdated = true

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(""))),
			}, nil
		}).Times(1)

	result := resource.UpdateContext(ctx, data, provider)
	r.Nil(result)
	r.True(policiesUpdated)
}

func TestAutoscalerResource_PoliciesUpdateAction_Fail(t *testing.T) {
	currentPolicies := `
		{
		    "enabled": true,
		    "isScopedMode": false,
		    "unschedulablePods": {
		        "enabled": true,
		        "headroom": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "headroomSpot": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "nodeConstraints": {
		            "minCpuCores": 2,
		            "maxCpuCores": 32,
		            "minRamMib": 4096,
		            "maxRamMib": 262144,
		            "enabled": false
		        },
		        "diskGibToCpuRatio": 25
		    },
		    "clusterLimits": {
		        "enabled": false,
		        "cpu": {
		            "minCores": 1,
		            "maxCores": 20
		        }
		    },
		    "nodeDownscaler": {
		        "emptyNodes": {
		            "enabled": false,
		            "delaySeconds": 0
		        }
		    }
		}`

	policyChanges := `{
		"isScopedMode":true,
		"unschedulablePods": {
			"nodeConstraints": {
				"enabled": true,
				"maxCpuCores": 96
			}
		}
	}`

	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceAutoscaler()

	clusterId := "cluster_id"
	val := cty.ObjectVal(map[string]cty.Value{
		FieldAutoscalerPoliciesJSON: cty.StringVal(policyChanges),
		FieldClusterId:              cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	body := io.NopCloser(bytes.NewReader([]byte(currentPolicies)))
	response := &http.Response{StatusCode: 200, Body: body}

	mockClient.EXPECT().PoliciesAPIGetClusterPolicies(gomock.Any(), clusterId, gomock.Any()).Return(response, nil).Times(1)
	mockClient.EXPECT().PoliciesAPIUpsertClusterPoliciesWithBody(gomock.Any(), clusterId, "application/json", gomock.Any()).
		DoAndReturn(func(ctx context.Context, clusterId string, contentType string, body io.Reader) (*http.Response, error) {
			return &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"message":"policies config: Evictor policy management is not allowed: Evictor installed externally. Uninstall Evictor first and try again.","fieldViolations":[]`))),
			}, nil
		}).Times(1)

	result := resource.UpdateContext(ctx, data, provider)
	r.NotNil(result)
	r.True(result.HasError())
	r.Equal(`expected status code 200, received: status=400 body={"message":"policies config: Evictor policy management is not allowed: Evictor installed externally. Uninstall Evictor first and try again.","fieldViolations":[]`, result[0].Summary)
}

func Test_validateAutoscalerPolicyJSON(t *testing.T) {
	type testData struct {
		json            string
		valid           bool
		expectedMessage string
	}
	tests := map[string]testData{
		"should return no diagnostic error for valid autoscaler policies JSON": {
			json: ` {
					     "enabled": true,
					     "unschedulablePods": {
					         "enabled": true
					     },
					    "nodeDownscaler": {
					         "enabled": true,
					         "emptyNodes": {
					             "enabled": true
					         },
					         "evictor": {
					             "aggressiveMode": true,
					             "cycleInterval": "5m10s",
					             "dryRun": false,
					             "enabled": true,
					             "nodeGracePeriodMinutes": 10,
					             "scopedMode": false
					         }
					     }
					}`,
			valid: true,
		},
		"should return diagnostic error if spot instances block is present in JSON": {
			json: ` {
					     "enabled": true,
					     "unschedulablePods": {
					         "enabled": true
					     },
					     "spotInstances": {
					         "enabled": true,
					         "clouds": ["gcp"],
					         "spotBackups": {
					             "enabled": true
					         }
					     },
					     "nodeDownscaler": {
					         "enabled": true,
					         "emptyNodes": {
					             "enabled": true
					         },
					         "evictor": {
					             "aggressiveMode": true,
					             "cycleInterval": "5m10s",
					             "dryRun": false,
					             "enabled": true,
					             "nodeGracePeriodMinutes": 10,
					             "scopedMode": false
					         }
					     }
					}`,
			valid:           false,
			expectedMessage: "'spotInstances' field was removed from policies JSON in 5.0.0. The configuration was migrated to default node template.",
		},
		"should return diagnostic error if custom instance enabled attribute is present in JSON": {
			json: ` {
					     "enabled": true,
					     "unschedulablePods": {
					         "enabled": true,
					         "customInstancesEnabled": true
					     },
					     "nodeDownscaler": {
					         "enabled": true,
					         "emptyNodes": {
					             "enabled": true
					         },
					         "evictor": {
					             "aggressiveMode": true,
					             "cycleInterval": "5m10s",
					             "dryRun": false,
					             "enabled": true,
					             "nodeGracePeriodMinutes": 10,
					             "scopedMode": false
					         }
					     }
					}`,
			valid:           false,
			expectedMessage: "'customInstancesEnabled' field was removed from policies JSON in 5.0.0. The configuration was migrated to default node template.",
		},

		"should return diagnostic error if node constraints attribute is present in JSON": {
			json: ` {
					     "enabled": true,
					     "unschedulablePods": {
					         "enabled": true,
					         "nodeConstraints": {}
					     },
					     "nodeDownscaler": {
					         "enabled": true,
					         "emptyNodes": {
					             "enabled": true
					         },
					         "evictor": {
					             "aggressiveMode": true,
					             "cycleInterval": "5m10s",
					             "dryRun": false,
					             "enabled": true,
					             "nodeGracePeriodMinutes": 10,
					             "scopedMode": false
					         }
					     }
					}`,
			valid:           false,
			expectedMessage: "'nodeConstraints' field was removed from policies JSON in 5.0.0. The configuration was migrated to default node template.",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := validateAutoscalerPolicyJSON()(tt.json, []cty.PathStep{cty.PathStep(nil)})
			require.Equal(t, tt.valid, !result.HasError())
			if !tt.valid {
				for _, d := range result {
					require.True(t, strings.Contains(d.Summary, tt.expectedMessage))
				}
			}
		})
	}
}

func TestAutoscalerResource_ReadPoliciesAction(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)
	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	currentPoliciesBytes, err := normalizeJSON([]byte(`
		{
		    "enabled": true,
		    "isScopedMode": false,
		    "unschedulablePods": {
		        "enabled": true,
		        "headroom": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "headroomSpot": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "nodeConstraints": {
		            "minCpuCores": 2,
		            "maxCpuCores": 32,
		            "minRamMib": 4096,
		            "maxRamMib": 262144,
		            "enabled": false
		        },
		        "diskGibToCpuRatio": 25
		    },
		    "clusterLimits": {
		        "enabled": false,
		        "cpu": {
		            "minCores": 1,
		            "maxCores": 20
		        }
		    },
		    "nodeDownscaler": {
		        "emptyNodes": {
		            "enabled": false,
		            "delaySeconds": 0
		        }
		    }
		}`))
	r.NoError(err)

	currentPolicies := string(currentPoliciesBytes)
	resource := resourceAutoscaler()

	clusterId := "cluster_id"
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	body := io.NopCloser(bytes.NewReader([]byte(currentPolicies)))
	response := &http.Response{StatusCode: 200, Body: body}

	mockClient.EXPECT().PoliciesAPIGetClusterPolicies(gomock.Any(), clusterId, gomock.Any()).Return(response, nil).Times(1)
	mockClient.EXPECT().PoliciesAPIUpsertClusterPoliciesWithBody(gomock.Any(), clusterId, "application/json", gomock.Any()).
		Times(0)

	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.Equal(currentPolicies, data.Get(FieldAutoscalerPolicies))
}

func TestAutoscalerResource_CustomizeDiff(t *testing.T) {
	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)
	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	currentPoliciesBytes, err := normalizeJSON([]byte(`
		{
		    "enabled": true,
		    "isScopedMode": false,
		    "unschedulablePods": {
		        "enabled": true,
		        "headroom": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "headroomSpot": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "nodeConstraints": {
		            "minCpuCores": 2,
		            "maxCpuCores": 32,
		            "minRamMib": 4096,
		            "maxRamMib": 262144,
		            "enabled": false
		        },
		        "diskGibToCpuRatio": 25
		    },
		    "clusterLimits": {
		        "enabled": false,
		        "cpu": {
		            "minCores": 1,
		            "maxCores": 20
		        }
		    },
		    "nodeDownscaler": {
		        "emptyNodes": {
		            "enabled": false,
		            "delaySeconds": 0
		        }
		    }
		}`))
	r.NoError(err)

	policyChangeBytes, err := normalizeJSON([]byte(`
		{
		    "enabled": false,
		    "unschedulablePods": {
		        "enabled": false
			}
		}`))
	r.NoError(err)

	expectedPoliciesBytes, err := normalizeJSON([]byte(`
		{
		    "enabled": false,
		    "isScopedMode": false,
		    "unschedulablePods": {
		        "enabled": false,
		        "headroom": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "headroomSpot": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "nodeConstraints": {
		            "minCpuCores": 2,
		            "maxCpuCores": 32,
		            "minRamMib": 4096,
		            "maxRamMib": 262144,
		            "enabled": false
		        },
		        "diskGibToCpuRatio": 25
		    },
		    "clusterLimits": {
		        "enabled": false,
		        "cpu": {
		            "minCores": 1,
		            "maxCores": 20
		        }
		    },
		    "nodeDownscaler": {
		        "emptyNodes": {
		            "enabled": false,
		            "delaySeconds": 0
		        }
		    }
		}`))
	r.NoError(err)

	currentPolicies := string(currentPoliciesBytes)
	policyChanges := string(policyChangeBytes)
	expectedPolicies := string(expectedPoliciesBytes)
	resource := resourceAutoscaler()

	clusterId := "cluster_id"
	val := cty.ObjectVal(map[string]cty.Value{
		FieldAutoscalerPoliciesJSON: cty.StringVal(policyChanges),
		FieldClusterId:              cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)
	r.NoError(err)

	body := io.NopCloser(bytes.NewReader([]byte(currentPolicies)))
	response := &http.Response{StatusCode: 200, Body: body}

	mockClient.EXPECT().PoliciesAPIGetClusterPolicies(gomock.Any(), clusterId, gomock.Any()).Return(response, nil).Times(1)
	mockClient.EXPECT().PoliciesAPIUpsertClusterPoliciesWithBody(gomock.Any(), clusterId, "application/json", gomock.Any()).
		Times(0)

	result, err := getChangedPolicies(ctx, data, provider, clusterId)
	r.NoError(err)
	r.Equal(expectedPolicies, string(result))
}

func TestAutoscalerResource_ToAutoscalerPolicy(t *testing.T) {
	tt := map[string]struct {
		data        cty.Value
		expected    *types.AutoscalerPolicy
		shouldFail  bool
		expectedErr error
	}{
		"should return nil when data is nil": {
			data:     cty.NilVal,
			expected: nil,
		},
		"should handle nested objects": {
			data: cty.ObjectVal(
				map[string]cty.Value{
					FieldAutoscalerSettings: cty.ListVal(
						[]cty.Value{
							cty.ObjectVal(
								map[string]cty.Value{
									"enabled": cty.BoolVal(true),
									"unschedulable_pods": cty.ListVal(
										[]cty.Value{
											cty.ObjectVal(
												map[string]cty.Value{
													"enabled": cty.BoolVal(true),
													"pod_pinner": cty.ListVal(
														[]cty.Value{
															cty.ObjectVal(
																map[string]cty.Value{
																	"enabled": cty.BoolVal(true),
																},
															),
														},
													),
												},
											),
										},
									),
								},
							),
						},
					),
				},
			),
			expected: &types.AutoscalerPolicy{
				Enabled: true,
				UnschedulablePods: &types.UnschedulablePods{
					Enabled:         true,
					CustomInstances: lo.ToPtr(false),
					PodPinner: &types.PodPinner{
						Enabled: true,
					},
				},
			},
		},
	}

	for testName, test := range tt {
		r := require.New(t)

		t.Run(testName, func(t *testing.T) {
			resource := resourceAutoscaler()
			state := terraform.NewInstanceStateShimmedFromValue(test.data, 0)
			actual, err := translateSettingsDataToPolicy(resource.Data(state))

			if test.shouldFail {
				r.Error(err)
				r.Error(err, test.expectedErr)
				return
			}

			r.NoError(err)
			r.Equal(test.expected, actual)
		})
	}
}

func TestAutoscalerResource_GetChangePolicies_ComparePolicyJsonAndDef(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockCtrl)

	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}
	tt := []struct {
		name           string
		current        string
		policyJson     string
		policyStruct   cty.Value
		expectedPolicy string
	}{
		// for list types values we can't unset primitive values. So, we need to set them to false. explicitly
		// we can't make new and old fields %100 consistent because of this.
		{
			name:           "simple policy",
			current:        `{"enabled":false}`,
			policyJson:     `{"enabled":false,"isScopedMode":false,"nodeTemplatesPartialMatchingEnabled":false}`,
			policyStruct:   cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{"enabled": cty.BoolVal(false)})}),
			expectedPolicy: `{"enabled":false,"isScopedMode":false,"nodeTemplatesPartialMatchingEnabled":false}`,
		},
		{
			name:           "with empty current policy",
			current:        `{}`,
			policyJson:     `{"enabled":false,"isScopedMode":false,"nodeTemplatesPartialMatchingEnabled":false}`,
			policyStruct:   cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{"enabled": cty.BoolVal(false)})}),
			expectedPolicy: `{"enabled":false,"isScopedMode":false,"nodeTemplatesPartialMatchingEnabled":false}`,
		},
		{
			name: "policy with nested objects",
			current: `{
				"enabled": true,
				"unschedulablePods": {
					"enabled": true,
					"headroom": {
						"cpuPercentage": 10,
						"memoryPercentage": 10,
						"enabled": true
					}
				},
				"nodeDownscaler": {
					"emptyNodes": {
						"enabled": false,
						"delaySeconds": 0
					}
				}
			}`,
			policyJson: `{
				"enabled": false,
				"isScopedMode": false,
				"nodeTemplatesPartialMatchingEnabled": false,
				"unschedulablePods": {
					"customInstancesEnabled":false,
					"enabled": false,
					"headroom": {
						"cpuPercentage": 100
					},
					"headroomSpot": {
						"cpuPercentage": 10,
						"memoryPercentage": 10,
						"enabled": true
					}
				}
			}`,
			policyStruct: cty.ListVal(
				[]cty.Value{
					cty.ObjectVal(
						map[string]cty.Value{
							"enabled": cty.BoolVal(false),
							"unschedulable_pods": cty.ListVal(
								[]cty.Value{
									cty.ObjectVal(
										map[string]cty.Value{
											"enabled": cty.BoolVal(false),
											"headroom": cty.ListVal(
												[]cty.Value{
													cty.ObjectVal(
														map[string]cty.Value{
															"cpu_percentage":    cty.NumberIntVal(100),
															"memory_percentage": cty.NumberIntVal(10),
															"enabled":           cty.BoolVal(true),
														},
													),
												},
											),
											"headroom_spot": cty.ListVal(
												[]cty.Value{
													cty.ObjectVal(
														map[string]cty.Value{
															"cpu_percentage":    cty.NumberIntVal(10),
															"memory_percentage": cty.NumberIntVal(10),
															"enabled":           cty.BoolVal(true),
														},
													),
												},
											),
										},
									),
								},
							),
						},
					),
				},
			),
			expectedPolicy: `{
				"enabled": false,
				"isScopedMode": false,
				"nodeTemplatesPartialMatchingEnabled": false,
				"unschedulablePods": {
					"customInstancesEnabled": false,
					"enabled": false,
					"headroom": {
						"cpuPercentage": 100,
						"memoryPercentage": 10,
						"enabled": true
					},
					"headroomSpot": {
						"cpuPercentage": 10,
						"memoryPercentage": 10,
						"enabled": true
					}
				},
				"nodeDownscaler": {
					"emptyNodes": {
						"enabled": false,
						"delaySeconds": 0
					}
				}
			}`,
		},
	}

	for _, test := range tt {
		r := require.New(t)

		t.Run(test.name, func(t *testing.T) {
			current, err := normalizeJSON([]byte(test.current))
			r.NoError(err)

			mockClient.EXPECT().
				PoliciesAPIGetClusterPolicies(gomock.Any(), gomock.Any(), gomock.Any()).
				Times(2).
				DoAndReturn(
					func(_ context.Context, _ string) (*http.Response, error) {
						return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(current))}, nil
					},
				)

			policyJSON, err := normalizeJSON([]byte(test.policyJson))
			r.NoError(err)

			clusterID := "cluster_id"

			valueWithPolicyJson := cty.ObjectVal(map[string]cty.Value{
				FieldClusterId:              cty.StringVal(clusterID),
				FieldAutoscalerPoliciesJSON: cty.StringVal(string(policyJSON)),
			})

			stateWithPolicyJson := terraform.NewInstanceStateShimmedFromValue(valueWithPolicyJson, 0)

			valueWithPolicyDefinition := cty.ObjectVal(map[string]cty.Value{
				FieldClusterId:          cty.StringVal(clusterID),
				FieldAutoscalerSettings: test.policyStruct,
			})

			stateWithPolicyDefinition := terraform.NewInstanceStateShimmedFromValue(valueWithPolicyDefinition, 0)

			resource := resourceAutoscaler()

			resultPolicyJson, err := getChangedPolicies(context.Background(), resource.Data(stateWithPolicyJson), provider, clusterID)
			r.NoError(err)

			resultPolicyDefinition, err := getChangedPolicies(context.Background(), resource.Data(stateWithPolicyDefinition), provider, clusterID)
			r.NoError(err)

			expectedPolicy, err := normalizeJSON([]byte(test.expectedPolicy))
			r.NoError(err)

			r.Equal(string(expectedPolicy), string(resultPolicyDefinition))
			r.Equal(string(expectedPolicy), string(resultPolicyJson))
		})
	}
}

// Checks if the value of custom_instances_enabled is retained as received from the API
// in case when it is not specified (null) in the resource configuration
func TestAutoscalerResource_GetChangePolicies_AdjustPolicyForDrift(t *testing.T) {
	r := require.New(t)
	mockCtrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockCtrl)
	clusterID := "cluster_id"
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}
	originalPolicyStr := `{
		"enabled": true,
		"isScopedMode": false,
		"nodeTemplatesPartialMatchingEnabled": false,
		"unschedulablePods": {
			"enabled": true,
			"custom_instances_enabled": true
		}
	}`
	expectedPolicyStr := `{
		"enabled": false,
		"isScopedMode": false,
		"nodeTemplatesPartialMatchingEnabled": false,
		"unschedulablePods": {
			"enabled": false,
			"custom_instances_enabled": true
		}
	}`

	policyJSON, err := normalizeJSON([]byte(originalPolicyStr))
	r.NoError(err)

	expectedPolicyJSON, err := normalizeJSON([]byte(expectedPolicyStr))
	r.NoError(err)

	mockClient.EXPECT().
		PoliciesAPIGetClusterPolicies(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(
			func(_ context.Context, _ string) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(policyJSON))}, nil
			},
		)

	value := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterID),
		FieldAutoscalerSettings: cty.ListVal(
			[]cty.Value{
				cty.ObjectVal(
					map[string]cty.Value{
						"enabled": cty.BoolVal(false),
						"unschedulable_pods": cty.ListVal(
							[]cty.Value{
								cty.ObjectVal(
									map[string]cty.Value{
										"enabled": cty.BoolVal(false),
									},
								),
							},
						),
					},
				),
			},
		),
	})
	rawConfig := cty.ObjectVal(
		map[string]cty.Value{
			"autoscaler_settings": cty.ListVal(
				[]cty.Value{
					cty.ObjectVal(
						map[string]cty.Value{
							"enabled": cty.BoolVal(false),
							"unschedulable_pods": cty.ListVal(
								[]cty.Value{
									cty.ObjectVal(
										map[string]cty.Value{
											"enabled":                  cty.BoolVal(false),
											"custom_instances_enabled": cty.NullVal(cty.Bool),
										},
									),
								},
							),
						},
					),
				},
			),
		},
	)

	stateWithPolicyDefinition := terraform.NewInstanceStateShimmedFromValue(value, 0)
	stateWithPolicyDefinition.RawConfig = rawConfig

	resource := resourceAutoscaler()
	data := resource.Data(stateWithPolicyDefinition)
	result, err := getChangedPolicies(context.Background(), data, provider, clusterID)

	r.NoError(err)
	r.Equal(string(expectedPolicyJSON), string(result))
}

func JSONBytesEqual(a, b []byte) (bool, error) {
	var j, j2 interface{}
	if err := json.Unmarshal(a, &j); err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, &j2); err != nil {
		return false, err
	}
	return reflect.DeepEqual(j2, j), nil
}

func TestAutoscalerResource_GetContext(t *testing.T) {
	clusterId := uuid.NewString()

	testCases := map[string]struct {
		ClusterPolicyRaw string
		State            cty.Value
		VerifyResult     func(testing.TB, *schema.ResourceData)
	}{
		"autoscaler_policies to contain all values received from the API": {
			State: cty.ObjectVal(map[string]cty.Value{
				FieldClusterId: cty.StringVal(clusterId),
			}),
			ClusterPolicyRaw: `{
				"some-unrecognised-value": true
			}`,
			VerifyResult: func(t testing.TB, data *schema.ResourceData) {
				value, exists := data.GetOk(FieldAutoscalerPolicies)
				require.True(t, exists)
				require.Contains(t, value, "some-unrecognised-value")
			},
		},
		"cluster_id falls back to id value": {
			State: cty.ObjectVal(map[string]cty.Value{
				FieldId: cty.StringVal(clusterId),
			}),
			ClusterPolicyRaw: `{}`,
			VerifyResult: func(t testing.TB, data *schema.ResourceData) {
				value, exists := data.GetOk(FieldClusterId)
				require.True(t, exists)
				require.Equal(t, clusterId, value)
			},
		},
		"autoscaler_settings populated (basic sample)": {
			State: cty.ObjectVal(map[string]cty.Value{
				FieldClusterId: cty.StringVal(clusterId),
			}),
			ClusterPolicyRaw: `{
				"enabled": true,
				"unschedulablePods": {
					"enabled": true
				},
				"clusterLimits": {
					"cpu": {
						"minCores": 4
					}
				}
			}`,
			VerifyResult: func(t testing.TB, data *schema.ResourceData) {
				var value any
				value = data.Get(FieldAutoscalerSettings + ".0." + FieldEnabled)
				require.Equal(t, true, value)
				value = data.Get(FieldAutoscalerSettings + ".0." + FieldUnschedulablePods + ".0." + FieldEnabled)
				require.Equal(t, true, value)
				value = data.Get(FieldAutoscalerSettings + ".0." + FieldClusterLimits + ".0." + FieldCPU + ".0." + FieldMinCores)
				require.Equal(t, 4, value)
			},
		},
	}

	for name, tt := range testCases {
		t.Run(name, func(t *testing.T) {
			mockctrl := gomock.NewController(t)
			ctx := t.Context()
			resource := resourceAutoscaler()

			mockClient := mock_sdk.NewMockClientInterface(mockctrl)
			givenGetClusterPoliciesResponse(mockClient, clusterId, tt.ClusterPolicyRaw)

			provider := &ProviderConfig{
				api: &sdk.ClientWithResponses{
					ClientInterface: mockClient,
				},
			}

			state := terraform.NewInstanceStateShimmedFromValue(tt.State, 0)
			data := resource.Data(state)
			diagnostics := resource.ReadContext(ctx, data, provider)
			require.Empty(t, diagnostics)

			if tt.VerifyResult != nil {
				tt.VerifyResult(t, data)
			}
		})
	}
}

func givenGetClusterPoliciesResponse(m *mock_sdk.MockClientInterface, clusterId, data string) {
	response := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(data)),
	}
	m.EXPECT().
		PoliciesAPIGetClusterPolicies(gomock.Any(), clusterId, gomock.Any()).
		Return(response, nil).
		AnyTimes()
}
