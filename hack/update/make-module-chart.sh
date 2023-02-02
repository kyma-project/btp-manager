#!/bin/bash
set -e
set -o pipefail

cd "$(dirname "$0")"

readonly CHART_PATH="../../module-chart/chart"

tag=$1
if [[ -z $tag ]]; then
  tag=$(sh get-latest-chart-version.sh)
fi

curl -L https://github.com/SAP/sap-btp-service-operator/releases/download/$tag/sap-btp-operator-$tag.tgz > charts.tgz
tar zxf charts.tgz
rm -r $CHART_PATH
rsync -a sap-btp-operator/ $CHART_PATH
rm -r sap-btp-operator
rm charts.tgz