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
kubectl apply -f ${YAML_DIR}/e2e-test-btpoperator.yaml
kubectl label -f ${YAML_DIR}/e2e-test-btpoperator.yaml force-delete=true

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

restarts=$(kubectl get po -n kyma-system -l app.kubernetes.io/component=btp-manager.kyma-project.io -o 'jsonpath={..items[0].status.containerStatuses[0].restartCount}')
if [ "${restarts}" != '0' ]
then
  echo "BTP manager was restarted $restarts times"
  exit 1
fi

restarts=$(kubectl get po -n kyma-system -l app.kubernetes.io/name=sap-btp-operator -o 'jsonpath={..items[0].status.containerStatuses[?(@.name=="manager")].restartCount}')
if [ "${restarts}" != '0' ]
then
  echo "BTP operator was restarted $restarts times"
  exit 1
fi

echo -e "\n---Deleting e2e-test-btpoperator"
kubectl delete btpoperators/e2e-test-btpoperator
