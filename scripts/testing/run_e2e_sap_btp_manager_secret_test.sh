#!/usr/bin/env bash

# The script requires BTP Manager to be installed and running in the cluster.
# Run install_module.sh script before running this script.

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

echo -e "\n--- SAP BTP Manager secret customization test ---"

# Set environment variables
## Resources names
RELEASE_NAMESPACE=kyma-system
BTP_MANAGER_SECRET_NAME=sap-btp-manager
SAP_BTP_OPERATOR_SECRET_NAME=sap-btp-service-operator
SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME=sap-btp-operator-clusterid
SAP_BTP_OPERATOR_CONFIGMAP_NAME=sap-btp-operator-config

## Fake values for the secret
CLIENT_ID=$(echo -n "client-id" | base64)
CLIENT_SECRET=$(echo -n "client-secret" | base64)
SM_URL=$(echo -n "sm-url" | base64)
TOKEN_URL=$(echo -n "token-url" | base64)
CLUSTER_ID="cluster-id"
MANAGEMENT_NAMESPACE="management-namespace"
ENCODED_CLUSTER_ID=$(echo -n ${CLUSTER_ID} | base64)
ENCODED_MANAGEMENT_NAMESPACE=$(echo -n ${MANAGEMENT_NAMESPACE} | base64)

## Check secret existence in the release namespace
kubectl get secret -n ${RELEASE_NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} && echo "${SAP_BTP_OPERATOR_SECRET_NAME} secret exists in ${RELEASE_NAMESPACE} namespace" || \
echo "could not get ${SAP_BTP_OPERATOR_SECRET_NAME} secret in ${RELEASE_NAMESPACE} namespace, command return code: $?" && exit 1

## Save current resourceVersion of the resources to be updated
SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION=$(kubectl get configmap -n ${RELEASE_NAMESPACE} ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} -o jsonpath="{.metadata.resourceVersion}")

## Save current ID of the resource to be recreated
SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID=$(kubectl get secret -n ${RELEASE_NAMESPACE} ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} -o jsonpath="{.metadata.uid}")

## Variables to track resources changes
ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION=${SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION}
ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID=${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID}

## Conditionals and loops control variables
SAP_BTP_OPERATOR_SECRET_CHANGED=false
SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION_CHANGED=false
SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID_CHANGED=false
RESOURCES_CHANGED=false

# Patch the secret with fake values
echo -e "\n--- Customizing ${BTP_MANAGER_SECRET_NAME} secret with fake values"
kubectl patch secret -n ${RELEASE_NAMESPACE} ${BTP_MANAGER_SECRET_NAME} -p "{\"data\":{\"clientid\":\"${CLIENT_ID}\",\"clientsecret\":\"${CLIENT_SECRET}\",\"sm_url\":\"${SM_URL}\",\"tokenurl\":\"${TOKEN_URL}\",\"cluster_id\":\"${ENCODED_CLUSTER_ID}\",\"management_namespace\":\"${ENCODED_MANAGEMENT_NAMESPACE}\"}}"

# Wait until resources are reconciled
echo -e "\n--- Waiting for SAP BTP service operator secrets and configmap changes"
SECONDS=0
TIMEOUT=60
until $RESOURCES_CHANGED
do
  if [[ ${SECONDS} -ge ${TIMEOUT} ]]; then
    echo "timed out after ${TIMEOUT}s" && exit 1
  fi
  if ! ${SAP_BTP_OPERATOR_SECRET_CHANGED}; then
    if ! kubectl get secret -n ${MANAGEMENT_NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME}; then
      echo "${SAP_BTP_OPERATOR_SECRET_NAME} secret exists in ${MANAGEMENT_NAMESPACE} namespace"
      SAP_BTP_OPERATOR_SECRET_CHANGED=true
    fi
  fi
  if ! ${SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION_CHANGED}; then
    ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION=$(kubectl get configmap -n ${RELEASE_NAMESPACE} ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} -o jsonpath="{.metadata.resourceVersion}")
    if [[ "${ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION}" != "${SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION}" ]]; then
      echo "${SAP_BTP_OPERATOR_CONFIGMAP_NAME} configmap version changed"
      SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION_CHANGED=true
    fi
  fi
  if ! ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID_CHANGED}; then
    if [[ "$(kubectl get secret -n ${RELEASE_NAMESPACE} ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} 2>&1)" != *"Error from server (NotFound)"* ]]; then
      ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID=$(kubectl get secret -n ${RELEASE_NAMESPACE} ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} -o jsonpath="{.metadata.uid}")
      if [[ "${ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID}" != "${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID}" ]]; then
        echo "${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secret ID changed"
        SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID_CHANGED=true
      fi
    else
      echo "${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} secret does not exist in ${RELEASE_NAMESPACE} namespace"
    fi
  fi
  if ${SAP_BTP_OPERATOR_SECRET_CHANGED} && ${SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION_CHANGED} && ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID_CHANGED}; then
    RESOURCES_CHANGED=true
  fi
  sleep 2
