#!/usr/bin/env bash
# The script has the following arguments:
#                       - amount of memory we expect to be increased in BTP Operator in Mi after creating large and labeled Secrets (e.g. 1 Mi)
# Example:
#           ./check_top_increased.sh 1

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

MEM_INCREASE=${1/Mi/}

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
measure_pod_resources app.kubernetes.io/name=sap-btp-operator| tee fin.mem.txt

OPERATOR_MEM=$( awk '/sap-btp-operator-controller-manager.*manager/{print $4}' <top.mem.txt | sed 's/Mi//')
FIN_OPERATOR_MEM=$( awk '/sap-btp-operator-controller-manager.*manager/{print $4}' <fin.mem.txt | sed 's/Mi//')

TARGET_MEM=$((OPERATOR_MEM + MEM_INCREASE))

if [ -n "$OPERATOR_MEM" ] && [ -n "$FIN_OPERATOR_MEM" ] && [ "$FIN_OPERATOR_MEM" -ge "$TARGET_MEM" ]; then
     echo "Memory usage of BTP Operator is within the expected limit: ${FIN_OPERATOR_MEM}Mi >= ${TARGET_MEM}Mi"
else
     echo "Memory usage of BTP Operator does not exceed the expected limit: ${FIN_OPERATOR_MEM}Mi < ${TARGET_MEM}Mi"
     exit 1
fi