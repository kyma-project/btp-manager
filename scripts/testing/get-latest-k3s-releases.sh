#!/usr/bin/env bash

LIST_LEN=${1:-5}

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked


REPOSITORY=k3s-io/k3s
GITHUB_URL=https://api.github.com/repos/${REPOSITORY}


LATEST_RELEASES=($(curl -sS "${GITHUB_URL}/releases" | jq '.[] | select(.name|test("v[0-9]{1,2}.[0-9]{1,3}.[0-9]{1,3}\\+")) | .name' | sort -V| tail -${LIST_LEN}))

echo ${LATEST_RELEASES[*]}

