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
SAP_BTP_OPERATOR_DEPLOYMENT_NAME=sap-btp-operator-controller-manager
BTP_MANAGER_DEPLOYMENT_NAME=btp-manager-controller-manager
EXPECTED_SAP_BTP_SERVICE_OPERATOR_CHART_VER=$(yq '.version' module-chart/chart/Chart.yaml)
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

echo -e "\n--- PREPARING ENVIRONMENT"

# deploy base image
scripts/testing/install_module.sh "${BASE_IMAGE}" dummy

BASE_SAP_BTP_OPERATOR_CHART_VER=$(kubectl get -n kyma-system deployment/${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} -o jsonpath='{.metadata.labels.chart-version}')
echo -e "\n--- SAP BTP Service Operator chart version before upgrade: ${BASE_SAP_BTP_OPERATOR_CHART_VER}"

SI_NAME=auditlog-management-si-dummy
export SI_NAME

echo -e "\n--- Creating ServiceInstance: ${SI_NAME}"
envsubst <${YAML_DIR}/e2e-test-service-instance.yaml | kubectl apply -f -

echo -e "\n--- Waiting for ServiceInstance existence"
until kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME}; do sleep 5; done

# set BtpOperator CR in Deleting state
echo -e "\n--- Deleting BtpOperator CR (setting Deleting state)"
kubectl delete btpoperators/e2e-test-btpoperator &

echo -e "\n--- Waiting for ServiceInstancesAndBindingsNotCleaned reason"
while [[ $(kubectl get btpoperators/e2e-test-btpoperator -o json| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs) != "FalseServiceInstancesAndBindingsNotCleaned" ]];
do sleep 5; done

echo -e "\n--- UPGRADING MODULE"

# deploy upgrade image
echo -e "\n--- Deploying module with image: ${UPGRADE_IMAGE} - invoking make"
IMG=${UPGRADE_IMAGE} make deploy

echo -e "\n--- Waiting for BTP Manager deployment to be available"
while [[ $(kubectl get deployment/${BTP_MANAGER_DEPLOYMENT_NAME} -n kyma-system -o 'jsonpath={..status.conditions[?(@.type=="Available")].status}') != "True" ]];
do sleep 5; done

echo -e "\n--- Expected SAP BTP Service Operator chart version after upgrade: ${EXPECTED_SAP_BTP_SERVICE_OPERATOR_CHART_VER}"

ACTUAL_SAP_BTP_SERVICE_OPERATOR_CHART_VER=""

echo -e "\n--- Waiting for SAP BTP Service Operator deployment reconciliation"
while [[ "${ACTUAL_SAP_BTP_SERVICE_OPERATOR_CHART_VER}" != "${EXPECTED_SAP_BTP_SERVICE_OPERATOR_CHART_VER}" ]];
do ACTUAL_SAP_BTP_SERVICE_OPERATOR_CHART_VER=$(kubectl get -n kyma-system deployment/${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} -o jsonpath='{.metadata.labels.chart-version}');
sleep 5; done

echo -e "\n--- SAP BTP Service Operator deployment has been reconciled. Current chart version: ${ACTUAL_SAP_BTP_SERVICE_OPERATOR_CHART_VER}"

echo -e "\n--- CLEANING UP"

echo -e "\n--- Adding force delete label"
kubectl label -f ${YAML_DIR}/e2e-test-btpoperator.yaml force-delete=true

while [[ "$(kubectl get btpoperators/e2e-test-btpoperator 2>&1)" != *"Error from server (NotFound)"* ]];
do echo -e "\n--- Waiting for BtpOperator CR to be removed"; sleep 5; done

echo -e "\n--- BtpOperator CR has been removed"

echo -e "\n--- Checking if ServiceInstance CRD was removed"
[[ "$(kubectl get crd serviceinstances 2>&1)" != *"Error from server (NotFound)"* ]] \
&& echo "ServiceInstance CRD still exists when it should have been removed" && exit 1

echo -e "\n--- ServiceInstance CRD has been removed"

echo -e "\n--- Checking if ServiceBinding CRD was removed"
[[ "$(kubectl get crd servicebindings 2>&1)" != *"Error from server (NotFound)"* ]] \
&& echo "ServiceBinding CRD still exists when it should have been removed" && exit 1

echo -e "\n--- ServiceBinding CRD has been removed"

echo -e "\n--- BTP Operator deprovisioning succeeded"

echo -e "\n--- Uninstalling BTP Manager"

# uninstall btp-manager
make undeploy

#clean up and ignore errors
kubectl delete -f ./examples/btp-manager-secret.yaml || echo "ignoring failure during secret removal"
kubectl delete -f ./deployments/prerequisites.yaml || echo "ignoring failure during prerequisites removal"
