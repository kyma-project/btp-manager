# BTP Manager

## Overview

BTP Manager is an operator for [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator) based on [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator](https://github.com/kyma-project/btp-manager/blob/main/operator/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) CRD which allows to manage SAP BTP Service Operator resource through CR.

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io)) 

## Installation
The following steps describe how to run BTP Manager. All `make` commands are referring to [Makefile](./operator/Makefile) in `operator` directory.

1. Install [BtpOperator CRD](./operator/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) by running `make install`.
2. Run `make run` to start BTP Manager locally.

## Usage

Create `BtpOperator` CR to provision SAP BTP Service Operator. You can use [sample CR](./default.yaml):
```sh
kubectl apply -f $BTPOPERATOR_CR_MANIFEST_PATH
```
e.g.:
```sh
kubectl apply -f default.yaml
```

Delete `BtpOperator` CR to remove SAP BTP Service Operator instance:
```sh
kubectl delete btpoperators.operator.kyma-project.io $NAME_OF_BTPOPERATOR_CR
```
e.g.:
```sh
kubectl delete btpoperators.operator.kyma-project.io btpoperator
```