#!/usr/bin/env bash

set -x

sudo systemctl stop k3s
sudo k3s server --cluster-reset
sudo /usr/local/bin/k3s-uninstall.sh

echo "Starting K3s cluster (K3s version: ${K3S_VERSION})"
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=${K3S_VERSION} K3S_KUBECONFIG_MODE=777 INSTALL_K3S_EXEC="server --disable traefik" sh -
mkdir -p ~/.kube
cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
chmod 600 ~/.kube/config
　