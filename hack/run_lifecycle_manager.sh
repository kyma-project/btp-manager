#!/usr/bin/env bash

# This script has the following argument: a link to a template file, for example:
# ./hack/run_lifecycle_manager.sh https://github.com/kyma-project/btp-manager/releases/latest/download/template.yaml

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked


echo -e "\n--- Download latest kyma CLI"
curl -Lo kyma https://storage.googleapis.com/kyma-cli-unstable/kyma-darwin
chmod +x kyma
mv kyma temp
echo -e "\n--- kyma CLI usage: ./temp/kyma --help"

echo -e "\n--- Create k3d cluster"
kyma provision k3d --ci

echo -e "\n--- Deploy Lifecycle Manager"
kyma alpha deploy --ci

echo -e "\n--- Apply BTP Manager template.yaml"
kubectl apply -f $1

echo -e "\n--- List Kyma modules template.yaml"
kyma alpha list module -n kcp-system

echo -e "\n--- Create dummy secret for btp-operator module"
kubectl apply -f examples/btp-manager-secret.yaml

echo -e "\n--- Enable btp-operator module"
kyma alpha enable module btp-operator -c alpha -w

echo -e "\n--- List BTP Manager pods"
kubectl get pods -n kyma-system