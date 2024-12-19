#!/usr/bin/env bash

# The script requires BTP Manager to be installed and running in the cluster.
# Run install_module.sh script before running this script.

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

NAMESPACE=kyma-system
BTP_MANAGER_SECRET_NAME=sap-btp-manager
SAP_BTP_OPERATOR_SECRET_NAME=sap-btp-service-operator
SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME=sap-btp-operator-clusterid
SAP_BTP_OPERATOR_CONFIGMAP_NAME=sap-btp-operator-config

# Environment variables with fake values for the secret
CLIENT_ID=$(echo "client-id" | base64)
CLIENT_SECRET=$(echo "client-secret" | base64)
SM_URL=$(echo "sm-url" | base64)
TOKEN_URL=$(echo "token-url" | base64)
CLUSTER_ID=$(echo "cluster-id" | base64)
MANAGEMENT_NAMESPACE=$(echo "management-namespace" | base64)

# Save current resourceVersion of the resources to be updated
SAP_BTP_OPERATOR_SECRET_RESOURCE_VERSION=$(kubectl get secret -n ${NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} -o jsonpath="{.metadata.resourceVersion}")
SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION=$(kubectl get configmap -n ${NAMESPACE} ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} -o jsonpath="{.metadata.resourceVersion}")

# Save current ID of the resource to be recreated
SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID=$(kubectl get secret -n ${NAMESPACE} ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} -o jsonpath="{.metadata.uid}")

# Patch the secret with fake values
kubectl patch secret -n ${NAMESPACE} ${BTP_MANAGER_SECRET_NAME} -p "{\"data\":{\"clientid\":\"${CLIENT_ID}\",\"clientsecret\":\"${CLIENT_SECRET}\",\"sm_url\":\"${SM_URL}\",\"tokenurl\":\"${TOKEN_URL}\",\"cluster_id\":\"${CLUSTER_ID}\",\"management_namespace\":\"${MANAGEMENT_NAMESPACE}\"}}"

ACTUAL_SAP_BTP_OPERATOR_SECRET_RESOURCE_VERSION=${SAP_BTP_OPERATOR_SECRET_RESOURCE_VERSION}
ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION=${SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION}
ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID=${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID}

SAP_BTP_OPERATOR_SECRET_RESOURCE_VERSION_CHANGED=false
SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION_CHANGED=false
SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID_CHANGED=false
RESOURCES_CHANGED=false

# Wait until resources are reconciled
echo -e "\n--- Waiting for SAP BTP service operator secrets and configmap changes"
SECONDS=0
TIMEOUT=120
until $RESOURCES_CHANGED
 [[ "${ACTUAL_SAP_BTP_OPERATOR_SECRET_RESOURCE_VERSION}" == "${SAP_BTP_OPERATOR_SECRET_RESOURCE_VERSION}" || "${ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION}" == "${SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION}" || "${ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID}" == "${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID}" ]]
do
  if [[ ${SECONDS} -ge ${TIMEOUT} ]]; then
    echo "timed out after ${TIMEOUT}s" && exit 1
  fi
  if ! ${SAP_BTP_OPERATOR_SECRET_RESOURCE_VERSION_CHANGED}; then
    ACTUAL_SAP_BTP_OPERATOR_SECRET_RESOURCE_VERSION=$(kubectl get secret -n ${NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} -o jsonpath="{.metadata.resourceVersion}")
    if [[ "${ACTUAL_SAP_BTP_OPERATOR_SECRET_RESOURCE_VERSION}" != "${SAP_BTP_OPERATOR_SECRET_RESOURCE_VERSION}" ]]; then
      echo "${SAP_BTP_OPERATOR_SECRET_NAME} resource version changed"
      SAP_BTP_OPERATOR_SECRET_RESOURCE_VERSION_CHANGED=true
    fi
  fi
  if ! ${SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION_CHANGED}; then
    ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION=$(kubectl get configmap -n ${NAMESPACE} ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} -o jsonpath="{.metadata.resourceVersion}")
    if [[ "${ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION}" != "${SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION}" ]]; then
      echo "${SAP_BTP_OPERATOR_CONFIGMAP_NAME} resource version changed"
      SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION_CHANGED=true
    fi
  fi
  if ! ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID_CHANGED}; then
    ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID=$(kubectl get secret -n ${NAMESPACE} ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} -o jsonpath="{.metadata.uid}")
    if [[ "${ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID}" != "${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID}" ]]; then
      echo "${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_NAME} ID changed"
      SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID_CHANGED=true
    fi
  fi
  if ${SAP_BTP_OPERATOR_SECRET_RESOURCE_VERSION_CHANGED} && ${SAP_BTP_OPERATOR_CONFIGMAP_RESOURCE_VERSION_CHANGED} && ${SAP_BTP_OPERATOR_CLUSTER_ID_SECRET_ID_CHANGED}; then
    RESOURCES_CHANGED=true
  fi
  sleep 2
