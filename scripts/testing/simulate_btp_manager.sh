#!/usr/bin/env bash
# Simulates one btp-manager reconcile cycle for the CA bundle probe.
# Runs the probe Job, reads its annotations from BtpOperator CR, restarts
# sap-btp-operator if warranted, and advances tls-probe-last-hash.
#
# TODO: remove this script once btp-manager implements probe Job creation
# and annotation-driven reconciliation.
#
# Usage:
#   ./scripts/testing/simulate_btp_manager.sh           # single cycle
#   INTERVAL=10 ./scripts/testing/simulate_btp_manager.sh --loop  # repeat every 10s
#
# Environment variables (all optional, defaults match E2E test setup):
#   NAMESPACE            — Kubernetes namespace (default: kyma-system)
#   PROBE_IMAGE          — probe Job image (default: localhost:5000/ca-bundle-probe:latest)
#   FAKE_SERVER_URL      — token URL override for probe (default: https://fake-server.kyma-system.svc.cluster.local/health)
#   BTPOPERATOR_NAME     — BtpOperator CR name (default: btpoperator)
#   BTP_OPERATOR_DEPLOYMENT — sap-btp-operator Deployment name (default: sap-btp-operator-controller-manager)
#   INTERVAL             — seconds between cycles in --loop mode (default: 30)

set -o nounset
set -o errexit
set -E
set -o pipefail

NAMESPACE=${NAMESPACE:-"kyma-system"}
PROBE_IMAGE=${PROBE_IMAGE:-"localhost:5000/ca-bundle-probe:latest"}
FAKE_SERVER_URL=${FAKE_SERVER_URL:-"https://fake-server.${NAMESPACE}.svc.cluster.local/health"}
BTPOPERATOR_NAME=${BTPOPERATOR_NAME:-"btpoperator"}
BTP_OPERATOR_DEPLOYMENT=${BTP_OPERATOR_DEPLOYMENT:-"sap-btp-operator-controller-manager"}
INTERVAL=${INTERVAL:-30}
JOB_NAME="ca-bundle-probe-sim"

# ─── helpers ──────────────────────────────────────────────────────────────────

waitForJobCompletion() {
  local job=$1 timeout=${2:-120} seconds=0
  while [[ $seconds -lt $timeout ]]; do
    local status
    status=$(kubectl get job/"$job" -n "$NAMESPACE" \
      -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}' 2>/dev/null || echo "")
    [[ "$status" == "True" ]] && return 0
    local failed
    failed=$(kubectl get job/"$job" -n "$NAMESPACE" \
      -o jsonpath='{.status.conditions[?(@.type=="Failed")].status}' 2>/dev/null || echo "")
    if [[ "$failed" == "True" ]]; then
      echo "[sim] ERROR: probe job failed"
      kubectl logs job/"$job" -n "$NAMESPACE" || true
      return 1
    fi
    sleep 5; seconds=$((seconds + 5))
  done
  echo "[sim] ERROR: probe job timed out after ${timeout}s"
  kubectl logs job/"$job" -n "$NAMESPACE" || true
  return 1
}

runProbe() {
  kubectl delete job/"$JOB_NAME" -n "$NAMESPACE" --ignore-not-found 2>/dev/null
  kubectl create -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: $JOB_NAME
  namespace: $NAMESPACE
spec:
  template:
    spec:
      serviceAccountName: btp-manager-ca-bundle-probe
      restartPolicy: Never
      containers:
        - name: probe
          image: $PROBE_IMAGE
          env:
            - name: PROBE_NAMESPACE
              value: "$NAMESPACE"
            - name: PROBE_TOKENURL_OVERRIDE
              value: "$FAKE_SERVER_URL"
EOF
  waitForJobCompletion "$JOB_NAME"
  echo "[sim] probe output:"
  kubectl logs job/"$JOB_NAME" -n "$NAMESPACE" 2>/dev/null | sed 's/^/  [probe] /' || true
}

reconcile() {
  local status hash lastHash
  status=$(kubectl get btpoperator/"$BTPOPERATOR_NAME" -n "$NAMESPACE" \
    -o jsonpath='{.metadata.annotations.tls-probe-status}' 2>/dev/null || echo "")
  hash=$(kubectl get btpoperator/"$BTPOPERATOR_NAME" -n "$NAMESPACE" \
    -o jsonpath='{.metadata.annotations.tls-probe-hash}' 2>/dev/null || echo "")
  lastHash=$(kubectl get btpoperator/"$BTPOPERATOR_NAME" -n "$NAMESPACE" \
    -o jsonpath='{.metadata.annotations.tls-probe-last-hash}' 2>/dev/null || echo "")

  echo "[sim] status='$status' hash=${hash:0:16}... lastHash=${lastHash:0:16}..."

  # Restart sap-btp-operator when: TLS ok, mount present, hash changed from last known value.
  # Empty lastHash means first probe run (initialization) — no restart.
  if [[ "$status" == "ok" && -n "$hash" && -n "$lastHash" && "$hash" != "$lastHash" ]]; then
    echo "[sim] hash changed, TLS ok — restarting $BTP_OPERATOR_DEPLOYMENT"
    kubectl delete pod -n "$NAMESPACE" -l app.kubernetes.io/instance=sap-btp-operator \
      --wait=false 2>/dev/null || true
  fi

  # Advance last-hash only when probe wrote a non-empty hash (matches probe_runner.go behaviour)
  if [[ -n "$hash" ]]; then
    kubectl annotate btpoperator/"$BTPOPERATOR_NAME" -n "$NAMESPACE" \
      --overwrite "tls-probe-last-hash=$hash" 2>/dev/null
  fi
}

runCycle() {
  echo "[sim] --- cycle start $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  runProbe
  reconcile
  echo "[sim] --- cycle done"
}

# ─── main ─────────────────────────────────────────────────────────────────────

if [[ "${1:-}" == "--loop" ]]; then
  echo "[sim] running in loop mode (interval=${INTERVAL}s); Ctrl-C to stop"
  while true; do
    runCycle
    sleep "$INTERVAL"
  done
else
  runCycle
fi
