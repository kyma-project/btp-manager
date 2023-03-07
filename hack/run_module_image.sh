#!/usr/bin/env bash

# This script has the following arguments:
#     the mandatory link to a module image,
#     optional ci to indicate call from CI pipeline
# Example:
# ./run_module_image.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:v0.2.3

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

DOWNLOADED_DIR=downloaded_module

echo -e "\n--- Downloading module image"
rm -rf ${DOWNLOADED_DIR}
mkdir ${DOWNLOADED_DIR}

# tls setting to allow local access over http
if [ -z "$2" ]
then
  TLS_OPTIONS=--tls-verify=false
fi

skopeo copy ${TLS_OPTIONS} docker://$1 dir:${DOWNLOADED_DIR}

FILENAME=$(cat ${DOWNLOADED_DIR}/manifest.json  | jq -c '.layers[] | select(.mediaType=="application/gzip").digest[7:]' | tr -d \")

echo -e "\n--- Extracting resources from file:" ${FILENAME}

mkdir ${DOWNLOADED_DIR}/chart
tar -xzf ${DOWNLOADED_DIR}/${FILENAME} -C ${DOWNLOADED_DIR}/chart

echo -e "\n--- Installing BTP Manager"

# install by helm
helm upgrade --install btp-manager ${DOWNLOADED_DIR}/chart -n kyma-system --create-namespace