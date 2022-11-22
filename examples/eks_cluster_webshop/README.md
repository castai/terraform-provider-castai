# EKS cluster with ALB Loadbalancer

This example showcase how you can create EKS cluster with NAT Gateway and Application Loadbalancer from AWS

## Example configuration

Save these variable to your local terraform.tfvars file, replacing the sample values below. In order to obtain the value of aws_account_id, run the following command and copy the value of "Account" into the variable file below.

```shell
aws sts get-caller-identity
```

```hcl
# terraform.tfvars
castai_api_token = "<API>"
aws_account_id = "ID"
aws_access_key_id = "KEY"
aws_secret_access_key = "SECRET" 
cluster_region = "eu-central-1"
cluster_name = "my-cluster-25-04-1"
delete_nodes_on_disconnect = true
loki_bucket_name = "loki-logs-bucket"
```

## Using

```shell
terraform init
terraform apply -target module.vpc # create vpc first
terraform apply -target module.eks # create EKS cluster
terraform apply # apply the rest
```

## Getting a Kubeconfig File

The EKS cluster will be created once the above Terraform completes. In order to access the newly created
cluster, run the following eksctl command. The command will append your local kubeconfig file and set the
correct context.

```shell
aws eks update-kubeconfig --region <cluster_region_code> --name <cluster_name>
```

## Deployed components 

### Loki + promtail

For logs this example uses Loki from `loki-simple-scalable` helm chart which deploys a few read and write components. 
Keep in mind that in scope of this example wasn't to have s3 buckket configured for Loki

```shell
$ kubectl get pod -n tools | grep loki
loki-gateway-7479f46545-pp7c9                            1/1     Running   0          6h27m
loki-read-0                                              1/1     Running   0          6h27m
loki-read-1                                              1/1     Running   0          6h27m
loki-read-2                                              1/1     Running   0          6h27m
loki-write-0                                             1/1     Running   0          6h27m
loki-write-1                                             1/1     Running   0          6h27m
loki-write-2                                             1/1     Running   0          6h27m
```

Loki for storing logs uses S3 bucket created by terraform (`loki_bucket_name` variable contains bucket name). To access the s3 bucket this example uses [IAM for Service Account](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)


### kube-prometheus-stack

Alertmanager + Prometheus + Grafana

More info about prometheus can be found [here](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/user-guides/getting-started.md)

```shell
$ kubectl get pod -n tools | grep "prom-stack"
alertmanager-prom-stack-kube-prometheus-alertmanager-0   2/2     Running   0          6h16m
prom-stack-grafana-7dfbb4f775-966tg                      3/3     Running   0          6h16m
prom-stack-kube-prometheus-operator-7d6fd799db-th94m     1/1     Running   0          6h16m
prom-stack-kube-state-metrics-776f694dcc-snmjm           1/1     Running   0          6h16m
prom-stack-prometheus-node-exporter-gnx5c                1/1     Running   0          6h16m
prom-stack-prometheus-node-exporter-qcv45                1/1     Running   0          6h16m
prom-stack-prometheus-node-exporter-vj4qx                1/1     Running   0          6h16m
prometheus-prom-stack-kube-prometheus-prometheus-0       2/2     Running   0          6h16m
```

Grafana is pre-installed with useful dashboard for checking PODs and nodes stats. Password
```shell
terraform output --json | jq ".grafana_password.value"
```

Please note that grafana is lacking of persistence (PVC has to be in ReadWriteMany accessMode) - adding database layer is required for production usage

Accessing grafana 

```shell
kubectl port-forward -n tools svc/prom-stack-grafana 8081:80                                                                                                                         
```
In the browser you can use http://localhost:8081

### cert-manager

