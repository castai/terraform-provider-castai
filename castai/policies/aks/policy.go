package aks
	import (
	_ "embed" // use go:embed
	"encoding/json"
	)

var (
	//go:embed policy.json
	Policy []byte
)

type pols struct {
	Policies []string `json:"Policies"`
}

func GetUserPolicy() ([]string, error) {
	var p pols
	err := json.Unmarshal(Policy, &p)
	if err != nil {
		return nil, err
	}

	return p.Policies, nil
}
