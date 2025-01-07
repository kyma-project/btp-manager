#!/usr/bin/env bash

# This script has the following arguments:
#     - release tag,
# ./get_manifest.sh 1.0.0

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

TAG=${1}
MANIFEST_FILENAME="btp-manager.yaml"
BTP_MANAGER_RELEASES_URL="https://github.com/kyma-project/btp-manager/releases"

curl -sL ${BTP_MANAGER_RELEASES_URL}/download/"${TAG}"/${MANIFEST_FILENAME}
