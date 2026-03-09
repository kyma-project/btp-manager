#!/usr/bin/env bash
# Stress testing in regard to memory consumption - could cause OOM (but should not).
# Creates secrets and configmaps in the current context.
#
# The script has the following arguments:
#     - number of Config Maps and Secrets to create (default: 100)
#     - number of large (64KB) secrets to create (default: 0)
# Example (creates 1000 CMs and Secrets):
#     ./stress-mem-secrets-configmaps.sh 100

N=${1-100}
M=${2-0}

SIZE=4000
YAML_DIR=./scripts/testing/yaml

echo -e "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: kyma-system" | kubectl apply -f -

echo -e "\n---Creating templates"

cat ${YAML_DIR}/secret.tmpl.yaml > secret.yaml
cat ${YAML_DIR}/cm.tmpl.yaml > cm.yaml
cat ${YAML_DIR}/large.secret.tmpl.yaml > large.secret.yaml
for ((i=1; i <= SIZE ; i++))
do
  echo "  key$i: \"data-01234567890123456789-abcdefgh-$i\"" >> secret.yaml
  echo "  key$i: \"data-01234567890123456789-abcdefgh-$i\"" >> cm.yaml
done

echo -e "\n---Creating ${N} ConfigMaps and Secrets"
for ((i=1; i <= N ; i++))
do
  CM_NAME=cm-$i
  SECRET_NAME=secret-$i

  export CM_NAME
  export SECRET_NAME

  envsubst <cm.yaml | kubectl apply -f - >/dev/null
  envsubst <secret.yaml | kubectl apply -f - >/dev/null
done

echo -e "\n---Creating ${M} large Secrets"
for ((i=1; i <= M ; i++))
do
  SECRET_NAME=large-secret-$i

  export SECRET_NAME

  envsubst <large.secret.yaml | kubectl apply -f - >/dev/null
done

./scripts/testing/check_pod_restarts.sh