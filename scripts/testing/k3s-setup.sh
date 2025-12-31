#!/usr/bin/env bash

DEFAULT_K3S_VERSION="v1.33.5+k3s1"

if [ "${1}" == "--wait" ]
then
  WAIT_OPT=$1
  K3S_VERSION=${2:-${DEFAULT_K3S_VERSION}}
else
  WAIT_OPT=$2
  K3S_VERSION=${1:-${DEFAULT_K3S_VERSION}}
fi

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

echo "Starting docker registry"
sudo mkdir -p /etc/rancher/k3s
sudo cp scripts/testing/yaml/registries.yaml /etc/rancher/k3s
docker run -d \
-p 5000:5000 \
--restart=always \
--name registry.localhost \
-v "$PWD/registry:/var/lib/registry" \
europe-docker.pkg.dev/kyma-project/prod/external/registry:2.8.3

echo "Starting K3s cluster (K3s version: ${K3S_VERSION})"
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=${K3S_VERSION} K3S_KUBECONFIG_MODE=777 INSTALL_K3S_EXEC="server --disable traefik" sh -
mkdir -p ~/.kube
cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
chmod 600 ~/.kube/config

if [ "${WAIT_OPT}" == "--wait" ]
then
  while [[ $(kubectl get nodes -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]];
  do echo "Waiting for cluster nodes to be ready"; sleep 1; done
fi