done

# Compare secrets and configmap data
echo -e "\n--- Checking SAP BTP service operator secrets and configmap data"

ACTUAL_SAP_BTP_OPERATOR_SECRET_CLIENT_ID=$(kubectl get secret -n ${NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} -o jsonpath="{.data.clientid}")
ACTUAL_SAP_BTP_OPERATOR_SECRET_CLIENT_SECRET=$(kubectl get secret -n ${NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} -o jsonpath="{.data.clientsecret}")
ACTUAL_SAP_BTP_OPERATOR_SECRET_SM_URL=$(kubectl get secret -n ${NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} -o jsonpath="{.data.sm_url}")
ACTUAL_SAP_BTP_OPERATOR_SECRET_TOKEN_URL=$(kubectl get secret -n ${NAMESPACE} ${SAP_BTP_OPERATOR_SECRET_NAME} -o jsonpath="{.data.tokenurl}")
ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_CLUSTER_ID=$(kubectl get configmap -n ${NAMESPACE} ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} -o jsonpath="{.data.CLUSTER_ID}")
ACTUAL_SAP_BTP_OPERATOR_CONFIGMAP_MANAGEMENT_NAMESPACE=$(kubectl get configmap -n ${NAMESPACE} ${SAP_BTP_OPERATOR_CONFIGMAP_NAME} -o jsonpath="{.data.MANAGEMENT_NAMESPACE}")

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
SAP_BTP_OPERATOR_POD_NAME=$(kubectl get pod -n ${NAMESPACE} -l app.kubernetes.io/name=sap-btp-operator -o jsonpath="{.items[*].metadata.name}")

# Check envs in the SAP BTP service operator pod
echo -e "\n--- Checking ${SAP_BTP_OPERATOR_POD_NAME} pod CLUSTER_ID and MANAGEMENT_NAMESPACE environment variables"

ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID=$(kubectl exec -n ${NAMESPACE} ${SAP_BTP_OPERATOR_POD_NAME} -c manager -- printenv CLUSTER_ID)
ACTUAL_SAP_BTP_OPERATOR_MANAGEMENT_NAMESPACE=$(kubectl exec -n ${NAMESPACE} ${SAP_BTP_OPERATOR_POD_NAME} -c manager -- printenv MANAGEMENT_NAMESPACE)
ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID=$(${ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID} | base64)
ACTUAL_SAP_BTP_OPERATOR_MANAGEMENT_NAMESPACE=$(${ACTUAL_SAP_BTP_OPERATOR_MANAGEMENT_NAMESPACE} | base64)

if [[ "${ACTUAL_SAP_BTP_OPERATOR_CLUSTER_ID}" == "${CLUSTER_ID}" && \
      "${ACTUAL_SAP_BTP_OPERATOR_MANAGEMENT_NAMESPACE}" == "${MANAGEMENT_NAMESPACE}" ]]; then
  echo "Environment variables match"
else
  echo "Environment variables do not match"
  exit 1
fi
