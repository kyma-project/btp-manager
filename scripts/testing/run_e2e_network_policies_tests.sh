#!/usr/bin/env bash

# This script tests the complete network policies lifecycle with BTP Manager:
# 1. Install the module (with network policies enabled by default)
# 2. Disable network policies via BtpOperator CR annotation
# 3. Delete BTP manager and SAP BTP operator pods
# 4. Wait for pods to be ready
# 5. Create a service instance
# 6. Re-enable network policies by removing the annotation
#
# Arguments:
#     - link to a binary image (required)
#     - credentials mode (required): dummy | real
# 
# Usage: ./run_e2e_network_policies_tests.sh europe-docker.pkg.dev/kyma-project/dev/btp-manager:PR-999 real

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

IMAGE_NAME=$1
CREDENTIALS=$2
YAML_DIR="scripts/testing/yaml"
SAP_BTP_OPERATOR_DEPLOYMENT_NAME=sap-btp-operator-controller-manager
BTP_MANAGER_DEPLOYMENT_NAME=btp-manager-controller-manager

[[ -z ${GITHUB_RUN_ID:-} ]] && GITHUB_RUN_ID="local-$(date +%s)"
[[ -z ${GITHUB_JOB:-} ]] && GITHUB_JOB="network-policies-test"

waitForBtpOperatorCrReadiness() {
  echo -e "\n--- Waiting for BtpOperator CR to be ready"
  while true; do
    operator_status=$(kubectl get btpoperators/btpoperator -n kyma-system -o json 2>/dev/null || echo '{}')
    state_status=$(echo $operator_status | jq -r '.status.state // "Unknown"')
    if [[ $state_status == "Ready" ]]; then
      echo -e "--- BtpOperator CR is ready"
      break
    else
      echo -e "--- BtpOperator CR state: $state_status, waiting..."
      sleep 5
    fi
  done
}

waitForDeploymentReady() {
  local deployment_name=$1
  local namespace=${2:-kyma-system}
  local timeout=${3:-300}
  
  echo -e "\n--- Waiting for deployment $deployment_name to be ready"
  local seconds=0
  while [[ $seconds -lt $timeout ]]; do
    local available=$(kubectl get deployment/$deployment_name -n $namespace -o 'jsonpath={..status.conditions[?(@.type=="Available")].status}' 2>/dev/null || echo "False")
    if [[ "$available" == "True" ]]; then
      echo -e "--- Deployment $deployment_name is ready"
      return 0
    fi
    echo -e "--- Waiting for deployment $deployment_name to be available (${seconds}s/${timeout}s)"
    sleep 5
    seconds=$((seconds + 5))
  done
  echo -e "--- ERROR: Deployment $deployment_name did not become ready within ${timeout}s"
  return 1
}

waitForPodsReady() {
  local label_selector=$1
  local namespace=${2:-kyma-system}
  local timeout=${3:-300}
  
  echo -e "\n--- Waiting for pods with selector '$label_selector' to be ready"
  local seconds=0
  while [[ $seconds -lt $timeout ]]; do
    local ready_pods=$(kubectl get pods -l "$label_selector" -n $namespace --field-selector=status.phase=Running -o name 2>/dev/null | wc -l)
    local total_pods=$(kubectl get pods -l "$label_selector" -n $namespace -o name 2>/dev/null | wc -l)
    
    if [[ $ready_pods -gt 0 && $ready_pods -eq $total_pods ]]; then
      echo -e "--- All pods ($ready_pods/$total_pods) with selector '$label_selector' are ready"
      return 0
    fi
    echo -e "--- Waiting for pods: $ready_pods/$total_pods ready (${seconds}s/${timeout}s)"
    sleep 5
    seconds=$((seconds + 5))
  done
  echo -e "--- ERROR: Pods with selector '$label_selector' did not become ready within ${timeout}s"
  return 1
}

checkNetworkPoliciesExist() {
  echo -e "\n--- Checking if network policies exist"
  local policies=(
    "kyma-project.io--btp-operator-allow-to-apiserver"
    "kyma-project.io--btp-operator-to-dns"
    "kyma-project.io--allow-btp-operator-metrics"
    "kyma-project.io--btp-operator-allow-to-webhook"
  )
  
  for policy in "${policies[@]}"; do
    if kubectl get networkpolicy "$policy" -n kyma-system >/dev/null 2>&1; then
      echo -e "--- ✓ Network policy '$policy' exists"
    else
      echo -e "--- ✗ Network policy '$policy' does not exist"
      return 1
    fi
  done
  echo -e "--- All network policies exist"
}

checkNetworkPoliciesDeleted() {
  echo -e "\n--- Checking if network policies are deleted"
  local policies=(
    "kyma-project.io--btp-operator-allow-to-apiserver"
    "kyma-project.io--btp-operator-to-dns"
    "kyma-project.io--allow-btp-operator-metrics"
    "kyma-project.io--btp-operator-allow-to-webhook"
  )
  
  for policy in "${policies[@]}"; do
    if kubectl get networkpolicy "$policy" -n kyma-system >/dev/null 2>&1; then
      echo -e "--- ✗ Network policy '$policy' still exists"
      return 1
    else
      echo -e "--- ✓ Network policy '$policy' does not exist"
    fi
  done
  echo -e "--- All network policies are deleted"
}

# Set unique names for test resources
K8S_VER=$(kubectl version -o json | jq .serverVersion.gitVersion -r | cut -d + -f 1)
SI_NAME=${GITHUB_JOB}-${K8S_VER}-${GITHUB_RUN_ID}
SB_NAME=${GITHUB_JOB}-${K8S_VER}-${GITHUB_RUN_ID}

export SI_NAME
export SB_NAME

