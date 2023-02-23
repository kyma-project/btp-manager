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

# for local runs
rm -rf ${TARGET_DIRECTORY}

TLS_OPTIONS=
TARGET_DIRECTORY=${TARGET_DIRECTORY:=downloaded_module}
DOWNLOADED_DIR=downloaded_module

echo -e "\n--- Downloading module image"

mkdir ${TARGET_DIRECTORY}

# tls setting to allow local access over http
TLS_OPTIONS=
if [ $# -lt 2 ]
then
  TLS_OPTIONS=--src-tls-verify=false
fi

skopeo copy ${TLS_OPTIONS} docker://$1 dir:${TARGET_DIRECTORY}

FILENAME=$(cat ${TARGET_DIRECTORY}/manifest.json  | jq -c '.layers[] | select(.mediaType=="application/gzip").digest[7:]' | tr -d \")

echo -e "\n--- Extracting resources from file:" ${FILENAME}

mkdir ${TARGET_DIRECTORY}/chart
tar -xzf ${TARGET_DIRECTORY}/${FILENAME} -C ${TARGET_DIRECTORY}/chart

echo -e "\n--- Installing BTP Manager"

# install by helm
helm upgrade --install btp-manager ${TARGET_DIRECTORY}/chart -n kyma-system --create-namespace