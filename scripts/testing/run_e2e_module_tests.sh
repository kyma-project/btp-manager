#!/usr/bin/env bash
set -x #TODO remove
# This script has the following arguments:
#     - link to a module image (required),
#     - credentials mode, allowed values (required):
#         dummy - dummy credentials passed
#         real - real credentials passed
#     - ci to indicate call from CI pipeline (optional)
# ./run_e2e_module_tests.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:v0.0.0-PR-999 real ci
#
# The script requires the following environment variable set - this value is used to create unique SI and SB names:
#      GITHUB_RUN_ID - A unique number for each workflow run within a repository
# The script requires the following environment variables if is called with "real" parameter - these should be real credentials base64 encoded:
#      SM_CLIENT_ID - client ID
#      SM_CLIENT_SECRET - client secret
#      SM_URL - service manager url
#      SM_TOKEN_URL - token url

CI=${3-manual}  # if called from any workflow "ci" is expected here

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

MODULE_IMAGE_NAME=$1
CREDENTIALS=$2
YAML_DIR="scripts/testing/yaml"

[[ -z ${GITHUB_RUN_ID} ]] && echo "required variable GITHUB_RUN_ID not set" && exit 1

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

SI_NAME=e2e-test-service-instance-${GITHUB_RUN_ID}
SB_NAME=e2e-test-service-binding-${GITHUB_RUN_ID}

echo -e "\n---Creating service instance: ${SI_NAME}"
envsubst <${YAML_DIR}/e2e-test-service-instance.yaml | kubectl apply -f -

echo -e "\n---Creating service binding: ${SB_NAME}"
envsubst <${YAML_DIR}/e2e-test-service-binding.yaml | kubectl apply -f -

if [[ "${CREDENTIALS}" == "real" ]]
then
  echo -e "\n---Using real credentials"
  while [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]];
  do echo -e "\n---Waiting for service instance to be ready"; sleep 5; done

  while [[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]];
  do echo -e "\n---Waiting for service binding to be ready"; sleep 5; done
else
  echo -e "\n---Using dummy credentials"
  while [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o json | jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs) != "FalseNotProvisioned" ]] \
  && [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o json | jq '.status.conditions[] | select(.type=="Succeeded") |.reason'|xargs) != "CreateInProgress" ]];
  do echo -e "\n---Waiting for service instance to be not ready due to invalid credentials"; sleep 5; done

  while [[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o json | jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs) != "FalseNotProvisioned" ]] \
  && [[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o json | jq '.status.conditions[] | select(.type=="Succeeded") |.reason'|xargs) != "CreateInProgress" ]];
  do echo -e "\n---Waiting for service binding to be not ready due to invalid credentials"; sleep 5; done
fi

echo -e "\n---Uninstalling..."

# remove btp-operator (service binding and service instance will be deleted if these were created)
kubectl delete btpoperators/e2e-test-btpoperator || echo "ignoring failure during btp-operator removal"

# uninstall btp-manager
./scripts/uninstall_btp_manager.sh

#clean up and ignore errors
kubectl delete -f ./examples/btp-manager-secret.yaml || echo "ignoring failure during secret removal"
kubectl delete -f ./deployments/prerequisites.yaml || echo "ignoring failure during prerequisites removal"
