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

echo -e "\n---Creating ${N} config maps and secrets"

for ((i=1; i <= N ; i++))
  do
      CM_NAME=cm-$i
      SECRET_NAME=secret-$i

      export CM_NAME
      export SECRET_NAME

      envsubst <${YAML_DIR}/stress-cm.yaml | kubectl apply -f - >/dev/null
      envsubst <${YAML_DIR}/stress-secret.yaml | kubectl apply -f - >/dev/null
done