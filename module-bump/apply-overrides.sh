#!/bin/bash
cd "$(dirname "$0")"
ARG1=${1:-../module-resources}
helm template $2 ../module-chart --output-dir rendered --values public-overrides.yaml
rsync -a rendered/sap-btp-operator/templates/ $ARG1
rm -r rendered/