package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	GKEWorkspace string `envconfig:"GKE_WORKSPACE"`
	APIURL       string `envconfig:"CASTAI_URL" default:"https://api.cast.ai"`
	Token        string `envconfig:"CASTAI_TOKEN" defualt:""`
}

func Load() (*Config, error) {
	config := &Config{}
	if err := envconfig.Process("E2E", config); err != nil {
		return nil, err
	}

	return config, nil
}
