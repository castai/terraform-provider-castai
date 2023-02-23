package policies

import (
	"bytes"
	_ "embed" // use go:embed
	"fmt"
	"text/template"
)

var (
	//go:embed iam-policy.json
	IAMPolicy string
	//go:embed user-policy.json
	UserPolicy string
)

type policy struct {
	Policies []string `json:"Policies"`
}

func GetIAMPolicy(accountNumber string) (string, error) {
	tmpl, err := template.New("json").Parse(IAMPolicy)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	type tmplValues struct {
		AccountNumber string
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, tmplValues{
		AccountNumber: accountNumber,
	}); err != nil {
		return "", fmt.Errorf("interpolating template: %w", err)
	}

	return buf.String(), nil
}

func GetUserInlinePolicy(clusterName, arn, vpc string) (string, error) {
	tmpl, err := template.New("json").Parse(UserPolicy)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	type tmplValues struct {
		ClusterName string
		ARN         string
		VPC         string
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, tmplValues{
		ClusterName: clusterName,
		ARN:         arn,
		VPC:         vpc,
	}); err != nil {
		return "", fmt.Errorf("interpolating template: %w", err)
	}

	return buf.String(), nil
}
