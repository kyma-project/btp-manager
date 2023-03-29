#! /bin/bash

RELEASE_TAG=$1

REPOSITORY='kyma-project/btp-manager'
LATEST_RELEASE_TAG=$(curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/repos/$REPOSITORY/releases/latest | jq -r '.tag_name')
COMMITS_SINCE_LATEST_RELEASE=($(git log $LATEST_RELEASE_TAG..HEAD --pretty=format:"%h"))

for line in "${COMMITS_SINCE_LATEST_RELEASE[@]}"; do
    git log $LATEST_RELEASE_TAG..HEAD --pretty=format:"* %s by @ %h" | grep "$line" | awk '{$NF=""; print $0}' | tr -d "\n" | sed 's/.$//' >> CHANGELOG.txt
    curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/repos/$REPOSITORY/commits/$line | jq -r '.author.login' >> CHANGELOG.txt
done
echo "## What's Changed" >> CHANGELOG.md
if [ -e CHANGELOG.txt ]; then
    tac CHANGELOG.txt >> CHANGELOG.md
fi

CONTRIBUTORS_BEFORE_NEW_RELEASE=()
while IFS= read -r line; do
    CONTRIBUTORS_BEFORE_NEW_RELEASE+=("$line")
done < <(curl -H "Authorization: token $GITHUB_TOKEN" -s "https://api.github.com/repos/$REPOSITORY/compare/$(git rev-list --max-parents=0 HEAD)...$LATEST_RELEASE_TAG" | jq -r '.commits[].author.login' | sort -u)

RELEASE_CONTRIBUTORS=()
while IFS= read -r line; do
    RELEASE_CONTRIBUTORS+=("$line")
done < <(curl -H "Authorization: token $GITHUB_TOKEN" -s "https://api.github.com/repos/$REPOSITORY/compare/$LATEST_RELEASE_TAG...HEAD" | jq -r '.commits[].author.login' | sort -u)

NEW_CONTRIBUTORS=false
    for i in "${RELEASE_CONTRIBUTORS[@]}"; do
        if [[ ! " ${CONTRIBUTORS_BEFORE_NEW_RELEASE[@]} " =~ " ${i} " ]]; then
            if [ "$NEW_CONTRIBUTORS" = false ] ; then
                echo -e "\n## New Contributors" >> CHANGELOG.md
                NEW_CONTRIBUTORS=true
            fi
            echo "* @$i made their first contribution in " | tr -d "\n" >> CHANGELOG.md
            grep @$i CHANGELOG.md | head -1 | grep -o " (#[0-9]\+)" >> CHANGELOG.md
        fi
    done
echo -e "\n**Full Changelog**: https://github.com/$REPOSITORY/compare/$LATEST_RELEASE_TAG...$RELEASE_TAG" >> CHANGELOG.md