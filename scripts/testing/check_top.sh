#!/usr/bin/env bash
# The script has the following arguments:
#                       - memory limit for btp-operator in Mi
#                       - reference link to be printed in case of failure
# Example:
#           ./check_top.sh 100 http://link-to-reference.com


# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

REF=$2
MEM_LIMIT=${1/Mi/}

function measure_pod_resources() {
     TIMEOUT=30
     NEXT_TRY_WAIT=5
     LABEL_SELECTOR=$1

     SECONDS=0
     while (($SECONDS < $TIMEOUT )); do
          kubectl top pod -l "$LABEL_SELECTOR" -n kyma-system --containers
          [ $? == 0 ] && break
          sleep $NEXT_TRY_WAIT
     done
}

echo -e "\n--- BTP Manager checking kubectl top" 
measure_pod_resources app.kubernetes.io/component=btp-manager.kyma-project.io

echo -e "\n--- BTP Operator checking kubectl top" 
measure_pod_resources app.kubernetes.io/name=sap-btp-operator| tee top.mem.txt

OPERATOR_MEM=$( awk '/sap-btp-operator-controller-manager.*manager/{print $4}' <top.mem.txt | sed 's/Mi//')
if [ -n "$OPERATOR_MEM" ] && [ "$OPERATOR_MEM" -le "$MEM_LIMIT" ]; then
     echo "Memory usage of BTP Operator is within the limit: ${OPERATOR_MEM}Mi <= ${MEM_LIMIT}Mi"
else
     echo "Memory usage of BTP Operator exceeds the limit: ${OPERATOR_MEM}Mi > ${MEM_LIMIT}Mi"
     echo "See ${REF} for reference."
     exit 1
fi