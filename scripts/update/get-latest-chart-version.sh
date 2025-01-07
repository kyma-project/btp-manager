#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

curl -sL -H "Accept: application/vnd.github+json" https://api.github.com/repos/sap/sap-btp-service-operator/releases/latest | jq -r '.tag_name'