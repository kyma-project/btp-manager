#!/usr/bin/env bash
# Stress testing in regard to memory consumption - could cause OOM (but should not).
# Creates secrets and configmaps in the current context.
#
# The script has the following arguments:
#     - number of large (64KB) Secrets to create (default: 16)
# Example (creates 10 Secrets):
#     ./stress-mem-large-labeled-secrets-configmaps.sh 10

M=${1-16}

echo -e "\n---Creating ${M} large and labeled Secrets"
for ((i=1; i <= M ; i++))
do
  SECRET_NAME=large-labeled-secret-$i

  export SECRET_NAME

  envsubst <large.labeled.secret.tmpl.yaml | kubectl apply -f - >/dev/null
done

./scripts/testing/check_pod_restarts.sh

sleep 60