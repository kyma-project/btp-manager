#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

#From Github API Docs:
#   You can use the REST API to create comments on issues and pull requests. Every pull request is an issue, but not every issue is a pull request.

PR_ID=$1

supported_labels=()

help_message="**Add one of following label** <br/><br/>"

while IFS= read -r label; do
  label_part=$(echo "$label" | cut -d "#" -f 1); help_message_part=$(echo "$label" | cut -d "#" -f 2)
  help_message="${help_message} - $label_part -> $help_message_part <br/><br/>"
  supported_labels+=("$label_part")
done <<< "$(yq eval '.changelog.categories.[].labels' ./.github/release.yml | grep "\- kind"| sed -e 's/- //g')"


comments=$(curl -sL \
            -H "Accept: application/vnd.github+json" \
            -H "X-GitHub-Api-Version: 2022-11-28" \
            https://api.github.com/repos/ukff/btp-manager/issues/${PR_ID}/comments |
            jq -r '.[] | objects | .body')

if [[ ! " ${comments[*]} " =~ " ${help_message} " ]]; then

  payload=$(jq -n \
    --arg body "$help_message" \
    '{
      "body": $body,
    }') 

  http_code=$(curl -sL \
              -w "%{http_code}" --output /dev/null \
              -X POST \
              -H "Accept: application/vnd.github+json" \
              -H "Authorization: Bearer $GITHUB_TOKEN" \
              -H "X-GitHub-Api-Version: 2022-11-28" \
              https://api.github.com/repos/ukff/btp-manager/issues/${PR_ID}/comments \
              -d "$payload")
  if [[ "$http_code" != "201" ]]; then
    echo "Unable to create comment with help text. $http_code"
    exit 1
  fi
fi

present_labels=$(curl -sL \
                  -H "Accept: application/vnd.github+json" \
                  -H "X-GitHub-Api-Version: 2022-11-28" \
                  https://api.github.com/repos/ukff/btp-manager/issues/${PR_ID} | 
                  jq -r '.labels[] | objects | .name')

count_of_required_labels=$(grep -o -w -F -c "${supported_labels[*]}" <<< "$present_labels")
if [[ $count_of_required_labels -eq 1 ]]; then 
  echo "label validation OK"
  exit 0
fi

echo "error: only 1 of following labels must be added to each PR before merge but found $count_of_required_labels:"
echo "${supported_labels[@]}"
exit 1