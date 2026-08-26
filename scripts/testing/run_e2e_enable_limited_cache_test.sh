#!/usr/bin/env bash
# E2E test for EnableLimitedCache config change triggering SAP BTP Service Operator pod restart.
# Preconditions: k3d cluster running, btp-manager + btp-operator installed via install_module.sh
# Usage: ./run_e2e_enable_limited_cache_test.sh

set -o nounset
set -o errexit
set -E
set -o pipefail

NAMESPACE="${NAMESPACE:-kyma-system}"
BTP_MANAGER_CONFIG_MAP="sap-btp-manager"
SAP_BTP_OPERATOR_CONFIG_MAP="sap-btp-operator-config"
ENABLE_LIMITED_CACHE_KEY="ENABLE_LIMITED_CACHE"

# ─── helpers ─────────────────────────────────────────────────────────────────

assertSapBtpOperatorRestarted() {
  local before_pod=$1 timeout=60 seconds=0
  echo "--- Waiting for sap-btp-operator pod to change from '$before_pod'"
  while [[ $seconds -lt $timeout ]]; do
    local current_pod
    current_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/instance=sap-btp-operator \
      --field-selector=status.phase=Running \
      --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || echo "")
    if [[ -n "$current_pod" && "$current_pod" != "$before_pod" ]]; then
      echo "--- PASS: sap-btp-operator restarted (new pod: '$current_pod')"
      return 0
    fi
    sleep 5; seconds=$((seconds + 5))
  done
  echo "--- FAIL: sap-btp-operator not restarted within ${timeout}s (pod unchanged: '$before_pod')"
  return 1
}

getSapBtpOperatorPodName() {
  kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/instance=sap-btp-operator \
    --field-selector=status.phase=Running \
    --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || echo ""
}

getEnableLimitedCacheValue() {
  kubectl get cm "$SAP_BTP_OPERATOR_CONFIG_MAP" -n "$NAMESPACE" \
    -o jsonpath="{.data.${ENABLE_LIMITED_CACHE_KEY}}" 2>/dev/null || echo ""
}

assertEnableLimitedCacheValue() {
  local expected=$1 timeout=30 seconds=0
  echo "--- Waiting for $ENABLE_LIMITED_CACHE_KEY=$expected in $SAP_BTP_OPERATOR_CONFIG_MAP"
  while [[ $seconds -lt $timeout ]]; do
    local actual
    actual=$(getEnableLimitedCacheValue)
    if [[ "$actual" == "$expected" ]]; then
      echo "--- PASS: $ENABLE_LIMITED_CACHE_KEY=$actual in $SAP_BTP_OPERATOR_CONFIG_MAP"
      return 0
    fi
    sleep 3; seconds=$((seconds + 3))
  done
  echo "--- FAIL: expected $ENABLE_LIMITED_CACHE_KEY=$expected in $SAP_BTP_OPERATOR_CONFIG_MAP, got '$(getEnableLimitedCacheValue)'"
  return 1
}

# ─── test ─────────────────────────────────────────────────────────────────────

echo "════════════════════════════════════════════════════"
echo "║  E2E: EnableLimitedCache change triggers restart  ║"
echo "════════════════════════════════════════════════════"

BEFORE_POD=$(getSapBtpOperatorPodName)
if [[ -z "$BEFORE_POD" ]]; then
  echo "--- FAIL: no running sap-btp-operator pod found in $NAMESPACE"
  exit 1
fi
echo "--- Initial pod: '$BEFORE_POD'"

echo ""
echo "══  Step 1: change EnableLimitedCache to false  ══"
kubectl patch cm "$BTP_MANAGER_CONFIG_MAP" -n "$NAMESPACE" \
  --type=merge -p '{"data":{"EnableLimitedCache":"false"}}'

assertSapBtpOperatorRestarted "$BEFORE_POD"
AFTER_POD=$(getSapBtpOperatorPodName)

assertEnableLimitedCacheValue "false"

echo ""
echo "══  Step 2: restore EnableLimitedCache to true  ══"
kubectl patch cm "$BTP_MANAGER_CONFIG_MAP" -n "$NAMESPACE" \
  --type=merge -p '{"data":{"EnableLimitedCache":"true"}}'

assertSapBtpOperatorRestarted "$AFTER_POD"

assertEnableLimitedCacheValue "true"

echo ""
echo "--- PASS: EnableLimitedCache change triggers pod restart"
