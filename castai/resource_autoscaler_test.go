package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	testingterraform "github.com/hashicorp/terraform-plugin-testing/terraform"
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

	// API returns full policies including deprecated/synced fields
	apiPolicies := `{
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

	// Expected output has volatile/synced fields filtered out (nodeConstraints removed)
	expectedPoliciesBytes, err := normalizeJSON([]byte(`{
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
	expectedPolicies := string(expectedPoliciesBytes)

	resource := resourceAutoscaler()

	clusterId := "cluster_id"
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId: cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	body := io.NopCloser(bytes.NewReader([]byte(apiPolicies)))
	response := &http.Response{StatusCode: 200, Body: body}

	mockClient.EXPECT().PoliciesAPIGetClusterPolicies(gomock.Any(), clusterId, gomock.Any()).Return(response, nil).Times(1)
	mockClient.EXPECT().PoliciesAPIUpsertClusterPoliciesWithBody(gomock.Any(), clusterId, "application/json", gomock.Any()).
		Times(0)

	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.Equal(expectedPolicies, data.Get(FieldAutoscalerPolicies))
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
			actual, err := toAutoscalerPolicy(resource.Data(state))

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

func TestAccEKS_ResourceAutoscaler_basic(t *testing.T) {
	rName := fmt.Sprintf("%v-autoscaler-%v", ResourcePrefix, acctest.RandString(8))
	clusterName, _ := lo.Coalesce(os.Getenv("CLUSTER_NAME"), "cost-terraform")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			// Step 1: Create autoscaler with initial settings
			{
				Config: testAccAutoscalerConfig(rName, clusterName, true, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.enabled", "true"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.is_scoped_mode", "false"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.node_templates_partial_matching_enabled", "false"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.unschedulable_pods.0.enabled", "true"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.cluster_limits.0.enabled", "true"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.cluster_limits.0.cpu.0.min_cores", "1"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.cluster_limits.0.cpu.0.max_cores", "100"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.node_downscaler.0.enabled", "true"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.node_downscaler.0.empty_nodes.0.enabled", "true"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.node_downscaler.0.empty_nodes.0.delay_seconds", "120"),
				),
			},
			// Step 2: Update autoscaler settings
			{
				Config: testAccAutoscalerConfig(rName, clusterName, false, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.enabled", "false"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.is_scoped_mode", "true"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.node_templates_partial_matching_enabled", "true"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.unschedulable_pods.0.enabled", "false"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.cluster_limits.0.enabled", "false"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.cluster_limits.0.cpu.0.min_cores", "2"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.cluster_limits.0.cpu.0.max_cores", "200"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.node_downscaler.0.enabled", "false"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.node_downscaler.0.empty_nodes.0.enabled", "false"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.node_downscaler.0.empty_nodes.0.delay_seconds", "300"),
				),
			},
			// Step 3: Import the resource
			{
				ResourceName: "castai_autoscaler.test",
				ImportStateIdFunc: func(s *testingterraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["castai_eks_cluster.test"]
					if !ok {
						return "", fmt.Errorf("castai_eks_cluster.test not found in state")
					}
					return rs.Primary.ID, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 4: Modify default node template and verify autoscaler doesn't drift
			// This tests the policy - node template sync behavior - when the default node template
			// is modified, the autoscaler should not show drift due to defaultNodeTemplateVersion changing.
			{
				Config: testAccAutoscalerWithNodeTemplateConfig(rName, clusterName),
				Check: resource.ComposeTestCheckFunc(
					// Verify autoscaler state is unchanged despite node template modification
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.enabled", "false"),
					resource.TestCheckResourceAttr("castai_autoscaler.test", "autoscaler_settings.0.is_scoped_mode", "true"),
					// Verify node template was created/updated
					resource.TestCheckResourceAttr("castai_node_template.default", "name", "default-by-castai"),
					resource.TestCheckResourceAttr("castai_node_template.default", "constraints.0.spot", "true"),
				),
			},
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "~> 5.0",
			},
		},
	})
}

func testAccAutoscalerConfig(rName, clusterName string, enabled bool, updated bool) string {
	isScopedMode := "false"
	nodeTemplatesPartialMatchingEnabled := "false"
	unschedulablePodsEnabled := "true"
	clusterLimitsEnabled := "true"
	minCores := 1
	maxCores := 100
	nodeDownscalerEnabled := "true"
	emptyNodesEnabled := "true"
	delaySeconds := 120

	if updated {
		isScopedMode = "true"
		nodeTemplatesPartialMatchingEnabled = "true"
		unschedulablePodsEnabled = "false"
		clusterLimitsEnabled = "false"
		minCores = 2
		maxCores = 200
		nodeDownscalerEnabled = "false"
		emptyNodesEnabled = "false"
		delaySeconds = 300
	}

	return ConfigCompose(testAccEKSClusterConfig(rName, clusterName), fmt.Sprintf(`
resource "castai_autoscaler" "test" {
  cluster_id = castai_eks_cluster.test.id

  autoscaler_settings {
    enabled                                = %t
    is_scoped_mode                         = %s
    node_templates_partial_matching_enabled = %s

    unschedulable_pods {
      enabled = %s
    }

    cluster_limits {
      enabled = %s

      cpu {
        min_cores = %d
        max_cores = %d
      }
    }

    node_downscaler {
      enabled = %s

      empty_nodes {
        enabled       = %s
        delay_seconds = %d
      }
    }
  }
}
`, enabled, isScopedMode, nodeTemplatesPartialMatchingEnabled,
		unschedulablePodsEnabled, clusterLimitsEnabled, minCores, maxCores,
		nodeDownscalerEnabled, emptyNodesEnabled, delaySeconds))
}

