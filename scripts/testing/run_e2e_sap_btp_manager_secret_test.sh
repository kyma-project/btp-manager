#!/usr/bin/env bash

# The script requires BTP Manager to be installed and running in the cluster.
# Run install_module.sh script before running this script.

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

checkResourcesReconciliation () {
  local ACTUAL_SAP_BTP_OPERATOR_SECRET=""
  local ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP=""
  local ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET=""

  local SAP_BTP_OPERATOR_SECRET_CHANGED=false
  local SAP_BTP_OPERATOR_CONFIGMAP_CHANGED=false
  local SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_CHANGED=false
  local RESOURCES_CHANGED=false

  SECONDS=0
  TIMEOUT=60
  until $RESOURCES_CHANGED
  do
    if [[ ${SECONDS} -ge ${TIMEOUT} ]]; then
      echo "timed out after ${TIMEOUT}s" && exit 1
    fi
    if ! ${SAP_BTP_OPERATOR_SECRET_CHANGED}; then
      ACTUAL_SAP_BTP_OPERATOR_SECRET=$(kubectl get secret -n $1 ${SAP_BTP_OPERATOR_SECRET_NAME} -o json) && \
      [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_SECRET} | jq -r .data.clientid)" == "${ENCODED_CLIENT_ID}" ]] && \
      [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_SECRET} | jq -r .data.clientsecret)" == "${ENCODED_CLIENT_SECRET}" ]] && \
      [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_SECRET} | jq -r .data.sm_url)" == "${ENCODED_SM_URL}" ]] && \
      [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_SECRET} | jq -r .data.tokenurl)" == "${ENCODED_TOKEN_URL}" ]] && \
      echo "${SAP_BTP_OPERATOR_SECRET_NAME} secret exists in $1 namespace and contains updated values" && \
      SAP_BTP_OPERATOR_SECRET_CHANGED=true
    fi
    if ! ${SAP_BTP_OPERATOR_CONFIGMAP_CHANGED}; then
      ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP=$(kubectl get configmap -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} -o json) && \
      [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP} | jq -r .data.CLUSTER_ID)" == "${CLUSTER_ID}" ]] && \
      [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP} | jq -r .data.RELEASE_NAMESPACE)" == "$1" ]] && \
      [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP} | jq -r .data.MANAGEMENT_NAMESPACE)" == "$1" ]] && \
      echo "${SAP_BTP_OPERATOR_CONFIGMAP_NAME} ConfigMap in ${KYMA_NAMESPACE} namespace contains updated values" && \
      SAP_BTP_OPERATOR_CONFIGMAP_CHANGED=true
    fi
    if ! ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_CHANGED}; then
      ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET=$(kubectl get secret -n $1 ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} -o json) && \
      [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET} | jq -r .data.INITIAL_CLUSTER_ID)" == "${ENCODED_CLUSTER_ID}" ]] && \
      echo "${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secret exists in $1 namespace and contains updated value" && \
      SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_CHANGED=true
    fi
    if ${SAP_BTP_OPERATOR_SECRET_CHANGED} && ${SAP_BTP_OPERATOR_CONFIGMAP_CHANGED} && ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_CHANGED}; then
      RESOURCES_CHANGED=true
    fi
    sleep 2
  done
}

checkSecretsRemovalFromPreviousNamespace () {
  echo -e "\n--- Checking if ${SAP_BTP_OPERATOR_SECRET_NAME} has been removed from $1 namespace"
  ([[ "$(kubectl get secret -n $1 ${SAP_BTP_OPERATOR_SECRET_NAME} 2>&1)" = *"Error from server (NotFound)"* ]] && echo "secret has been removed") || \
  (echo "secret has not been removed" && exit 1)

  echo -e "\n--- Checking if ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} has been removed from $1 namespace"
  ([[ "$(kubectl get secret -n $1 ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} 2>&1)" = *"Error from server (NotFound)"* ]] && echo "secret has been removed") || \
  (echo "secret has not been removed" && exit 1)
}

