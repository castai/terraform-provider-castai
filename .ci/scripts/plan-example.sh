#!/bin/bash
set -eo pipefail

EXAMPLE_DIR="${1:-}"

if [ -z "$EXAMPLE_DIR" ]; then
    echo "Error: Missing argument."
    echo "Usage: $0 <example-folder>"
    exit 1
fi

PWD_ROOT="${PWD}"
cat > "${PWD_ROOT}/localbuild.tfrc" << EOF
provider_installation {
  dev_overrides {
    "registry.terraform.io/castai/castai" = "${PWD_ROOT}"
  }
  direct {}
}
EOF
echo "${PWD_ROOT}/localbuild.tfrc"
export TF_CLI_CONFIG_FILE="${PWD_ROOT}/localbuild.tfrc"

cd "${PWD_ROOT}/${EXAMPLE_DIR}"
terraform init || terraform init -upgrade
terraform plan 1> /dev/null