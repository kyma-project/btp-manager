#!/usr/bin/env bash

# This script has the following arguments:
#     - number of iterations of switching between which btpOperator is reconciled
# ./multiple_btpoperators_exist.sh 10

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

iterations=$1
YAML_DIR="scripts/testing/yaml"

apply_btp_operator() {
    local namespace=$1
    echo -e "\n---Creating BTP Operator in $namespace namespace"
    kubectl apply -f ${YAML_DIR}/e2e-test-btpoperator.yaml -n $namespace
}

wait_for_condition() {
    local condition=$1
    local namespace=$2
    local message=$3
    while [[ $(kubectl get btpoperators/e2e-test-btpoperator -n $namespace -ojson| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != $condition ]];
    do
        echo -e "\n---Waiting for BTP Operator in $namespace namespace to be $message"; sleep 5;
    done
}

delete_btp_operator() {
    local namespace=$1
    echo -e "\n---Deleting BTP Operator in $namespace namespace"
    kubectl delete btpoperators e2e-test-btpoperator -n $namespace &
    kubectl patch btpoperator e2e-test-btpoperator -n $namespace -p '{"metadata":{"finalizers":[]}}' --type=merge &
    wait
}

echo -e "\n---Testing multiple BTP Operators handling"

for ((i=1; i<=$iterations; i++))
do        
    apply_btp_operator "kyma-system"
    wait_for_condition "FalseOlderCRExists" "kyma-system" "ready and in error OlderCRExists"
    delete_btp_operator "default"
    wait_for_condition "TrueReconcileSucceeded" "kyma-system" "ready and reconciled"
    apply_btp_operator "default"
    wait_for_condition "FalseOlderCRExists" "default" "ready and in error OlderCRExists"
    delete_btp_operator "kyma-system"
    wait_for_condition "TrueReconcileSucceeded" "default" "ready and reconciled"
done

count=$(kubectl get serviceinstances -o json | jq '.items | length')
if [[ $count -ne 1 ]]; then
    echo "Error: Expected 1 service instance, but found $count"
    exit 1
fi

count=$(kubectl get servicebindings -o json | jq '.items | length')
if [[ $count -ne 1 ]]; then
    echo "Error: Expected 1 service binding, but found $count"
    exit 1
fi

echo -e "\n---Multiple BTP Operators handling finished successfully"
