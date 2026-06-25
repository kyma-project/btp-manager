#!/usr/bin/env bash
# E2E test for CA bundle probe v2.
# Preconditions: k3d cluster running, btp-manager + btp-operator installed via install_module.sh
# Usage: ./run_e2e_ca_bundle_probe_test.sh
#
# Environment variables:
#   FAKE_SERVER_IMAGE — fake HTTPS server image (default: localhost:5000/fake-server:latest)
#   WEBHOOK_IMAGE     — fake webhook image (default: localhost:5000/ca-bundle-webhook:latest)

set -o nounset
set -o errexit
set -E
set -o pipefail

NAMESPACE="kyma-system"
FAKE_SERVER_IMAGE=${FAKE_SERVER_IMAGE:-"localhost:5000/fake-server:latest"}
WEBHOOK_IMAGE=${WEBHOOK_IMAGE:-"localhost:5000/ca-bundle-webhook:latest"}
BTPOPERATOR_NAME="btpoperator"
BTP_MANAGER_DEPLOYMENT="btp-manager-controller-manager"
BTP_OPERATOR_DEPLOYMENT="sap-btp-operator-controller-manager"
CERTS_DIR="/tmp/ca-bundle-probe-certs"

# ─── helpers ─────────────────────────────────────────────────────────────────

waitForCRAnnotation() {
  local key=$1 expected=$2 timeout=${3:-120} seconds=0
  echo "--- Waiting for annotation $key=$expected"
  while [[ $seconds -lt $timeout ]]; do
    local actual
    actual=$(kubectl get btpoperator/"$BTPOPERATOR_NAME" -n "$NAMESPACE" \
      -o go-template="{{index .metadata.annotations \"$key\"}}" 2>/dev/null || echo "")
    if [[ "$actual" == "$expected" ]]; then
      echo "--- PASS: annotation $key=$expected"
      return 0
    fi
    sleep 5; seconds=$((seconds + 5))
  done
  local actual
  actual=$(kubectl get btpoperator/"$BTPOPERATOR_NAME" -n "$NAMESPACE" \
    -o go-template="{{index .metadata.annotations \"$key\"}}" 2>/dev/null || echo "")
  echo "--- FAIL: annotation $key: expected='$expected' got='$actual' (timed out after ${timeout}s)"
  return 1
}

assertBtpOperatorRestarted() {
  local before_ts=$1
  local ts
  ts=$(kubectl get deployment/"$BTP_OPERATOR_DEPLOYMENT" -n "$NAMESPACE" \
    -o jsonpath='{.spec.template.metadata.annotations.kubectl\.kubernetes\.io/restartedAt}' \
    2>/dev/null || echo "")
  if [[ "$ts" > "$before_ts" ]]; then
    echo "--- PASS: btp-operator restarted at $ts"
    return 0
  fi
  echo "--- FAIL: btp-operator not restarted (restartedAt='$ts', before='$before_ts')"
  return 1
}

assertBtpOperatorNotRestarted() {
  local before_ts=$1
  local ts
  ts=$(kubectl get deployment/"$BTP_OPERATOR_DEPLOYMENT" -n "$NAMESPACE" \
    -o jsonpath='{.spec.template.metadata.annotations.kubectl\.kubernetes\.io/restartedAt}' \
    2>/dev/null || echo "")
  if [[ "$ts" > "$before_ts" ]]; then
    echo "--- FAIL: btp-operator was restarted unexpectedly at $ts"
    return 1
  fi
  echo "--- PASS: btp-operator not restarted"
}

waitForDeploymentReady() {
  local dep=$1 timeout=${2:-120}
  kubectl rollout status deployment/"$dep" -n "$NAMESPACE" --timeout="${timeout}s"
}

# ─── setup ────────────────────────────────────────────────────────────────────

echo "=== SETUP ==="
kubectl apply -f tests/webhook/manifests/rbac.yaml
kubectl apply -f tests/webhook/manifests/service.yaml
sed "s|WEBHOOK_IMAGE|$WEBHOOK_IMAGE|g" tests/webhook/manifests/deployment.yaml | kubectl apply -f -
sed "s|FAKE_SERVER_IMAGE|$FAKE_SERVER_IMAGE|g" tests/fake-server/manifests/deployment.yaml | kubectl apply -f -
kubectl apply -f tests/fake-server/manifests/service.yaml
kubectl apply -f tests/probe/manifests/rbac.yaml
kubectl apply -f tests/webhook/manifests/mutatingwebhookconfiguration.yaml

# Generate certs: ca-A, ca-B, server-A (signed by ca-A), server-B (signed by ca-B), webhook TLS
./tests/probe/setup-certs.sh "$NAMESPACE"

waitForDeploymentReady ca-bundle-webhook
waitForDeploymentReady fake-server

# ─── phase A: baseline, no mount, system CA ──────────────────────────────────

echo "=== PHASE A: baseline (no mount, system CA) ==="

# Step 1 (row 1): system CA trusts server → inject ca-A into k3d node system trust store
K3D_NODE=${K3D_NODE:-k3d-k3s-default-server-0}
docker exec "$K3D_NODE" sh -c "cp /dev/stdin /usr/local/share/ca-certificates/ca-A.crt" \
  < "$CERTS_DIR/ca-A.crt"
docker exec "$K3D_NODE" update-ca-certificates

BEFORE_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "--- Step 1 (row 1): no mount, system CA, TLS ok"
waitForCRAnnotation "tls-probe-mount" "false"
waitForCRAnnotation "tls-probe-tls-result" "ok"
waitForCRAnnotation "tls-probe-signal" ""
assertBtpOperatorNotRestarted "$BEFORE_TS"

