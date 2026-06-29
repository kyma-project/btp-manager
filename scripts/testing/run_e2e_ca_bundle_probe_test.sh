#!/usr/bin/env bash
# E2E test for CA bundle probe v2.
# Preconditions: k3d cluster running, btp-manager + btp-operator installed via install_module.sh
# Usage: ./run_e2e_ca_bundle_probe_test.sh
#
# Requires simulate_btp_manager.sh --loop running in a separate terminal:
#   INTERVAL=5 ./scripts/testing/simulate_btp_manager.sh --loop
#
# Environment variables:
#   PROBE_IMAGE       — probe Job image (default: localhost:5000/ca-bundle-probe:latest)
#   FAKE_SERVER_IMAGE — fake HTTPS server image (default: localhost:5000/fake-server:latest)
#   WEBHOOK_IMAGE     — fake webhook image (default: localhost:5000/ca-bundle-webhook:latest)

set -o nounset
set -o errexit
set -E
set -o pipefail

NAMESPACE="kyma-system"
PROBE_IMAGE=${PROBE_IMAGE:-"localhost:5000/ca-bundle-probe:latest"}
FAKE_SERVER_IMAGE=${FAKE_SERVER_IMAGE:-"localhost:5000/fake-server:latest"}
WEBHOOK_IMAGE=${WEBHOOK_IMAGE:-"localhost:5000/ca-bundle-webhook:latest"}
FAKE_SERVER_URL="https://fake-server.${NAMESPACE}.svc.cluster.local/health"
BTPOPERATOR_NAME="btpoperator"
BTP_MANAGER_DEPLOYMENT="btp-manager-controller-manager"
BTP_OPERATOR_DEPLOYMENT="sap-btp-operator-controller-manager"
CERTS_DIR="/tmp/ca-bundle-probe-certs"
FORCE_REGEN=${FORCE_REGEN:-"false"}

# ─── helpers ─────────────────────────────────────────────────────────────────

assertCRAnnotation() {
  local key=$1 expected=$2
  local actual
  actual=$(kubectl get btpoperator/"$BTPOPERATOR_NAME" -n "$NAMESPACE" \
    -o jsonpath="{.metadata.annotations.$key}" 2>/dev/null || echo "")
  if [[ "$actual" != "$expected" ]]; then
    echo "--- FAIL: annotation $key: expected='$expected' got='$actual'"
    return 1
  fi
  echo "--- PASS: annotation $key=$expected"
}

assertBtpOperatorRestarted() {
  local before_pod=$1 timeout=60 seconds=0
  echo "--- Waiting for btp-operator pod to change from '$before_pod'"
  while [[ $seconds -lt $timeout ]]; do
    local current_pod
    current_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/instance=sap-btp-operator \
      --field-selector=status.phase=Running \
      --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || echo "")
    if [[ -n "$current_pod" && "$current_pod" != "$before_pod" ]]; then
      echo "--- PASS: btp-operator restarted (new pod: '$current_pod')"
      return 0
    fi
    sleep 5; seconds=$((seconds + 5))
  done
  echo "--- FAIL: btp-operator not restarted within ${timeout}s (pod unchanged: '$before_pod')"
  return 1
}

assertBtpOperatorNotRestarted() {
  local before_pod=$1
  local current_pod
  current_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/instance=sap-btp-operator \
    --field-selector=status.phase=Running \
    --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || echo "")
  if [[ "$current_pod" == "$before_pod" ]]; then
    echo "--- PASS: btp-operator not restarted"
    return 0
  fi
  echo "--- FAIL: btp-operator was restarted unexpectedly (new pod: '$current_pod')"
  return 1
}

getBtpOperatorRestartedAt() {
  kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/instance=sap-btp-operator \
    --field-selector=status.phase=Running \
    --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || echo ""
}

getRunId() {
  local val
  val=$(kubectl get btpoperator/"$BTPOPERATOR_NAME" -n "$NAMESPACE" \
    -o jsonpath='{.metadata.annotations.tls-probe-run-id}' 2>/dev/null || echo "")
  echo "${val:-0}"
}

