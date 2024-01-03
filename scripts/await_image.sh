#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# This script has the following arguments:
#                       - BTP Manager binary image tag - mandatory
#
# ./await_image.sh 1.1.0

# Expected variables:
#             BTP_MANAGER_REPO - btp-operator binary image repository
#             GITHUB_TOKEN - github token


export IMAGE_TAG=$1

PROTOCOL=docker://

until $(skopeo list-tags ${PROTOCOL}${BTP_MANAGER_REPO} | jq '.Tags|any(. == env.IMAGE_TAG)'); do
  echo "Waiting for BTP Manager binary image: ${BTP_MANAGER_REPO}:${IMAGE_TAG}"
  sleep 10
done

echo "BTP Manager binary image available"
