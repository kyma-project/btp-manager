#!/usr/bin/env bash

# This script has the following arguments:
#                       optional ci to indicate call from CI pipeline
# The script uses the following environment variables (if not provided script defines default values):
#                       LOCAL_REGISTRY
#                       K3D_REGISTRY
#                       PR_NAME
#                       MODULE_PREFIX
# ./run_e2e_on_k3d.sh ci

CI=${2-manual}  # if called from any workflow "ci" is expected here

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being maskedPORT=5001

echo "Building images and pushing to the local registry"

# sets MODULE_REFERENCE variable as a side effect - hence sourcing
. ./scripts/build_and_push_locally.sh

echo "Running E2E tests"

./scripts/testing/run_e2e_module_tests.sh "${MODULE_REFERENCE}" dummy "${CI}"

