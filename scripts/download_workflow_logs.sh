#!/usr/bin/env bash
# This script downloads logs from all attempts of the latest workflow run by workflow name and title.

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked


if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <workflow_name> <workflow_title>"
  exit 1
fi

WORKFLOW_NAME="$1"
WORKFLOW_TITLE="$2"

REPO="${GITHUB_REPOSITORY:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}"

workflow_id=$(gh api \
  -H "Accept: application/vnd.github+json" \
  "/repos/${REPO}/actions/workflows" | jq -r --arg name "$WORKFLOW_NAME" '.workflows[] | select(.name == $name) | .id')
if [ -z "$workflow_id" ] || [ "$workflow_id" = "null" ]; then
  echo "Workflow '$WORKFLOW_NAME' not found."
  exit 1
fi

run_id=$(gh api \
  -H "Accept: application/vnd.github+json" \
  "/repos/${REPO}/actions/workflows/${workflow_id}/runs" | jq -r --arg workflow_title_filter "$WORKFLOW_TITLE" '.workflow_runs[] | select(.display_title | test($workflow_title_filter; "i")) | .id' | head -n 1)
if [ -z "$run_id" ] || [ "$run_id" = "null" ]; then
  echo "No runs found for workflow: $WORKFLOW_NAME with title filter: $WORKFLOW_TITLE"
  exit 1
fi

attempts=$(gh api \
  -H "Accept: application/vnd.github+json" \
  "/repos/${REPO}/actions/runs/${run_id}" | jq -r '.run_attempt')
if [ -z "$attempts" ] || [ "$attempts" = "null" ]; then
  echo "Attempts not found."
  exit 1
fi

for attempt in $(seq 1 $attempts); do
  echo "Downloading logs for attempt $attempt..."
  gh api \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    "/repos/${REPO}/actions/runs/${run_id}/attempts/${attempt}/logs" > "logs_attempt_${attempt}.zip"
done

echo "Downloaded logs for $attempts attempt(s)."
