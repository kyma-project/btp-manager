#!/usr/bin/env bash
# move changes to the dedicated branch created from the remote main and create link on the Gopher dashboard

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked# link the PR from ^^ to gopher project board

# Expected variables passed (passed from CI):
#   GH_TOKEN                      - GitHub token for GitHub CLI
#   GIT_EMAIL                     - email setting for PR to be created
#   GIT_NAME                      - user name setting for PR to be created
#   KYMA_BTP_MANAGER_REPO         - Kyma repository
#   SAP_BTP_SERVICE_OPERATOR_REPO - upstream repository with new chart versions
#   BRANCH_NAME                   - branch with updated resources
#   TAG                           - new chart version

# add changed files to stage
git add module-chart/*
git add module-resources/*
git add controllers/btpoperator_controller.go
git add config/rbac/role.yaml

#stash staged changes
git stash push --staged

#pass changes to branch created from main
git checkout --force -B main refs/remotes/origin/main
git checkout -B ${BRANCH_NAME}

#apply stashed changes
git stash apply
git add module-chart/*
git add module-resources/*
git add controllers/btpoperator_controller.go
git add config/rbac/role.yaml

#configure git
git config --global user.email ${GIT_EMAIL}
git config --global user.name ${GIT_NAME}

#commit and push changes
git commit -m "$MSG"
git remote set-url origin https://x-access-token:${GH_TOKEN}@github.com/${KYMA_BTP_MANAGER_REPO}.git
git push --set-upstream origin ${BRANCH_NAME} -f

#create PR
pr_link=$(gh pr create -B main --title "${MSG}" --body "${SAP_BTP_SERVICE_OPERATOR_REPO}/releases/tag/${TAG}" | tail -n 1)
echo "Link for created PR: ${pr_link}"

pr_number=$(echo "${pr_link}" | awk -F '/' '{print($NF)}')
pr_id=$(gh api repos/kyma-project/btp-manager/pulls/"${pr_number}" | jq -r '.node_id')

# Gopher board node_id
readonly project_board_id=PVT_kwDOAlVvc84AEv0v
# "To Do" column on Gopher board node_id
readonly todo_column_id=834c7033
# order in "To Do" column on Gopher board node_id
readonly status_field=PVTSSF_lADOAlVvc84AEv0vzgCvCtY

# insert projectv2 item (card on the gopher board)
resp=$(gh api graphql -f query='mutation{ addProjectV2ItemById(input:{projectId: "'${project_board_id}'" contentId: "'${pr_id}'"}){ item{id} }}' )
echo "response from inserting projectv2 item: $resp"
card_id=$(echo "$resp" | jq -r '.data.addProjectV2ItemById.item.id')

# move projectv2 item (card on the gopher board) to the top of the "To Do" column
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
