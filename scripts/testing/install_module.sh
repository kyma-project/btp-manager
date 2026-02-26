#!/usr/bin/env bash

# This script has the following arguments:
#     - link to a binary image (required),
#     - credentials mode, allowed values (required):
#         dummy - dummy credentials passed
#         real - real credentials passed
#     - EnableLimitCache in configmap, allowed values (optional, default: false):
#         true - cache enabled
#         false - cache disabled
# ./install_module.sh europe-docker.pkg.dev/kyma-project/dev/btp-manager:PR-999 real

# The script requires the following environment variables if is called with "real" parameter - these should be real credentials base64 encoded:
#      SM_CLIENT_ID - client ID
#      SM_CLIENT_SECRET - client secret
#      SM_URL - service manager url
#      SM_TOKEN_URL - token url

set -x
LIMIT_CACHE=${4:false}
set +x

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

IMAGE_NAME=$1
CREDENTIALS=$2
YAML_DIR="scripts/testing/yaml"

# installing prerequisites, on production environment these are present before chart is used
kubectl apply -f ./deployments/prerequisites.yaml

# creating secret
if [[ "${CREDENTIALS}" == "real" ]]
then
  [ -n "${SM_CLIENT_ID}" ] && [ -n "${SM_CLIENT_SECRET}" ] && [ -n "${SM_URL}" ] && [ -n "${SM_TOKEN_URL}" ] || (echo "Missing credentials - failing test" && exit 1)
  envsubst <${YAML_DIR}/e2e-test-secret.yaml | kubectl apply -f -
else
  # shortening HardDeleteTimeout to make cleanup faster
  kubectl apply -f ${YAML_DIR}/e2e-test-configmap.yaml
  kubectl apply -f ./examples/btp-manager-secret.yaml
fi

echo -e "\n--- Setting EnableLimitCache=${LIMIT_CACHE} in configmap"
if [[ "${LIMIT_CACHE}" == "true" ]]
  kubectl patch configmap sap-btp-manager -n kyma-system --type merge -p '{"data":{"EnableLimitCache":"true"}}'
then
  kubectl patch configmap sap-btp-manager -n kyma-system --type merge -p '{"data":{"EnableLimitCache":"false"}}'
fi
kubectl get configmap sap-btp-manager -n kyma-system -ojson | jq '.data.EnableLimitCache' | xargs -I{} echo "EnableLimitCache is set to {}"

echo -e "\n--- Deploying module with image: ${IMAGE_NAME} - invoking make"
IMG=${IMAGE_NAME} make deploy

# check if deployment is available
while [[ $(kubectl get deployment/btp-manager-controller-manager -n kyma-system -o 'jsonpath={..status.conditions[?(@.type=="Available")].status}') != "True" ]];
do echo -e "\n---Waiting for deployment to be available"; sleep 5; done

echo -e "\n---Deployment available"

echo -e "\n---Installing BTP operator"
kubectl apply -f ${YAML_DIR}/e2e-test-btpoperator.yaml

while [[ $(kubectl get btpoperators/btpoperator -n kyma-system -ojson| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != "TrueReconcileSucceeded" ]];
do echo -e "\n---Waiting for BTP Operator to be ready and reconciled"; sleep 5; done

# verifying whether service instance and service binding custom resources were created
echo -e "\n---Checking if serviceinstances and servicebindings CRDs are created"

CRDS=$(kubectl get crds|awk '/(servicebindings|serviceinstances)/{print $1}')
if [[ $(wc -w <<< ${CRDS}) -ne 2 ]]
then
  echo "Missing CR definitions - failing tests"
  exit 1
fi
