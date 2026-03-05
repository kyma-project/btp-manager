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

waitForBtpOperatorCrReadiness () {
  echo -e "\n--- Waiting for BtpOperator CR to be ready"
  while true; do
    operator_status=$(kubectl get btpoperators/btpoperator -n kyma-system -o json)
    state_status=$(echo $operator_status | jq -r '.status.state')
    if [[ $state_status == "Ready" ]]; then
      break
    else
      echo -e "\n--- Waiting for BtpOperator CR to be ready"; sleep 5;
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
    "kyma-project.io--allow-btp-operator-webhook"
  )
  
  for policy in "${policies[@]}"; do
    if kubectl get networkpolicy "$policy" -n kyma-system >/dev/null 2>&1; then
      echo -e "--- Network policy '$policy' exists"
    else
      echo -e "--- Network policy '$policy' does not exist"
      return 1
    fi
  done
  echo -e "--- All network policies exist"
  return 0
}

checkNetworkPoliciesDeleted() {
  echo -e "\n--- Checking if network policies are deleted"
  local policies=(
    "kyma-project.io--btp-operator-allow-to-apiserver"
    "kyma-project.io--btp-operator-to-dns"
    "kyma-project.io--allow-btp-operator-metrics"
    "kyma-project.io--allow-btp-operator-webhook"
  )
  
  for policy in "${policies[@]}"; do
    if kubectl get networkpolicy "$policy" -n kyma-system >/dev/null 2>&1; then
      echo -e "--- Network policy '$policy' still exists"
      return 1
    else
      echo -e "--- Network policy '$policy' does not exist"
    fi
  done
  echo -e "--- All network policies are deleted"
  return 0
}

checkContainerImages () {
  echo -e "\n--- Checking SAP BTP service operator container images and BTP Manager environment variables with container images"
  sap_btp_service_operator_images=( $(kubectl get deployment ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} -n kyma-system -o json | jq -r '.spec.template.spec.containers[].image') )
  btp_manager_envs_values=( $(kubectl get deployment ${BTP_MANAGER_DEPLOYMENT_NAME} -n kyma-system -o json | jq -r '.spec.template.spec.containers[].env[].value') )

  missing_elements=()
  for img in "${sap_btp_service_operator_images[@]}"; do
    found=false
    for env in "${btp_manager_envs_values[@]}"; do
      if [[ "$img" == "$env" ]]; then
        found=true
        break
      fi
    done
    if ! $found; then
      missing_elements+=("$img")
    fi
  done
  if [ ${#missing_elements[@]} -gt 0 ]; then
    echo -e "Missing container images in BTP Manager envs: ${missing_elements[@]}"
    exit 1
  else
    echo -e "BTP Manager envs include all SAP BTP service operator container images"
  fi
}

CREDENTIALS=$1
YAML_DIR="scripts/testing/yaml"
SAP_BTP_OPERATOR_DEPLOYMENT_NAME=sap-btp-operator-controller-manager
BTP_MANAGER_DEPLOYMENT_NAME=btp-manager-controller-manager

[[ -z ${GITHUB_RUN_ID} ]] && echo "required variable GITHUB_RUN_ID not set" && exit 1

checkContainerImages

echo -e "\n--- Verifying network policies are present initially"
if checkNetworkPoliciesExist; then
  echo -e "--- Network policies correctly present after module installation"
else
  echo -e "--- ERROR: Network policies should exist by default but they don't"
  exit 1
fi

echo -e "\n--- Applying deny-all NetworkPolicy to test connectivity"
kubectl apply -f - <<'EOF'
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kyma-project.io--deny-all-test
  namespace: kyma-system
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
EOF

echo -e "\n--- Testing pod restart with network policies enabled"
echo -e "\n--- Deleting BTP Manager and SAP BTP Operator pods"
kubectl delete pods -l kyma-project.io/module=btp-operator -n kyma-system

echo -e "\n--- Waiting for pods to be recreated and ready with network policies enabled"
waitForPodsReady "kyma-project.io/module=btp-operator"

echo -e "\n--- Waiting for deployments to be ready after pod restart"
waitForDeploymentReady $BTP_MANAGER_DEPLOYMENT_NAME
waitForDeploymentReady $SAP_BTP_OPERATOR_DEPLOYMENT_NAME

echo -e "\n--- Waiting for BtpOperator CR to be ready after pod restart"
waitForBtpOperatorCrReadiness

echo -e "\n--- Testing network policies disable/enable lifecycle"

echo -e "\n--- Disabling network policies via annotation"
kubectl annotate btpoperators/btpoperator -n kyma-system operator.kyma-project.io/btp-operator-disable-network-policies=true

echo -e "\n--- Waiting for network policies to be cleaned up"
sleep 10

if checkNetworkPoliciesDeleted; then
  echo -e "--- Network policies correctly deleted after disable annotation"
else
  echo -e "--- ERROR: Network policies should be deleted but they still exist"
  exit 1
fi

echo -e "\n--- Re-enabling network policies by removing the disable annotation"
kubectl annotate btpoperators/btpoperator -n kyma-system operator.kyma-project.io/btp-operator-disable-network-policies-

echo -e "\n--- Waiting for network policies to be recreated"
sleep 10

if checkNetworkPoliciesExist; then
  echo -e "--- Network policies correctly recreated after removing disable annotation"
else
  echo -e "--- ERROR: Network policies should be recreated but they don't exist"
  exit 1
fi

K8S_VER=$(kubectl version -o json | jq .serverVersion.gitVersion -r | cut -d + -f 1)

SI_NAME=${GITHUB_JOB}-${K8S_VER}-${GITHUB_RUN_ID}
SB_NAME=${GITHUB_JOB}-${K8S_VER}-${GITHUB_RUN_ID}
SI_PARAMS_SECRET_NAME=params-secret

export SI_NAME
export SB_NAME
export SI_PARAMS_SECRET_NAME

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

./scripts/testing/multiple_btpoperators.sh

echo -e "\n--- Patching ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment with non-existing image"
kubectl patch deployment ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} -n kyma-system --patch '{"spec": {"template": {"spec": {"containers": [{"name": "manager", "image": "non-existing-image:0.0.00000"}]}}}}'

