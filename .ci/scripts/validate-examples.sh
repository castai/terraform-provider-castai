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
  # Clean up .terraform directory to free disk space
  rm -rf .terraform
done < <(find examples/eks examples/gke examples/aks -mindepth 1 -maxdepth 1 -type d -print0)

rm "${TF_CLI_CONFIG_FILE}"
unset TF_CLI_CONFIG_FILE

# Clean up Terraform plugin cache to free disk space
# Terraform stores plugins in ~/.terraform.d/plugins or TF_PLUGIN_CACHE_DIR
if [ -n "${TF_PLUGIN_CACHE_DIR}" ]; then
  echo "Cleaning up Terraform plugin cache at ${TF_PLUGIN_CACHE_DIR}"
  rm -rf "${TF_PLUGIN_CACHE_DIR}"/*
elif [ -d "${HOME}/.terraform.d/plugins" ]; then
  echo "Cleaning up Terraform plugin cache at ${HOME}/.terraform.d/plugins"
  rm -rf "${HOME}/.terraform.d/plugins"/*
fi

# Also clean up any temporary Terraform files in /tmp
echo "Cleaning up temporary Terraform files in /tmp"
find /tmp -name "terraform-provider*" -type f -delete 2>/dev/null || true
