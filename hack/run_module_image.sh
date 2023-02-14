#!/usr/bin/env bash

# This script has the following argument: a link to a module image, for example:
# ./run_module_image.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:0.0.32

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

echo "Downloading module image"

mkdir downloaded_module
skopeo copy docker://$1 dir:downloaded_module

cd downloaded_module
filename=$(cat manifest.json  | jq -c '.layers[] | select(.mediaType=="application/gzip").digest[7:]' | tr -d \")

echo "Extracting $filename"

mkdir chart
tar -xzf $filename -C chart

echo "Installing helm chart"

helm install btp-manager chart