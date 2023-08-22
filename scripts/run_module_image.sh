#!/usr/bin/env bash

# This script has the following arguments:
#     the mandatory link to a module image,
#     optional ci to indicate call from CI pipeline
# Example:
# ./run_module_image.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:v0.2.3 ci

CI=${2-manual}  # if called from any workflow "ci" is expected here

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

IMAGE_NAME=$1
TARGET_DIRECTORY=${TARGET_DIRECTORY:-downloaded_module}
CHART_DIRECTORY=${CHART_DIRECTORY:-chart}
NAMESPACE=${NAMESPACE:-kyma-system}

# for local runs
rm -rf ${TARGET_DIRECTORY}

echo -e "\n--- Downloading module image: ${IMAGE_NAME}"
mkdir ${TARGET_DIRECTORY}

# tls setting to allow local access over http, when invoked from CI https is used
TLS_OPTIONS=
if [ "${CI}" != "ci" ]
then
  TLS_OPTIONS=--src-tls-verify=false
fi

skopeo copy ${TLS_OPTIONS} docker://${IMAGE_NAME} dir:${TARGET_DIRECTORY}

FILENAME=$(./scripts/check_if_k8s_yaml.sh)
echo $FILENAME
if [[ -n "$FILENAME" ]]; 
then
  echo -e "\n--- Installing BTP Manager in ${NAMESPACE} namespace using kubectl apply"

  kubectl apply -f ${TARGET_DIRECTORY}/${FILENAME}
else
  FILENAME=$(cat ${TARGET_DIRECTORY}/manifest.json  | jq -c '.layers[] | select(.mediaType=="application/gzip").digest[7:]' | tr -d \")
  echo -e "\n--- Extracting resources from file:" ${FILENAME}

  mkdir ${TARGET_DIRECTORY}/$CHART_DIRECTORY
  tar xzvf ${TARGET_DIRECTORY}/${FILENAME} -C ${TARGET_DIRECTORY}/${CHART_DIRECTORY}
  echo -e "\n--- Installing BTP Manager in ${NAMESPACE} namespace using helm"

  # install by helm
  helm upgrade --install btp-manager ${TARGET_DIRECTORY}/${CHART_DIRECTORY} -n ${NAMESPACE} --create-namespace
fi
