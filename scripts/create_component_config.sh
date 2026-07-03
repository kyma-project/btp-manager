#!/usr/bin/env bash

# Updates component-config.yaml with the given release tag.
# Usage: ./scripts/create_component_config.sh <release-tag>

TAG=${1}

# standard bash error handling
set -o nounset
set -o errexit
set -E
set -o pipefail

CONTROLLER_TAG=$(yq '.images[0].source' external-images.yaml | awk -F':' '{print $NF}')

echo "Updating component-config.yaml for release ${TAG}:"

cat <<EOF | tee component-config.yaml
name: kyma-project.io/kyma-runtime/kcp-components/btp-manager
team: kyma/gophers
images:
- europe-docker.pkg.dev/kyma-project/prod/btp-manager:${TAG}
- europe-docker.pkg.dev/kyma-project/prod/external/controller:${CONTROLLER_TAG}
EOF