checkPodEnvs() {
  # Get environment variables from the SAP BTP service operator pod
  ACTUAL_SAP_BTP_OPERATOR_POD_CLUSTER_ID=$(kubectl exec -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_POD_NAME} -c manager -- printenv CLUSTER_ID)
  ACTUAL_SAP_BTP_OPERATOR_POD_RELEASE_NAMESPACE=$(kubectl exec -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_POD_NAME} -c manager -- printenv RELEASE_NAMESPACE)
  ACTUAL_SAP_BTP_OPERATOR_POD_MANAGEMENT_NAMESPACE=$(kubectl exec -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_POD_NAME} -c manager -- printenv MANAGEMENT_NAMESPACE)

  # Check envs in the SAP BTP service operator pod
  echo -e "\n--- Checking ${SAP_BTP_OPERATOR_POD_NAME} pod's CLUSTER_ID, RELEASE_NAMESPACE, MANAGEMENT_NAMESPACE environment variables"

  if [[ "${ACTUAL_SAP_BTP_OPERATOR_POD_CLUSTER_ID}" == "$1" && \
        "${ACTUAL_SAP_BTP_OPERATOR_POD_RELEASE_NAMESPACE}" == "$2" && \
        "${ACTUAL_SAP_BTP_OPERATOR_POD_MANAGEMENT_NAMESPACE}" == "$2" ]]; then
    echo "Environment variables match"
  else
    echo "Environment variables do not match"
    exit 1
  fi
}

waitForResourceExistence() {
  local resource_type=$1
  local resource_name=$2
  local namespace=${3:-kyma-system}
  local timeout=${4:-10}

  echo -e "\n--- Checking $resource_type/$resource_name existence in $namespace namespace"
  local seconds=0
  while [[ $seconds -lt $timeout ]]; do
    local exists="$(kubectl get -n $namespace $resource_type/$resource_name 2>&1)"
    if [[ $exists != *"Error from server (NotFound)"* ]]; then
      echo -e "--- $resource_type/$resource_name exists in $namespace namespace"
      return 0
    fi
    echo -e "--- Waiting for $resource_type/$resource_name existence in $namespace namespace (${seconds}s/${timeout}s)"
    sleep 2
  done
  echo -e "--- ERROR: Timed out waiting for $resource_type/$resource_name existence in $namespace namespace"
  return 1
}

# Set environment variables
YAML_DIR="scripts/testing/yaml"
SECRET_RESOURCE=secret
CONFIGMAP_RESOURCE=configmap

## Resources names
KYMA_NAMESPACE=kyma-system
BTP_MANAGER_SECRET_NAME=sap-btp-manager
SAP_BTP_OPERATOR_SECRET_NAME=sap-btp-service-operator
SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME=sap-btp-operator-clusterid
SAP_BTP_OPERATOR_CONFIGMAP_NAME=sap-btp-operator-config

## Fake values for the secret
CLIENT_ID="new-client-id"
CLIENT_SECRET="new-client-secret"
SM_URL="new-sm-url"
TOKEN_URL="new-token-url"
CLUSTER_ID="new-cluster-id"
CREDENTIALS_NAMESPACE="credentials-namespace"
ENCODED_CLIENT_ID=$(echo -n ${CLIENT_ID} | base64)
ENCODED_CLIENT_SECRET=$(echo -n ${CLIENT_SECRET} | base64)
ENCODED_SM_URL=$(echo -n ${SM_URL} | base64)
ENCODED_TOKEN_URL=$(echo -n ${TOKEN_URL} | base64)
ENCODED_CLUSTER_ID=$(echo -n ${CLUSTER_ID} | base64)
ENCODED_CREDENTIALS_NAMESPACE=$(echo -n ${CREDENTIALS_NAMESPACE} | base64)
ENCODED_KYMA_NAMESPACE=$(echo -n ${KYMA_NAMESPACE} | base64)

echo -e "\n--- BTP Manager secret customization test ---\n"

## Check SAP BTP service operator secret existence
waitForResourceExistence $SECRET_RESOURCE $SAP_BTP_OPERATOR_SECRET_NAME

## Check SAP BTP service operator configmap existence
waitForResourceExistence $CONFIGMAP_RESOURCE $SAP_BTP_OPERATOR_CONFIGMAP_NAME

## Check SAP BTP service operator cluster ID secret existence
waitForResourceExistence $SECRET_RESOURCE $SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME

# Create credentials namespace if required
echo -e "\n--- Creating ${CREDENTIALS_NAMESPACE} if required"
kubectl create namespace ${CREDENTIALS_NAMESPACE} || echo "${CREDENTIALS_NAMESPACE} namespace already exists"

