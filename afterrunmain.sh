#!/bin/bash

kubectl apply -f deployments/prerequisites.yaml &&
chmod +x hack/create-secret-file.sh &&
hack/create-secret-file.sh &&
kubectl apply -f hack/operator-secret.yaml &&
kubectl apply -f examples/btp-operator.yaml