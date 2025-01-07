#!/usr/bin/env bash

LIST_LEN=${1:-5}

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked


REPOSITORY=k3s-io/k3s
GITHUB_URL=https://api.github.com/repos/${REPOSITORY}

# selecting at most ${LIST_LEN} recent minor version releases with maximal patch number for the given minor
# Example
#    given:
#             "v1.24.17+k3s1"
#             "v1.25.13+k3s1"
#             "v1.25.14+k3s1"
#             "v1.26.8+k3s1"
#             "v1.26.9+k3s1"
#             "v1.27.5+k3s1"
#             "v1.27.6+k3s1"
#             "v1.28.1+k3s1"
#             "v1.28.2+k3s1"
#   outputs:
#             "v1.25.14+k3s1" "v1.26.9+k3s1" "v1.27.6+k3s1" "v1.28.1+k3s1"
LATEST_RELEASES=($(curl -sS "${GITHUB_URL}/releases" \
| jq '.[] | select(.prerelease == false) | select(.name|test("v[0-9]{1,2}.[0-9]{1,3}.[0-9]{1,3}\\+")) | .name' \
| sort -rV\
| awk -F. '/k3s1/ {if ($2 != p) {print $0; p=$2}}' \
| head -n ${LIST_LEN}))

echo ${LATEST_RELEASES[*]}
