#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

set -x

sudo systemctl stop k3s
sudo k3s server --cluster-reset
sudo systemctl start k3s

while [[ $(kubectl get nodes -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]];
do echo "Waiting for cluster nodes to be ready"; sleep 1; done
