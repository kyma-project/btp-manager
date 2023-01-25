#!/bin/bash
ls -la
helm template $1 module-chart --output-dir rendered --values module-overrides/public-overrides.yaml
mv rendered/sap-btp-operator/templates/ module-resources
rm -r rendered/
ls -la