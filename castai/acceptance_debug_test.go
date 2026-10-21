package castai

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// watchClusterDuringTest polls the CAST AI cluster record while the
// acceptance test runs and logs every state change with a timestamp, so
// lifecycle transitions (e.g. ready -> deleting) can be correlated with
// the test step markers and provider API calls visible in the same log.
// Enabled by setting CASTAI_CLUSTER_WATCH.
func watchClusterDuringTest(t *testing.T, clusterName string) {
	if os.Getenv("CASTAI_CLUSTER_WATCH") == "" {
		return
	}

	client := testAccProvider.Meta().(*ProviderConfig).api
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	log.Printf("[CLUSTER-WATCH] watching cluster %q", clusterName)

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		var clusterID string
		var lastState string
		tick := 0

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			tick++

			var cluster *sdk.ExternalclusterV1Cluster
			if clusterID != "" {
				resp, err := client.ExternalClusterAPIGetClusterWithResponse(ctx, clusterID)
				if err != nil {
					log.Printf("[CLUSTER-WATCH] get cluster: %v", err)
					continue
				}
				if resp.StatusCode() == http.StatusNotFound {
					log.Printf("[CLUSTER-WATCH] cluster %s gone (404), falling back to list by name", clusterID)
					clusterID = ""
					lastState = ""
					continue
				}
				cluster = resp.JSON200
			} else {
				resp, err := client.ExternalClusterAPIListClustersWithResponse(ctx)
				if err != nil {
					log.Printf("[CLUSTER-WATCH] list clusters: %v", err)
					continue
				}
				if resp.JSON200 == nil || resp.JSON200.Items == nil {
					log.Printf("[CLUSTER-WATCH] list clusters: status=%d body=%s", resp.StatusCode(), resp.Body)
					continue
				}
				for i := range *resp.JSON200.Items {
					if c := (*resp.JSON200.Items)[i]; c.Name != nil && *c.Name == clusterName {
						cluster = &(*resp.JSON200.Items)[i]
						break
					}
				}
			}

			if cluster == nil {
				if tick%10 == 0 {
					log.Printf("[CLUSTER-WATCH] heartbeat: no cluster named %q", clusterName)
				}
				continue
			}

			if cluster.Id != nil && *cluster.Id != clusterID {
				clusterID = *cluster.Id
				log.Printf("[CLUSTER-WATCH] tracking cluster id %s", clusterID)
			}

			state, err := json.Marshal(cluster)
			if err != nil {
				continue
			}
			if string(state) != lastState {
				lastState = string(state)
				log.Printf("[CLUSTER-WATCH] cluster changed: %s", state)
			} else if tick%10 == 0 {
				log.Printf("[CLUSTER-WATCH] heartbeat: status=%s agent_status=%s credentials_id=%s",
					toString(cluster.Status), toString(cluster.AgentStatus), toString(cluster.CredentialsId))
			}
		}
	}()
}

// withStepLogging adds a timestamped log line before every test step so the
// step boundaries are visible next to the CLUSTER-WATCH and provider logs.
func withStepLogging(steps []resource.TestStep) []resource.TestStep {
	for i := range steps {
		stepNum := i + 1
		steps[i].PreConfig = func() {
			log.Printf("[TEST-STEP] starting step %d/%d", stepNum, len(steps))
		}
	}
	return steps
}
