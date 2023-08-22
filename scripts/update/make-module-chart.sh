#!/usr/bin/env bash

TAG=$1

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables passed e.g. from CI:
#   SAP_BTP_SERVICE_OPERATOR_REPO - repository to fetch the new chart

cd "$(dirname "$0")"

readonly CHART_PATH="../../module-chart/chart"

if [[ -z ${TAG} ]]; then
   TAG=$(./get-latest-chart-version.sh)
fi

curl -sL ${SAP_BTP_SERVICE_OPERATOR_REPO}/releases/download/${TAG}/sap-btp-operator-${TAG}.tgz | tar zx

rm -r ${CHART_PATH}
cp -R sap-btp-operator/ ${CHART_PATH}

#cleanup
rm -r sap-btp-operator