echo -e "\n--- Deleting ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} pod"
kubectl delete pod -l app.kubernetes.io/name=sap-btp-operator -n kyma-system

echo -e "\n--- Waiting for ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment to be in error"
SECONDS=0
TIMEOUT=30
until [[ "$(kubectl get deployment ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} -n kyma-system -o json | jq -r '.status.conditions[] | select(.type=="Available") | .status')" == "False" ]] && \
  [[ "$(kubectl get deployment ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} -n kyma-system -o json | jq -r '.status.conditions[] | select(.type=="Progressing") | .status')" == "True" ]]; do
  echo -e "\n--- Waiting for ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment to be in error"
  if [[ ${SECONDS} -ge ${TIMEOUT} ]]; then
    echo "timed out after ${TIMEOUT}s" && exit 1
  fi
  sleep 5
done

echo -e "\n--- Waiting for ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment to be reconciled and ready"
SECONDS=0
TIMEOUT=30
until [[ "$(kubectl get deployment ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} -n kyma-system -o json | jq -r '.status.conditions[] | select(.type=="Available") | .status')" == "True" ]] && \
  [[ "$(kubectl get deployment ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} -n kyma-system -o json | jq -r '.status.conditions[] | select(.type=="Progressing") | .status')" == "True" ]]; do
  echo -e "\n--- Waiting for ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment to be reconciled and ready"
  if [[ ${SECONDS} -ge ${TIMEOUT} ]]; then
    echo "timed out after ${TIMEOUT}s" && exit 1
  fi
  sleep 5
done

waitForBtpOperatorCrReadiness

echo -e "\n--- Verifying network policies exist after deployment error/recovery"
if checkNetworkPoliciesExist; then
  echo -e "--- Network policies correctly persist through deployment error/recovery"
else
  echo -e "--- ERROR: Network policies should persist through deployment error/recovery"
  exit 1
fi

echo -e "\n--- Saving lastTransitionTime of BtpOperator CR"
last_transition_time=$(kubectl get btpoperators/btpoperator -n kyma-system -o json | jq -r '.status.conditions[] | select(.type=="Ready") | .lastTransitionTime')

echo -e "\n--- Creating sap-btp-manager configmap with ReadyTimeout 10s"
kubectl apply -f ${YAML_DIR}/e2e-test-configmap.yaml
kubectl patch configmap sap-btp-manager -n kyma-system --type merge -p '{"data":{"ReadyTimeout":"10s"}}'

echo -e "\n--- Deleting ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment to trigger reconciliation and change BtpOperator CR status"
kubectl delete deployment ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} -n kyma-system

