#!/usr/bin/env bash

# This script has the following arguments:
#     - link to the upgrade image (optional),
#     - link to the base image (optional),
# ./run_e2e_module_upgrade_during_deletion_tests.sh [upgrade-image] [base-image]
# ./run_e2e_module_upgrade_during_deletion_tests.sh europe-docker.pkg.dev/kyma-project/prod/btp-manager:1.1.2 europe-docker.pkg.dev/kyma-project/prod/btp-manager:1.0.0

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

REGISTRY=europe-docker.pkg.dev/kyma-project/prod/btp-manager
YAML_DIR="scripts/testing/yaml"

if [[ $# -eq 2 ]]; then
  # upgrade from one given version to another given version
  UPGRADE_IMAGE=${1}
  BASE_IMAGE=${2}
elif [[ $# -eq 1 ]]; then
  # upgrade from the latest release to the given version
  UPGRADE_IMAGE=${1}
  REPOSITORY=${REPOSITORY:-kyma-project/btp-manager}
  GITHUB_URL=https://api.github.com/repos/${REPOSITORY}
  BASE_IMAGE_TAG=$(curl -sS "${GITHUB_URL}/releases/latest" | jq -r '.tag_name')
  BASE_IMAGE=${REGISTRY}:${BASE_IMAGE_TAG}
elif [[ $# -eq 0 ]]; then
  # upgrade from the pre-latest release to the latest release
  REPOSITORY=${REPOSITORY:-kyma-project/btp-manager}
  GITHUB_URL=https://api.github.com/repos/${REPOSITORY}
  UPGRADE_IMAGE_TAG=$(curl -sS "${GITHUB_URL}/releases/latest" | jq -r '.tag_name')
  BASE_IMAGE_TAG=$(curl -sS "${GITHUB_URL}/tags" | jq -r '.[].name' | grep -A1 "${UPGRADE_IMAGE_TAG}" | grep -v "${UPGRADE_IMAGE_TAG}")
  UPGRADE_IMAGE=${REGISTRY}:${UPGRADE_IMAGE_TAG}
  BASE_IMAGE=${REGISTRY}:${BASE_IMAGE_TAG}
else
  echo "wrong number of arguments" && exit 1
fi

echo "--- E2E Module Upgrade Test when BtpOperator CR is in Deleting state"
echo "-- FROM: ${BASE_IMAGE}"
echo "-- TO: ${UPGRADE_IMAGE}"

scripts/testing/install_module.sh "${BASE_IMAGE}" dummy

SI_NAME=auditlog-management-si-dummy
export SI_NAME

echo -e "\n--- Creating ServiceInstance: ${SI_NAME}"
envsubst <${YAML_DIR}/e2e-test-service-instance.yaml | kubectl apply -f -

until kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME}; do echo -e "\n--- Waiting for ServiceInstance existence"; sleep 5; done

# set BtpOperator CR in Deleting state
echo -e "\n--- Deleting BtpOperator CR (setting Deleting state)"
kubectl delete btpoperators/e2e-test-btpoperator &

while [[ $(kubectl get btpoperators/e2e-test-btpoperator -o json| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != "FalseServiceInstancesAndBindingsNotCleaned" ]];
do echo -e "\n--- Waiting for ServiceInstancesAndBindingsNotCleaned reason"; sleep 5; done