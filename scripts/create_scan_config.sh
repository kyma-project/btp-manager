#!/usr/bin/env bash

# This script has the following arguments:
#                       binary image reference (mandatory)
#                       filename of file to be created (optional)
#                       release tag (optional)
# ./create_scan_config image temp_scan_congitfig.yaml          - use when building module image
# ./create_scan_config image temp_scan_congitfig.yaml tag      - use when bumping the config on the main branch

FILENAME=${2-../sec-scanners-config.yaml}
TAG=${3:-}

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being maskedPORT=5001

IMAGE=${1}
echo "Creating security scan configuration file:"

# add rc-tag when creating the config on the main branch
if [ -n "${TAG}" ]; then
  cat <<EOF | tee ${FILENAME}
module-name: btp-operator
rc-tag: ${TAG}
protecode:
  - ${IMAGE}
whitesource:
  language: golang-mod
  subprojects: false
  exclude:
    - "**/*_test.go"
EOF
else
  cat <<EOF | tee ${FILENAME}
module-name: btp-operator
protecode:
  - ${IMAGE}
whitesource:
  language: golang-mod
  subprojects: false
  exclude:
    - "**/*_test.go"
EOF
fi
