#!/usr/bin/env bash

# The script uses the following environment variables (if not provided script defines default values):
#                       LOCAL_REGISTRY
#                       K3D_REGISTRY
#                       PR_NAME
# ./build_push_deploy.sh

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being maskedPORT=5001

# registry access from local machine
LOCAL_REGISTRY=${LOCAL_REGISTRY:-localhost:5001}

# registry access from k3d cluster
K3D_REGISTRY=${K3D_REGISTRY:-k3d-kyma-registry:5000}

PR_NAME=${PR_NAME:-PR-undefined}
IMG_NAME=btp-manager:${PR_NAME}

echo "Creating binary image and pushing to registry: ${LOCAL_REGISTRY}"
make module-image IMG=${LOCAL_REGISTRY}/${IMG_NAME}

echo "Creating prerequisites"
kubectl apply -f deployments/prerequisites.yaml

echo "Deploying binary image from local registry: ${K3D_REGISTRY}"
make deploy IMG=${K3D_REGISTRY}/${IMG_NAME}