// testAccAutoscalerWithNodeTemplateConfig returns a config that includes both autoscaler and
// the default node template. This tests the policy↔node template sync behavior.
func testAccAutoscalerWithNodeTemplateConfig(rName, clusterName string) string {
	return ConfigCompose(testAccAutoscalerConfig(rName, clusterName, false, true), `
resource "castai_node_template" "default" {
  cluster_id = castai_eks_cluster.test.id
  name       = "default-by-castai"
  is_default = true
  is_enabled = true

  constraints {
    on_demand = true
    spot      = true
  }

  # Ignore computed fields that are populated by the API
  lifecycle {
    ignore_changes = [
      configuration_id,
      constraints[0].azs,
      constraints[0].cpu_manufacturers,
    ]
  }
}
`)
}

func TestAutoscalerResource_FlattenAutoscalerSettings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name: "basic policy with top-level fields",
			input: `{
				"enabled": true,
				"isScopedMode": false,
				"nodeTemplatesPartialMatchingEnabled": true
			}`,
			expected: map[string]interface{}{
				"enabled":        true,
				"is_scoped_mode": false,
				"node_templates_partial_matching_enabled": true,
			},
		},
		{
			name: "policy with deprecated fields omitted",
			input: `{
				"enabled": true,
				"unschedulablePods": {
					"enabled": true,
					"headroom": {"cpuPercentage": 10, "memoryPercentage": 10, "enabled": true},
					"headroomSpot": {"cpuPercentage": 5, "memoryPercentage": 5, "enabled": false},
					"nodeConstraints": {"minCpuCores": 2, "maxCpuCores": 32, "enabled": true},
					"customInstancesEnabled": true,
					"podPinner": {"enabled": true}
				},
				"spotInstances": {"enabled": true, "maxReclaimRate": 10}
			}`,
			expected: map[string]interface{}{
				"enabled": true,
				"unschedulable_pods": []interface{}{
					map[string]interface{}{
						"enabled": true,
						"pod_pinner": []interface{}{
							map[string]interface{}{
								"enabled": true,
							},
						},
					},
				},
			},
		},
		{
			name: "policy with cluster limits",
			input: `{
				"enabled": true,
				"clusterLimits": {
					"enabled": true,
					"cpu": {
						"minCores": 1,
						"maxCores": 100
					}
				}
			}`,
			expected: map[string]interface{}{
				"enabled": true,
				"cluster_limits": []interface{}{
					map[string]interface{}{
						"enabled": true,
						"cpu": []interface{}{
							map[string]interface{}{
								"min_cores": float64(1),
								"max_cores": float64(100),
							},
						},
					},
				},
			},
		},
		{
			name: "policy with node downscaler and evictor",
			input: `{
				"enabled": true,
				"nodeDownscaler": {
					"enabled": true,
					"emptyNodes": {
						"enabled": true,
						"delaySeconds": 300
					},
					"evictor": {
						"enabled": true,
						"dryRun": false,
						"aggressiveMode": true,
						"scopedMode": false,
						"cycleInterval": "5m",
						"nodeGracePeriodMinutes": 10,
						"podEvictionFailureBackOffInterval": "10s",
						"ignorePodDisruptionBudgets": false
					}
				}
			}`,
			expected: map[string]interface{}{
				"enabled": true,
				"node_downscaler": []interface{}{
					map[string]interface{}{
						"enabled": true,
						"empty_nodes": []interface{}{
							map[string]interface{}{
								"enabled":       true,
								"delay_seconds": float64(300),
							},
						},
						"evictor": []interface{}{
							map[string]interface{}{
								"enabled":                                true,
								"dry_run":                                false,
								"aggressive_mode":                        true,
								"scoped_mode":                            false,
								"cycle_interval":                         "5m",
								"node_grace_period_minutes":              float64(10),
								"pod_eviction_failure_back_off_interval": "10s",
								"ignore_pod_disruption_budgets":          false,
							},
						},
					},
				},
			},
		},
		{
			name: "disabled evictor and pod_pinner are not included",
			input: `{
				"enabled": true,
				"unschedulablePods": {
					"enabled": true,
					"podPinner": {"enabled": false}
				},
				"nodeDownscaler": {
					"enabled": true,
					"emptyNodes": {"enabled": true, "delaySeconds": 120},
					"evictor": {"enabled": false, "dryRun": false}
				}
			}`,
			expected: map[string]interface{}{
				"enabled": true,
				"unschedulable_pods": []interface{}{
					map[string]interface{}{
						"enabled": true,
					},
				},
				"node_downscaler": []interface{}{
					map[string]interface{}{
						"enabled": true,
						"empty_nodes": []interface{}{
							map[string]interface{}{
								"enabled":       true,
								"delay_seconds": float64(120),
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)

			result, err := flattenAutoscalerSettings([]byte(tt.input))
			r.NoError(err)
			r.NotNil(result)
			r.Len(result, 1)

			// Compare each key in expected with result
			for key, expectedValue := range tt.expected {
				actualValue, ok := result[0][key]
				r.True(ok, "expected key %s not found in result", key)
				r.Equal(expectedValue, actualValue, "mismatch for key %s", key)
			}
		})
	}
}

func TestAutoscalerResource_FlattenAutoscalerSettings_InvalidJSON(t *testing.T) {
	r := require.New(t)

	_, err := flattenAutoscalerSettings([]byte("invalid json"))
	r.Error(err)
	r.Contains(err.Error(), "unmarshaling policies JSON")
}

func TestAutoscalerResource_FlattenEvictor(t *testing.T) {
	r := require.New(t)

	input := map[string]interface{}{
		"enabled":                           true,
		"dryRun":                            false,
		"aggressiveMode":                    true,
		"scopedMode":                        false,
		"cycleInterval":                     "1m",
		"nodeGracePeriodMinutes":            5,
		"podEvictionFailureBackOffInterval": "5s",
		"ignorePodDisruptionBudgets":        false,
	}

	result := flattenEvictor(input)

	r.Equal(true, result["enabled"])
	r.Equal(false, result["dry_run"])
	r.Equal(true, result["aggressive_mode"])
	r.Equal(false, result["scoped_mode"])
	r.Equal("1m", result["cycle_interval"])
	r.Equal(5, result["node_grace_period_minutes"])
	r.Equal("5s", result["pod_eviction_failure_back_off_interval"])
	r.Equal(false, result["ignore_pod_disruption_budgets"])
}

func TestAutoscalerResource_FilterVolatileFields(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldExist []string
		shouldGone  []string
	}{
		{
			name:        "removes defaultNodeTemplateVersion",
			input:       `{"enabled":true,"defaultNodeTemplateVersion":"5","isScopedMode":false}`,
			shouldExist: []string{"enabled", "isScopedMode"},
			shouldGone:  []string{"defaultNodeTemplateVersion"},
		},
		{
			name:        "removes spotInstances (synced with node template)",
			input:       `{"enabled":true,"spotInstances":{"enabled":true,"maxReclaimRate":50}}`,
			shouldExist: []string{"enabled"},
			shouldGone:  []string{"spotInstances"},
		},
		{
			name:        "handles missing defaultNodeTemplateVersion",
			input:       `{"enabled":true,"isScopedMode":false}`,
			shouldExist: []string{"enabled", "isScopedMode"},
			shouldGone:  []string{"defaultNodeTemplateVersion"},
		},
		{
			name:        "preserves nested structures except deprecated fields",
			input:       `{"enabled":true,"defaultNodeTemplateVersion":"10","nodeDownscaler":{"enabled":true}}`,
			shouldExist: []string{"enabled", "nodeDownscaler"},
			shouldGone:  []string{"defaultNodeTemplateVersion"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)

			result := filterVolatileFields([]byte(tt.input))

			var filtered map[string]interface{}
			err := json.Unmarshal(result, &filtered)
			r.NoError(err)

			for _, key := range tt.shouldExist {
				_, exists := filtered[key]
				r.True(exists, "expected key %q to exist", key)
			}

			for _, key := range tt.shouldGone {
				_, exists := filtered[key]
				r.False(exists, "expected key %q to be removed", key)
			}
		})
	}
}