echo -e "\n--- Waiting for BtpOperator CR lastTransitionTime to change"
while true; do
  operator_status=$(kubectl get btpoperators/btpoperator -n kyma-system -o json)
  state_status=$(echo $operator_status | jq -r '.status.state')
  current_last_transition_time=$(echo $operator_status | jq -r '.status.conditions[] | select(.type=="Ready") | .lastTransitionTime')
  if [[ $current_last_transition_time != $last_transition_time ]]; then
    echo -e "\n--- lastTransitionTime has changed"
    break
  else
    echo -e "\n--- Waiting for BtpOperator CR lastTransitionTime to change"; sleep 1;
  fi
done

waitForBtpOperatorCrReadiness

echo -e "\n--- Testing EnableLimitedCache ConfigMap propagation"

echo -e "\n--- Verifying default ENABLE_LIMITED_CACHE value (false) in sap-btp-operator-config ConfigMap"
operator_limited_cache_default=$(kubectl get configmap sap-btp-operator-config -n kyma-system -o jsonpath='{.data.ENABLE_LIMITED_CACHE}' 2>/dev/null || echo "")
echo -e "Current ENABLE_LIMITED_CACHE in sap-btp-operator-config: ${operator_limited_cache_default}"
if [[ "${operator_limited_cache_default}" != "false" ]]; then
  echo "Expected ENABLE_LIMITED_CACHE to be 'false' by default, but got: ${operator_limited_cache_default}" && exit 1
fi

echo -e "\n--- Enabling limited cache in sap-btp-manager ConfigMap"
kubectl patch configmap sap-btp-manager -n kyma-system --type merge -p '{"data":{"EnableLimitedCache":"true"}}'

echo -e "\n--- Verifying EnableLimitedCache=true was set in sap-btp-manager ConfigMap"
manager_limited_cache=$(kubectl get configmap sap-btp-manager -n kyma-system -o jsonpath='{.data.EnableLimitedCache}')
echo -e "sap-btp-manager ConfigMap EnableLimitedCache: ${manager_limited_cache}"

echo -e "\n--- Waiting for ENABLE_LIMITED_CACHE=true to propagate to sap-btp-operator-config ConfigMap"
SECONDS=0
TIMEOUT=60
while true; do
  operator_limited_cache=$(kubectl get configmap sap-btp-operator-config -n kyma-system -o jsonpath='{.data.ENABLE_LIMITED_CACHE}' 2>/dev/null || echo "")
  if [[ "${operator_limited_cache}" == "true" ]]; then
    echo -e "ENABLE_LIMITED_CACHE=true propagated to sap-btp-operator-config ConfigMap"
    break
  else
    if [[ ${SECONDS} -ge ${TIMEOUT} ]]; then
      echo "FAILED: ENABLE_LIMITED_CACHE did not propagate to 'true' in sap-btp-operator-config within ${TIMEOUT}s. Current value: ${operator_limited_cache}" && exit 1
    fi
    echo -e "--- Waiting for ENABLE_LIMITED_CACHE=true to propagate (current: ${operator_limited_cache}, elapsed: ${SECONDS}s)"
    sleep 5
    SECONDS=$((SECONDS + 5))
  fi
done

echo -e "\n--- Disabling limited cache in sap-btp-manager ConfigMap"
kubectl patch configmap sap-btp-manager -n kyma-system --type merge -p '{"data":{"EnableLimitedCache":"false"}}'

echo -e "\n--- Waiting for ENABLE_LIMITED_CACHE=false to propagate back to sap-btp-operator-config"
SECONDS=0
while true; do
  operator_limited_cache=$(kubectl get configmap sap-btp-operator-config -n kyma-system -o jsonpath='{.data.ENABLE_LIMITED_CACHE}' 2>/dev/null || echo "")
  if [[ "${operator_limited_cache}" == "false" ]]; then
    echo -e "ENABLE_LIMITED_CACHE=false propagated back to sap-btp-operator-config ConfigMap"
    break
  else
    if [[ ${SECONDS} -ge ${TIMEOUT} ]]; then
      echo "FAILED: ENABLE_LIMITED_CACHE did not propagate back to 'false' in sap-btp-operator-config within ${TIMEOUT}s. Current value: ${operator_limited_cache}" && exit 1
    fi
    echo -e "Waiting for ENABLE_LIMITED_CACHE=false to propagate (current: ${operator_limited_cache}, elapsed: ${SECONDS}s)"
    sleep 5
    SECONDS=$((SECONDS + 5))
  fi
done

echo -e "\n--- EnableLimitedCache ConfigMap propagation test completed successfully"

echo -e "\n--- Removing sap-btp-manager configmap"
kubectl delete -f ${YAML_DIR}/e2e-test-configmap.yaml

