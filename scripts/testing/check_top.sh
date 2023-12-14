#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

TIMEOUT=30
NEXT_TRY_WAIT=5

echo -e "\n--- BTP Manager checking kubectl top" 

SECONDS=0
while ((SECONDS < $TIMEOUT )); do
     kubectl top pod -l app.kubernetes.io/component=btp-manager.kyma-project.io -n kyma-system --containers
     [ $? == 0 ] && break
     sleep $NEXT_TRY_WAIT
done

echo -e "\n--- BTP Operator checking kubectl top" 

SECONDS=0
while ((SECONDS < $TIMEOUT )); do
     kubectl top pod -l app.kubernetes.io/name=sap-btp-operator -n kyma-system --containers
     [ $? == 0 ] && break
     sleep $NEXT_TRY_WAIT
done