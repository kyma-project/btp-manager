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

echo "PULL_BASE_REF ${PULL_BASE_REF}"

MODULE_VERSION=${PULL_BASE_REF} make module-build

echo "Generated template.yaml:"
cat template.yaml

sed 's/target: remote/target: control-plane/g' <template.yaml >template_control_plane.yaml

echo "Generated template_control_plane.yaml:"
cat template_control_plane.yaml

echo "Updating github release with template.yaml, template_control_plane.yaml, rendered.yaml"

echo "Finding release: ${PULL_BASE_REF}"
releases=$(curl -sL \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $BOT_GITHUB_TOKEN"\
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/kyma-project/btp-manager/releases/tags/${PULL_BASE_REF})

release_id=$(echo $releases | jq -r '.id')
echo "Got release ID: ${release_id}"

TEMPLATE_GH_ASSET="https://uploads.github.com/repos/kyma-project/btp-manager/releases/${release_id}/assets?name=template.yaml"

uploadFile "template.yaml" $TEMPLATE_GH_ASSET

TEMPLATE_CONTROL_PLANE_GH_ASSET="https://uploads.github.com/repos/kyma-project/btp-manager/releases/${release_id}/assets?name=template_control_plane.yaml"

uploadFile "template_control_plane.yaml" $TEMPLATE_CONTROL_PLANE_GH_ASSET

RENDERED_GH_ASSET="https://uploads.github.com/repos/kyma-project/btp-manager/releases/${release_id}/assets?name=rendered.yaml"

uploadFile "charts/btp-operator/templates/rendered.yaml" $RENDERED_GH_ASSET

