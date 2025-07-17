#!/usr/bin/env bash

set -euo pipefail

DEPLOYMENT_YAML="./module-resources/apply/deployment.yml"
MANIFEST_YAML="./manifests/btp-operator/btp-manager.yaml"

echo "Updating environment variables in Deployment manifest based on images from $DEPLOYMENT_YAML"

# Extract images from deployment.yml
export SAP_BTP_SERVICE_OPERATOR=$(yq -r '.spec.template.spec.containers[] | select(.name=="manager") | .image' "$DEPLOYMENT_YAML")
export KUBE_RBAC_PROXY=$(yq -r '.spec.template.spec.containers[] | select(.name=="kube-rbac-proxy") | .image' "$DEPLOYMENT_YAML")

echo "Extracted images from $DEPLOYMENT_YAML:"
echo "- SAP_BTP_SERVICE_OPERATOR: $SAP_BTP_SERVICE_OPERATOR"
echo "- KUBE_RBAC_PROXY: $KUBE_RBAC_PROXY"

# Add environment variables to the manager container in the deployment manifest only
# Use yq to select only the Deployment kind and update the env section
yq -i '
  select(.kind == "Deployment") |
  (.spec.template.spec.containers[] | select(.name == "manager").env) += [
	{"name": "sap-btp-service-operator", "value": strenv(SAP_BTP_SERVICE_OPERATOR)},
	{"name": "kube-rbac-proxy", "value": strenv(KUBE_RBAC_PROXY)}
  ]
' "$MANIFEST_YAML"

echo "Environment variables added to manager container in Deployment manifest"
