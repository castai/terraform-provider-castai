package gcp

import (
	_ "embed" // use go:embed
	"encoding/json"
)

var (
	//go:embed iam-policy.json
	Policy []byte
)

type pols struct {
	Policies []string `json:"Policies"`
}

func GetIAMPolicy() ([]string, error) {
	var p pols
	err := json.Unmarshal(Policy, &p)
	if err != nil {
		return nil, err
	}

	return p.Policies, nil
}