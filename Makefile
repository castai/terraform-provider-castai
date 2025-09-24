SHELL := /bin/bash

export API_TAGS ?= ExternalClusterAPI,PoliciesAPI,NodeConfigurationAPI,NodeTemplatesAPI,AuthTokenAPI,ScheduledRebalancingAPI,InventoryAPI,UsersAPI,OperationsAPI,EvictorAPI,SSOAPI,CommitmentsAPI,WorkloadOptimizationAPI,ServiceAccountsAPI,RbacServiceAPI,RuntimeSecurityAPI,AllocationGroupAPI
export SWAGGER_LOCATION ?= https://api.cast.ai/v1/spec/openapi.json

export CLUSTER_AUTOSCALER_API_TAGS ?= HibernationSchedulesAPI
export CLUSTER_AUTOSCALER_SWAGGER_LOCATION ?= https://api.cast.ai/spec/cluster-autoscaler/openapi.yaml

export ORGANIZATION_MANAGEMENT_API_TAGS ?= EnterpriseAPI
export ORGANIZATION_MANAGEMENT_SWAGGER_LOCATION ?= https://api.cast.ai/spec/organization-management/openapi.yaml

default: build

.PHONY: format-tf
format-tf:
	terraform fmt -recursive -list=false

.PHONY: generate-sdk
generate-sdk:
	echo "==> Generating castai sdk client"
	go generate castai/sdk/generate.go

# The following command also rewrites existing documentation
.PHONY: generate-docs
generate-docs:
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.14.1
	tfplugindocs generate --rendered-provider-name "CAST AI" --ignore-deprecated --provider-name terraform-provider-castai

.PHONY: generate-all
generate-all: generate-sdk generate-docs

.PHONY: build
build: generate-sdk
build: generate-docs
build:
	@echo "==> Building terraform-provider-castai"
	go build

.PHONY: lint
lint:
	@echo "==> Running lint"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

.PHONY: test
test:
	@echo "==> Running tests"
	go test $$(go list ./... | grep -v vendor/ | grep -v e2e)  -timeout=1m -parallel=4

.PHONY: testacc
testacc:
	@echo "==> Running acceptance tests"
	TF_ACC=1 go test ./castai/... '-run=^TestAcc' -v -timeout 50m

.PHONY: init-examples
init-examples: build
init-examples:
	@echo "==> Creating symlinks for example/ projects to terraform-provider-castai binary"
	@.ci/scripts/init-examples.sh

.PHONY: validate-terraform-examples
validate-terraform-examples:
validate-terraform-examples:
	@.ci/scripts/validate-examples.sh
