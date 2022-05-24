package policies

import (
	_ "embed" // use go:embed
	"encoding/json"
)

var (
	//go:embed aks/policy.json
	AKSPolicy []byte
	//go:embed gke/policy.json
	GKEPolicy []byte
)

type pols struct {
	Policies []string `json:"Policies"`
}

func GetAKSPolicy() ([]string, error) {
	var p pols
	err := json.Unmarshal(AKSPolicy, &p)
	if err != nil {
		return nil, err
	}

	return p.Policies, nil
}

func GetGKEPolicy() ([]string, error) {
	var p pols
	err := json.Unmarshal(GKEPolicy, &p)
	if err != nil {
		return nil, err
	}

	return p.Policies, nil
}
