#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables:
#   PULL_BASE_REF - name of the tag
#   BOT_GITHUB_TOKEN - github token used to upload the template yaml

uploadFile() {
  filePath=${1}
  ghAsset=${2}

  response=$(curl -s -o output.txt -w "%{http_code}" \
                  --request POST --data-binary @"$filePath" \
                  -H "Authorization: token $BOT_GITHUB_TOKEN" \
                  -H "Content-Type: text/yaml" \
                   $ghAsset)
  if [[ "$response" != "201" ]]; then
    echo "Unable to upload the asset ($filePath): "
    echo "HTTP Status: $response"
    cat output.txt
    exit 1
  else
    echo "$filePath uploaded"
  fi
}

IMG=${IMG_REGISTRY}/btp-manager:$PULL_BASE_REF
echo "Referring to image: ${IMG}"

## prepare security scanning configuration for the module template
SCAN_CONFIG_FILE=module_scanners_config.yaml
scripts/create_scan_config.sh ${IMG} ${SCAN_CONFIG_FILE}

MODULE_VERSION=${PULL_BASE_REF} SECURITY_SCAN_OPTIONS="--sec-scanners-config ${SCAN_CONFIG_FILE}" make module-build

rm -rf ${SCAN_CONFIG_FILE}

echo "Updating github release with btp-manager.yaml, btp-operator-default-cr.yaml"

echo "Finding release id for: ${PULL_BASE_REF}"
CURL_RESPONSE=$(curl -w "%{http_code}" -sL \
                -H "Accept: application/vnd.github+json" \
                -H "Authorization: Bearer $BOT_GITHUB_TOKEN"\
                https://api.github.com/repos/kyma-project/btp-manager/releases)

JSON_RESPONSE=$(sed '$ d' <<< "${CURL_RESPONSE}")
HTTP_CODE=$(tail -n1 <<< "${CURL_RESPONSE}")
[[ "${HTTP_CODE}" != "200" ]] && echo "${JSON_RESPONSE}" && exit 1

RELEASE_ID=$(jq <<< ${JSON_RESPONSE} --arg tag "${PULL_BASE_REF}" '.[] | select(.tag_name == $ARGS.named.tag) | .id')

[[ -z "${RELEASE_ID}" ]] && echo "No release with tag: ${PULL_BASE_REF}" && exit 1

UPLOAD_URL="https://uploads.github.com/repos/kyma-project/btp-manager/releases/${RELEASE_ID}/assets"

[[ ! -e "manifests/btp-operator/btp-manager.yaml" ]] && echo "Manifest file does not exist" && exit 1

uploadFile "manifests/btp-operator/btp-manager.yaml" "${UPLOAD_URL}?name=btp-manager.yaml"

[[ ! -e "examples/btp-operator.yaml" ]] && echo "BTP operator CR does not exist" && exit 1

uploadFile "examples/btp-operator.yaml" "${UPLOAD_URL}?name=btp-operator-default-cr.yaml"
