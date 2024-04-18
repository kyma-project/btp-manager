#!/usr/bin/env bash

# This script has the following arguments:
#     - credentials mode, allowed values (required):
#         dummy - dummy credentials passed
#         real - real credentials passed
# ./run_e2e_module_tests.sh real
#
# The script requires the following environment variable set - these values are used to create unique SI and SB names:
#      GITHUB_RUN_ID - a unique number for each workflow run within a repository
#      GITHUB_JOB - the ID of the current job from the workflow

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

CREDENTIALS=$1
YAML_DIR="scripts/testing/yaml"
SAP_BTP_OPERATOR_DEPLOYMENT_NAME=sap-btp-operator-controller-manager

[[ -z ${GITHUB_RUN_ID} ]] && echo "required variable GITHUB_RUN_ID not set" && exit 1

SI_NAME=auditlog-management-si-${GITHUB_JOB}-${GITHUB_RUN_ID}
SB_NAME=auditlog-management-sb-${GITHUB_JOB}-${GITHUB_RUN_ID}

export SI_NAME
export SB_NAME

echo -e "\n---Creating service instance: ${SI_NAME}"
envsubst <${YAML_DIR}/e2e-test-service-instance.yaml | kubectl apply -f -

echo -e "\n---Creating service binding: ${SB_NAME}"
envsubst <${YAML_DIR}/e2e-test-service-binding.yaml | kubectl apply -f -

if [[ "${CREDENTIALS}" == "real" ]]
then
  echo -e "\n---Using real credentials"
  while [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]];
  do echo -e "\n---Waiting for service instance to be ready"; sleep 5; done

  echo -e "\n---Service instance is ready"

  while [[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]];
  do echo -e "\n---Waiting for service binding to be ready"; sleep 5; done

  echo -e "\n---Service binding is ready"
else
  echo -e "\n---Using dummy credentials"
  while [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o json | jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs) != "FalseNotProvisioned" ]] \
  && [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o json | jq '.status.conditions[] | select(.type=="Succeeded") |.reason'|xargs) != "CreateInProgress" ]];
  do echo -e "\n---Waiting for service instance to be not ready due to invalid credentials"; sleep 5; done

  echo -e "\n---Service instance is not ready due to dummy/invalid credentials (Ready: NotProvisioned, Succeeded: CreateInProgress)"

  while [[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o json | jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs) != "FalseNotProvisioned" ]] \
  && [[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o json | jq '.status.conditions[] | select(.type=="Succeeded") |.reason'|xargs) != "CreateInProgress" ]];
  do echo -e "\n--- Waiting for service binding to be not ready due to invalid credentials"; sleep 5; done

  echo -e "\n--- Service binding is not ready due to dummy/invalid credentials (Ready: NotProvisioned, Succeeded: CreateInProgress)"
fi

echo -e "\n---Uninstalling..."

# remove btp-operator (ServiceInstance and ServiceBinding should be deleted as well)
kubectl delete btpoperators/e2e-test-btpoperator &

echo -e "\n--- Checking deprovisioning without force delete label"

while true; do
  operator_status=$(kubectl get btpoperators/e2e-test-btpoperator -o json)
  condition_status=$(echo $operator_status | jq -r '.status.conditions[] | select(.type=="Ready") | .status+.reason')
  state_status=$(echo $operator_status | jq -r '.status.state')

  if [[ $condition_status == "FalseServiceInstancesAndBindingsNotCleaned" && $state_status == "Warning" ]]; then
    break
  else
    echo -e "\n--- Waiting for ServiceInstancesAndBindingsNotCleaned reason and state"; sleep 5;
  fi
done

echo -e "\n--- Condition reason is correct"

echo -e "\n--- Checking if ServiceInstance still exists"
[[ "$(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} 2>&1)" = *"Error from server (NotFound)"* ]] \
&& echo "ServiceInstance was removed when it shouldn't have been" && exit 1

echo -e "\n--- Checking if ServiceBinding still exists"
[[ "$(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} 2>&1)" = *"Error from server (NotFound)"* ]] \
&& echo "ServiceBinding was removed when it shouldn't have been" && exit 1

if [[ "${CREDENTIALS}" == "real" ]]
then
  echo -e "\n--- Checking if ServiceInstance is in Ready state"
  [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]] \
  && echo "ServiceInstance is not in Ready state" && exit 1

  echo -e "\n--- ServiceInstance exists and is in Ready state"

  echo -e "\n--- Checking if ServiceBinding is in Ready state"
  [[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]] \
  && echo "ServiceBinding is not in Ready state" && exit 1

  echo -e "\n--- ServiceBinding exists and is in Ready state"
else
  [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o json | jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs) != "FalseNotProvisioned" ]] \
  && [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o json | jq '.status.conditions[] | select(.type=="Succeeded") |.reason'|xargs) != "CreateInProgress" ]] \
  && echo -e "\n--- ServiceInstance is not in expected state Ready: NotProvisioned, Succeeded: CreateInProgress"

  echo -e "\n--- ServiceInstance exists and is not ready due to dummy/invalid credentials (Ready: NotProvisioned, Succeeded: CreateInProgress)"

  [[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o json | jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs) != "FalseNotProvisioned" ]] \
  && [[ $(kubectl get servicebindings.services.cloud.sap.com/${SB_NAME} -o json | jq '.status.conditions[] | select(.type=="Succeeded") |.reason'|xargs) != "CreateInProgress" ]] \
  && echo -e "\n--- ServiceBinding is not in expected state Ready: NotProvisioned, Succeeded: CreateInProgress"

  echo -e "\n--- ServiceBinding exists and is not ready due to dummy/invalid credentials (Ready: NotProvisioned, Succeeded: CreateInProgress)"
fi

echo -e "\n--- Deprovisioning safety measures work"

echo -e "\n--- Checking module resources reconciliation when BtpOperator CR is in Deleting state"

echo "Deleting ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment"
kubectl delete -n kyma-system deployment/${SAP_BTP_OPERATOR_DEPLOYMENT_NAME}

while [[ "$(kubectl get -n kyma-system deployment/${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} 2>&1)" != *"Error from server (NotFound)"* ]];
do echo -e "\n--- Waiting for ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment deletion"; sleep 5; done

echo -e "\n--- Triggering reconciliation by annotating BtpOperator CR"
kubectl annotate --overwrite -f ${YAML_DIR}/e2e-test-btpoperator.yaml last-manual-reconciliation-timestamp="$(date -u -Iseconds)"

echo -e "\n--- Waiting for reconciliation (${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment existence)"
SECONDS=0
TIMEOUT=120
until kubectl get -n kyma-system deployment/${SAP_BTP_OPERATOR_DEPLOYMENT_NAME}
do
  if [[ ${SECONDS} -ge ${TIMEOUT} ]]; then
    echo "timed out after ${TIMEOUT}s" && exit 1
  fi
  sleep 5
done

echo -e "\n--- ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment has been reconciled"

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
make undeploy

#clean up and ignore errors
kubectl delete -f ./examples/btp-manager-secret.yaml || echo "ignoring failure during secret removal"
kubectl delete -f ./deployments/prerequisites.yaml || echo "ignoring failure during prerequisites removal"
