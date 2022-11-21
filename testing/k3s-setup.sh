#!/bin/bash
set -o errexit

echo "Starting docker registry"
sudo mkdir -p /etc/rancher/k3s
sudo cp testing/registries.yaml /etc/rancher/k3s
docker run -d \
-p 5000:5000 \
--restart=always \
--name registry.localhost \
-v "$PWD/registry:/var/lib/registry" \
registry:2

echo "Starting K3s cluster"
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="v1.24.7+k3s1" K3S_KUBECONFIG_MODE=777 INSTALL_K3S_EXEC="server --disable traefik" sh -
mkdir -p ~/.kube
cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
chmod 600 ~/.kube/config