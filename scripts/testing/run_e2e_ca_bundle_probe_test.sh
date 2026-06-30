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

printAnnotations() {
  echo "--- BtpOperator CR annotations:"
  kubectl get btpoperator/"$BTPOPERATOR_NAME" -n "$NAMESPACE" \
    -o jsonpath='{.metadata.annotations}' 2>/dev/null \
    | tr ',' '\n' | tr -d '{}' | grep tls-probe | sed 's/^/      /'
  echo ""
}

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
  local before_pod=$1 window=${2:-20} seconds=0
  echo "--- Watching btp-operator pod for ${window}s (expecting no restart)"
  while [[ $seconds -lt $window ]]; do
    local current_pod
    current_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/instance=sap-btp-operator \
      --field-selector=status.phase=Running \
      --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || echo "")
    if [[ -n "$current_pod" && "$current_pod" != "$before_pod" ]]; then
      echo "--- FAIL: btp-operator was restarted unexpectedly (new pod: '$current_pod')"
      return 1
    fi
    sleep 5; seconds=$((seconds + 5))
  done
  echo "--- PASS: btp-operator not restarted within ${window}s"
}

getBtpOperatorPodName() {
  kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/instance=sap-btp-operator \
    --field-selector=status.phase=Running \
    --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || echo ""
}

getUpdatedAt() {
  kubectl get btpoperator/"$BTPOPERATOR_NAME" -n "$NAMESPACE" \
    -o jsonpath='{.metadata.annotations.tls-probe-updated-at}' 2>/dev/null || echo ""
}

waitForNextCycle() {
  local beforeTimestamp=$1 timeout=${2:-180} seconds=0
  echo "--- Waiting for simulate loop cycle (last updated-at: ${beforeTimestamp:-<none>})"
  while [[ $seconds -lt $timeout ]]; do
    local current
    current=$(getUpdatedAt)
    if [[ -n "$current" && "$current" > "$beforeTimestamp" ]]; then
      echo "--- Simulate loop cycle complete (updated-at=$current)"
      return 0
    fi
    sleep 5; seconds=$((seconds + 5))
  done
  echo "--- ERROR: timed out waiting for simulate loop (updated-at stuck at '${beforeTimestamp}' after ${timeout}s)"
  echo "--- Is 'INTERVAL=5 ./scripts/testing/simulate_btp_manager.sh --loop' running?"
  return 1
}

waitForDeploymentReady() {
  local dep=$1 timeout=${2:-120}
  kubectl rollout status deployment/"$dep" -n "$NAMESPACE" --timeout="${timeout}s"
}

# ─── setup ────────────────────────────────────────────────────────────────────

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  CA BUNDLE PROBE E2E TEST                                        ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
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
waitForNextCycle "$(getUpdatedAt)"
echo "--- Setup complete, environment ready"

# ─── phase A: baseline, no mount, distroless system CA ──────────────────────
# The probe runs in a distroless container whose x509.SystemCertPool() reads the CA bundle
# baked into the image layer. That bundle contains real root CAs and will never trust our
# self-signed fake-server cert. So: mount=false, tls=failed-x509, signal=error — always.
# This tests that the mount detection and signal logic work correctly when no volume is injected.

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  PHASE A: no mount — distroless system CA                        ║"
echo "║  Expected: mount=false, tls=failed-x509, signal=error            ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo "--- World state: no ca-bundle Secret, webhook not injecting mount"
BEFORE_UPDATED_AT=$(getUpdatedAt)
BEFORE_POD=$(getBtpOperatorPodName)
waitForNextCycle "$BEFORE_UPDATED_AT"
assertCRAnnotation "tls-probe-status" "error"
assertBtpOperatorNotRestarted "$BEFORE_POD"
printAnnotations

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  PHASE B step 1: mount injected (customCA1=ca-A), TLS ok         ║"
echo "║  Expected: status=ok, no restart                                 ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
BEFORE_POD=$(getBtpOperatorPodName)
echo "--- World state: applying ca-bundle Secret with ca-A (webhook will inject mount)"
CA_A_B64=$(base64 -w0 < "$CERTS_DIR/ca-A.crt")
CA_BUNDLE_B64=$CA_A_B64 envsubst < scripts/testing/yaml/ca-bundle-probe/ca-bundle-secret.yaml | kubectl apply -f -
# Capture timestamp AFTER Secret is applied so the next cycle probes with the mount already injected
BEFORE_UPDATED_AT=$(getUpdatedAt)

