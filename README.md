# BTP Manager

## Overview

BTP Manager is an operator for [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator) based on [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator](https://github.com/kyma-project/btp-manager/blob/main/operator/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) CRD which allows to manage SAP BTP Service Operator resource through CR.

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io)) 

## Install BTP Manager locally
Use the following commands to run the BTP Manager controller from your host. All the `make` commands refer to [Makefile](./Makefile) in the `operator` directory.

```sh
make install
make run
```

## Install BTP Manager in your cluster

There are two ways to install BTP Manager in your cluster.

### Installation with btp-operator module image
Use the following command to download and install BTP manager from OCI Image in your cluster.

```shell
./hack/run_module_image.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:v0.2.3
```
> **NOTE:** Before using the script, you must install [Helm](https://github.com/helm/helm#install), [skopeo](https://github.com/containers/skopeo) and [jq](https://github.com/stedolan/jq).

### Install BTP Manager with `template.yaml`

To install BTP Manager using a template file (the output of the [kyma alpha create module](https://github.com/kyma-project/cli/blob/main/docs/gen-docs/kyma_alpha_create_module.md) command) in your cluster, use the following command:

```shell
./hack/run_template.sh https://github.com/kyma-project/btp-manager/releases/download/0.2.3/template.yaml
```

> **NOTE:** Before using the script, you must install [Helm](https://github.com/helm/helm#install), [skopeo](https://github.com/containers/skopeo), [jq](https://github.com/stedolan/jq) and [yq](https://github.com/mikefarah/yq).

### Uninstall BTP Manager from your cluster

Use the following command to uninstall BTP Manager from your cluster.
```shell
helm uninstall btp-manager -n kyma-system
```

## Usage

#### Install SAP BTP Service Operator

To install SAP BTP Service Operator, run the following commands:
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

Check `BtpOperator` CR status by running the following command:
```sh
kubectl get btpoperators btpoperator-sample
```

The expected result is:
```
NAME                 STATE
btpoperator-sample   Ready
```

#### Uninstall SAP BTP Service Operator

To uninstall SAP BTP Service Operator, run the following commands:
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
