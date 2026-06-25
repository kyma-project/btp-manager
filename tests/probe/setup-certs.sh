#!/usr/bin/env bash
# Generates ca-A and ca-B keypairs, creates k8s Secrets for test use.
# Usage: ./setup-certs.sh [namespace]
set -o nounset
set -o errexit
set -o pipefail

NAMESPACE=${1:-kyma-system}
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "--- Generating ca-A keypair"
openssl req -x509 -newkey rsa:2048 -keyout $TMPDIR/ca-A.key -out $TMPDIR/ca-A.crt \
  -days 365 -nodes -subj "/CN=test-ca-A"

echo "--- Generating ca-A server cert (signed by ca-A)"
openssl req -newkey rsa:2048 -keyout $TMPDIR/server-A.key -out $TMPDIR/server-A.csr \
  -nodes -subj "/CN=fake-server.kyma-system.svc.cluster.local"
openssl x509 -req -in $TMPDIR/server-A.csr -CA $TMPDIR/ca-A.crt -CAkey $TMPDIR/ca-A.key \
  -CAcreateserial -out $TMPDIR/server-A.crt -days 365 \
  -extfile <(printf "subjectAltName=DNS:fake-server.kyma-system.svc,DNS:fake-server.kyma-system.svc.cluster.local")

echo "--- Generating ca-B keypair"
openssl req -x509 -newkey rsa:2048 -keyout $TMPDIR/ca-B.key -out $TMPDIR/ca-B.crt \
  -days 365 -nodes -subj "/CN=test-ca-B"

echo "--- Generating ca-B server cert (signed by ca-B)"
openssl req -newkey rsa:2048 -keyout $TMPDIR/server-B.key -out $TMPDIR/server-B.csr \
  -nodes -subj "/CN=fake-server.kyma-system.svc.cluster.local"
openssl x509 -req -in $TMPDIR/server-B.csr -CA $TMPDIR/ca-B.crt -CAkey $TMPDIR/ca-B.key \
  -CAcreateserial -out $TMPDIR/server-B.crt -days 365 \
  -extfile <(printf "subjectAltName=DNS:fake-server.kyma-system.svc,DNS:fake-server.kyma-system.svc.cluster.local")

echo "--- Generating webhook TLS cert (self-signed)"
openssl req -x509 -newkey rsa:2048 -keyout $TMPDIR/webhook.key -out $TMPDIR/webhook.crt \
  -days 365 -nodes -subj "/CN=ca-bundle-webhook.kyma-system.svc.cluster.local" \
  -addext "subjectAltName=DNS:ca-bundle-webhook.kyma-system.svc,DNS:ca-bundle-webhook.kyma-system.svc.cluster.local"

echo "--- Creating k8s Secrets"
kubectl create secret tls fake-server-tls \
  --cert=$TMPDIR/server-A.crt --key=$TMPDIR/server-A.key \
  --namespace=$NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret tls ca-bundle-webhook-tls \
  --cert=$TMPDIR/webhook.crt --key=$TMPDIR/webhook.key \
  --namespace=$NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Patch caBundle in MutatingWebhookConfiguration
CA_BUNDLE_B64=$(base64 -w0 < $TMPDIR/webhook.crt)
kubectl patch mutatingwebhookconfiguration ca-bundle-webhook \
  --type=json -p="[{\"op\":\"replace\",\"path\":\"/webhooks/0/clientConfig/caBundle\",\"value\":\"${CA_BUNDLE_B64}\"}]"

# Export cert files for use by e2e script
mkdir -p /tmp/ca-bundle-probe-certs
cp $TMPDIR/ca-A.crt /tmp/ca-bundle-probe-certs/ca-A.crt
cp $TMPDIR/ca-B.crt /tmp/ca-bundle-probe-certs/ca-B.crt
cp $TMPDIR/server-A.crt /tmp/ca-bundle-probe-certs/server-A.crt
cp $TMPDIR/server-A.key /tmp/ca-bundle-probe-certs/server-A.key
cp $TMPDIR/server-B.crt /tmp/ca-bundle-probe-certs/server-B.crt
cp $TMPDIR/server-B.key /tmp/ca-bundle-probe-certs/server-B.key

echo "--- Cert setup complete. CA certs in /tmp/ca-bundle-probe-certs/"
