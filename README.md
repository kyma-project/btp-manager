# BTP Manager

## Overview

BTP Manager is an operator for [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator) based on the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator](https://github.com/kyma-project/btp-manager/blob/main/operator/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) CRD which allows to manage SAP BTP Service Operator resource through CR. For more information, see the [BTP Manager documentation](./docs/README.md).

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io)) 

## Install BTP Manager

 You can install BTP Manager locally or in your cluster. For more information, read the [Install and uninstall BTP Manager](./docs/installation.md) document.

## Usage

### Install SAP BTP Service Operator

To install SAP BTP Service Operator, run the following commands:
```sh
kubectl apply -f deployments/prerequisites.yaml
kubectl apply -f examples/btp-manager-secret.yaml
kubectl apply -f examples/btp-operator.yaml
```

Check `BtpOperator` CR status by running the following command:
```sh
kubectl get btpoperators btpoperator
```

The expected result is:
```
NAME                 STATE
btpoperator   Ready
```

### Uninstall SAP BTP Service Operator

To uninstall SAP BTP Service Operator, run the following commands:
```sh
kubectl delete -f examples/btp-operator.yaml
kubectl delete -f examples/btp-manager-secret.yaml
kubectl delete -f deployments/prerequisites.yaml
```
