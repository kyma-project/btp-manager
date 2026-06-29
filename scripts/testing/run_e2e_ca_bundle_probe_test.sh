#!/usr/bin/env bash
# E2E test for CA bundle probe v2.
# Preconditions: k3d cluster running, btp-manager + btp-operator installed via install_module.sh
# Usage: ./run_e2e_ca_bundle_probe_test.sh
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

# ─── helpers ─────────────────────────────────────────────────────────────────

waitForJobCompletion() {
  local job=$1 timeout=${2:-120} seconds=0
  echo "--- Waiting for job $job to complete"
  while [[ $seconds -lt $timeout ]]; do
    local status
    status=$(kubectl get job/"$job" -n "$NAMESPACE" \
      -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}' 2>/dev/null || echo "")
    [[ "$status" == "True" ]] && echo "--- Job $job completed" && return 0
    local failed
    failed=$(kubectl get job/"$job" -n "$NAMESPACE" \
      -o jsonpath='{.status.conditions[?(@.type=="Failed")].status}' 2>/dev/null || echo "")
    if [[ "$failed" == "True" ]]; then
      echo "--- ERROR: job $job failed"
      kubectl logs job/"$job" -n "$NAMESPACE" || true
      return 1
    fi
    sleep 5; seconds=$((seconds + 5))
  done
  echo "--- ERROR: job $job timed out after ${timeout}s"
  kubectl logs job/"$job" -n "$NAMESPACE" || true
  return 1
}

