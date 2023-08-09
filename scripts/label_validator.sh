#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

PR_ID=$1

kind_labels=()

relase_notes_supported_labels=$(yq eval '.changelog.categories.[].labels' ./.github/release.yml)
while IFS= read -r label; do
  clean_label=$(echo "$label" | sed -e 's/-//g ; s/ //g')
  if [[ $clean_label == kind* ]]; then
    kind_labels+=("$clean_label")
  fi
done <<< "$relase_notes_supported_labels"

msg="Please use one of following labels for this PR: ${kind_labels[*]}"

comments=$(curl -L \
            -H "Accept: application/vnd.github+json" \
            -H "X-GitHub-Api-Version: 2022-11-28" \
            https://api.github.com/repos/ukff/btp-manager/pulls/comments |
            jq -r '.[] | objects | .body')

if [[ ! " ${comments[*]} " =~ " ${msg} " ]]; then
  response=$(curl -L \
              -X POST \
              -H "Accept: application/vnd.github+json" \
              -H "Authorization: Bearer $GITHUB_TOKEN" \
              -H "X-GitHub-Api-Version: 2022-11-28" \
              https://api.github.com/repos/ukff/btp-manager/pulls/${PR_ID}/comments \
              -d "{'body':'$msg'}")
  echo "$response"
fi

present_labels=$(curl -L \
                  -H "Accept: application/vnd.github+json" \
                  -H "X-GitHub-Api-Version: 2022-11-28" \
                  https://api.github.com/repos/ukff/btp-manager/pulls/${PR_ID} | 
                  jq -r '.labels[] | objects | .name')

count_of_required_labels=0
while IFS= read -r label; do
    if [[ " ${kind_labels[*]} " =~ " ${label} " ]]; then
      echo "found label: ${label}"
      ((count_of_required_labels=count_of_required_labels+1))
    fi
done <<< "$present_labels"

if [[ $count_of_required_labels -eq 1 ]]; then 
  echo "label validation OK"
  exit 0
fi

echo "error: only 1 of following labels must be added to each PR before merge but found $count_of_required_labels:"
echo "${kind_labels[@]}"
exit 1