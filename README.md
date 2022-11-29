# BTP Manager

## Overview

BTP Manager is an operator for [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator) based on [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator](https://github.com/kyma-project/btp-manager/blob/main/operator/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) CRD which allows to manage SAP BTP Service Operator resource through CR.

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io)) 

## Installation
The following commands describe how to run BTP Manager locally. All `make` commands are referring to [Makefile](./operator/Makefile) in `operator` directory.

```sh
cd operator
make install
make run
```

## Usage

#### The following commands describe how to install SAP BTP Service Operator.
```sh
kubectl apply -f deployments/prerequisites.yaml
kubectl apply -f examples/btp-manager-secret.yaml
kubectl apply -f examples/btp-operator.yaml
```
```
namespace/kyma-system created
priorityclass.scheduling.k8s.io/kyma-system created
secret/sap-btp-manager created
btpoperator.operator.kyma-project.io/btpoperator-sample created
```

Check `BtpOperator` CR status.
```sh
kubectl get btpoperators btpoperator-sample
```

Expected result.
```
NAME                 STATE
btpoperator-sample   Ready
```

#### The following commands describe how to uninstall SAP BTP Service Operator.
```sh
kubectl delete -f examples/btp-operator.yaml
kubectl delete -f examples/btp-manager-secret.yaml
kubectl delete -f deployments/prerequisites.yaml
```
```
btpoperator.operator.kyma-project.io "btpoperator-sample" deleted
secret "sap-btp-manager" deleted
namespace "kyma-system" deleted
priorityclass.scheduling.k8s.io "kyma-system" deleted
```