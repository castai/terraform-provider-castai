default: build

init-examples:
	@echo "==> Creating symlinks for example/ projects to terraform-provider-castai binary"; \
	TF_PROVIDER_FILENAME=terraform-provider-castai; \
	GOOS=`go tool dist env | awk -F'=' '/^GOOS/ { print $$2}' | tr -d '"'`; \
	GOARCH=`go tool dist env | awk -F'=' '/^GOARCH/ { print $$2}' | tr -d '"'`; \
	for tfproject in examples/* ; do \
		TF_PROJECT_PLUGIN_PATH="$${tfproject}/terraform.d/plugins/registry.terraform.io/castai/castai/0.0.0-local/$${GOOS}_$${GOARCH}"; \
		echo "creating $${TF_PROVIDER_FILENAME} symlink to $${TF_PROJECT_PLUGIN_PATH}/$${TF_PROVIDER_FILENAME}"; \
		mkdir -p "${PWD}/$${TF_PROJECT_PLUGIN_PATH}"; \
		ln -sf "${PWD}/terraform-provider-castai" "$${TF_PROJECT_PLUGIN_PATH}"; \
	done

generate-sdk:
	@echo "==> Generating castai sdk client"
	go generate castai/sdk/generate.go

# The following command also rewrites existing documentation
generate-docs:
	go generate main.go

build: init-examples
build: generate-sdk
build:
	@echo "==> Building terraform-provider-castai"
	go build

test:
	@echo "==> Running tests"
	go test $$(go list ./... | grep -v vendor/ | grep -v e2e)  -timeout=1m -parallel=4

testacc:
	@echo "==> Running acceptance tests"
	TF_ACC=1 go test ./... -v -timeout 120m
