#!/usr/bin/env bash

# This script has the following arguments:
#     - number of iterations of switching between which btpOperator is reconciled
# ./multiple_btpoperators_exist.sh 10

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

YAML_DIR="scripts/testing/yaml"

apply_btp_operator() {
    local filename=$1
    echo -e "\n---Creating BTP Operator in $namespace namespace"
    kubectl apply -f ${YAML_DIR}/e2e-test-btpoperator.yaml -n $namespace
}

wait_for_condition() {
    local condition=$1
    local name=$2
    local namespace=$3
    local message=$4
    sleep 1
    while [[ $(kubectl get btpoperators/$name -n $namespace -ojson| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != $condition ]];
    do
        echo -e "\n---Waiting for BTP Operator $name in $namespace namespace to be $message"; sleep 3;
    done
}

delete_btp_operator() {
    local name=$1
    local namespace=$2
    echo -e "\n---Deleting BTP Operator $name in $namespace namespace"
    kubectl delete btpoperators $name -n $namespace &
    kubectl patch btpoperator $name -n $namespace -p '{"metadata":{"finalizers":[]}}' --type=merge &
    wait
}

echo -e "\n---Testing multiple BTP Operators handling"

echo -e "\n---Creating BTP Operator in default namespace"
kubectl apply -f ${YAML_DIR}/e2e-test-btpoperator-wrong-namespace.yaml
wait_for_condition "FalseWrongNamespaceOrName" "btpoperator" "default" "ready and in error WrongNameOrNamespace"

echo -e "\n---Creating BTP Operator with wrong name"
kubectl apply -f ${YAML_DIR}/e2e-test-btpoperator-wrong-name.yaml
wait_for_condition "FalseWrongNamespaceOrName" "btpoperator-test" "kyma-system" "ready and in error WrongNameOrNamespace"

echo -e "\n---Creating correct BTP Operator"
kubectl apply -f ${YAML_DIR}/e2e-test-btpoperator.yaml
wait_for_condition "TrueReconcileSucceeded" "btpoperator" "kyma-system" "ready and reconciled"

echo -e "\n---BTP Operator created successfully"

echo -e "\n---Deleting btpopertors"
delete_btp_operator "btpoperator" "kyma-system"
delete_btp_operator "btpoperator-test" "kyma-system"

echo -e "\n---Multiple BTP Operators handling finished successfully"
