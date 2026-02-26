#!/usr/bin/env bash

set -x

sudo systemctl stop k3s
sudo k3s server --cluster-reset
sudo systemctl restart k3s

while [[ $(kubectl get nodes -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]];
do echo "Waiting for cluster nodes to be ready"; sleep 1; done
