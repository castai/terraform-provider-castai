#!/bin/bash

RESOURCE_GROUP="<your-resource-group>"
VMSS_LIST=()
INSTANCE_LIST=()

for vmss in $(az vmss list --resource-group "$RESOURCE_GROUP" --query "[].name" -o tsv); do
  for instance in $(az vmss list-instances --resource-group "$RESOURCE_GROUP" --name "$vmss" --query "[].instanceId" -o tsv); do
    is_latest=$(az vmss get-instance-view --resource-group "$RESOURCE_GROUP" --name "$vmss" --instance-id "$instance" --query "latestModelApplied" -o tsv)
    if [ "$is_latest" != "true" ]; then
      VMSS_LIST+=("$vmss")
      INSTANCE_LIST+=("$instance")
    fi
  done
done

if [ ${#VMSS_LIST[@]} -eq 0 ]; then
  echo "All VM instances are using the latest model."
  exit 0
fi

echo "VM instances that need upgrade:"
for i in "${!VMSS_LIST[@]}"; do
  echo "  ${VMSS_LIST[$i]}: ${INSTANCE_LIST[$i]}"
done

read -p "Approve upgrade for all listed instances? (y/n): " approve
if [ "$approve" == "y" ]; then
  for i in "${!VMSS_LIST[@]}"; do
    az vmss update-instances --resource-group "$RESOURCE_GROUP" --name "${VMSS_LIST[$i]}" --instance-ids "${INSTANCE_LIST[$i]}"
    echo "Upgraded instance ${INSTANCE_LIST[$i]} in ${VMSS_LIST[$i]}"
  done
else
  echo "Upgrade cancelled."
fi

