#!/usr/bin/env bash

# This script has the following arguments:
#     the mandatory link to a module image,
#     optional ci to indicate call from CI pipeline
# Example:
# ./run_module_image.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:v0.2.3

CI=$2

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

echo -e "\n--- Downloading module image"
mkdir ${TARGET_DIRECTORY}

# tls setting to allow local access over http, when invoked from CI https is used
TLS_OPTIONS=
if [ -z "${CI}" ]
then
  TLS_OPTIONS=--src-tls-verify=false
fi

skopeo copy ${TLS_OPTIONS} docker://${IMAGE_NAME} dir:${TARGET_DIRECTORY}

FILENAME=$(cat ${TARGET_DIRECTORY}/manifest.json  | jq -c '.layers[] | select(.mediaType=="application/gzip").digest[7:]' | tr -d \")

# if no application/gzip files found that means that we have a raw Kubernetes YAML file and we need to identify it before apply
if [ -z "${FILENAME}" ]
then
  POTENTIAL_YAML_FILES=$(cat ${TARGET_DIRECTORY}/manifest.json  | jq -c '.layers[] | select(.mediaType=="application/octet-stream").digest[7:]' | tr -d \")
  POTENTIAL_YAMLS_ARRAY=($POTENTIAL_YAML_FILES)

  for RAW_K8S_YAML in "${POTENTIAL_YAMLS_ARRAY[@]}"
  do
    KIND_VALUE=$(cat ${TARGET_DIRECTORY}/${RAW_K8S_YAML} | yq e '.kind' -)    
    if echo "$KIND_VALUE" | grep -q "CustomResourceDefinition"; 
    then
      echo -e "\n--- Installing BTP Manager in ${NAMESPACE} namespace"

      kubectl apply -f ${TARGET_DIRECTORY}/${RAW_K8S_YAML}
      break
    fi
  done
else
  echo -e "\n--- Extracting resources from file:" ${FILENAME}

  mkdir ${TARGET_DIRECTORY}/$CHART_DIRECTORY
  tar xzvf ${TARGET_DIRECTORY}/${FILENAME} -C ${TARGET_DIRECTORY}/${CHART_DIRECTORY}
  echo -e "\n--- Installing BTP Manager in ${NAMESPACE} namespace"

  # install by helm
  helm upgrade --install btp-manager ${TARGET_DIRECTORY}/${CHART_DIRECTORY} -n ${NAMESPACE} --create-namespace
fi