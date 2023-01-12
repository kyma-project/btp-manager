#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked


get_new_release_version() {
    # get the list of tags in a reverse chronological order
    TAG_LIST=($(git tag --sort=-creatordate))
    NEW_RELEASE_VERSION=${TAG_LIST[0]}
}

get_current_release_version() {
    # get the list of tags in a reverse chronological order excluding release candidate tags
    TAG_LIST_WITHOUT_RC=($(git tag --sort=-creatordate | grep -v -e "-rc"))
    if [[ $NEW_RELEASE_VERSION == *"-rc"* ]]; then
        CURRENT_RELEASE_VERSION=${TAG_LIST_WITHOUT_RC[0]}
    else
        CURRENT_RELEASE_VERSION=${TAG_LIST_WITHOUT_RC[1]}
    fi
}


if [[ "${1-}" == "ci" ]]; then
      git remote add origin git@github.com:kyma-project/cli.git
fi

get_new_release_version

echo "Preparing release ${NEW_RELEASE_VERSION}"

MODULE_VERSION=${NEW_RELEASE_VERSION} make docker-build docker-push module-build
