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

func GetIAMPolicy(accountNumber, partition string) (string, error) {
	tmpl, err := template.New("json").Parse(IAMPolicy)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	type tmplValues struct {
		AccountNumber string
		Partition     string
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, tmplValues{
		AccountNumber: accountNumber,
		Partition:     partition,
	}); err != nil {
		return "", fmt.Errorf("interpolating template: %w", err)
	}

	return buf.String(), nil
}

func GetUserInlinePolicy(clusterName, arn, vpc, partition, sharedVPCArn string) (string, error) {
	tmpl, err := template.New("json").Parse(UserPolicy)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	// If sharedVPCArn is not provided, use the main ARN for VPC/subnet resources
	if sharedVPCArn == "" {
		sharedVPCArn = arn
	}

	type tmplValues struct {
		ClusterName  string
		ARN          string
		VPC          string
		Partition    string
		SharedVPCArn string
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, tmplValues{
		ClusterName:  clusterName,
		ARN:          arn,
		VPC:          vpc,
		Partition:    partition,
		SharedVPCArn: sharedVPCArn,
	}); err != nil {
		return "", fmt.Errorf("interpolating template: %w", err)
	}

	return buf.String(), nil
}

func GetManagedPolicies(partition string) []string {
	return []string{
		"arn:" + partition + ":iam::aws:policy/AmazonEC2ReadOnlyAccess",
		"arn:" + partition + ":iam::aws:policy/IAMReadOnlyAccess",
	}
}
