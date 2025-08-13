#!/usr/bin/env bash

# This script updates the channel mapping configuration file

[[ -z "$CHANNEL" || -z "$TAG" ]] && echo "Error: CHANNEL and TAG environment variables must be set." && exit 1

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables:
#             CHANNEL - fast or regular channel
#             TAG - version to be set for the channel

CHANNEL_MAPPING_FILENAME=$1

echo "Updating channel mapping configuration file: ${CHANNEL_MAPPING_FILENAME} for channel: ${CHANNEL} with tag: ${TAG}"

if [ -f "${CHANNEL_MAPPING_FILENAME}" ]; then
  echo "File exists, updating it"
  yq -i '.channels |= map(select(.channel == env(CHANNEL)).version = env(TAG))' ${CHANNEL_MAPPING_FILENAME}
  cat ${CHANNEL_MAPPING_FILENAME}
else
  echo "File does not exist, §creating it"
  cat <<EOF | tee ${CHANNEL_MAPPING_FILENAME}
channels:
 - channel: regular
   version: ${TAG}
 - channel: fast
   version: ${TAG}
EOF
fi
