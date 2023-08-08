#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

PR_ID=$1

REQUIRED_LABELS=("kind/feature" "kind/enhancement" "kind/bug")

present_labels=$(curl -L \
                -H "Accept: application/vnd.github+json" \
                -H "X-GitHub-Api-Version: 2022-11-28" \
                https://api.github.com/repos/ukff/btp-manager/pulls/${PR_ID} | 
                jq -r '.labels[] | objects | .name')

count_of_required_labels=0
while IFS= read -r label; do
    if [[ " ${REQUIRED_LABELS[*]} " =~ " ${label} " ]]; then
      echo "found label: ${label}"
      ((count_of_required_labels=count_of_required_labels+1))
    fi
done <<< "$present_labels"

if [[ $count_of_required_labels -eq 1 ]]; then 
  echo "label validation OK"
  exit 0
fi

echo "only one of following labels must be added to each PR before merge:"
echo "${REQUIRED_LABELS[@]}"
exit 1