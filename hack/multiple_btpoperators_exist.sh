#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

apply_btp_operator() {
    local namespace=$1
    kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-operator-default-cr.yaml -n $namespace
}

wait_for_condition() {
    local condition=$1
    local namespace=$2
    local message=$3
    while [[ $(kubectl get btpoperators btpoperator -n $namespace -ojson| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != $condition ]];
    do
        echo -e "\n---Waiting for BTP Operator in $namespace namespace to be $message"; sleep 5;
    done
}

delete_and_patch_btp_operator() {
    local namespace=$1
    kubectl delete btpoperators btpoperator -n $namespace &
    kubectl patch btpoperator btpoperator -n $namespace -p '{"metadata":{"finalizers":[]}}' --type=merge &
    wait
}

iterations=$1

kubectl apply -f deployments/prerequisites.yaml  
kubectl apply -f examples/btp-manager-secret.yaml
kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-manager.yaml
apply_btp_operator "default"
wait_for_condition "TrueReconcileSucceeded" "default" "ready and reconciled"

export SI_NAME="test"
envsubst < scripts/testing/yaml/e2e-test-service-instance.yaml | kubectl apply -f -
export SB_NAME="test"
envsubst < scripts/testing/yaml/e2e-test-service-binding.yaml | kubectl apply -f -

for ((i=1; i<=$iterations; i++))
do        
    apply_btp_operator "kyma-system"
    wait_for_condition "FalseOlderCRExists" "kyma-system" "ready and in error OlderCRExists"
    delete_and_patch_btp_operator "default"
    wait_for_condition "TrueReconcileSucceeded" "kyma-system" "ready and reconciled"
    apply_btp_operator "default"
    wait_for_condition "FalseOlderCRExists" "default" "ready and in error OlderCRExists"
    delete_and_patch_btp_operator "kyma-system"
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

echo "Test finished successfully"
