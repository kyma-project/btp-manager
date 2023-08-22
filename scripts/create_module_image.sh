#!/usr/bin/env bash

# This script has the following argument:
#     optional ci to indicate call from CI pipeline
# ./create_module_image.sh ci

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables passed e.g. from CI:
#   PULL_NUMBER     - number of the PR
#   MODULE_REGISTRY - docker registry to push the module to
#   IMG_REGISTRY    -  docker registry to push the image to

echo "PULL_NUMBER ${PULL_NUMBER}"

PULL_REQUEST_NAME="PR-${PULL_NUMBER}"

# To satisfy component name validation
HARD_WIRED_RELEASE="0.0.0"
MODULE_VERSION=${HARD_WIRED_RELEASE}-${PULL_REQUEST_NAME}

IMAGE_NAME=/btp-manager
IMAGE_REFERENCE=${IMG_REGISTRY}${IMAGE_NAME}:${PULL_REQUEST_NAME}

echo "MODULE_VERSION ${MODULE_VERSION} - No security scanning configuration in module template"
echo "IMAGE_REFERENCE ${IMAGE_REFERENCE}"

MODULE_VERSION=${MODULE_VERSION} IMG=${IMAGE_REFERENCE} make module-build

echo "Generated template.yaml:"
cat template.yaml
