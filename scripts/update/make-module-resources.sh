#!/bin/zsh

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

cd "$(dirname "$0")"

readonly MODULE_CHART_PATH="../../module-chart"
readonly CHART_PATH="${MODULE_CHART_PATH}/chart"
readonly CHART_OVERRIDES_PATH="${MODULE_CHART_PATH}/overrides.yaml"
readonly EXISTING_RESOURCES_PATH="../../module-resources"
readonly EXISTING_RESOURCES_DELETE_PATH="${EXISTING_RESOURCES_PATH}/delete"
readonly EXISTING_RESOURCES_APPLY_PATH="${EXISTING_RESOURCES_PATH}/apply"
readonly EXCLUDED_RESOURCES_PATH="${EXISTING_RESOURCES_PATH}/excluded"
readonly HELM_OUTPUT_PATH="rendered"
readonly NEW_RESOURCES_PATH="rendered/sap-btp-operator/templates"
readonly RBAC_FILE_PATH="../../controllers/btpoperator_controller.go"

TAG=$1
helm template ${TAG} ${CHART_PATH} --output-dir ${HELM_OUTPUT_PATH} --values ${CHART_OVERRIDES_PATH} --namespace "kyma-system"

trap 'rm -rf -- "temp"' EXIT

declare -A rbac
declare -A resourcePlural
# pluralization exceptions have to be done manually because discovery client is not setup
# https://github.com/kubernetes/kubernetes/issues/18622
resourcePlural["componentstatuss"]=componentstatuses
resourcePlural["csistoragecapacitys"]=csistoragecapacities
resourcePlural["endpointss"]=endpoints
resourcePlural["ingressclasss"]=ingressclasses
resourcePlural["ingresss"]=ingresses
resourcePlural["networkpolicys"]=networkpolicies
resourcePlural["priorityclasss"]=priorityclasses
resourcePlural["runtimeclasss"]=runtimeclasses
resourcePlural["storageclasss"]=storageclasses

resource() {
  r=$(yq '.kind' "$1" | tr '[:upper:]' '[:lower:]')s
  if [[ "${resourcePlural[$r]+x}" ]]; then
    r=${resourcePlural[$r]}
  fi
  echo -n "$r"
}

group() {
  echo -n "$(yq '.apiVersion' "$1" | awk -F '/' '/\//{print($1)}' )"
}

# Extract all RBAC rules from the existing rbac.yml file and populate the rbac array
extractRbacFromFile() {
  local rbac_file="${EXISTING_RESOURCES_APPLY_PATH}/rbac.yml"
  
  if [ ! -f "$rbac_file" ]; then
    echo "Warning: No RBAC file found at $rbac_file" >&2
    return
  fi
  
  # Process both ClusterRole and Role resources
  local total_count=$(yq eval 'select(.kind == "ClusterRole" or .kind == "Role") | .metadata.name' "$rbac_file" | wc -l | tr -d ' ')
  
  for ((i=0; i<total_count; i++)); do
    local rules_count=$(yq eval "select(.kind == \"ClusterRole\" or .kind == \"Role\") | select(di == $i) | .rules | length" "$rbac_file" 2>/dev/null)
    
    if [ -z "$rules_count" ] || [ "$rules_count" = "0" ]; then
      continue
    fi
    
    for ((j=0; j<rules_count; j++)); do
      local groups=$(yq eval "select(.kind == \"ClusterRole\" or .kind == \"Role\") | select(di == $i) | .rules[$j].apiGroups[]" "$rbac_file" 2>/dev/null)
      local resources=$(yq eval "select(.kind == \"ClusterRole\" or .kind == \"Role\") | select(di == $i) | .rules[$j].resources[]" "$rbac_file" 2>/dev/null)
      local verbs=$(yq eval "select(.kind == \"ClusterRole\" or .kind == \"Role\") | select(di == $i) | .rules[$j].verbs[]" "$rbac_file" 2>/dev/null | sort -u | tr '\n' ';' | sed 's/;$//')
      
      # Generate RBAC annotation for each group+resource combination
      while IFS= read -r group; do
        while IFS= read -r resource; do
          if [ -n "$resource" ] && [ -n "$verbs" ]; then
            local gvk="${group}/${resource}"
            rbac[$gvk]='//+kubebuilder:rbac:groups="'$group'",resources="'$resource'",verbs='$verbs
          fi
        done <<< "$resources"
      done <<< "$groups"
    done
  done
}