waitForNextCycle() {
  local beforeRunId=$1 timeout=${2:-180} seconds=0
  echo "--- Waiting for simulate loop cycle (run-id currently $beforeRunId)"
  while [[ $seconds -lt $timeout ]]; do
    local current
    current=$(getRunId)
    if [[ "$current" -gt "$beforeRunId" ]]; then
      echo "--- Simulate loop cycle complete (run-id=$current)"
      return 0
    fi
    sleep 5; seconds=$((seconds + 5))
  done
  echo "--- ERROR: timed out waiting for simulate loop (run-id stuck at $beforeRunId after ${timeout}s)"
  echo "--- Is 'INTERVAL=5 ./scripts/testing/simulate_btp_manager.sh --loop' running?"
  return 1
}

waitForDeploymentReady() {
  local dep=$1 timeout=${2:-120}
  kubectl rollout status deployment/"$dep" -n "$NAMESPACE" --timeout="${timeout}s"
}

# ─── setup ────────────────────────────────────────────────────────────────────

echo "=== SETUP ==="
echo "--- NOTE: requires simulate_btp_manager.sh --loop in a separate terminal:"
echo "---   INTERVAL=5 PROBE_IMAGE=$PROBE_IMAGE FAKE_SERVER_URL=$FAKE_SERVER_URL \\"
echo "---     ./scripts/testing/simulate_btp_manager.sh --loop"

# Clean up any leftover state from a previous run
kubectl delete secret/ca-bundle -n "$NAMESPACE" --ignore-not-found
kubectl delete mutatingwebhookconfiguration/ca-bundle-webhook --ignore-not-found

kubectl apply -f tests/webhook/manifests/rbac.yaml
kubectl apply -f tests/webhook/manifests/service.yaml
sed "s|WEBHOOK_IMAGE|$WEBHOOK_IMAGE|g" tests/webhook/manifests/deployment.yaml | kubectl apply -f -
sed "s|FAKE_SERVER_IMAGE|$FAKE_SERVER_IMAGE|g" tests/fake-server/manifests/deployment.yaml | kubectl apply -f -
kubectl apply -f tests/fake-server/manifests/service.yaml
kubectl apply -f tests/probe/manifests/rbac.yaml
kubectl apply -f tests/webhook/manifests/mutatingwebhookconfiguration.yaml

if [[ "$FORCE_REGEN" == "true" || ! -f "$CERTS_DIR/ca-A.crt" || ! -f "$CERTS_DIR/ca-bundle-webhook.key" ]]; then
  echo "--- Generating certs (FORCE_REGEN=$FORCE_REGEN)"
  ./tests/probe/setup-certs.sh "$NAMESPACE"
  kubectl rollout restart deployment/ca-bundle-webhook -n "$NAMESPACE"
  kubectl rollout restart deployment/fake-server -n "$NAMESPACE"
  waitForDeploymentReady ca-bundle-webhook
  waitForDeploymentReady fake-server
else
  echo "--- Reusing cached certs from $CERTS_DIR (set FORCE_REGEN=true to regenerate)"
  # Recreate webhook TLS secret from cached cert (deleted during teardown)
  kubectl create secret tls ca-bundle-webhook-tls \
    --cert="$CERTS_DIR/ca-bundle-webhook.crt" --key="$CERTS_DIR/ca-bundle-webhook.key" \
    --namespace="$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
  # Patch MWC caBundle from the cached webhook cert so API server can verify the webhook TLS
  CA_BUNDLE_B64=$(base64 -w0 < "$CERTS_DIR/ca-bundle-webhook.crt")
  kubectl patch mutatingwebhookconfiguration ca-bundle-webhook \
    --type=json -p="[{\"op\":\"replace\",\"path\":\"/webhooks/0/clientConfig/caBundle\",\"value\":\"${CA_BUNDLE_B64}\"}]"
  # Restore fake-server to server-A baseline in case a previous run left it on server-B
  TLS_CRT_B64=$(base64 -w0 < "$CERTS_DIR/server-A.crt")
  TLS_KEY_B64=$(base64 -w0 < "$CERTS_DIR/server-A.key")
  TLS_CRT_B64=$TLS_CRT_B64 TLS_KEY_B64=$TLS_KEY_B64 \
    envsubst < scripts/testing/yaml/ca-bundle-probe/fake-server-tls-secret.yaml | kubectl apply -f -
  kubectl rollout restart deployment/fake-server -n "$NAMESPACE"
  waitForDeploymentReady ca-bundle-webhook
  waitForDeploymentReady fake-server
fi

# Ensure btp-manager has no stale rt-bootstrapper-certs mount from a previous run
kubectl rollout restart deployment/"$BTP_MANAGER_DEPLOYMENT" -n "$NAMESPACE"
waitForDeploymentReady "$BTP_MANAGER_DEPLOYMENT"