func TestAutoscalerResource_FilterVolatileFields_NestedDeprecated(t *testing.T) {
	r := require.New(t)

	// Test that nested deprecated fields in unschedulablePods are removed
	input := `{
		"enabled": true,
		"unschedulablePods": {
			"enabled": true,
			"nodeConstraints": {"minCpuCores": 2, "maxCpuCores": 96},
			"customInstancesEnabled": true,
			"headroom": {"cpuPercentage": 10}
		}
	}`

	result := filterVolatileFields([]byte(input))

	var filtered map[string]interface{}
	err := json.Unmarshal(result, &filtered)
	r.NoError(err)

	// unschedulablePods should still exist
	up, ok := filtered["unschedulablePods"].(map[string]interface{})
	r.True(ok, "unschedulablePods should exist")

	// enabled should be preserved
	_, exists := up["enabled"]
	r.True(exists, "unschedulablePods.enabled should be preserved")

	// headroom should be preserved (not a synced field)
	_, exists = up["headroom"]
	r.True(exists, "unschedulablePods.headroom should be preserved")

	// nodeConstraints should be removed (synced with node template)
	_, exists = up["nodeConstraints"]
	r.False(exists, "unschedulablePods.nodeConstraints should be removed")

	// customInstancesEnabled should be removed (synced with node template)
	_, exists = up["customInstancesEnabled"]
	r.False(exists, "unschedulablePods.customInstancesEnabled should be removed")
}