waitForNextCycle "$BEFORE_UPDATED_AT"
assertCRAnnotation "tls-probe-status" "ok"
assertBtpOperatorNotRestarted "$BEFORE_POD"
printAnnotations

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  PHASE B step 2: customCA1 mounted, server cert rotated to ca-B  ║"
echo "║  Expected: status=alert, no restart                              ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo "--- World state: rotating fake-server TLS cert to one signed by ca-B (not trusted by ca-A)"
BEFORE_POD=$(getBtpOperatorPodName)
TLS_CRT_B64=$(base64 -w0 < "$CERTS_DIR/server-B.crt")
TLS_KEY_B64=$(base64 -w0 < "$CERTS_DIR/server-B.key")
TLS_CRT_B64=$TLS_CRT_B64 TLS_KEY_B64=$TLS_KEY_B64 \
  envsubst < scripts/testing/yaml/ca-bundle-probe/fake-server-tls-secret.yaml | kubectl apply -f -
kubectl rollout restart deployment/fake-server -n "$NAMESPACE"
waitForDeploymentReady fake-server
# Capture timestamp AFTER server is ready so the next cycle definitely probes the new server
BEFORE_UPDATED_AT=$(getUpdatedAt)

waitForNextCycle "$BEFORE_UPDATED_AT"
assertCRAnnotation "tls-probe-status" "alert"
assertBtpOperatorNotRestarted "$BEFORE_POD"
printAnnotations

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  PHASE C: CA bundle rotated to customCA2 (ca-B), TLS ok          ║"
echo "║  Expected: status=ok, btp-operator RESTARTED                     ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo "--- World state: updating ca-bundle Secret to ca-B (matches server cert)"
BEFORE_UPDATED_AT=$(getUpdatedAt)
BEFORE_POD=$(getBtpOperatorPodName)
CA_B_B64=$(base64 -w0 < "$CERTS_DIR/ca-B.crt")
CA_BUNDLE_B64=$CA_B_B64 envsubst < scripts/testing/yaml/ca-bundle-probe/ca-bundle-secret.yaml | kubectl apply -f -

waitForNextCycle "$BEFORE_UPDATED_AT"
assertCRAnnotation "tls-probe-status" "ok"
assertBtpOperatorRestarted "$BEFORE_POD"
printAnnotations

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  PHASE D: new bundle (ca-B2) that doesn't trust server           ║"
echo "║  Expected: status=alert, no restart                              ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo "--- World state: rotating server cert back to ca-A, deploying unrelated ca-B2 bundle"
BEFORE_POD=$(getBtpOperatorPodName)
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
# Capture timestamp AFTER server is ready and bundle is updated
BEFORE_UPDATED_AT=$(getUpdatedAt)

waitForNextCycle "$BEFORE_UPDATED_AT"
assertCRAnnotation "tls-probe-status" "alert"
assertBtpOperatorNotRestarted "$BEFORE_POD"
printAnnotations

# ─── teardown ─────────────────────────────────────────────────────────────────

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  TEARDOWN                                                        ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
kubectl delete deployment/ca-bundle-webhook deployment/fake-server -n "$NAMESPACE" --ignore-not-found
kubectl delete service/ca-bundle-webhook service/fake-server -n "$NAMESPACE" --ignore-not-found
kubectl delete mutatingwebhookconfiguration/ca-bundle-webhook --ignore-not-found
kubectl delete secret/ca-bundle secret/fake-server-tls secret/ca-bundle-webhook-tls -n "$NAMESPACE" --ignore-not-found
kubectl delete clusterrole/ca-bundle-probe clusterrole/ca-bundle-webhook --ignore-not-found
kubectl delete clusterrolebinding/ca-bundle-probe clusterrolebinding/ca-bundle-webhook --ignore-not-found
kubectl delete serviceaccount/ca-bundle-probe serviceaccount/ca-bundle-webhook -n "$NAMESPACE" --ignore-not-found

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  ALL TESTS PASSED                                                ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
