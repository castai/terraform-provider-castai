## EKS and CAST AI example using custom IAM policies

Following example shows how can you onboard EKS cluster to CAST AI using custom IAM policies that can be tailored for
your needs.

IAM policies in the example allows CAST AI:

- start new instances only in private subnets of cluster VPC
- start new instances only with predefined security groups
- manage existing instances that are part of the cluster
- other necessary permissions scoped to specific cluster
- all permissions are scoped to specific EKS cluster

Most of the permissions are mandatory for CAST AI to work properly.
Optional permissions include:

- `ec2.ImportKeyPair` - needed only if you will provide public SSH key for CAST nodes
- `autoscaling:SuspendProcesses` - needed only if you will use pause/resume functionality
- `autoscaling:ResumeProcesses` - needed only if you will use pause/resume functionality