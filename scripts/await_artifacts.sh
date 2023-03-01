#!/usr/bin/env bash

SKIP_TEMPLATES=$3

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# This script has the following arguments:
#                       - BTP Manager binary image tag - mandatory
#                       - BTP Operator OCI module image tag - mandatory
#                       --skip-template - optional
#
# ./await_artifacts.sh 1.1.0 v1.1.0

# Expected variables:
#             BTP_MANAGER_REPO - btp-operator binary image repository
#             BTP_OPERATOR_REPO - btp-operator OCI module image repository

export IMAGE_TAG=$1
export MODULE_TAG=$2

PROTOCOL=docker://

TEMPLATE_URL="https://github.com/kyma-project/btp-manager/releases/download/${IMAGE_TAG}/template.yaml"

if [ "${SKIP_TEMPLATES}" != "--skip-templates" ]
then
  until $(curl --output /dev/null --silent --head --fail ${TEMPLATE_URL}); do
    echo 'waiting for template.yaml'
    sleep 10
  done
  echo "template.yaml available"
fi

until $(skopeo list-tags ${PROTOCOL}${BTP_OPERATOR_REPO} | jq '.Tags|any(. == env.MODULE_TAG)'); do
  echo "Waiting for BTP Operator OCI module image: ${BTP_OPERATOR_REPO}:${MODULE_TAG}"
  sleep 10
done

echo "BTP Operator OCI module image available"

until $(skopeo list-tags ${PROTOCOL}${BTP_MANAGER_REPO} | jq '.Tags|any(. == env.IMAGE_TAG)'); do
  echo "Waiting for BTP Manager binary image: ${BTP_MANAGER_REPO}:${IMAGE_TAG}"
  sleep 10
done

echo "BTP Manager binary image available"
