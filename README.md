# BTP Manager

## Overview

BTP Manager is an operator for [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator) based on [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator](https://github.com/kyma-project/btp-manager/blob/main/operator/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) CRD which allows to manage SAP BTP Service Operator resource through CR.

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io)) 

## Install BTP Manager locally

Use the following commands to run the BTP Manager controller from your host. Both `make` commands refer to [Makefile](./Makefile) in the `operator` directory.

```sh
make install
make run
```

## Install BTP Manager in your cluster

There are two ways to install BTP Manager in your cluster:

<details>
<summary>With kubectl and <code>rendered.yaml</code> (recommended)</summary>
<br>

Use the following command to download and install BTP Manager from Kubernetes resources in your cluster.

```shell
kubectl apply -f deployments/prerequisites.yaml
kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/rendered.yaml
```

Use the following command to uninstall BTP Manager from your cluster.

```shell
kubectl delete -f https://github.com/kyma-project/btp-manager/releases/latest/download/rendered.yaml
kubectl delete -f deployments/prerequisites.yaml
```
</details>

<details>
<summary>With Helm and <code>template.yaml</code></summary>
<br>

To install BTP Manager using a template file (the output of the [kyma alpha create module](https://github.com/kyma-project/cli/blob/main/docs/gen-docs/kyma_alpha_create_module.md) command) in your cluster, use the following command:

```shell
./scripts/run_template.sh https://github.com/kyma-project/btp-manager/releases/latest/download/template.yaml
```

> **NOTE:** Before using the script, you must install [Helm](https://github.com/helm/helm#install), [skopeo](https://github.com/containers/skopeo), [jq](https://github.com/stedolan/jq) and [yq](https://github.com/mikefarah/yq).

Use the following command to uninstall BTP Manager from your cluster.
```shell
helm uninstall btp-manager -n kyma-system
```

</details>

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
