SHELL := /bin/bash

export API_TAGS ?= ExternalClusterAPI,PoliciesAPI,NodeConfigurationAPI,NodeTemplatesAPI,AuthTokenAPI,ScheduledRebalancingAPI,InventoryAPI,UsersAPI,OperationsAPI,EvictorAPI,SSOAPI,CommitmentsAPI,WorkloadOptimizationAPI,ServiceAccountsAPI,RbacServiceAPI,RuntimeSecurityAPI,AllocationGroupAPI
export SWAGGER_LOCATION ?= https://api.cast.ai/v1/spec/openapi.json

#  To add a new SDK, add a line here in the format: package_name:ApiTagName:spec_location
SDK_SPECS := \
	cluster_autoscaler:HibernationSchedulesAPI:https://api.cast.ai/spec/cluster-autoscaler/openapi.yaml \
	organization_management:EnterpriseAPI:https://api.cast.ai/spec/organization-management/openapi.yaml \
	omni:EdgeLocationsAPI,ClustersAPI:https://api.cast.ai/spec/omni/openapi.yaml

export AI_OPTIMIZER_API_TAGS ?= APIKeysAPI,AnalyticsAPI,SettingsAPI,HostedModelsAPI,ComponentsAPI
export AI_OPTIMIZER_SWAGGER_LOCATION ?= https://api.cast.ai/spec/ai-optimizer/openapi.yaml

default: build

.PHONY: format-tf
format-tf:
	terraform fmt -recursive -list=false

.PHONY: generate-sdk 
generate-sdk: generate-sdk-new
	@echo "==> Generating main sdk client"
	go generate castai/sdk/generate.go

.PHONY: generate-sdk-new
generate-sdk-new:
	@echo "==> Generating api sdk clients"
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1
	@go install github.com/golang/mock/mockgen
	@cd castai/sdk && for spec in $(SDK_SPECS); do \
		IFS=':' read -r pkg tag loc <<< "$$spec"; \
		[ -z "$$pkg" ] && continue; \
		echo "generating sdk for: $$tag from $$loc"; \
		mkdir -p $$pkg/mock && \
		oapi-codegen -o $$pkg/api.gen.go --old-config-style -generate types -include-tags $$tag -package $$pkg $$loc && \
		oapi-codegen -o $$pkg/client.gen.go --old-config-style -templates codegen/templates -generate client -include-tags $$tag -package $$pkg $$loc && \
		mockgen -source $$pkg/client.gen.go -destination $$pkg/mock/client.go . ClientInterface; \
	done

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
test: build
	@echo "==> Running tests"
	go test $$(go list ./... | grep -v vendor/ | grep -v e2e)  -timeout=1m -parallel=4

.PHONY: testacc-eks
testacc-eks: build
	@echo "==> Running EKS acceptance tests"
	TF_ACC=1 go test ./castai/... '-run=^TestAccEKS_' -v -timeout 50m

.PHONY: testacc-gke
testacc-gke: build
	@echo "==> Running GKE acceptance tests"
	TF_ACC=1 go test ./castai/... '-run=^TestAccGKE_' -v -timeout 50m

.PHONY: testacc-aks
testacc-aks: build
	@echo "==> Running AKS acceptance tests"
	TF_ACC=1 go test ./castai/... '-run=^TestAccAKS_' -v -timeout 50m

.PHONY: testacc-cloud-agnostic
testacc-cloud-agnostic: build
	@echo "==> Running cloud agnostic acceptance tests"
	TF_ACC=1 go test ./castai/... '-run=^TestAccCloudAgnostic_' -v -timeout 50m

.PHONY: validate-terraform-examples
validate-terraform-examples:
	@.ci/scripts/validate-examples.sh

.PHONY: plan-terraform-example
plan-terraform-example:
	@if [ -z "$(EXAMPLE_DIR)" ]; then \
		echo "Error: EXAMPLE_DIR is required. Usage: make plan-terraform-example EXAMPLE_DIR=examples/.../..."; \
		exit 1; \
	fi
	@.ci/scripts/plan-example.sh "$(EXAMPLE_DIR)"