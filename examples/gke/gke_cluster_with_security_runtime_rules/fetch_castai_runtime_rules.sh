#!/bin/bash

set -euo pipefail

# Configuration
BASE_URL=${CASTAI_BASE_URL:-"https://api.cast.ai"}
API_KEY=${CASTAI_API_KEY:-""}
OUTPUT_FILE="castai_runtime_rules_new.yaml"
ENABLED_ONLY=false

# Parse optional arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    --enabled-only)
      ENABLED_ONLY=true
      shift
      ;;
    *)
      echo "❌ Unknown option: $1"
      echo "Usage: $0 [--enabled-only]"
      exit 1
      ;;
  esac
done

# Check for API key
if [ -z "$API_KEY" ]; then
  echo "❌ Error: CASTAI_API_KEY environment variable not set."
  exit 1
fi

# Full API endpoint
ENDPOINT="$BASE_URL/v1/security/runtime/rules?search=&page.limit=5000&sort.field=severity&sort.order=desc"

echo "🔍 Fetching CAST AI Runtime Rules from: $ENDPOINT"

# Fetch data
response=$(curl -s "$ENDPOINT" \
  -H "Accept: application/json" \
  -H "X-API-Key: $API_KEY"\
  --fail -v)

if [ -z "$response" ]; then
  echo "❌ Failed to fetch data from CAST AI."
  exit 1
fi

# Require yq and jq
if ! command -v yq &> /dev/null; then
  echo "❌ 'yq' is required but not installed."
  exit 1
fi
if ! command -v jq &> /dev/null; then
  echo "❌ 'jq' is required but not installed."
  exit 1
fi

# Filter and reorder JSON fields
if [[ $ENABLED_ONLY == true ]]; then
  jq_filter='
    del(.nextCursor, .previousCursor, .count) |
    .rules |= map(
      select(.enabled == true) |
      del(.id) |
      {
        name: .name
      } + (del(.name))
    )
  '
else
  jq_filter='
    del(.nextCursor, .previousCursor, .count) |
    .rules |= map(
      del(.id) |
      {
        name: .name
      } + (del(.name))
    )
  '
fi

cleaned_json=$(echo "$response" | jq "$jq_filter")

# Convert to YAML
echo "$cleaned_json" | yq -P '.' > "$OUTPUT_FILE"
RULE_COUNT=$(yq e '.rules | length' "$OUTPUT_FILE")
echo "✅  Fetched and saved $RULE_COUNT rule(s) to $OUTPUT_FILE"

# Terraform import suggestions
echo "📦 Suggested Terraform import commands:"
echo "$cleaned_json" | jq -r '.rules[] | .name' | while read -r name; do
  tf_resource=$(echo "$name" | tr '[:upper:]' '[:lower:]' | tr ' ' '_' | tr -cd '[:alnum:]_')
  echo "terraform import castai_runtime_rule.${tf_resource} \"${name}\""
done