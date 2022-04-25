# EKS cluster with ALB Loadbalancer

This example showcase how you can create EKS cluster with NAT Gateway and Application Loadbalancer from AWS

## Example configuration

```hcl
# local.auto.tfvars
castai_api_token = "<API>"
aws_account_id = "ID"
aws_access_key_id = "KEY"
aws_secret_access_key = "SECRET" 
cluster_region = "eu-central-1"
cluster_name = "my-cluster-25-04-1"
delete_nodes_on_disconnect = true
```

## Using

```shell
terraform init
terraform apply -target module.vpc # create vpc first
terraform apply # apply the rest
```

After cluster creating you can add ingress/LB for application 

```shell
kubectl apply -f k8s/
```

Checking LB address
```shell
$ kubectl  get ingress -A
NAMESPACE    NAME         CLASS   HOSTS                                            ADDRESS                                                                 PORTS   AGE
echoserver   echoserver   alb     *.example.com,*.eu-central-1.elb.amazonaws.com   k8s-albdemogroup-ID-iD.eu-central-1.elb.amazonaws.com   80      59m
```
