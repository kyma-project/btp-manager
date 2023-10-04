#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables:
#             KYMA_BTP_MANAGER_REPO - Kyma repository
#             PR_NUMBER - Number of the PR with the changes to be merged


until  $(gh pr view ${PR_NUMBER} --json closed | jq -r '.closed'); do
  echo "Waiting for https://github.com/${KYMA_BTP_MANAGER_REPO}/pull/${PR_NUMBER} to be merged"
  sleep 5
done