# Flush any in-flight simulate loop cycle (probe may have run during setup against a
# partially-ready environment). Wait for one clean cycle to complete before testing.
echo "--- Flushing setup-time simulate loop cycles"
waitForNextCycle "$(getRunId)"
echo "--- Setup complete, environment ready"

# ─── phase A: baseline, no mount, distroless system CA ──────────────────────
# The probe runs in a distroless container whose x509.SystemCertPool() reads the CA bundle
# baked into the image layer. That bundle contains real root CAs and will never trust our
# self-signed fake-server cert. So: mount=false, tls=failed-x509, signal=warning — always.
# This tests that the mount detection and signal logic work correctly when no volume is injected.

echo "=== PHASE A: no mount, distroless system CA doesn't trust self-signed fake-server ==="
BEFORE_RUN_ID=$(getRunId)
BEFORE_RESTART_AT=$(getBtpOperatorRestartedAt)
waitForNextCycle "$BEFORE_RUN_ID"
assertCRAnnotation "tls-probe-mount" "false"
assertCRAnnotation "tls-probe-tls-result" "failed-x509"
assertCRAnnotation "tls-probe-signal" "warning"
assertBtpOperatorNotRestarted "$BEFORE_RESTART_AT"

# ─── phase B: mount injected (customCA1 = ca-A) ──────────────────────────────

echo "=== PHASE B: mount injected (customCA1 = ca-A) ==="
echo "--- Step 3 (row 3): mount=customCA1, TLS ok — first hash written, no restart"
BEFORE_RUN_ID=$(getRunId)
BEFORE_RESTART_AT=$(getBtpOperatorRestartedAt)
CA_A_B64=$(base64 -w0 < "$CERTS_DIR/ca-A.crt")
CA_BUNDLE_B64=$CA_A_B64 envsubst < scripts/testing/yaml/ca-bundle-probe/ca-bundle-secret.yaml | kubectl apply -f -

# Restart btp-manager so webhook fires and injects rt-bootstrapper-certs mount
kubectl rollout restart deployment/"$BTP_MANAGER_DEPLOYMENT" -n "$NAMESPACE"
waitForDeploymentReady "$BTP_MANAGER_DEPLOYMENT"

waitForNextCycle "$BEFORE_RUN_ID"
assertCRAnnotation "tls-probe-mount" "true"
assertCRAnnotation "tls-probe-tls-result" "ok"
assertCRAnnotation "tls-probe-signal" ""
assertBtpOperatorNotRestarted "$BEFORE_RESTART_AT"

echo "--- Step 4 (row 5): mount=customCA1, server cert signed by ca-B → TLS x509"
BEFORE_RUN_ID=$(getRunId)
BEFORE_RESTART_AT=$(getBtpOperatorRestartedAt)
TLS_CRT_B64=$(base64 -w0 < "$CERTS_DIR/server-B.crt")
TLS_KEY_B64=$(base64 -w0 < "$CERTS_DIR/server-B.key")
TLS_CRT_B64=$TLS_CRT_B64 TLS_KEY_B64=$TLS_KEY_B64 \
  envsubst < scripts/testing/yaml/ca-bundle-probe/fake-server-tls-secret.yaml | kubectl apply -f -
kubectl rollout restart deployment/fake-server -n "$NAMESPACE"
waitForDeploymentReady fake-server

waitForNextCycle "$BEFORE_RUN_ID"
assertCRAnnotation "tls-probe-mount" "true"
assertCRAnnotation "tls-probe-tls-result" "failed-x509"
assertCRAnnotation "tls-probe-signal" "alert"
assertBtpOperatorNotRestarted "$BEFORE_RESTART_AT"

# ─── phase C: bundle rotation (customCA2 = ca-B), TLS ok ─────────────────────

echo "=== PHASE C: bundle rotation to customCA2 (ca-B), TLS ok ==="
echo "--- Step 5 (row 4): mount=customCA2, hash changed, TLS ok → btp-operator restarted"
BEFORE_RUN_ID=$(getRunId)
BEFORE_RESTART_AT=$(getBtpOperatorRestartedAt)
CA_B_B64=$(base64 -w0 < "$CERTS_DIR/ca-B.crt")
CA_BUNDLE_B64=$CA_B_B64 envsubst < scripts/testing/yaml/ca-bundle-probe/ca-bundle-secret.yaml | kubectl apply -f -

