#!/usr/bin/env bash

# This script extracts all container images from module-chart/chart/values.yaml by
# searching for objects with both 'repository' and 'tag' keys and concatenating their values.
# It excludes images listed in the EXCLUDE_IMAGES array.
# The images are written to external-images.yaml in the format:
# images:
#   - source: "<IMAGE>"
# The script also prints the found images to the console.

set -euo pipefail

OUTPUT_FILE="external-images.yaml"
VALUES_YAML="./module-chart/chart/values.yaml"

# List of images to exclude from the output
EXCLUDE_IMAGES=("bitnami/kubectl" "bitnamisecure/kubectl")

echo "images:" > "$OUTPUT_FILE"

if [ -f "$VALUES_YAML" ]; then
  IMAGES=$(yq -r '.. | select(has("repository") and has("tag")) | .repository + ":" + .tag' "$VALUES_YAML" | sort -u)
  for exclude in "${EXCLUDE_IMAGES[@]}"; do
    IMAGES=$(echo "$IMAGES" | grep -v "$exclude")
  done
  echo "Images found in $VALUES_YAML (excluding: ${EXCLUDE_IMAGES[*]}):"
  echo "$IMAGES" | awk '{print "- "$0}'
  echo "$IMAGES" | awk '{print "  - source: \""$0"\""}' >> "$OUTPUT_FILE"
fi

echo "Images have been written to $OUTPUT_FILE"
