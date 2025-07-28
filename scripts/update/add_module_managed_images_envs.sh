#!/usr/bin/env bash

set -euo pipefail

EXTERNAL_IMAGES_YAML="./external-images.yaml"
EXTERNAL_IMAGES_REPO="europe-docker.pkg.dev/kyma-project/prod/external"
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
  if env_var_exists ${1}; then
    update_env_var ${1} ${2}
  else
    add_env_var ${1} ${2}
  fi
}

echo "Updating environment variables in the patch manifest for the deployment based on images from $EXTERNAL_IMAGES_YAML"

# Extract images from external-images.yaml
SAP_BTP_SERVICE_OPERATOR_EXTERNAL_IMAGE=$(yq '.images[] | select(.source | contains("sap-btp-service-operator")) | .source' "$EXTERNAL_IMAGES_YAML")
KUBE_RBAC_PROXY_EXTERNAL_IMAGE=$(yq '.images[] | select(.source | contains("kube-rbac-proxy")) | .source' "$EXTERNAL_IMAGES_YAML")
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

handle_env_var "sap-btp-service-operator" "$SAP_BTP_SERVICE_OPERATOR_IMAGE"
handle_env_var "kube-rbac-proxy" "$KUBE_RBAC_PROXY_IMAGE"

echo "Environment variables added to the patch manifest"
