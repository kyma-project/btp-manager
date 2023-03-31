#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

BTP_OPERATOR_REPO=https://api.github.com/repos/SAP/sap-btp-service-operator

curl -sL -H "Accept: application/vnd.github+json" ${BTP_OPERATOR_REPO}/releases/latest | jq -r '.tag_name'