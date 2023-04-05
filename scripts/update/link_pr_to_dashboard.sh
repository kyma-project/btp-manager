#!/usr/bin/env bash
# link PR it to Gopher dashboard

pr_link=$1

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked# link the PR from ^^ to gopher project board

# Expected variables passed (passed from CI via calling script):
#   GH_TOKEN                      - GitHub token for GitHub CLI

pr_number=$(echo "${pr_link}" | awk -F '/' '{print($NF)}')
pr_id=$(gh api repos/kyma-project/btp-manager/pulls/"${pr_number}" | jq -r '.node_id')

# Gopher board node_id
project_board_id=PVT_kwDOAlVvc84AEv0v

# "To Do" column on Gopher board node_id
todo_column_id=834c7033

# order in "To Do" column on Gopher board node_id
status_field=PVTSSF_lADOAlVvc84AEv0vzgCvCtY

# insert projectv2 item (card on the gopher board)
resp=$(gh api graphql -f query='mutation{ addProjectV2ItemById(input:{projectId: "'${project_board_id}'" contentId: "'${pr_id}'"}){ item{id} }}' )
echo "response from inserting projectv2 item: $resp"
card_id=$(echo "$resp" | jq -r '.data.addProjectV2ItemById.item.id')

# move projectv2 item (card on the gopher board) to the top of the "todo" column
# due to GitHub internal GraphQL limitation, adding item and update has to be two separate calls
# https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects#updating-projects
gh api graphql -f query="$(cat << EOF
  mutation {
    set_status: updateProjectV2ItemFieldValue(input: {
      projectId: "$project_board_id"
      itemId: "$card_id"
      fieldId: "$status_field"
      value: {
        singleSelectOptionId: "$todo_column_id"
      }
    }){projectV2Item {id}}
    set_position: updateProjectV2ItemPosition(input: {
      projectId: "$project_board_id"
      itemId: "$card_id"
    }){items {totalCount}}
  }
EOF
)"
