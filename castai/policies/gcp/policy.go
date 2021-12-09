package gcp

import (
	_ "embed" // use go:embed
)

var (
	//go:embed iam-policy.json
	Policy string
)

func GetIAMPolicy() (string, error) {
	return Policy, nil
}
