#!/usr/bin/env bash

# This script has the following argument: a link to a module image, for example:
# ./run_e2e_module_tests.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:0.0.0-PR-999

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# installing prerequisites, on production environment these are present before chart is used
kubectl apply -f ./deployments/prerequisites.yaml
kubectl apply -f ./examples/btp-manager-secret.yaml

# fetch OCI module image and install btp-manager in current cluster
./hack/run_module_image.sh "${IMAGE_NAME}"

# uninstall btp-manager
helm uninstall btp-manager

#clean up and ignore errors
kubectl delete -f ./examples/btp-manager-secret.yaml || echo "ignoring failure during secret removal"
kubectl delete -f ./deployments/prerequisites.yaml || echo "ignoring failure during prerequisites removal"