filterProhibitedFiles() {
  local directory=${1}
  if [ "$(ls -A $directory)" ]; then
      for yaml in $directory/*
      do
          yq '. | select(.metadata.annotations."helm.sh/hook" == "*pre-delete*")' $yaml >> to-exclude.yml

          yq -i '. | select(.metadata.annotations."helm.sh/hook" != "*pre-delete*")' $yaml
          if [ ! -s $yaml ]; then
              echo "Removing $yaml because of pre-delete helm hook"
              rm $yaml
          fi

      done
    else
      echo "$directory is empty"
    fi
}

runActionForEachYaml() {
  local directory=${1}
  local action=${2}

  if [ "$(ls -A $directory)" ]; then    
    for combinedYaml in $directory/*
    do
        mkdir 'temp' && cd 'temp'
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
        local gvk=$(yq '.apiVersion' "$yaml")/$(yq '.kind' "$yaml")
        local r=$(resource "$yaml")
        local g=$(group "$yaml")
        if ! [[ "${rbac[$gvk]+x}" ]]; then
          rbac[$gvk]='//+kubebuilder:rbac:groups=\"'$g'\",resources=\"'$r'\",verbs=\"delete\"'
        fi
    fi
}

updateRbac() {
    local rbac_string="$(echo "${rbac[@]}" | tr " " "\n" | sort | uniq)"
    local temp_rbac_file=$(mktemp)
    echo "$rbac_string" > "$temp_rbac_file"
    
    awk -v rbac_file="$temp_rbac_file" '
        !/\+kubebuilder:rbac/{print($0)} 
        /Autogenerated RBAC from the btp-operator chart/{
            a=1
            while((getline line < rbac_file) > 0) {
                print line
            }
            close(rbac_file)
        } 
        /\+kubebuilder:rbac/{
            if(a!=1){
                print($0)
            }
        }
    ' "$RBAC_FILE_PATH" > "$RBAC_FILE_PATH.tmp"
    
    mv "$RBAC_FILE_PATH.tmp" "$RBAC_FILE_PATH"
    rm "$temp_rbac_file"
    
    (cd ../../; make manifests)
}

set_images_to_empty_strings() {
  local deployment_file="${EXISTING_RESOURCES_APPLY_PATH}/deployment.yml"
  if [ -f "$deployment_file" ]; then
    yq -i '(.spec.template.spec.containers[].image) = ""' "$deployment_file"
  fi
}

incoming_resources=()
touch to-exclude.yml
filterProhibitedFiles ${NEW_RESOURCES_PATH}
runActionForEachYaml ${NEW_RESOURCES_PATH} actionForNewResource

touch to-delete.yml
runActionForEachYaml ${EXISTING_RESOURCES_APPLY_PATH} actionForExistingResource

extractRbacFromFile

updateRbac

rm -rf ${EXISTING_RESOURCES_PATH}
mkdir -p ${EXISTING_RESOURCES_APPLY_PATH}
mkdir -p ${EXISTING_RESOURCES_DELETE_PATH}
mkdir -p ${EXCLUDED_RESOURCES_PATH}

mv $NEW_RESOURCES_PATH/* $EXISTING_RESOURCES_APPLY_PATH

mv to-delete.yml $EXISTING_RESOURCES_DELETE_PATH
mv to-exclude.yml $EXCLUDED_RESOURCES_PATH

rm -r $HELM_OUTPUT_PATH

set_images_to_empty_strings
