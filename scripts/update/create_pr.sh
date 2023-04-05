#!/usr/bin/env bash
set -x
# Create PR with the new version of chart and link it to Gophers' dashboard

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables passed e.g. from CI:
#   BRANCH_NAME                   - branch
#   GH_TOKEN                      - GitHub token for GitHub CLI
#   KYMA_BTP_MANAGER_REPO         - repository to create PR adding the new chart
#   SAP_BTP_SERVICE_OPERATOR_REPO - repository to fetch the new chart
#   MSG                           - commit message and the title for the new PR
#   TAG                           - chart version

git status
git checkout -B ${BRANCH_NAME}
git stash apply
git add module-chart/*
git add module-resources/*
git add controllers/btpoperator_controller.go
git add config/rbac/role.yaml
git commit -m "$MSG"
git remote set-url origin https://x-access-token:${GH_TOKEN}@github.com/${KYMA_BTP_MANAGER_REPO}.git
git push --set-upstream origin ${BRANCH_NAME} -f

pr_link=$(gh pr create -B main --title "${MSG}" --body "https://${SAP_BTP_SERVICE_OPERATOR_REPO}/releases/tag/${TAG}" | tail -n 1)
echo "Link for created PR: ${pr_link}"

./scripts/update/link_pr_to_dashboard.sh "${pr_link}"


