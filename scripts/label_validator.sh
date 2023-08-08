#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

labels=("kind/feature" "kind/enhancement" "kind/bug")

PR_ID=$1

current_labels=$(curl -L \
                -H "Accept: application/vnd.github+json" \
                -H "X-GitHub-Api-Version: 2022-11-28" \
                https://api.github.com/repos/kyma-project/btp-manager/pulls/${PR_ID} | 
                jq -r '.labels[] | objects | .name')

while IFS= read -r label; do
    if [[ " ${labels[*]} " =~ " ${label} " ]]; then
      echo "found label: ${label}"
      exit 0
    fi
done <<< "$current_labels"

echo "one of following labels must be added to each PR before merge:"
echo "${labels[@]}"
exit 1