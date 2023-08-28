#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

   
  supported_labels=$(yq eval '.changelog.categories.[].labels' ./.github/release.yml | grep "\- kind"| sed -e 's/- //g' | cut -d "#" -f 1)
  #supported_labels=$(echo "${supported_labels[*]}" | tr " " "\n" )

    present_labels=$(curl -sL \
                    -H "Accept: application/vnd.github+json" \
                    -H "X-GitHub-Api-Version: 2022-11-28" \
                    https://api.github.com/repos/ukff/btp-manager/issues/101 | 
                    jq -r 'if (.labels | length) > 0 then .labels[] | objects | .name else empty end')
    
    echo $present_labels
    echo $supported_labels

    count_of_required_labels=$(grep -o -w -F -c -s "${supported_labels}" <<< "$present_labels" || true)
    if [[ $count_of_required_labels -eq 0 ]]; then 
      echo "PR 101 dosent have any /kind label"
    fi
    echo "end"