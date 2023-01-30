#!/bin/bash
cd "$(dirname "$0")"

readonly CHART_PATH="../../module-chart/chart"
readonly CHART_OVERRIDES_PATH="../../module-chart/public-overrides.yaml"
readonly EXSITING_RESOURCES_PATH="../../module-resources"
readonly EXSITING_RESOURCES_DELETE_PATH="../../module-resources/delete"
readonly EXSITING_RESOURCES_APPLY_PATH="../../module-resources/apply"
readonly HELM_OUTPUT_PATH="rendered"
readonly NEW_RESOURCES_PATH="rendered/sap-btp-operator/templates"

helm template $1 $CHART_PATH --output-dir $HELM_OUTPUT_PATH --values $CHART_OVERRIDES_PATH

trap 'rm -rf -- "temp"' EXIT
runActionForEachYaml() {
  local directory=${1}
  local action=${2}

  if [ "$(ls -A $directory)" ]; then    
    for combinedYaml in $directory/*
    do
        mkdir 'temp' && cd "$_"
        yq -s '"file_" + $index' "../$combinedYaml"
        for singleYaml in *
        do
          $action $singleYaml
        done
        cd .. && rm -r 'temp'
    done
	else
    echo "$directory is Empty"
  fi
}

actionForNewResource() {
  local yaml=${1}
  incoming_resources+=("$(yq '.metadata.name' $yaml):$(yq '.kind' $yaml)")
}

actionForExistingResource() {
    local yaml=${1}
    if [[ ! "${incoming_resources[*]}" =~ "$(yq '.metadata.name' $yaml):$(yq '.kind' $yaml)" ]] ; then
        cat $yaml >> ../to-delete.yml
    fi
}

incoming_resources=()

runActionForEachYaml $NEW_RESOURCES_PATH actionForNewResource

touch to-delete.yml
runActionForEachYaml $EXSITING_RESOURCES_APPLY_PATH actionForExistingResource

rm -r $EXSITING_RESOURCES_PATH
mkdir $EXSITING_RESOURCES_PATH
mkdir $EXSITING_RESOURCES_APPLY_PATH
mkdir $EXSITING_RESOURCES_DELETE_PATH
mv $NEW_RESOURCES_PATH/* $EXSITING_RESOURCES_APPLY_PATH
mv to-delete.yml $EXSITING_RESOURCES_DELETE_PATH
rm -r $HELM_OUTPUT_PATH