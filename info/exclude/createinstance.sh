#!/bin/sh

NAME_FROM_SERVICE_MARKETPLACE=$1
PLAN_FROM_SERVICE_MARKETPLACE=$2
POSTFIX=$3
INSTANCE_NAME="sill-${NAME_FROM_SERVICE_MARKETPLACE}${POSTFIX}"
BINDING_NAME="sbll-${NAME_FROM_SERVICE_MARKETPLACE}${POSTFIX}"

kubectl create -f - <<EOF
apiVersion: services.cloud.sap.com/v1alpha1
kind: ServiceInstance
metadata:
  name: $INSTANCE_NAME
  namespace: default
spec:
  serviceOfferingName: $NAME_FROM_SERVICE_MARKETPLACE
  servicePlanName: $PLAN_FROM_SERVICE_MARKETPLACE
  externalName: $INSTANCE_NAME
EOF

kubectl create -f - <<EOF
apiVersion: services.cloud.sap.com/v1alpha1
kind: ServiceBinding
metadata:
  name: $BINDING_NAME
  namespace: default
spec:
  serviceInstanceName: $INSTANCE_NAME
  externalName: $BINDING_NAME
  secretName: $BINDING_NAME
EOF
