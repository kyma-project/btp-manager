#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# This script has the following arguments:
#                       - BTP Manager binary image tag - mandatory
#
# ./check_artifacts_existence.sh 1.1.0

# Expected variables:
#             BTP_MANAGER_REPO - btp-operator binary image repository
#             GITHUB_TOKEN - github token

export IMAGE_TAG=$1

PROTOCOL=docker://

if [ $(skopeo list-tags ${PROTOCOL}${BTP_MANAGER_REPO} | jq '.Tags|any(. == env.IMAGE_TAG)') == "true" ]
then
  echo "::warning ::BTP Manager binary image for tag ${IMAGE_TAG} already exists"
else
  echo "No previous BTP Manager binary image found for tag ${IMAGE_TAG}"
fi
