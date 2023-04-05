#!/usr/bin/env bash
set -x
TAG=$1

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

cd "$(dirname "$0")"

readonly CHART_PATH="../../module-chart/chart"
readonly SAP_BTP_SERVICE_OPERATOR_REPO=https://github.com/SAP/sap-btp-service-operator

TAG=${TAG:-$(./get-latest-chart-version.sh)}

curl -sL ${SAP_BTP_SERVICE_OPERATOR_REPO}/releases/download/${TAG}/sap-btp-operator-${TAG}.tgz | tar zx

rm -r ${CHART_PATH}
cp -R sap-btp-operator/ ${CHART_PATH}

#cleanup
rm -r sap-btp-operator
