#!/bin/bash

kubectl apply -f deployments/prerequisites.yaml &&
kubectl apply -f examples/btp-manager-secret.yaml &&
kubectl apply -f examples/btp-operator.yaml