#!/usr/bin/env bash

# The script requires BTP Manager to be installed and running in the cluster.
# Run install_module.sh script before running this script.

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

echo -e "\n--- BTP Manager secret customization test ---"

# Set environment variables
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

## Check SAP BTP service operator secret existence
(kubectl get secret -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} && echo "${SAP_BTP_OPERATOR_SECRET_NAME} secret in ${KYMA_NAMESPACE} namespace exists") || \
(echo "could not get ${SAP_BTP_OPERATOR_SECRET_NAME} secret in ${KYMA_NAMESPACE} namespace, command return code: $?" && exit 1)

## Check SAP BTP service operator configmap existence
(kubectl get configmap -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} && echo "${SAP_BTP_OPERATOR_CONFIGMAP_NAME} ConfigMap in ${KYMA_NAMESPACE} namespace exists") || \
(echo "could not get ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} ConfigMap in ${KYMA_NAMESPACE} namespace, command return code: $?" && exit 1)

## Check SAP BTP service operator cluster ID secret existence
(kubectl get secret -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} && echo "${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secret in ${KYMA_NAMESPACE} namespace exists") || \
(echo "could not get ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secret in ${KYMA_NAMESPACE} namespace, command return code: $?" && exit 1)

## Variables to track resources changes
ACTUAL_SAP_BTP_OPERATOR_SECRET=""
ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP=""
ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET=""

## Conditionals and loops control variables
SAP_BTP_OPERATOR_SECRET_CHANGED=false
SAP_BTP_OPERATOR_CONFIGMAP_CHANGED=false
SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_CHANGED=false
RESOURCES_CHANGED=false

# Create credentials namespace if required
kubectl create namespace ${CREDENTIALS_NAMESPACE} || echo "${CREDENTIALS_NAMESPACE} namespace already exists"

# Patch the secret with fake values
echo -e "\n--- Customizing ${BTP_MANAGER_SECRET_NAME} secret with fake values"
kubectl patch secret -n ${KYMA_NAMESPACE} ${BTP_MANAGER_SECRET_NAME} -p "{\"data\":{\"clientid\":\"${ENCODED_CLIENT_ID}\",\"clientsecret\":\"${ENCODED_CLIENT_SECRET}\",\"sm_url\":\"${ENCODED_SM_URL}\",\"tokenurl\":\"${ENCODED_TOKEN_URL}\",\"cluster_id\":\"${ENCODED_CLUSTER_ID}\",\"credentials_namespace\":\"${ENCODED_CREDENTIALS_NAMESPACE}\"}}" || \
(echo "could not patch ${BTP_MANAGER_SECRET_NAME} secret in ${KYMA_NAMESPACE} namespace, command return code: $?" && exit 1)

# Wait until resources are reconciled
echo -e "\n--- Waiting for SAP BTP service operator secrets and configmap changes"
echo -e "\n-- Expected changes:" \
"\n- ${SAP_BTP_OPERATOR_SECRET_NAME} and ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secrets in ${CREDENTIALS_NAMESPACE} namespace" \
"\n- ${SAP_BTP_OPERATOR_SECRET_NAME} secret contains updated Service Manager credentials" \
"\n- ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secret contains updated value of the INITIAL_CLUSTER_ID key" \
"\n- ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} ConfigMap in ${KYMA_NAMESPACE} namespace contains the updated values of the following keys: CLUSTER_ID, RELEASE_NAMESPACE, MANAGEMENT_NAMESPACE"

SECONDS=0
TIMEOUT=60
until $RESOURCES_CHANGED
do
  if [[ ${SECONDS} -ge ${TIMEOUT} ]]; then
    echo "timed out after ${TIMEOUT}s" && exit 1
  fi
  if ! ${SAP_BTP_OPERATOR_SECRET_CHANGED}; then
    ACTUAL_SAP_BTP_OPERATOR_SECRET=$(kubectl get secret -n ${CREDENTIALS_NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} -o json) && \
    [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_SECRET} | jq -r .data.clientid)" == "${ENCODED_CLIENT_ID}" ]] && \
    [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_SECRET} | jq -r .data.clientsecret)" == "${ENCODED_CLIENT_SECRET}" ]] && \
    [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_SECRET} | jq -r .data.sm_url)" == "${ENCODED_SM_URL}" ]] && \
    [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_SECRET} | jq -r .data.tokenurl)" == "${ENCODED_TOKEN_URL}" ]] && \
    echo "${SAP_BTP_OPERATOR_SECRET_NAME} secret exists in ${CREDENTIALS_NAMESPACE} namespace and contains updated values" && \
    SAP_BTP_OPERATOR_SECRET_CHANGED=true
  fi
  if ! ${SAP_BTP_OPERATOR_CONFIGMAP_CHANGED}; then
    ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP=$(kubectl get configmap -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} -o json) && \
    [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP} | jq -r .data.CLUSTER_ID)" == "${CLUSTER_ID}" ]] && \
    [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP} | jq -r .data.RELEASE_NAMESPACE)" == "${CREDENTIALS_NAMESPACE}" ]] && \
    [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP} | jq -r .data.MANAGEMENT_NAMESPACE)" == "${CREDENTIALS_NAMESPACE}" ]] && \
    echo "${SAP_BTP_OPERATOR_CONFIGMAP_NAME} ConfigMap in ${KYMA_NAMESPACE} contains updated values" && \
    SAP_BTP_OPERATOR_CONFIGMAP_CHANGED=true
  fi
  if ! ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_CHANGED}; then
    ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET=$(kubectl get secret -n ${CREDENTIALS_NAMESPACE} ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} -o json) && \
    [[ "$(echo ${ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET} | jq -r .data.INITIAL_CLUSTER_ID)" == "${ENCODED_CLUSTER_ID}" ]] && \
    echo "${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secret exists in ${CREDENTIALS_NAMESPACE} namespace and contains updated value" && \
    SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_CHANGED=true
  fi
  if ${SAP_BTP_OPERATOR_SECRET_CHANGED} && ${SAP_BTP_OPERATOR_CONFIGMAP_CHANGED} && ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_CHANGED}; then
    RESOURCES_CHANGED=true
  fi
  sleep 2
