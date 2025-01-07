#! /bin/bash
echo "
apiVersion: v2
name: $OPERATOR_NAME
description: A Helm chart for the Operator in a Cluster based on a Kustomize Manifest
type: application
version: $MODULE_VERSION
appVersion: "$MODULE_VERSION"
"