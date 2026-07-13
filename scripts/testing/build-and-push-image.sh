#!/usr/bin/env bash

# Script to build an image locally and push it to k3s registry
# Usage: ./build-and-push-image.sh [IMAGE_NAME] [DOCKERFILE]
# DOCKERFILE defaults to Dockerfile in the current directory

IMAGE_NAME=$1
DOCKERFILE=${2:-Dockerfile}

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

echo "Building ${IMAGE_NAME}"
docker build -t "${IMAGE_NAME}" -f "${DOCKERFILE}" .

echo "Pushing ${IMAGE_NAME} to registry..."
docker push "${IMAGE_NAME}"

echo "${IMAGE_NAME} built and pushed successfully"
