#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

TARGET_DIRECTORY=${TARGET_DIRECTORY:-downloaded_module}

FILENAME=$(./scripts/check_if_k8s_yaml.sh)

if [[ -n "$FILENAME" ]]; 
then
  kubectl delete -f ${TARGET_DIRECTORY}/${FILENAME}
else
  helm uninstall btp-manager -n kyma-system
fi
