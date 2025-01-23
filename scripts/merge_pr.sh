#!/usr/bin/env bash

# This script merges a pull request

# Expected variables:
#             PR_NUMBER - Number of the PR with the changes to be merged
#             GH_TOKEN  - GitHub token for GitHub CLI
#             REPOSITORY - Repository where the PR is located

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

GITHUB_URL=https://api.github.com/repos/${REPOSITORY}
GITHUB_AUTH_HEADER="Authorization: Bearer ${GH_TOKEN}"

CURL_RESPONSE=$(curl -L \
  -X PUT \
  -H "Accept: application/vnd.github+json" \
  -H "${GITHUB_AUTH_HEADER}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  ${GITHUB_URL}/pulls/${PR_NUMBER}/merge \
  -d '{"merge_method": "squash"}')