kubectl rollout restart deployment/"$BTP_MANAGER_DEPLOYMENT" -n "$NAMESPACE"
waitForDeploymentReady "$BTP_MANAGER_DEPLOYMENT"

waitForNextCycle "$BEFORE_RUN_ID"
assertCRAnnotation "tls-probe-mount" "true"
assertCRAnnotation "tls-probe-tls-result" "ok"
assertCRAnnotation "tls-probe-signal" ""
assertBtpOperatorRestarted "$BEFORE_RESTART_AT"

# ─── phase D: new bundle doesn't trust server ────────────────────────────────

echo "=== PHASE D: new bundle (ca-B2) that doesn't trust server (signed by ca-A) ==="
echo "--- Step 6 (row 6): mount=customCA2-bad, hash changed, TLS x509 → alert, no restart"
BEFORE_RUN_ID=$(getRunId)
BEFORE_RESTART_AT=$(getBtpOperatorRestartedAt)
TLS_CRT_B64=$(base64 -w0 < "$CERTS_DIR/server-A.crt")
TLS_KEY_B64=$(base64 -w0 < "$CERTS_DIR/server-A.key")
TLS_CRT_B64=$TLS_CRT_B64 TLS_KEY_B64=$TLS_KEY_B64 \
  envsubst < scripts/testing/yaml/ca-bundle-probe/fake-server-tls-secret.yaml | kubectl apply -f -
kubectl rollout restart deployment/fake-server -n "$NAMESPACE"
waitForDeploymentReady fake-server

openssl req -x509 -newkey rsa:2048 -keyout /tmp/ca-B2.key -out /tmp/ca-B2.crt \
  -days 365 -nodes -subj "/CN=test-ca-B2" 2>/dev/null
CA_B2_B64=$(base64 -w0 < /tmp/ca-B2.crt)
CA_BUNDLE_B64=$CA_B2_B64 envsubst < scripts/testing/yaml/ca-bundle-probe/ca-bundle-secret.yaml | kubectl apply -f -

kubectl rollout restart deployment/"$BTP_MANAGER_DEPLOYMENT" -n "$NAMESPACE"
waitForDeploymentReady "$BTP_MANAGER_DEPLOYMENT"

waitForNextCycle "$BEFORE_RUN_ID"
assertCRAnnotation "tls-probe-mount" "true"
assertCRAnnotation "tls-probe-tls-result" "failed-x509"
assertCRAnnotation "tls-probe-signal" "alert"
assertBtpOperatorNotRestarted "$BEFORE_RESTART_AT"

# ─── phase E: connectivity failure ───────────────────────────────────────────

echo "=== PHASE E: connectivity failure ==="
echo "--- Step 7 (row 7): server unreachable → warning, no restart"
kubectl scale deployment/fake-server -n "$NAMESPACE" --replicas=0
# Wait for fake-server pod to terminate before recording BEFORE_RUN_ID
kubectl wait --for=delete pod -l app=fake-server -n "$NAMESPACE" --timeout=60s 2>/dev/null || true
BEFORE_RUN_ID=$(getRunId)
BEFORE_RESTART_AT=$(getBtpOperatorRestartedAt)

waitForNextCycle "$BEFORE_RUN_ID"
assertCRAnnotation "tls-probe-signal" "warning"
assertBtpOperatorNotRestarted "$BEFORE_RESTART_AT"

kubectl scale deployment/fake-server -n "$NAMESPACE" --replicas=1
waitForDeploymentReady fake-server

# ─── teardown ─────────────────────────────────────────────────────────────────

echo "=== TEARDOWN ==="
kubectl delete deployment/ca-bundle-webhook deployment/fake-server -n "$NAMESPACE" --ignore-not-found
kubectl delete service/ca-bundle-webhook service/fake-server -n "$NAMESPACE" --ignore-not-found
kubectl delete mutatingwebhookconfiguration/ca-bundle-webhook --ignore-not-found
kubectl delete secret/ca-bundle secret/fake-server-tls secret/ca-bundle-webhook-tls -n "$NAMESPACE" --ignore-not-found
kubectl delete clusterrole/ca-bundle-probe clusterrole/ca-bundle-webhook --ignore-not-found
kubectl delete clusterrolebinding/ca-bundle-probe clusterrolebinding/ca-bundle-webhook --ignore-not-found
kubectl delete serviceaccount/ca-bundle-probe serviceaccount/ca-bundle-webhook -n "$NAMESPACE" --ignore-not-found

echo "=== ALL TESTS PASSED ==="