if [[ "${CREDENTIALS}" == "real" ]]; then
  echo -e "\n--- Checking Service Instance reconciliation after parameters change in the referenced Secret"

  echo -e "\n-- Applying Secret with parameters"
  envsubst <${YAML_DIR}/e2e-test-si-param-secret.yaml | kubectl apply -f -

  echo -e "\n Current parameters in the Secret: $(kubectl get secret ${SI_PARAMS_SECRET_NAME} -o jsonpath="{.data.key1}" | base64 -d)"

  echo -e "\n-- Patching Service Instance to get parameters from the Secret"
  kubectl patch serviceinstances.services.cloud.sap.com/${SI_NAME} --type='json' -p="[{\"op\": \"add\", \"path\": \"/spec/watchParametersFromChanges\", \"value\":true}, {\"op\": \"add\", \"path\": \"/spec/parametersFrom\", \"value\": [{\"secretKeyRef\": {\"name\": \"${SI_PARAMS_SECRET_NAME}\", \"key\": \"key1\" } }] }]"

  while [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o 'jsonpath={..status.conditions[?(@.reason=="Updated")].status}') != "True" ]];
  do echo -e "\n-- Waiting for Service Instance to be updated"; sleep 5; done

  echo -e "\n-- Service Instance has been updated"

  echo -e "\n-- Saving current resource version of the Service Instance"
  SI_RESOURCE_VER=$(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o jsonpath="{.metadata.resourceVersion}")

  echo -e "\n Current resource version: ${SI_RESOURCE_VER}"

  echo -e "\n-- Patching Secret with new parameters"
  PARAM=$(echo '{"new-param": "new-value"}' | base64)
  kubectl patch secret ${SI_PARAMS_SECRET_NAME} -p "{\"data\":{\"key1\":\"$PARAM\"}}"

  echo -e "\n Current parameters in the Secret: $(kubectl get secret ${SI_PARAMS_SECRET_NAME} -o jsonpath="{.data.key1}" | base64 -d)"

  while [[ $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o 'jsonpath={..status.conditions[?(@.reason=="Updated")].status}') != "True" && \
          $(kubectl get serviceinstances.services.cloud.sap.com/${SI_NAME} -o jsonpath="{.metadata.resourceVersion}") == "${SI_RESOURCE_VER}" ]];
  do echo -e "\n-- Waiting for Service Instance to be updated"; sleep 5; done

  echo -e "\n-- Service Instance has been updated - reconciliation after parameters change succeeded"
fi

echo -e "\n---Uninstalling..."

# remove btp-operator (ServiceInstance and ServiceBinding should be deleted as well)
kubectl delete btpoperators/btpoperator -n kyma-system  &

echo -e "\n--- Checking deprovisioning without force delete label"

while true; do
  operator_status=$(kubectl get btpoperators/btpoperator -n kyma-system -o json)
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

OLD_SAP_BTP_OPERATOR_DEPLOYMENT_UID=$(kubectl get -n kyma-system deployment/${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} -o jsonpath="{.metadata.uid}")

echo "Deleting ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment"
kubectl delete -n kyma-system deployment/${SAP_BTP_OPERATOR_DEPLOYMENT_NAME}

echo -e "\n--- Waiting for reconciliation (new ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment)"
SECONDS=0
TIMEOUT=120
until [[ "$(kubectl get -n kyma-system deployment/${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} 2>&1)" != *"Error from server (NotFound)"* ]] && \
  [[ $(kubectl get -n kyma-system deployment/${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} -o jsonpath="{.metadata.uid}") != "${OLD_SAP_BTP_OPERATOR_DEPLOYMENT_UID}" ]]
do
  echo "Waiting for new ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment to be created"
  if [[ ${SECONDS} -ge ${TIMEOUT} ]]; then
    echo "timed out after ${TIMEOUT}s" && exit 1
  fi
  sleep 5
done

echo -e "\n--- ${SAP_BTP_OPERATOR_DEPLOYMENT_NAME} deployment has been reconciled"

if [[ "${CREDENTIALS}" != "real" ]]
then
  echo -e "\n--- Creating sap-btp-manager configmap with HardDeleteTimeout 10s"
  kubectl apply -f ${YAML_DIR}/e2e-test-configmap.yaml
fi

echo -e "\n--- Adding force delete label"
kubectl label -f ${YAML_DIR}/e2e-test-btpoperator.yaml force-delete=true

echo -e "\n--- Checking deprovisioning with force delete label"

while [[ "$(kubectl get btpoperators/btpoperator -n kyma-system 2>&1)" != *"Error from server (NotFound)"* ]];
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
kubectl delete secret ${SI_PARAMS_SECRET_NAME} || echo "ignoring failure during params secret removal"
