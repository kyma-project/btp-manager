#!/usr/bin/env bash

# This script has the following argument: ci, for example:
# ./create_module_image.sh ci

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables:
#   PULL_NUMBER - PR number

echo "PULL_NUMBER: ${PULL_NUMBER}"

echo "Creating module image - TBD"


