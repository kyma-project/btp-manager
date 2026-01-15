#!/usr/bin/env bash

# Script to build BTP Manager image locally and push it to k3s registry
# Usage: ./build-and-push-image.sh [IMAGE_NAME]

IMAGE_NAME=$1

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

echo "Building ${IMAGE_NAME}"
docker build -t "${IMAGE_NAME}" .

echo "Pushing ${IMAGE_NAME} to registry..."
docker push "${IMAGE_NAME}"

echo "${IMAGE_NAME} built and pushed successfully"