# Patch the secret with fake values
echo -e "\n--- Customizing ${BTP_MANAGER_SECRET_NAME} secret with fake values"
kubectl patch secret -n ${KYMA_NAMESPACE} ${BTP_MANAGER_SECRET_NAME} -p "{\"data\":{\"clientid\":\"${ENCODED_CLIENT_ID}\",\"clientsecret\":\"${ENCODED_CLIENT_SECRET}\",\"sm_url\":\"${ENCODED_SM_URL}\",\"tokenurl\":\"${ENCODED_TOKEN_URL}\",\"cluster_id\":\"${ENCODED_CLUSTER_ID}\",\"credentials_namespace\":\"${ENCODED_CREDENTIALS_NAMESPACE}\"}}" || \
(echo "could not patch ${BTP_MANAGER_SECRET_NAME} secret in ${KYMA_NAMESPACE} namespace, command return code: $?" && exit 1)

# Wait until resources are reconciled
echo -e "\n--- Waiting for SAP BTP service operator secrets and configmap changes"
echo -e "-- Expected changes:" \
"\n- ${SAP_BTP_OPERATOR_SECRET_NAME} and ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secrets exist in ${CREDENTIALS_NAMESPACE} namespace" \
"\n- ${SAP_BTP_OPERATOR_SECRET_NAME} secret contains updated Service Manager credentials" \
"\n- ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secret contains updated value of the INITIAL_CLUSTER_ID key" \
"\n- ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} ConfigMap in ${KYMA_NAMESPACE} namespace contains the updated values of the following keys: CLUSTER_ID, RELEASE_NAMESPACE, MANAGEMENT_NAMESPACE\n"
checkResourcesReconciliation ${CREDENTIALS_NAMESPACE}

# Check if secrets have been removed from the previous namespace
checkSecretsRemovalFromPreviousNamespace ${KYMA_NAMESPACE}

# Check SAP BTP service operator pod environment variables
SAP_BTP_OPERATOR_POD_NAME=$(kubectl get pod -n ${KYMA_NAMESPACE} -l app.kubernetes.io/name=sap-btp-operator -o jsonpath="{.items[*].metadata.name}")
checkPodEnvs ${CLUSTER_ID} ${CREDENTIALS_NAMESPACE}

echo -e "\n--- SAP BTP service operator secrets and configmap reconciliation succeeded!"

while [[ $(kubectl get btpoperators/btpoperator -n kyma-system -ojson| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != "TrueReconcileSucceeded" ]];
do echo -e "\n--- Waiting for BTP Operator to be ready and reconciled"; sleep 5; done

echo -e "\n--- BTP Manager secret customization succeeded!"

echo -e "\n--- Checking corner cases..."

# Patch the secret with kyma-system namespace as credentials namespace
echo -e "\n--- Customizing ${BTP_MANAGER_SECRET_NAME} secret with ${KYMA_NAMESPACE} namespace as credentials namespace"
kubectl patch secret -n ${KYMA_NAMESPACE} ${BTP_MANAGER_SECRET_NAME} -p "{\"data\":{\"credentials_namespace\":\"${ENCODED_KYMA_NAMESPACE}\"}}" || \
(echo "could not patch ${BTP_MANAGER_SECRET_NAME} secret in ${KYMA_NAMESPACE} namespace, command return code: $?" && exit 1)

# Wait until resources are reconciled
echo -e "\n--- Waiting for SAP BTP service operator secrets and configmap changes"
echo -e "-- Expected changes:" \
"\n- ${SAP_BTP_OPERATOR_SECRET_NAME} and ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secrets exist in ${KYMA_NAMESPACE} namespace" \
"\n- ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} ConfigMap in ${KYMA_NAMESPACE} namespace contains the updated values of the following keys: RELEASE_NAMESPACE, MANAGEMENT_NAMESPACE\n"
checkResourcesReconciliation ${KYMA_NAMESPACE}

# Check if secrets have been removed from the previous namespace
checkSecretsRemovalFromPreviousNamespace ${CREDENTIALS_NAMESPACE}

# Check SAP BTP service operator pod environment variables
SAP_BTP_OPERATOR_POD_NAME=$(kubectl get pod -n ${KYMA_NAMESPACE} -l app.kubernetes.io/name=sap-btp-operator -o jsonpath="{.items[*].metadata.name}")
checkPodEnvs ${CLUSTER_ID} ${KYMA_NAMESPACE}

while [[ $(kubectl get btpoperators/btpoperator -n kyma-system -ojson| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != "TrueReconcileSucceeded" ]];
do echo -e "\n--- Waiting for BTP Operator to be ready and reconciled"; sleep 5; done

echo -e "\n-- Changing INITIAL_CLUSTER_ID in ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secret"
kubectl patch secret -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} -p "{\"data\":{\"INITIAL_CLUSTER_ID\":\"$(echo -n 'different-cluster-id' | base64)\"}}" || \
(echo "could not patch ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secret in ${KYMA_NAMESPACE} namespace, command return code: $?" && exit 1)

echo -e "\n--- Creating sap-btp-manager configmap with ReadyTimeout 10s"
kubectl apply -f ${YAML_DIR}/e2e-test-configmap.yaml
kubectl patch configmap sap-btp-manager -n kyma-system --type merge -p '{"data":{"ReadyTimeout":"10s"}}'

SAP_BTP_OPERATOR_POD_NAME=$(kubectl get pod -n ${KYMA_NAMESPACE} -l app.kubernetes.io/name=sap-btp-operator -o jsonpath="{.items[*].metadata.name}")
echo -e "\n-- Deleting ${SAP_BTP_OPERATOR_POD_NAME} pod to enforce CrashLoopBackOff due to invalid cluster ID"
kubectl delete pod -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_POD_NAME} || \
(echo "could not delete ${SAP_BTP_OPERATOR_POD_NAME} pod in ${KYMA_NAMESPACE} namespace, command return code: $?" && exit 1)

