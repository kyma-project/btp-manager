#!/usr/bin/env bash
# This script checks if btp-manager and btp-operator was restarted (exit 1)

# Give some time to both controllers to process stress data
sleep 15

echo -e "\n--- BTP Manager checking pod restarts"
restarts=$(kubectl get po -n kyma-system -l app.kubernetes.io/component=btp-manager.kyma-project.io -o 'jsonpath={..items[0].status.containerStatuses[0].restartCount}')
if [ "${restarts}" != '0' ]
then
  echo "BTP Manager was restarted $restarts times"
  exit 1
fi
echo -e "\n--- BTP Manager no restarts"

echo -e "\n--- BTP Operator checking pod restarts"
restarts=$(kubectl get po -n kyma-system -l app.kubernetes.io/name=sap-btp-operator -o 'jsonpath={..items[0].status.containerStatuses[?(@.name=="manager")].restartCount}')
if [ "${restarts}" != '0' ]
then
  echo "BTP Operator was restarted $restarts times"
  exit 1
fi
echo -e "\n--- BTP Operator no restarts"