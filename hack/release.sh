#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables:
#   PULL_BASE_REF - name of the tag
#   BOT_GITHUB_TOKEN - github token used to upload the template yaml

echo "PULL_BASE_REF ${PULL_BASE_REF}"

MODULE_VERSION=${PULL_BASE_REF} make module-build

echo "Generated template.yaml:"
cat template.yaml

echo "\nUpdating github release with template.yaml"

echo "\nFinding release: ${PULL_BASE_REF}"
releases=$(curl -sL \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $BOT_GITHUB_TOKEN"\
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/kyma-project/btp-manager/releases/tags/${PULL_BASE_REF})

release_id=$(echo $releases | jq -r '.id')
echo "Got release ID: ${release_id}"

GH_ASSET="https://uploads.github.com/repos/kyma-project/btp-manager/releases/${release_id}/assets?name=$(basename $filename)"

echo "Release asset url: ${GH_ASSET}"

response=$(curl -s -o output.txt -w "%{http_code}" \
                --request POST --data-binary @"$filename" \
                -H "Authorization: token $BOT_GITHUB_TOKEN" \
                -H "Content-Type: text/yaml" \
                 $GH_ASSET)

if [[ "$response" != "201" ]]; then
  echo "Unable to upload the asset: "
  echo "HTTP Status: $response"
  cat output.txt
  exit 1
else
  echo "template.yaml uploaded"
fi

