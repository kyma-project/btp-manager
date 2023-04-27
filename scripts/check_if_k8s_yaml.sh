#!/usr/bin/env bash

# This script returns the name of the raw Kubernetes YAML if present

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

TARGET_DIRECTORY=${TARGET_DIRECTORY:-downloaded_module}

FILENAME=$(cat ${TARGET_DIRECTORY}/manifest.json  | jq -c '.layers[] | select(.mediaType=="application/gzip").digest[7:]' | tr -d \")

# if no application/gzip files found that means that we have a raw Kubernetes YAML file and we need to identify it
if [ -z "${FILENAME}" ]
then
  POTENTIAL_YAML_FILES=$(cat ${TARGET_DIRECTORY}/manifest.json  | jq -c '.layers[] | select(.mediaType=="application/octet-stream").digest[7:]' | tr -d \")
  POTENTIAL_YAMLS_ARRAY=($POTENTIAL_YAML_FILES)

  for RAW_K8S_YAML in "${POTENTIAL_YAMLS_ARRAY[@]}"
  do
    KIND_VALUE=$(cat ${TARGET_DIRECTORY}/${RAW_K8S_YAML} | yq e '.kind' -)    
    if echo "$KIND_VALUE" | grep -q "CustomResourceDefinition"; 
    then
      echo "${RAW_K8S_YAML}"
    fi
  done
fi
