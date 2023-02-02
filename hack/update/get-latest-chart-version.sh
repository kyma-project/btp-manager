#!/bin/bash
set -e
set -o pipefail

cd "$(dirname "$0")"
latest=$(curl \
        -H "Accept: application/vnd.github+json" \
        -H "X-GitHub-Api-Version: 2022-11-28" \
        https://api.github.com/repos/SAP/sap-btp-service-operator/releases/latest | jq -r '.tag_name')
echo $latest