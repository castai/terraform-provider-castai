#!/bin/bash

set -eo pipefail

PWD_ROOT="${PWD}"

while IFS='' read -r -d $'\0' TFPROJECT; do
  echo "Validating ${PWD_ROOT}/$TFPROJECT"
  cd "${PWD_ROOT}/$TFPROJECT"
  terraform init || terraform init -upgrade
  terraform validate
done < <(find examples/eks examples/gke examples/aks -mindepth 1 -maxdepth 1 -type d -print0)
