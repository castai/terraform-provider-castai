#!/bin/bash

set -eo pipefail

GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

git fetch --tags > /dev/null

NEXT_MINOR=$(git tag --list 'v*'|sort -V|tail -n 1| awk -F. -v OFS=. '{$NF += 1 ; print}')
echo "using next possible minor version without 'v' prefix: ${NEXT_MINOR:-1}"

while IFS='' read -r -d $'\0' TFPROJECT; do
  TF_PROJECT_PLUGIN_PATH="$TFPROJECT/terraform.d/plugins/registry.terraform.io/castai/castai/${NEXT_MINOR:-1}/${GOOS}_${GOARCH}"
  echo "creating symlink under $TF_PROJECT_PLUGIN_PATH"
  mkdir -p "${PWD}/$TF_PROJECT_PLUGIN_PATH"
  ln -sf "${PWD}/terraform-provider-castai" "$TF_PROJECT_PLUGIN_PATH"
done < <(find examples/eks examples/gke examples/aks -type d -depth 1 -print0)
