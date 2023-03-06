#!/usr/bin/env bash

# This script has the following argument: a link to a module image, for example:
# ./run_module_image.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:0.0.32

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

echo "--- Download template.yaml"
wget -nv -O template.yaml $1

component_name=$(cat template.yaml | yq '.spec.descriptor.component.name')
base_url=$(cat template.yaml | yq '.spec.descriptor.component.repositoryContexts[0].baseUrl')
version=$(cat template.yaml | yq '.spec.descriptor.component.version')

url="$base_url/component-descriptors/$component_name:$version"
echo -e "\nBTP operator module image:" $url
./hack/run_module_image.sh $url