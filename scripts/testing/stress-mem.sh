#!/usr/bin/env bash
# Stress testing in regard to memory consumption - could cause OOM (but should not).
# Creates btp-operator and numerous service instances and service bindings in current context.
#
# The script has the following arguments:
#                       - number of service instances and service bindings to be created - default 500
#                       - number of seconds to let them be before btp-operator is deleted - default 60
# Example:
#           ./stress-mem.sh 100 30

N=${1-500}
YAML_DIR=./scripts/testing/yaml
LIFE_SPAN=${2-60}

echo -e "\n---Installing BTP operator"
kubectl apply -f - <<EOF
apiVersion: operator.kyma-project.io/v1alpha1
kind: BtpOperator
metadata:
  labels:
    app.kubernetes.io/name: e2e-test-btpoperator
    app.kubernetes.io/instance: btpoperator
    app.kubernetes.io/part-of: btp-manager
    app.kubernetes.io/managed-by: btp-manager
    app.kubernetes.io/created-by: btp-manager
    force-delete: "true"
  name: e2e-test-btpoperator
spec:
# fields can be added here
EOF

while [[ $(kubectl get btpoperators/e2e-test-btpoperator -ojson| jq '.status.conditions[] | select(.type=="Ready") |.status+.reason'|xargs)  != "TrueReconcileSucceeded" ]];
do echo -e "\n---Waiting for BTP Operator to be ready and reconciled"; sleep 5; done

echo -e "\n---Creating ${N} service bindings and instances"

for ((i=1; i <= N ; i++))
do
    SI_NAME=auditlog-management-si-$i
    SB_NAME=auditlog-management-sb-$i

    export SI_NAME
    export SB_NAME

    envsubst <${YAML_DIR}/e2e-test-service-instance.yaml | kubectl apply -f - >/dev/null
    envsubst <${YAML_DIR}/e2e-test-service-binding.yaml | kubectl apply -f - >/dev/null
done

echo -e "\n---${N} service bindings and instances created - let them be for a while... ${LIFE_SPAN}s"
sleep ${LIFE_SPAN}

echo -e "\n---Deleting e2e-test-btpoperator"
kubectl delete btpoperators/e2e-test-btpoperator
