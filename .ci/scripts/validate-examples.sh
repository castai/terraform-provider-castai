#!/bin/bash
set -eo pipefail

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

while IFS='' read -r -d $'\0' TFPROJECT; do
  echo "Validating ${PWD_ROOT}/$TFPROJECT"
  cd "${PWD_ROOT}/$TFPROJECT"
  terraform init || terraform init -upgrade
  terraform validate
done < <(find examples/eks examples/gke examples/aks -mindepth 1 -maxdepth 1 -type d -print0)

rm "${TF_CLI_CONFIG_FILE}"
unset TF_CLI_CONFIG_FILE
