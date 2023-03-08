#!/usr/bin/env bash

# This script has the following argument: a link to a module image, for example:
# ./run_module_image.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:v0.2.3

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

echo -e "\n--- Downloading module image"
rm -rf downloaded_module
mkdir downloaded_module

downloaded_dir=downloaded_module
skopeo copy docker://$1 dir:$downloaded_dir
filename=$(cat $downloaded_dir/manifest.json  | jq -c '.layers[] | select(.mediaType=="application/gzip").digest[7:]' | tr -d \")

echo -e "\n--- Extracting resources from file:" $filename
mkdir $downloaded_dir/chart
tar -xzf $downloaded_dir/$filename -C $downloaded_dir/chart

echo -e "\n--- Installing BTP Manager"
# install by helm
helm upgrade --install btp-manager $downloaded_dir/chart -n kyma-system --create-namespace