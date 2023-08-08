#!/usr/bin/env bash
# Stress testing in regard to memory consumption - could cause OOM (but should not).
# Creates btp-operator and numerous Service Instances and Service Bindings in the current context.
#
# The script has the following arguments:
#     - number of Config Maps and Secrets (the provided number is multiplied by 10)
# Example (creates 1000 CMs and Secrets):
#     ./stress-mem.sh 100

N=${1-100}
YAML_DIR=./scripts/testing/yaml

echo -e "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: kyma-system" | kubectl apply -f -

echo -e "\n---Creating the secret template"

cat ${YAML_DIR}/secret.tmpl.yaml > secret.yaml
cat ${YAML_DIR}/cm.tmpl.yaml > cm.yaml
for ((i=1; i <= 4000 ; i++))
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