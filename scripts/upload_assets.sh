#!/usr/bin/env bash

# This script has the following argument:
#     - releaseID (mandatory)
# ./upload_assets.sh 12345678

RELEASE_ID=${1}

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables:
#   BOT_GITHUB_TOKEN - github token used to upload the template yaml
#   KYMA_BTP_MANAGER_REPO  - Kyma repository

uploadFile() {
  filePath=${1}
  ghAsset=${2}

  response=$(curl -s -o output.txt -w "%{http_code}" \
                  --request POST --data-binary @"$filePath" \
                  -H "Authorization: token $BOT_GITHUB_TOKEN" \
                  -H "Content-Type: text/yaml" \
                   $ghAsset)
  if [[ "$response" != "201" ]]; then
    echo "::error ::Unable to upload the asset ($filePath): "
    echo "::error ::HTTP Status: $response"
    cat output.txt
    exit 1
  else
    echo "$filePath uploaded"
  fi
}

MANIFEST_FILE="./manifests/btp-operator/btp-manager.yaml"
DEFAULT_CR_FILE="./examples/btp-operator.yaml"
UPLOAD_URL="https://uploads.github.com/repos/${KYMA_BTP_MANAGER_REPO}/releases/${RELEASE_ID}/assets"

echo -e "\n--- Updating GitHub release ${RELEASE_ID} with btp-manager.yaml and btp-operator-default-cr.yaml assets"

[[ ! -e ${MANIFEST_FILE} ]] && echo "::error ::Manifest file does not exist" && exit 1

uploadFile "${MANIFEST_FILE}" "${UPLOAD_URL}?name=btp-manager.yaml"

[[ ! -e ${DEFAULT_CR_FILE} ]] && echo "::error ::BTP operator CR does not exist" && exit 1

uploadFile "${DEFAULT_CR_FILE}" "${UPLOAD_URL}?name=btp-operator-default-cr.yaml"
