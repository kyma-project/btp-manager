#!/usr/bin/env bash

# This script has the following arguments:
#     the mandatory link to a module image,
#     optional ci to indicate call from CI pipeline
# ./run_e2e_module_tests.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:v0.0.0-PR-999 ci

CI=$2

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

MODULE_IMAGE_NAME=$1

# installing prerequisites, on production environment these are present before chart is used
kubectl apply -f ./deployments/prerequisites.yaml || echo "ignoring failure during prerequisites installation"
kubectl apply -f ./examples/btp-manager-secret.yaml || echo "ignoring failure during secret creation"

# fetch OCI module image and install btp-manager in current cluster
./hack/run_module_image.sh "${MODULE_IMAGE_NAME}" ${CI}

# check if deployment is available
while [[ $(kubectl get deployment/btp-manager-controller-manager -n kyma-system -o 'jsonpath={..status.conditions[?(@.type=="Available")].status}') != "True" ]];
do echo "Waiting for deployment to be available"; sleep 5; done

echo "Deployment available"

echo "Uninstalling..."

# uninstall btp-manager
helm uninstall btp-manager

#clean up and ignore errors
kubectl delete -f ./examples/btp-manager-secret.yaml || echo "ignoring failure during secret removal"
kubectl delete -f ./deployments/prerequisites.yaml || echo "ignoring failure during prerequisites removal"
