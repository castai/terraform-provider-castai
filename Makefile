SHELL := /bin/bash

default: build

.PHONY: init-examples
init-examples:
	@echo "==> Creating symlinks for example/ projects to terraform-provider-castai binary"; \
	TF_PROVIDER_FILENAME=terraform-provider-castai; \
	GOOS=`go env GOOS`; \
	GOARCH=`go env GOARCH`; \
	git fetch --tags > /dev/null; \
	NEXT_MINOR=`git tag --list 'v*'|sort|tail -n 1| awk -F. -v OFS=. '{$$NF += 1 ; print}'`; \
	echo "using next possible minor version without 'v' prefix: $${NEXT_MINOR:1}"; \
	for examples in examples/eks examples/gke examples/aks ; do \
		for tfproject in $$examples/* ; do \
			TF_PROJECT_PLUGIN_PATH="$${tfproject}/terraform.d/plugins/registry.terraform.io/castai/castai/$${NEXT_MINOR:1}/$${GOOS}_$${GOARCH}"; \
			echo "creating $${TF_PROVIDER_FILENAME} symlink to $${TF_PROJECT_PLUGIN_PATH}/$${TF_PROVIDER_FILENAME}"; \
			mkdir -p "${PWD}/$${TF_PROJECT_PLUGIN_PATH}"; \
			ln -sf "${PWD}/terraform-provider-castai" "$${TF_PROJECT_PLUGIN_PATH}"; \
		done \
	done

.PHONY: format-tf
format-tf:
	terraform fmt -recursive -list=false

.PHONY: generate-sdk
generate-sdk:
	@echo "==> Generating castai sdk client"
	@API_TAGS=ExternalClusterAPI,PoliciesAPI,NodeConfigurationAPI,NodeTemplatesAPI,AuthTokenAPI,ScheduledRebalancingAPI,InventoryAPI,UsersAPI,OperationsAPI,EvictorAPI,SSOAPI,CommitmentsAPI,WorkloadOptimizationAPI go generate castai/sdk/generate.go

# The following command also rewrites existing documentation
.PHONY: generate-docs
generate-docs:
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.14.1
	tfplugindocs generate --rendered-provider-name "CAST AI" --ignore-deprecated

.PHONY: generate-all
generate-all: generate-sdk generate-docs

.PHONY: build
build: init-examples
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
	TF_ACC=1 go test ./castai/... '-run=^TestAcc' -v -timeout 30m

.PHONY: validate-terraform-examples
validate-terraform-examples:
	for examples in examples/eks examples/gke examples/aks ; do \
		for tfproject in $$examples/* ; do \
			echo "==> Validating terraform example $$tfproject"; \
			cd $$tfproject; \
			terraform init; \
			terraform validate; \
			cd -; \
		done \
	done
