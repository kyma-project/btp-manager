#!/bin/bash

k3d cluster delete btp-tests &&
k3d cluster create btp-tests &&
kubectl config use k3d-btp-tests