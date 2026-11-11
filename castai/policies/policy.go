package policies

import (
	"bytes"
	_ "embed" // use go:embed
	"encoding/json"
	"fmt"
	"regexp"
	"text/template"
)

// iamRoleARNPattern matches an IAM role ARN: arn:<partition>:iam::<12 digit account>:role/<optional path/>name.
var iamRoleARNPattern = regexp.MustCompile(`^arn:(aws|aws-cn|aws-us-gov):iam::[0-9]{12}:role/([A-Za-z0-9+=,.@_-]+/)*[A-Za-z0-9+=,.@_-]+$`)

var (
	//go:embed iam-policy.json
	IAMPolicy string
	//go:embed user-policy.json
	UserPolicy string
)

// ValidateRoleARNs checks that every entry is an IAM role ARN belonging to the given partition.
func ValidateRoleARNs(partition string, roleARNs []string) error {
	for _, roleARN := range roleARNs {
		if !iamRoleARNPattern.MatchString(roleARN) {
			return fmt.Errorf("%q is not a valid IAM role ARN, expected format arn:<partition>:iam::<account-id>:role/<path/name>", roleARN)
		}
		if prefix := "arn:" + partition + ":"; roleARN[:len(prefix)] != prefix {
			return fmt.Errorf("IAM role ARN %q does not belong to partition %q", roleARN, partition)
		}
	}
	return nil
}

// passRoleResources renders the Resource value of the PassRole statements as JSON.
// Without roleARNs it allows any role in the partition.
func passRoleResources(partition string, roleARNs []string) (string, error) {
	if len(roleARNs) == 0 {
		return `"arn:` + partition + `:iam::*:role/*"`, nil
	}
	if err := ValidateRoleARNs(partition, roleARNs); err != nil {
		return "", err
	}

	var resources any = roleARNs
	if len(roleARNs) == 1 {
		resources = roleARNs[0]
	}
	out, err := json.Marshal(resources)
	if err != nil {
		return "", fmt.Errorf("marshaling resources: %w", err)
	}
	return string(out), nil
}

// GetIAMPolicy renders the IAM policy. If passRoleARNs is empty, iam:PassRole is allowed for any role in the partition,
// otherwise only for the given role ARNs.
func GetIAMPolicy(accountNumber, partition string, passRoleARNs []string) (string, error) {
	passRoleResources, err := passRoleResources(partition, passRoleARNs)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("json").Parse(IAMPolicy)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	type tmplValues struct {
		AccountNumber     string
		Partition         string
		PassRoleResources string
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, tmplValues{
		AccountNumber:     accountNumber,
		Partition:         partition,
		PassRoleResources: passRoleResources,
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
	var vpcArn string
	// If sharedVPCArn is not provided, use the main ARN for VPC/subnet resources
	if sharedVPCArn == "" {
		vpcArn = arn
	} else {
		vpcArn = sharedVPCArn
	}

	type tmplValues struct {
		ClusterName string
		ARN         string
		VPC         string
		Partition   string
		VPCArn      string
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, tmplValues{
		ClusterName: clusterName,
		ARN:         arn,
		VPC:         vpc,
		Partition:   partition,
		VPCArn:      vpcArn,
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
