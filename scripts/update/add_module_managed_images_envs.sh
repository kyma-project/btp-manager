#!/usr/bin/env bash

# This script updates environment variables in the BTP Manager deployment patch manifest
# (set_external_images.yaml) based on image references found in external-images.yaml.
# It extracts image names, constructs full image paths, and ensures the deployment manifest
# contains the correct environment variables for these images.
#
# Usage: Run this script from the repository root. Requires yq and bash.
#
# Expected variables passed (passed from CI):
#   EXTERNAL_IMAGES_REPO - Kyma external images repository


set -euo pipefail

EXTERNAL_IMAGES_YAML="./external-images.yaml"
BTP_MANAGER_DEPLOYMENT_PATCH_YAML="./config/manager/set_external_images.yaml"

update_env_var() {
  export ENV_NAME=${1}
  export ENV_VALUE=${2}
  yq -i '
    (.spec.template.spec.containers[] | select(.name == "manager").env[] | select(.name == env(ENV_NAME))).value = env(ENV_VALUE)
  ' "$BTP_MANAGER_DEPLOYMENT_PATCH_YAML"
}

add_env_var() {
  export ENV_NAME=${1}
  export ENV_VALUE=${2}
  yq -i '
    (.spec.template.spec.containers[] | select(.name == "manager").env) += [
	    {"name": env(ENV_NAME), "value": strenv(ENV_VALUE)}
  ]
  ' "$BTP_MANAGER_DEPLOYMENT_PATCH_YAML"
}

env_var_exists() {
  export ENV_NAME=${1}
  yq -e '
    .spec.template.spec.containers[] | select(.name == "manager").env[]? | select(.name == env(ENV_NAME))
  ' "$BTP_MANAGER_DEPLOYMENT_PATCH_YAML" > /dev/null 2>&1
}

handle_env_var() {
  if env_var_exists "${1}"; then
    update_env_var "${1}" "${2}"
  else
    add_env_var "${1}" "${2}"
  fi
}

echo "Updating environment variables in the patch manifest for the deployment based on images from $EXTERNAL_IMAGES_YAML"

# Extract images from external-images.yaml
SAP_BTP_SERVICE_OPERATOR_EXTERNAL_IMAGE=$(yq '.images[] | select(.source | contains("SAP_BTP_SERVICE_OPERATOR")) | .source' "$EXTERNAL_IMAGES_YAML")
KUBE_RBAC_PROXY_EXTERNAL_IMAGE=$(yq '.images[] | select(.source | contains("KUBE_RBAC_PROXY")) | .source' "$EXTERNAL_IMAGES_YAML")
if [[ -z "$SAP_BTP_SERVICE_OPERATOR_EXTERNAL_IMAGE" || -z "$KUBE_RBAC_PROXY_EXTERNAL_IMAGE" ]]; then
  echo "Error: Could not extract images from $EXTERNAL_IMAGES_YAML"
  exit 1
fi
echo "Extracted images from $EXTERNAL_IMAGES_YAML:"
echo "- $SAP_BTP_SERVICE_OPERATOR_EXTERNAL_IMAGE"
echo "- $KUBE_RBAC_PROXY_EXTERNAL_IMAGE"

# Set images environment variables for the patch manifest
SAP_BTP_SERVICE_OPERATOR_IMAGE="$EXTERNAL_IMAGES_REPO/$SAP_BTP_SERVICE_OPERATOR_EXTERNAL_IMAGE"
KUBE_RBAC_PROXY_IMAGE="$EXTERNAL_IMAGES_REPO/$KUBE_RBAC_PROXY_EXTERNAL_IMAGE"

handle_env_var "SAP_BTP_SERVICE_OPERATOR" "$SAP_BTP_SERVICE_OPERATOR_IMAGE"
handle_env_var "KUBE_RBAC_PROXY" "$KUBE_RBAC_PROXY_IMAGE"

echo "Environment variables added to the patch manifest"
