#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

echo -e "\n--- BTP Manager checking kubectl top"
kubectl top pod -l app.kubernetes.io/component=btp-manager.kyma-project.io -n kyma-system --containers

echo -e "\n--- BTP Operator checking kubectl top"
kubectl top pod -l app.kubernetes.io/name=sap-btp-operator -n kyma-system --containers