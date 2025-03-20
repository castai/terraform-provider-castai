package gke

import (
	_ "embed" // use go:embed
	"encoding/json"
)

var (
	//go:embed iam-policy.json
	Policy []byte

	//go:embed loadBalancers-targetBackendPools.json
	LoadBalancersTargetBackendPools []byte

	//go:embed loadBalancers-unmanagedInstanceGroups.json
	LoadBalancersUnmanagedInstanceGroups []byte
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

func GetLoadBalancersTargetBackendPoolsPolicy() ([]string, error) {
	var p pols
	err := json.Unmarshal(LoadBalancersTargetBackendPools, &p)
	if err != nil {
		return nil, err
	}

	return p.Policies, nil
}

func GetLoadBalancersUnmanagedInstanceGroupsPolicy() ([]string, error) {
	var p pols
	err := json.Unmarshal(LoadBalancersUnmanagedInstanceGroups, &p)
	if err != nil {
		return nil, err
	}

	return p.Policies, nil
}
