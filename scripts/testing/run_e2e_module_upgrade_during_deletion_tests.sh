#!/usr/bin/env bash

# This script has the following arguments:
#     - tag for the upgrade binary image (optional),
#     - tag for the base binary image (optional),
# ./run_e2e_module_upgrade_during_deletion_tests.sh 1.1.2 1.0.0

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

REGISTRY_PROD=europe-docker.pkg.dev/kyma-project/prod/btp-manager
REGISTRY_DEV=europe-docker.pkg.dev/kyma-project/dev/btp-manager

if [[ $# -eq 2 ]]; then
  # upgrade from one given version to another given version
  UPGRADE_TAG=${1}
  BASE_TAG=${2}
elif [[ $# -eq 1 ]]; then
  # upgrade from the latest release to the given version
  UPGRADE_TAG=${1}
  REPOSITORY=${REPOSITORY:-kyma-project/btp-manager}
  GITHUB_URL=https://api.github.com/repos/${REPOSITORY}
  BASE_TAG=$(curl -sS "${GITHUB_URL}/releases/latest" | jq -r '.tag_name')
elif [[ $# -eq 0 ]]; then
  # upgrade from the pre-latest release to the latest release
  REPOSITORY=${REPOSITORY:-kyma-project/btp-manager}
  GITHUB_URL=https://api.github.com/repos/${REPOSITORY}
  UPGRADE_TAG=$(curl -sS "${GITHUB_URL}/releases/latest" | jq -r '.tag_name')
  BASE_TAG=$(curl -sS "${GITHUB_URL}/tags" | jq -r '.[].name' | grep -A1 "${UPGRADE_TAG}" | grep -v "${UPGRADE_TAG}")
else
  echo "wrong number of arguments" && exit 1
fi
