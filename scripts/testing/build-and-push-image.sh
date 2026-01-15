#!/usr/bin/env bash

# Script to build BTP Manager image locally and push it to k3s registry
# Usage: ./build-and-push-image.sh [VERSION]

VERSION=${1:-"PR-999"}
REGISTRY="localhost:5000"
IMAGE_NAME="btp-manager"

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

echo "Building BTP Manager image with version: ${VERSION}"
docker build -t "${REGISTRY}/${IMAGE_NAME}:${VERSION}" .

echo "Pushing ${IMAGE_NAME}:${VERSION} to registry..."
docker push ${REGISTRY}/${IMAGE_NAME}:${VERSION}

echo "${IMAGE_NAME}:${VERSION} built and pushed successfully"
echo "Images are available in k3s registry at ${REGISTRY}"
