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

#SemVer regular expression, see https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
# bash does not support '\d' character class, so it has been replaced with '[0-9]' range
SEMVER_REGEX='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-((?:0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$'

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

if [[ ${UPGRADE_TAG} =~ ${SEMVER_REGEX} ]]; then
  UPGRADE_IMAGE_REF=${REGISTRY_PROD}:${UPGRADE_TAG}
else
  UPGRADE_IMAGE_REF=${REGISTRY_DEV}:${UPGRADE_TAG}
fi

if [[ ${BASE_TAG} =~ ${SEMVER_REGEX} ]]; then
  BASE_IMAGE_REF=${REGISTRY_PROD}:${BASE_TAG}
else
  BASE_IMAGE_REF=${REGISTRY_DEV}:${BASE_TAG}
fi