For managing certificates we are using [cert-manager](https://cert-manager.io/docs/installation/helm/)
You can issue a certificate using [let's encrypt](https://getbetterdevops.io/k8s-ingress-with-letsencrypt/) or [external provider](https://cert-manager.io/v1.0-docs/configuration/external/)
In this example only self-singed ClusterIssuer is created

### EKS CSI driver

EKS CSI driver is installed as addon for cluster. StorageClass that uses ebs-csi driver is called`ebs-sc`

```shell
$ kubectl get storageclasses.storage.k8s.io
NAME            PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
ebs-sc          ebs.csi.aws.com         Retain          WaitForFirstConsumer   true                   7h22m
gp2 (default)   kubernetes.io/aws-ebs   Delete          WaitForFirstConsumer   false                  7h26m
```

### nginx-ingress and ALB

#### NIGNX 

For ingress you can use nginx-ingress controller for which in terraform we are creating a static IP in terraform

IP for Loadbalancer for NGINX:

```shell
terraform output --json | jq ".ingress_ips.value"
```

#### Application Load Balancer

In case of ALB the controller is creating ALB in AWS for each group (annotation `alb.ingress.kubernetes.io/group.name`)

Note: For exposing service with type `ClusterIP` you need to add annotations: `alb.ingress.kubernetes.io/target-type: ip`

More info about available annotations for ALB can be found [here](https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.2/guide/ingress/annotations/)

# Demo application

Demo apps is a echo-server with additional container that uses PVC 

```shell
kubectl apply -f k8s/namespace.yaml # create namespaces
kubectl apply -f k8s # create other resources
```

Accessing service using ingress

```shell
$ curl -X GET http://demo.example.com  -x <IP_OF_LB>:80

CLIENT VALUES:
client_address=10.0.3.190
command=GET
real path=/
query=nil
request_version=1.1
request_uri=http://demo.example.com:8080/

SERVER VALUES:
server_version=nginx: 1.10.0 - lua: 10001

HEADERS RECEIVED:
accept=*/*
host=demo.example.com
proxy-connection=Keep-Alive
user-agent=curl/7.77.0
x-forwarded-for=10.0.3.198
x-forwarded-host=demo.example.com
x-forwarded-port=80
x-forwarded-proto=http
x-forwarded-scheme=http
x-real-ip=10.0.3.198
x-request-id=f35e9aa54092d3ebd3cdf922e1f217b8
x-scheme=http
BODY:
-no body in request-% 
```

# Allowing access to cluster for other users (assume role)

By default, full access to AWS EKS cluster is granted to a IAM user that created given cluster. If you want to add a new IAM user/role you need to edit [AWS auth-config map](https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html). 
This example contains simple path for granting a role an admin access to cluster, you can add variable `eks_user_role_arn` which should have ARN for role that you want to grant access to cluster (this role has to have EKS cluster describe).

## Example 

In this example we will add access to EKS cluster for role `arn:aws:iam::123456789012:role/assume-test`. We need to add this role to `aws-auth`. Setting variable `eks_user_role_arn` will be enough in our case. 
Then apply terraform changes.
Example local.auto.tfvars
```
castai_api_token = "<API>"
aws_account_id = "ID"
aws_access_key_id = "KEY"
aws_secret_access_key = "SECRET" 
cluster_region = "eu-central-1"
cluster_name = "my-cluster-25-04-1"
delete_nodes_on_disconnect = true
eks_user_role_arn = "arn:aws:iam::123456789012:role/assume-test"
```


To validate if role was added please run [eksctl](https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html)
```shell
 eksctl get iamidentitymapping --cluster <cluster_name> --region=<region>
 
2022-06-02 10:13:01 [ℹ]  eksctl version 0.89.0
2022-06-02 10:13:01 [ℹ]  using region <region>
ARN											USERNAME			GROUPS					ACCOUNT
arn:aws:iam::123456789012:role/castai-eks-instance-<cluster_name>			system:node:{{EC2PrivateDNSName}}system:bootstrappers,system:nodes
arn:aws:iam::123456789012:role/default_node_group-node-group-aa   system:node:{{EC2PrivateDNSName}}system:bootstrappers,system:nodes
arn:aws:iam::123456789012:role/worker-group-1-node-group-aa system:node:{{EC2PrivateDNSName}}system:bootstrappers,system:nodes
arn:aws:iam::123456789012:role/assume-test					admin				system:masters # <- role for our user
```

### Testing 

Example based on https://aws.amazon.com/premiumsupport/knowledge-center/iam-assume-role-cli/

1. Get env variables for role assume 
```shell
aws sts assume-role --role-arn "arn:aws:iam::123456789012:role/assume-test" --role-session-name AWS-Session
```
2. Export `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN` get values from assume-role command
```shell
   export AWS_ACCESS_KEY_ID=RoleAccessKeyID
   export AWS_SECRET_ACCESS_KEY=RoleSecretKey
   export AWS_SESSION_TOKEN=RoleSessionToken
```
3. Validate if you have correct role for aws cli
```shell
 aws sts get-caller-identity
{
    "UserId": "ABCDEFGHIJKLM:AWS-Session",
    "Account": "123456789012",
    "Arn": "arn:aws:sts::123456789012:assumed-role/assume-test/AWS-Session"
}
(
```
4. Verify configuration by running `kubectl get nodes`.
