#!/usr/bin/env bash
# This script checks if btp-manager was restarted (exit 1)

restarts=$(kubectl get po -n kyma-system -l app.kubernetes.io/component=btp-manager.kyma-project.io -o 'jsonpath={..items[0].status.containerStatuses[0].restartCount}')
if [ "${restarts}" != '0' ]
then
    echo "BTP manager was restarted $restarts times"
    exit 1
fi