# Step 2 (row 2): remove ca-A from system store → TLS x509
echo "--- Step 2 (row 2): no mount, system CA, TLS x509"
docker exec "$K3D_NODE" sh -c \
  "rm -f /usr/local/share/ca-certificates/ca-A.crt && update-ca-certificates"

waitForCRAnnotation "tls-probe-tls-result" "failed-x509"
waitForCRAnnotation "tls-probe-signal" "warning"
assertBtpOperatorNotRestarted "$BEFORE_TS"

# ─── phase B: mount injected (customCA1 = ca-A) ──────────────────────────────

echo "=== PHASE B: mount injected (customCA1 = ca-A) ==="
CA_A_B64=$(base64 -w0 < "$CERTS_DIR/ca-A.crt")
CA_BUNDLE_B64=$CA_A_B64 envsubst < scripts/testing/yaml/ca-bundle-probe/ca-bundle-secret.yaml | kubectl apply -f -

# Restart btp-manager so webhook fires and injects rt-bootstrapper-certs mount
kubectl rollout restart deployment/"$BTP_MANAGER_DEPLOYMENT" -n "$NAMESPACE"
waitForDeploymentReady "$BTP_MANAGER_DEPLOYMENT"

echo "--- Step 3 (row 3): mount=customCA1, TLS ok — first hash written, no restart"
BEFORE_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
waitForCRAnnotation "tls-probe-mount" "true"
waitForCRAnnotation "tls-probe-tls-result" "ok"
waitForCRAnnotation "tls-probe-signal" ""
assertBtpOperatorNotRestarted "$BEFORE_TS"

echo "--- Step 4 (row 5): mount=customCA1, TLS x509 (server cert signed by ca-B, not trusted by ca-A)"
TLS_CRT_B64=$(base64 -w0 < "$CERTS_DIR/server-B.crt")
TLS_KEY_B64=$(base64 -w0 < "$CERTS_DIR/server-B.key")
TLS_CRT_B64=$TLS_CRT_B64 TLS_KEY_B64=$TLS_KEY_B64 \
  envsubst < scripts/testing/yaml/ca-bundle-probe/fake-server-tls-secret.yaml | kubectl apply -f -

BEFORE_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
waitForCRAnnotation "tls-probe-tls-result" "failed-x509"
waitForCRAnnotation "tls-probe-signal" "alert"
assertBtpOperatorNotRestarted "$BEFORE_TS"

# ─── phase C: bundle rotation (customCA2 = ca-B), TLS ok ─────────────────────

echo "=== PHASE C: bundle rotation to customCA2 (ca-B), TLS ok ==="
CA_B_B64=$(base64 -w0 < "$CERTS_DIR/ca-B.crt")
CA_BUNDLE_B64=$CA_B_B64 envsubst < scripts/testing/yaml/ca-bundle-probe/ca-bundle-secret.yaml | kubectl apply -f -

kubectl rollout restart deployment/"$BTP_MANAGER_DEPLOYMENT" -n "$NAMESPACE"
waitForDeploymentReady "$BTP_MANAGER_DEPLOYMENT"

echo "--- Step 5 (row 4): mount=customCA2, hash changed, TLS ok → btp-operator restarted"
BEFORE_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
waitForCRAnnotation "tls-probe-tls-result" "ok"
waitForCRAnnotation "tls-probe-signal" ""
assertBtpOperatorRestarted "$BEFORE_TS"

# ─── phase D: new bundle delivered but doesn't trust server ──────────────────

echo "=== PHASE D: new bundle (ca-B2) that doesn't trust server (signed by ca-A) ==="
TLS_CRT_B64=$(base64 -w0 < "$CERTS_DIR/server-A.crt")
TLS_KEY_B64=$(base64 -w0 < "$CERTS_DIR/server-A.key")
TLS_CRT_B64=$TLS_CRT_B64 TLS_KEY_B64=$TLS_KEY_B64 \
  envsubst < scripts/testing/yaml/ca-bundle-probe/fake-server-tls-secret.yaml | kubectl apply -f -

# Generate a fresh CA (ca-B2) — different hash from ca-B, doesn't trust server-A
openssl req -x509 -newkey rsa:2048 -keyout /tmp/ca-B2.key -out /tmp/ca-B2.crt \
  -days 365 -nodes -subj "/CN=test-ca-B2" 2>/dev/null
CA_B2_B64=$(base64 -w0 < /tmp/ca-B2.crt)
CA_BUNDLE_B64=$CA_B2_B64 envsubst < scripts/testing/yaml/ca-bundle-probe/ca-bundle-secret.yaml | kubectl apply -f -

kubectl rollout restart deployment/"$BTP_MANAGER_DEPLOYMENT" -n "$NAMESPACE"
waitForDeploymentReady "$BTP_MANAGER_DEPLOYMENT"

echo "--- Step 6 (row 6): mount=customCA2-bad, hash changed, TLS x509 → alert, no restart"
BEFORE_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
waitForCRAnnotation "tls-probe-tls-result" "failed-x509"
waitForCRAnnotation "tls-probe-signal" "alert"
assertBtpOperatorNotRestarted "$BEFORE_TS"

# ─── phase E: connectivity failure ───────────────────────────────────────────

echo "=== PHASE E: connectivity failure ==="
kubectl scale deployment/fake-server -n "$NAMESPACE" --replicas=0

echo "--- Step 7 (row 7): server unreachable → warning, no restart"
BEFORE_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
waitForCRAnnotation "tls-probe-signal" "warning"
assertBtpOperatorNotRestarted "$BEFORE_TS"

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