done

echo -e "\n--- Checking if ${SAP_BTP_OPERATOR_SECRET_NAME} has been removed from ${KYMA_NAMESPACE} namespace"
([[ "$(kubectl get secret -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} 2>&1)" = *"Error from server (NotFound)"* ]] && echo "secret has been removed") || \
(echo "secret has not been removed" && exit 1)

echo -e "\n--- Checking if ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} has been removed from ${KYMA_NAMESPACE} namespace"
([[ "$(kubectl get secret -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} 2>&1)" = *"Error from server (NotFound)"* ]] && echo "secret has been removed") || \
(echo "secret has not been removed" && exit 1)

# Get SAP BTP service operator pod name
SAP_BTP_OPERATOR_POD_NAME=$(kubectl get pod -n ${KYMA_NAMESPACE} -l app.kubernetes.io/name=sap-btp-operator -o jsonpath="{.items[*].metadata.name}")

# Get environment variables from the SAP BTP service operator pod
ACTUAL_SAP_BTP_OPERATOR_POD_CLUSTER_ID=$(kubectl exec -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_POD_NAME} -c manager -- printenv CLUSTER_ID)
ACTUAL_SAP_BTP_OPERATOR_POD_RELEASE_NAMESPACE=$(kubectl exec -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_POD_NAME} -c manager -- printenv RELEASE_NAMESPACE)
ACTUAL_SAP_BTP_OPERATOR_POD_MANAGEMENT_NAMESPACE=$(kubectl exec -n ${KYMA_NAMESPACE} ${SAP_BTP_OPERATOR_POD_NAME} -c manager -- printenv MANAGEMENT_NAMESPACE)

# Check envs in the SAP BTP service operator pod
echo -e "\n--- Checking ${SAP_BTP_OPERATOR_POD_NAME} pod's CLUSTER_ID, RELEASE_NAMESPACE, MANAGEMENT_NAMESPACE environment variables"

if [[ "${ACTUAL_SAP_BTP_OPERATOR_POD_CLUSTER_ID}" == "${CLUSTER_ID}" && \
      "${ACTUAL_SAP_BTP_OPERATOR_POD_RELEASE_NAMESPACE}" == "${CREDENTIALS_NAMESPACE}" && \
      "${ACTUAL_SAP_BTP_OPERATOR_POD_MANAGEMENT_NAMESPACE}" == "${CREDENTIALS_NAMESPACE}" ]]; then
  echo "Environment variables match"
else
  echo "Environment variables do not match"
  exit 1
fi

echo -e "\n--- SAP BTP service operator secrets and configmap reconciliation succeeded!"

while [[ $(kubectl get btpoperators/e2e-test-btpoperator -ojson| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != "TrueReconcileSucceeded" ]];
do echo -e "\n---Waiting for BTP Operator to be ready and reconciled"; sleep 5; done

echo -e "\n--- BTP Manager secret customization succeeded!"

echo -e "\n--- Uninstalling..."

kubectl delete btpoperators/e2e-test-btpoperator &
while [[ "$(kubectl get btpoperators/e2e-test-btpoperator 2>&1)" != *"Error from server (NotFound)"* ]];
do echo -e "\n--- Waiting for BtpOperator CR to be removed"; sleep 5; done

echo -e "\n--- BTP Operator deprovisioning succeeded"

echo -e "\n--- Uninstalling BTP Manager"

# Uninstall BTP Manager
make undeploy

# Clean up and ignore errors
kubectl delete -f ./examples/btp-manager-secret.yaml || echo "ignoring failure during secret removal"
kubectl delete -f ./deployments/prerequisites.yaml || echo "ignoring failure during prerequisites removal"
