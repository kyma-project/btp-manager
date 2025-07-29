#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# This script accepts a single argument:
#   1. image repository with tag (e.g., repo/image:tag)
#
# Usage:
#   ./await_image.sh <image>
#
# Expected variables:
#   GITHUB_TOKEN - github token

PROTOCOL=docker://

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <image>"
  exit 1
fi

IMAGE_REPO_WITH_TAG="$1"
IMAGE_REPO="${IMAGE_REPO_WITH_TAG%:*}"
IMAGE_TAG="${IMAGE_REPO_WITH_TAG##*:}"

until $(skopeo list-tags ${PROTOCOL}${IMAGE_REPO} | jq --arg IMAGE_TAG "$IMAGE_TAG" '.Tags|any(. == $IMAGE_TAG)'); do
  echo "Waiting for binary image: ${IMAGE_REPO}:${IMAGE_TAG}"
  sleep 10
done

echo "Binary image: ${IMAGE_REPO}:${IMAGE_TAG} available"
