#!/bin/bash
cd "$(dirname "$0")"
ARG1=${1:-../module-chart}
latest=$(curl \
        -H "Accept: application/vnd.github+json" \
        -H "X-GitHub-Api-Version: 2022-11-28" \
        https://api.github.com/repos/SAP/sap-btp-service-operator/releases/latest | jq -r '.tag_name') 
curl -L https://github.com/SAP/sap-btp-service-operator/releases/download/$latest/sap-btp-operator-$latest.tgz > charts.tgz
tar zxf charts.tgz 
rsync -a sap-btp-operator/ $ARG1
rm -r sap-btp-operator
rm charts.tgz
echo $latest