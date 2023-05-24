#!/usr/bin/env bash

# This script has the following arguments:
#     - link to the upgrade module image (required),
#     - link to the base module image (optional),
#     - ci to indicate call from CI pipeline (optional)
# ./run_e2e_module_upgrade_tests.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:v0.4.1 ci
#
# The script requires the following environment variable set - these values are used to create unique SI and SB names:
#      GITHUB_RUN_ID - a unique number for each workflow run within a repository
#      GITHUB_JOB - the ID of the current job from the workflow
# The script requires the following environment variables - these should be real credentials base64 encoded:
#      SM_CLIENT_ID - client ID
#      SM_CLIENT_SECRET - client secret
#      SM_URL - service manager url
#      SM_TOKEN_URL - token url

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

[[ -z ${GITHUB_RUN_ID} ]] && echo "required variable GITHUB_RUN_ID not set" && exit 1
[[ -z ${GITHUB_JOB} ]] && echo "required variable GITHUB_JOB not set" && exit 1

if [[ $# -eq 3 ]]; then
  NEW_MODULE_IMAGE_NAME=$1
  OLD_MODULE_IMAGE_NAME=$2
  CI=${3-manual} # if called from any workflow "ci" is expected here
elif [[ $# -eq 2 ]]; then
  # get the latest release version
  REPOSITORY=${REPOSITORY:-kyma-project/btp-manager}
  GITHUB_URL=https://api.github.com/repos/${REPOSITORY}
  LATEST_RELEASE=$(curl -sS "${GITHUB_URL}/releases/latest" | jq -r '.tag_name')
  NEW_MODULE_IMAGE_NAME=$1
  OLD_MODULE_IMAGE_NAME=${NEW_MODULE_IMAGE_NAME/:*/:v$LATEST_RELEASE}
  CI=${2-manual} # if called from any workflow "ci" is expected here
else
  echo "wrong number of arguments" && exit 1
fi

YAML_DIR="scripts/testing/yaml"

# installing prerequisites, on production environment these are present before chart is used
kubectl apply -f ./deployments/prerequisites.yaml

# creating secret
[ -n "${SM_CLIENT_ID}" ] && [ -n "${SM_CLIENT_SECRET}" ] && [ -n "${SM_URL}" ] && [ -n "${SM_TOKEN_URL}" ] || (echo "Missing credentials - failing test" && exit 1)
envsubst <${YAML_DIR}/e2e-test-secret.yaml | kubectl apply -f -

# fetch the latest OCI module image and install btp-manager in current cluster
echo -e "\n--- Running module image: ${OLD_MODULE_IMAGE_NAME}"
./scripts/run_module_image.sh "${OLD_MODULE_IMAGE_NAME}" ${CI}

# check if deployment is available
while [[ $(kubectl get deployment/btp-manager-controller-manager -n kyma-system -o 'jsonpath={..status.conditions[?(@.type=="Available")].status}') != "True" ]];
do echo -e "\n--- Waiting for deployment to be available"; sleep 5; done

echo -e "\n--- Deployment available"

echo -e "\n---Installing BTP operator"
kubectl apply -f ${YAML_DIR}/e2e-test-btpoperator.yaml

while [[ $(kubectl get btpoperators/e2e-test-btpoperator -o json| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != "TrueReconcileSucceeded" ]];
do echo -e "\n--- Waiting for BTP Operator to be ready and reconciled"; sleep 5; done

echo -e "\n--- BTP Operator is ready"

# verifying whether service instance and service binding crds were created
echo -e "\n--- Checking if serviceinstances and servicebindings CRDs are created"
CRDS=$(kubectl get crds|awk '/(servicebindings|serviceinstances)/{print $1}')
if [[ $(wc -w <<< ${CRDS}) -ne 2 ]]
then
  echo "Missing CR definitions - failing tests"
  exit 1
fi

SI_NAME=auditlog-management-si-${GITHUB_JOB}-${GITHUB_RUN_ID}
SB_NAME=auditlog-management-sb-${GITHUB_JOB}-${GITHUB_RUN_ID}

export SI_NAME
export SB_NAME

echo -e "\n--- Creating ServiceInstance: ${SI_NAME}"
envsubst <${YAML_DIR}/e2e-test-service-instance.yaml | kubectl apply -f -

echo -e "\n--- Creating ServiceBinding: ${SB_NAME}"
envsubst <${YAML_DIR}/e2e-test-service-binding.yaml | kubectl apply -f -

while [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]];
do echo -e "\n--- Waiting for ServiceInstance to be ready"; sleep 5; done

echo -e "\n--- ServiceInstance is ready"

while [[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]];
do echo -e "\n--- Waiting for ServiceBinding to be ready"; sleep 5; done

echo -e "\n--- ServiceBinding is ready"

echo -e "\n--- Upgrading the module"
echo -e "\n--- Running module image: ${NEW_MODULE_IMAGE_NAME}"
./scripts/run_module_image.sh "${NEW_MODULE_IMAGE_NAME}" ${CI}

# check if deployment is available
while [[ $(kubectl get deployment/btp-manager-controller-manager -n kyma-system -o 'jsonpath={..status.conditions[?(@.type=="Available")].status}') != "True" ]];
do echo -e "\n--- Waiting for deployment to be available"; sleep 5; done

echo -e "\n--- Deployment available"

while [[ $(kubectl get btpoperators/e2e-test-btpoperator -o json| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != "TrueReconcileSucceeded" ]];
do echo -e "\n--- Waiting for BTP Operator to be ready and reconciled"; sleep 5; done

echo -e "\n--- BTP Operator is ready"

echo -e "\n--- Checking readiness of previously created ServiceInstance and ServiceBinding"

while [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]];
do echo -e "\n--- Waiting for ServiceInstance to be ready"; sleep 5; done

echo -e "\n--- ServiceInstance is ready"

while [[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]];
do echo -e "\n--- Waiting for ServiceBinding to be ready"; sleep 5; done

echo -e "\n--- ServiceBinding is ready"

SB_NAME=auditlog-management-sb2-${GITHUB_JOB}-${GITHUB_RUN_ID}

echo -e "\n--- Creating new ServiceBinding: ${SB_NAME}"
envsubst <${YAML_DIR}/e2e-test-service-binding.yaml | kubectl apply -f -

while [[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]];
do echo -e "\n--- Waiting for new ServiceBinding to be ready"; sleep 5; done

echo -e "\n--- New ServiceBinding is ready"

echo -e "\n--- Upgrade succeeded"

echo -e "\n--- Uninstalling..."

# remove btp-operator (ServiceInstance and ServiceBinding should be deleted as well)
kubectl delete btpoperators/e2e-test-btpoperator &

echo -e "\n--- Checking deprovisioning without force delete label"

while [[ $(kubectl get btpoperators/e2e-test-btpoperator -o json| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != "FalseServiceInstancesAndBindingsNotCleaned" ]];
do echo -e "\n--- Waiting for ServiceInstancesAndBindingsNotCleaned reason"; sleep 5; done

echo -e "\n--- Condition reason is correct"

echo -e "\n--- Checking if ServiceInstance still exists"
[[ "$(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} 2>&1)" = *"Error from server (NotFound)"* ]] \
&& echo "ServiceInstance was removed when it shouldn't have been" && exit 1

echo -e "\n--- Checking if ServiceInstance is in Ready state"
[[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]] \
&& echo "ServiceInstance is not in Ready state" && exit 1

echo -e "\n--- ServiceInstance exists and is in Ready state"

SB_NAME=auditlog-management-sb-${GITHUB_JOB}-${GITHUB_RUN_ID}

echo -e "\n--- Checking if ServiceBinding still exists"
[[ "$(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} 2>&1)" = *"Error from server (NotFound)"* ]] \
&& echo "ServiceBinding was removed when it shouldn't have been" && exit 1

echo -e "\n--- Checking if ServiceBinding is in Ready state"
[[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]] \
&& echo "ServiceBinding is not in Ready state" && exit 1

echo -e "\n--- ServiceBinding exists and is in Ready state"

SB_NAME=auditlog-management-sb2-${GITHUB_JOB}-${GITHUB_RUN_ID}

echo -e "\n--- Checking if new ServiceBinding still exists"
[[ "$(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} 2>&1)" = *"Error from server (NotFound)"* ]] \
&& echo "New ServiceBinding was removed when it shouldn't have been" && exit 1

echo -e "\n--- Checking if new ServiceBinding is in Ready state"
[[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]] \
&& echo "New ServiceBinding is not in Ready state" && exit 1

echo -e "\n--- New ServiceBinding exists and is in Ready state"

echo -e "\n--- Deprovisioning safety measures work"

echo -e "\n--- Adding force delete label"
kubectl label -f ${YAML_DIR}/e2e-test-btpoperator.yaml force-delete=true

echo -e "\n--- Checking deprovisioning with force delete label"

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
./scripts/uninstall_btp_manager.sh

#clean up and ignore errors
kubectl delete -f ./examples/btp-manager-secret.yaml || echo "ignoring failure during secret removal"
kubectl delete -f ./deployments/prerequisites.yaml || echo "ignoring failure during prerequisites removal"
