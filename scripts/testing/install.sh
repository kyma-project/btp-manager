#!/usr/bin/env bash

# This script has the following arguments:
#     - link to a module image (required),
#     - credentials mode, allowed values (required):
#         dummy - dummy credentials passed
#         real - real credentials passed
#     - ci to indicate call from CI pipeline (optional)
# ./install.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:v0.0.0-PR-999 real ci

CI=${3-manual}  # if called from any workflow "ci" is expected here

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

MODULE_IMAGE_NAME=$1
YAML_DIR="scripts/testing/yaml"
CREDENTIALS=$2

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

# fetch OCI module image and install btp-manager in current cluster
echo -e "\n--- Running module image: ${MODULE_IMAGE_NAME}"
./scripts/run_module_image.sh "${MODULE_IMAGE_NAME}" ${CI}

# check if deployment is available
while [[ $(kubectl get deployment/btp-manager-controller-manager -n kyma-system -o 'jsonpath={..status.conditions[?(@.type=="Available")].status}') != "True" ]];
do echo -e "\n---Waiting for deployment to be available"; sleep 5; done

echo -e "\n---Deployment available"

echo -e "\n---Installing BTP operator"
kubectl apply -f ${YAML_DIR}/e2e-test-btpoperator.yaml

while [[ $(kubectl get btpoperators/e2e-test-btpoperator -ojson| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != "TrueReconcileSucceeded" ]];
do echo -e "\n---Waiting for BTP Operator to be ready and reconciled"; sleep 5; done

# verifying whether service instance and service binding resources were created
echo -e "\n---Checking if serviceinstances and servicebindings CRDs are created"

CRDS=$(kubectl get crds|awk '/(servicebindings|serviceinstances)/{print $1}')
if [[ $(wc -w <<< ${CRDS}) -ne 2 ]]
then
  echo "Missing CR definitions - failing tests"
  exit 1
fi