# Wait until the pod is recreated
echo -e "\n-- Waiting for pod to be recreated"
SAP_BTP_OPERATOR_POD_NAME=""
until [[ -n "${SAP_BTP_OPERATOR_POD_NAME}" ]]; do
  SAP_BTP_OPERATOR_POD_NAME=$(kubectl get pod -n ${KYMA_NAMESPACE} -l app.kubernetes.io/name=sap-btp-operator -o jsonpath="{.items[*].metadata.name}")
  sleep 2
done

# Wait until the pod is in CrashLoopBackOff state
until [[ "$(kubectl get pod -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_POD_NAME} -o json | jq -r '.status.containerStatuses[] | select(.state.waiting.reason == "CrashLoopBackOff") | .state.waiting.reason')" == "CrashLoopBackOff" ]]; do
  echo -e "\n-- Waiting for ${SAP_BTP_OPERATOR_POD_NAME} pod to be in the CrashLoopBackOff state..."
  SAP_BTP_OPERATOR_POD_NAME=$(kubectl get pod -n ${KYMA_NAMESPACE} -l app.kubernetes.io/name=sap-btp-operator -o jsonpath="{.items[*].metadata.name}")
  sleep 1
done

# Wait until resources are reconciled
echo -e "\n--- Waiting for SAP BTP service operator secrets and configmap changes"
echo -e "-- Expected changes:" \
"\n- ${SAP_BTP_OPERATOR_SECRET_NAME} and ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secrets exist in ${KYMA_NAMESPACE} namespace" \
"\n- ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secret is recreated and contains the correct value of the INITIAL_CLUSTER_ID key\n"
checkResourcesReconciliation ${KYMA_NAMESPACE}

# Check SAP BTP service operator pod environment variables
SAP_BTP_OPERATOR_POD_NAME=$(kubectl get pod -n ${KYMA_NAMESPACE} -l app.kubernetes.io/name=sap-btp-operator -o jsonpath="{.items[*].metadata.name}")
checkPodEnvs ${CLUSTER_ID} ${KYMA_NAMESPACE}

while [[ $(kubectl get btpoperators/btpoperator -n kyma-system -ojson| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != "TrueReconcileSucceeded" ]];
do echo -e "\n--- Waiting for BTP Operator to be ready and reconciled"; sleep 5; done

echo -e "\n--- Corner cases check succeeded!"

echo -e "\n--- Uninstalling..."

kubectl delete btpoperators/btpoperator -n kyma-system &
while [[ "$(kubectl get btpoperators/btpoperator -n kyma-system  2>&1)" != *"Error from server (NotFound)"* ]];
do echo -e "\n--- Waiting for BtpOperator CR to be removed"; sleep 5; done

echo -e "\n--- BTP Operator deprovisioning succeeded"

echo -e "\n--- Uninstalling BTP Manager"

# Uninstall BTP Manager
make undeploy

# Clean up and ignore errors
kubectl delete -f ./examples/btp-manager-secret.yaml || echo "ignoring failure during secret removal"
kubectl delete -f ./deployments/prerequisites.yaml || echo "ignoring failure during prerequisites removal"
