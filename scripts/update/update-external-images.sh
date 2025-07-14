#!/usr/bin/env bash

set -euo pipefail

OUTPUT_FILE="external-images.yaml"

echo "images:" > "$OUTPUT_FILE"

# Extract images from module-resources/apply/deployment.yml if the file exists
DEPLOYMENT_YAML="./module-resources/apply/deployment.yml"
if [ -f "$DEPLOYMENT_YAML" ]; then
  IMAGES=$(yq '.spec.template.spec.containers[].image' "$DEPLOYMENT_YAML" | uniq)
  echo "Images found in $DEPLOYMENT_YAML:"
  echo "$IMAGES" | awk '{print "- "$0""}'
  echo "$IMAGES" | awk '{print "  - source: \""$0"\""}' >> "$OUTPUT_FILE"
fi

echo "Images have been written to $OUTPUT_FILE"