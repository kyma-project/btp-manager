#!/usr/bin/env bash

SKIP_ASSETS=$3

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# This script has the following arguments:
#                       - BTP Manager binary image tag - mandatory
#                       - BTP Operator OCI module image tag - mandatory
#                       --skip-template - optional
#
# ./await_artifacts.sh 1.1.0 v1.1.0

# Expected variables:
#             BTP_MANAGER_REPO - btp-operator binary image repository
#             BTP_OPERATOR_REPO - btp-operator OCI module image repository

export IMAGE_TAG=$1
export MODULE_TAG=$2

PROTOCOL=docker://

if [ "${SKIP_ASSETS}" != "--skip-templates" ]
then
  echo "Finding assets for: ${IMAGE_TAG}"
  curl -sL \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN"\
  https://api.github.com/repos/kyma-project/btp-manager/releases | jq --arg tag "${IMAGE_TAG}" '.[] | select(.tag_name == $ARGS.named.tag) | .assets[] | .browser_download_url | split("/") | last ' > $$.assets
  until grep "rendered.yaml"<$$.assets && grep "template.yaml"<$$.assets && grep "template_control_plane.yaml"<$$.assets; do
    echo 'waiting for the assets'
    sleep 10
    curl -sL \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GITHUB_TOKEN"\
    https://api.github.com/repos/kyma-project/btp-manager/releases | jq --arg tag "${IMAGE_TAG}" '.[] | select(.tag_name == $ARGS.named.tag) | .assets[] | .browser_download_url | split("/") | last ' > $$.assets
  done >/dev/null
  echo "assets available"
fi

until $(skopeo list-tags ${PROTOCOL}${BTP_OPERATOR_REPO} | jq '.Tags|any(. == env.MODULE_TAG)'); do
  echo "Waiting for BTP Operator OCI module image: ${BTP_OPERATOR_REPO}:${MODULE_TAG}"
  sleep 10
done

echo "BTP Operator OCI module image available"

until $(skopeo list-tags ${PROTOCOL}${BTP_MANAGER_REPO} | jq '.Tags|any(. == env.IMAGE_TAG)'); do
  echo "Waiting for BTP Manager binary image: ${BTP_MANAGER_REPO}:${IMAGE_TAG}"
  sleep 10
done

echo "BTP Manager binary image available"