runProbeJob() {
  local run_name=$1
  kubectl delete job/"$run_name" -n "$NAMESPACE" --ignore-not-found 2>/dev/null
  kubectl create -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: $run_name
  namespace: $NAMESPACE
spec:
  template:
    spec:
      serviceAccountName: ca-bundle-probe
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
  waitForJobCompletion "$run_name"
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
# Clean up any leftover state from a previous run
kubectl delete secret/ca-bundle -n "$NAMESPACE" --ignore-not-found
kubectl delete mutatingwebhookconfiguration/ca-bundle-webhook --ignore-not-found
for job in probe-a1 probe-b1 probe-b2 probe-c1 probe-d1 probe-e1; do
  kubectl delete job/"$job" -n "$NAMESPACE" --ignore-not-found 2>/dev/null
done
# Restore fake-server TLS to server-A (ca-A-signed) as baseline
if [[ -f "$CERTS_DIR/server-A.crt" ]]; then
  TLS_CRT_B64=$(base64 -w0 < "$CERTS_DIR/server-A.crt")
  TLS_KEY_B64=$(base64 -w0 < "$CERTS_DIR/server-A.key")
  TLS_CRT_B64=$TLS_CRT_B64 TLS_KEY_B64=$TLS_KEY_B64 \
    envsubst < scripts/testing/yaml/ca-bundle-probe/fake-server-tls-secret.yaml | kubectl apply -f - 2>/dev/null || true
fi

kubectl apply -f tests/webhook/manifests/rbac.yaml
kubectl apply -f tests/webhook/manifests/service.yaml
sed "s|WEBHOOK_IMAGE|$WEBHOOK_IMAGE|g" tests/webhook/manifests/deployment.yaml | kubectl apply -f -
sed "s|FAKE_SERVER_IMAGE|$FAKE_SERVER_IMAGE|g" tests/fake-server/manifests/deployment.yaml | kubectl apply -f -
kubectl apply -f tests/fake-server/manifests/service.yaml
kubectl apply -f tests/probe/manifests/rbac.yaml
kubectl apply -f tests/webhook/manifests/mutatingwebhookconfiguration.yaml

# Generate certs: ca-A, ca-B, server-A (signed by ca-A), server-B (signed by ca-B), webhook TLS
./tests/probe/setup-certs.sh "$NAMESPACE"

# Restart webhook so it picks up the freshly generated TLS secret
kubectl rollout restart deployment/ca-bundle-webhook -n "$NAMESPACE"
waitForDeploymentReady ca-bundle-webhook
waitForDeploymentReady fake-server

# Ensure btp-manager has no stale rt-bootstrapper-certs mount from a previous run
kubectl rollout restart deployment/"$BTP_MANAGER_DEPLOYMENT" -n "$NAMESPACE"
waitForDeploymentReady "$BTP_MANAGER_DEPLOYMENT"

# ─── phase A: baseline, no mount, distroless system CA ──────────────────────
# The probe runs in a distroless container whose x509.SystemCertPool() reads the CA bundle
# baked into the image layer. That bundle contains real root CAs and will never trust our
# self-signed fake-server cert. So: mount=false, tls=failed-x509, signal=warning — always.
# This tests that the mount detection and signal logic work correctly when no volume is injected.

echo "=== PHASE A: no mount, distroless system CA doesn't trust self-signed fake-server ==="
BEFORE_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
runProbeJob "probe-a1"
assertCRAnnotation "tls-probe-mount" "false"
assertCRAnnotation "tls-probe-tls-result" "failed-x509"
assertCRAnnotation "tls-probe-signal" "warning"
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
runProbeJob "probe-b1"
assertCRAnnotation "tls-probe-mount" "true"
assertCRAnnotation "tls-probe-tls-result" "ok"
assertCRAnnotation "tls-probe-signal" ""
assertBtpOperatorNotRestarted "$BEFORE_TS"

echo "--- Step 4 (row 5): mount=customCA1, server cert signed by ca-B → TLS x509"
TLS_CRT_B64=$(base64 -w0 < "$CERTS_DIR/server-B.crt")
TLS_KEY_B64=$(base64 -w0 < "$CERTS_DIR/server-B.key")
TLS_CRT_B64=$TLS_CRT_B64 TLS_KEY_B64=$TLS_KEY_B64 \
  envsubst < scripts/testing/yaml/ca-bundle-probe/fake-server-tls-secret.yaml | kubectl apply -f -
sleep 3  # give fake-server time to reload cert

BEFORE_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
runProbeJob "probe-b2"
assertCRAnnotation "tls-probe-mount" "true"
assertCRAnnotation "tls-probe-tls-result" "failed-x509"
assertCRAnnotation "tls-probe-signal" "alert"
assertBtpOperatorNotRestarted "$BEFORE_TS"

# ─── phase C: bundle rotation (customCA2 = ca-B), TLS ok ─────────────────────

echo "=== PHASE C: bundle rotation to customCA2 (ca-B), TLS ok ==="
CA_B_B64=$(base64 -w0 < "$CERTS_DIR/ca-B.crt")
CA_BUNDLE_B64=$CA_B_B64 envsubst < scripts/testing/yaml/ca-bundle-probe/ca-bundle-secret.yaml | kubectl apply -f -

kubectl rollout restart deployment/"$BTP_MANAGER_DEPLOYMENT" -n "$NAMESPACE"
waitForDeploymentReady "$BTP_MANAGER_DEPLOYMENT"

echo "--- Step 5 (row 4): mount=customCA2, hash changed, TLS ok → btp-operator restarted"
BEFORE_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
runProbeJob "probe-c1"
assertCRAnnotation "tls-probe-mount" "true"
assertCRAnnotation "tls-probe-tls-result" "ok"
assertCRAnnotation "tls-probe-signal" ""
assertBtpOperatorRestarted "$BEFORE_TS"

# ─── phase D: new bundle doesn't trust server ────────────────────────────────

echo "=== PHASE D: new bundle (ca-B2) that doesn't trust server (signed by ca-A) ==="
TLS_CRT_B64=$(base64 -w0 < "$CERTS_DIR/server-A.crt")
TLS_KEY_B64=$(base64 -w0 < "$CERTS_DIR/server-A.key")
TLS_CRT_B64=$TLS_CRT_B64 TLS_KEY_B64=$TLS_KEY_B64 \
  envsubst < scripts/testing/yaml/ca-bundle-probe/fake-server-tls-secret.yaml | kubectl apply -f -

openssl req -x509 -newkey rsa:2048 -keyout /tmp/ca-B2.key -out /tmp/ca-B2.crt \
  -days 365 -nodes -subj "/CN=test-ca-B2" 2>/dev/null
CA_B2_B64=$(base64 -w0 < /tmp/ca-B2.crt)
CA_BUNDLE_B64=$CA_B2_B64 envsubst < scripts/testing/yaml/ca-bundle-probe/ca-bundle-secret.yaml | kubectl apply -f -

kubectl rollout restart deployment/"$BTP_MANAGER_DEPLOYMENT" -n "$NAMESPACE"
waitForDeploymentReady "$BTP_MANAGER_DEPLOYMENT"
sleep 3  # give fake-server time to reload cert

echo "--- Step 6 (row 6): mount=customCA2-bad, hash changed, TLS x509 → alert, no restart"
BEFORE_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
runProbeJob "probe-d1"
assertCRAnnotation "tls-probe-mount" "true"
assertCRAnnotation "tls-probe-tls-result" "failed-x509"
assertCRAnnotation "tls-probe-signal" "alert"
assertBtpOperatorNotRestarted "$BEFORE_TS"

# ─── phase E: connectivity failure ───────────────────────────────────────────

echo "=== PHASE E: connectivity failure ==="
kubectl scale deployment/fake-server -n "$NAMESPACE" --replicas=0
sleep 5

echo "--- Step 7 (row 7): server unreachable → warning, no restart"
BEFORE_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
runProbeJob "probe-e1"
assertCRAnnotation "tls-probe-signal" "warning"
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
for job in probe-a1 probe-b1 probe-b2 probe-c1 probe-d1 probe-e1; do
  kubectl delete job/"$job" -n "$NAMESPACE" --ignore-not-found
done

echo "=== ALL TESTS PASSED ==="
