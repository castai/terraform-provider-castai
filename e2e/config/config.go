package config

import (
	"os"
	"os/user"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

type Config struct {
	GKEClusterName       string `envconfig:"GKE_CLUSTER_NAME" default:""`
	GKENetworkRegion     string `envconfig:"GKE_NETWORK_REGION" default:""`
	GKEProjectID         string `envconfig:"GKE_PROJECT_ID" default:""`
	GKEClusterLocation   string `envconfig:"GKE_CLUSTER_LOCATION" default:"eu-central"`
	GCPCredentialsBase64 string `envconfig:"GCP_CREDENTIALS_BASE64"`
	GKEWorkspace         string `envconfig:"GKE_WORKSPACE"`
	APIURL               string `envconfig:"CASTAI_URL" default:"https://api.cast.ai"`
	Token                string `envconfig:"CASTAI_TOKEN" defualt:""`
}

func getDefaultUserName() string {
	// are we running in CI? gitlab sets CI variable by default
	if os.Getenv("CI") != "" {
		return "ci"
	}

	// use OS level user name.
	u, err := user.Current()
	if err != nil {
		return "unknown-user"
	}

	return strings.Replace(u.Username, " ", "", -1)
}

func Load() (*Config, error) {
	config := &Config{}
	if err := envconfig.Process("E2E", config); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) PrintConfig(log logrus.FieldLogger) {
	log.Info("########################## Test Suite Config ##########################")
	log.Info("Console API URL: ", c.APIURL)
	log.Info("#######################################################################")
}