echo -e "\n--- Starting Network Policies E2E Test"
echo -e "Image: $IMAGE_NAME"
echo -e "Credentials: $CREDENTIALS"
echo -e "Service Instance: $SI_NAME"
echo -e "Service Binding: $SB_NAME"

# Install the module
echo -e "\n--- Installing BTP Manager Module"

# Install prerequisites
echo -e "\n--- Installing prerequisites"
kubectl apply -f ./deployments/prerequisites.yaml

# Create secret based on credentials mode
if [[ "${CREDENTIALS}" == "real" ]]; then
  echo -e "\n--- Using real credentials"
  [ -n "${SM_CLIENT_ID:-}" ] && [ -n "${SM_CLIENT_SECRET:-}" ] && [ -n "${SM_URL:-}" ] && [ -n "${SM_TOKEN_URL:-}" ] || (echo "Missing credentials - failing test" && exit 1)
  envsubst <${YAML_DIR}/e2e-test-secret.yaml | kubectl apply -f -
else
  echo -e "\n--- Using dummy credentials"
  kubectl apply -f ${YAML_DIR}/e2e-test-configmap.yaml
  kubectl apply -f ./examples/btp-manager-secret.yaml
fi

# Deploy BTP Manager
echo -e "\n--- Deploying BTP Manager with image: ${IMAGE_NAME}"
IMG=${IMAGE_NAME} make deploy

# Wait for BTP Manager deployment
waitForDeploymentReady $BTP_MANAGER_DEPLOYMENT_NAME

# Install BTP Operator CR (with network policies disabled by default)
echo -e "\n--- Installing BTP Operator CR"
kubectl apply -f ./examples/btp-operator.yaml

# Wait for BTP Operator to be ready
waitForBtpOperatorCrReadiness

# Wait for SAP BTP Operator deployment
waitForDeploymentReady $SAP_BTP_OPERATOR_DEPLOYMENT_NAME

echo -e "\n--- Module installed successfully"

# Verify network policies are NOT present initially (since they're disabled by default)
echo -e "\n--- Verifying network policies are not present initially"
sleep 5
if checkNetworkPoliciesDeleted; then
  echo -e "--- ✓ Network policies correctly not present initially"
else
  echo -e "--- ✗ ERROR: Network policies should not exist initially but they do"
  exit 1
fi

# Enable network policies
echo -e "\n--- Enabling Network Policies"

echo -e "\n--- Applying deny-all NetworkPolicy"
kubectl apply -f - <<'EOF'
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kyma-project.io--deny-all-ingress
  namespace: kyma-system
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
EOF

echo -e "\n--- Network policies should be enabled by default, checking if they exist"
sleep 10

# Check if network policies exist (they should be created automatically by default)
checkNetworkPoliciesExist

echo -e "\n--- Network policies enabled"

# Delete BTP manager and SAP BTP operator pods
echo -e "\n--- Deleting Manager and Operator Pods"

echo -e "\n--- Deleting BTP Manager and SAP BTP Operator pods"
kubectl delete pods -l kyma-project.io/module=btp-operator -n kyma-system

echo -e "\n--- Pods deleted"

# Wait for pods to be ready
echo -e "\n--- Waiting for Pods to be Ready"
waitForPodsReady "kyma-project.io/module=btp-operator"

# Verify deployments are ready
waitForDeploymentReady $BTP_MANAGER_DEPLOYMENT_NAME
waitForDeploymentReady $SAP_BTP_OPERATOR_DEPLOYMENT_NAME

# Verify BTP Operator CR is still ready
waitForBtpOperatorCrReadiness

# Verify network policies still exist after pod restart
checkNetworkPoliciesExist

echo -e "\n--- All pods are ready"

# Create a service instance
echo -e "\n--- Creating Service Instance"

echo -e "\n--- Creating service instance: ${SI_NAME}"
envsubst <${YAML_DIR}/e2e-test-service-instance.yaml | kubectl apply -f -

if [[ "${CREDENTIALS}" == "real" ]]; then
  echo -e "\n--- Waiting for service instance to be ready with real credentials"
  while [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "False") != "True" ]]; do
    echo -e "--- Waiting for service instance to be ready..."
    sleep 5
  done
  echo -e "--- Service instance is ready"
else
  echo -e "\n--- Waiting for service instance to reach expected state with dummy credentials"
  while [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o json 2>/dev/null | jq -r '.status.conditions[] | select(.type=="Ready") |.status+.reason' 2>/dev/null || echo "") != "FalseNotProvisioned" ]] && \
        [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o json 2>/dev/null | jq -r '.status.conditions[] | select(.type=="Succeeded") |.reason' 2>/dev/null || echo "") != "CreateInProgress" ]]; do
    echo -e "--- Waiting for service instance to reach expected state..."
    sleep 5
  done
  echo -e "--- Service instance reached expected state (not ready due to dummy credentials)"
fi

echo -e "\n--- Service instance created"

echo -e "\n--- Disabling Network Policies"

echo -e "\n--- Disabling network policies in BTP Operator CR via annotation"
kubectl annotate btpoperators/btpoperator -n kyma-system operator.kyma-project.io/btp-operator-disable-network-policies=true

echo -e "\n--- Waiting for network policies to be cleaned up"
sleep 10

checkNetworkPoliciesDeleted

echo -e "\n--- Re-enabling network policies by removing the disable annotation"
kubectl annotate btpoperators/btpoperator -n kyma-system operator.kyma-project.io/btp-operator-disable-network-policies-

echo -e "\n--- Waiting for network policies to be recreated"
sleep 10

checkNetworkPoliciesCreated

echo -e "\n--- Removing deny-all NetworkPolicy"
kubectl delete networkpolicy kyma-project.io--deny-all-ingress -n kyma-system --ignore-not-found=true

echo -e "\n--- Network policies disabled and deny-all policy removed"