#!/usr/bin/env bash

# Updates component-config.yaml with the given release tag.
# Usage: ./scripts/create_component_config.sh <release-tag>

# standard bash error handling
set -o nounset
set -o errexit
set -E
set -o pipefail

TAG=${1}

CONTROLLER_SOURCE=$(yq '.images[] | select(.source | contains("sap-btp-service-operator")) | .source' external-images.yaml)

echo "Updating component-config.yaml for release ${TAG}:"

cat <<EOF | tee component-config.yaml
name: kyma-project.io/kyma-runtime/kcp-components/btp-manager
team: kyma/gophers
images:
- europe-docker.pkg.dev/kyma-project/prod/btp-manager:${TAG}
- europe-docker.pkg.dev/kyma-project/prod/external/${CONTROLLER_SOURCE}
EOF
