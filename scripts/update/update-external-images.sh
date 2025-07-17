#!/usr/bin/env bash

# This script extracts container images from module-chart/chart/values.yaml by
# concatenating manager.image.repository and manager.image.tag for the first image,
# and manager.rbacProxy.image.repository and manager.rbacProxy.image.tag for the second image.
# The images are written to external-images.yaml in the format:
# images:
#   - source: "<IMAGE>"

set -euo pipefail

OUTPUT_FILE="external-images.yaml"
VALUES_YAML="./module-chart/chart/values.yaml"

echo "images:" > "$OUTPUT_FILE"

if [ -f "$VALUES_YAML" ]; then
  MANAGER_IMAGE=$(yq -r '.manager.image.repository + ":" + .manager.image.tag' "$VALUES_YAML")
  RBAC_PROXY_IMAGE=$(yq -r '.manager.rbacProxy.image.repository + ":" + .manager.rbacProxy.image.tag' "$VALUES_YAML")
  echo "Images found in $VALUES_YAML:"
  echo "- $MANAGER_IMAGE"
  echo "- $RBAC_PROXY_IMAGE"
  echo "  - source: \"$MANAGER_IMAGE\"" >> "$OUTPUT_FILE"
  echo "  - source: \"$RBAC_PROXY_IMAGE\"" >> "$OUTPUT_FILE"
fi

echo "Images have been written to $OUTPUT_FILE"