done

echo -e "\n--- Checking if ${SAP_BTP_OPERATOR_SECRET_NAME} has been removed from ${RELEASE_NAMESPACE} namespace"
[[ "$(kubectl get secret -n ${RELEASE_NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} 2>&1)" = *"Error from server (NotFound)"* ]] && \
echo "secret has been removed" || echo "secret has not been removed" && exit 1

# Save the current data from secret and configmap
ACTUAL_SAP_BTP_OPERATOR_SECRET_CLIENT_ID=$(kubectl get secret -n ${MANAGEMENT_NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} -o jsonpath="{.data.clientid}")
ACTUAL_SAP_BTP_OPERATOR_SECRET_CLIENT_SECRET=$(kubectl get secret -n ${MANAGEMENT_NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} -o jsonpath="{.data.clientsecret}")
ACTUAL_SAP_BTP_OPERATOR_SECRET_SM_URL=$(kubectl get secret -n ${MANAGEMENT_NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} -o jsonpath="{.data.sm_url}")
ACTUAL_SAP_BTP_OPERATOR_SECRET_TOKEN_URL=$(kubectl get secret -n ${MANAGEMENT_NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} -o jsonpath="{.data.tokenurl}")
ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_CLUSTER_ID=$(kubectl get configmap -n ${RELEASE_NAMESPACE} ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} -o jsonpath="{.data.CLUSTER_ID}")
ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_MANAGEMENT_NAMESPACE=$(kubectl get configmap -n ${RELEASE_NAMESPACE} ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} -o jsonpath="{.data.MANAGEMENT_NAMESPACE}")

# Compare secrets and configmap data
echo -e "\n--- Checking SAP BTP service operator secret and configmap data"

if [[ "${ACTUAL_SAP_BTP_OPERATOR_SECRET_CLIENT_ID}" == "${CLIENT_ID}" && \
      "${ACTUAL_SAP_BTP_OPERATOR_SECRET_CLIENT_SECRET}" == "${CLIENT_SECRET}" && \
      "${ACTUAL_SAP_BTP_OPERATOR_SECRET_SM_URL}" == "${SM_URL}" && \
      "${ACTUAL_SAP_BTP_OPERATOR_SECRET_TOKEN_URL}" == "${TOKEN_URL}" ]]; then
  echo "Secret data matches"
else
  echo "Secret data does not match"
  exit 1
fi

if [[ "${ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_CLUSTER_ID}" == "${CLUSTER_ID}" && \
      "${ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_MANAGEMENT_NAMESPACE}" == "${MANAGEMENT_NAMESPACE}" ]]; then
  echo "ConfigMap data matches"
else
  echo "ConfigMap data does not match"
  exit 1
fi

# Get SAP BTP service operator pod name
SAP_BTP_OPERATOR_POD_NAME=$(kubectl get pod -n ${RELEASE_NAMESPACE} -l app.kubernetes.io/name=sap-btp-operator -o jsonpath="{.items[*].metadata.name}")

# Get environment variables from the SAP BTP service operator pod
ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID=$(kubectl exec -n ${RELEASE_NAMESPACE} ${SAP_BTP_OPERATOR_POD_NAME} -c manager -- printenv CLUSTER_ID)
ACTUAL_SAP_BTP_OPERATOR_MANAGEMENT_NAMESPACE=$(kubectl exec -n ${RELEASE_NAMESPACE} ${SAP_BTP_OPERATOR_POD_NAME} -c manager -- printenv MANAGEMENT_NAMESPACE)

# Check envs in the SAP BTP service operator pod
echo -e "\n--- Checking ${SAP_BTP_OPERATOR_POD_NAME} pod CLUSTER_ID and MANAGEMENT_NAMESPACE environment variables"

if [[ "${ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID}" == "${CLUSTER_ID}" && \
      "${ACTUAL_SAP_BTP_OPERATOR_MANAGEMENT_NAMESPACE}" == "${MANAGEMENT_NAMESPACE}" ]]; then
  echo "Environment variables match"
else
  echo "Environment variables do not match"
  exit 1
fi

echo -e "\n--- SAP BTP service operator secrets and configmap reconciliation succeeded!"

while [[ $(kubectl get btpoperators/e2e-test-btpoperator -ojson| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != "TrueReconcileSucceeded" ]];
do echo -e "\n---Waiting for BTP Operator to be ready and reconciled"; sleep 5; done

echo -e "\n--- SAP BTP Manager secret customization succeeded!"

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
