#!/usr/bin/env bash


# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

PR_NUMBER=$1

timeout 5m bash -c "
  until $(gh pr view ${PR_NUMBER} --json closed | jq -r '.closed'); do
    echo 'Waiting for PR #${PR_NUMBER} to be merged'
    sleep 10
  done
"