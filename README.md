# BTP Manager

## Overview

BTP Manager is an operator for [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator) based on [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator](https://github.com/kyma-project/btp-manager/blob/main/operator/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) CRD which allows to manage SAP BTP Service Operator resource through CR.

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io)) 

## Installation
Use the following commands to run BTP Manager locally. All `make` commands refer to [Makefile](./Makefile) in the `operator` directory.

```sh
make install
make run
```

## Installation with btp-operator module image

Use the following command to download and run btp-manager from OCI image.

```shell
./hack/run_module_image.sh europe-docker.pkg.dev/kyma-project/prod/unsigned/component-descriptors/kyma.project.io/module/btp-operator:0.0.32
```
> **NOTE:** Before using the script, you must install [skopeo](https://github.com/containers/skopeo) and [jq](https://github.com/stedolan/jq).

## Installation using a template.yaml

To install btp-operator module using a template file (an output of the [kyma alpha create module](https://github.com/kyma-project/cli/blob/main/docs/gen-docs/kyma_alpha_create_module.md) command), use the following commands:

1. Download the template file, for example:
```shell
wget https://github.com/kyma-project/btp-manager/releases/download/0.0.32/template.yaml
```

2. Deploy the module:
```shell
component_name=$(cat template.yaml | yq '.spec.descriptor.component.name')
base_url=$(cat template.yaml | yq '.spec.descriptor.component.repositoryContexts[0].baseUrl')
version=$(cat template.yaml | yq '.spec.descriptor.component.version')

url="$base_url/component-descriptors/$component_name:$version"

./hack/run_module_image.sh $url
```

> **NOTE:** Before using the script, you must install [skopeo](https://github.com/containers/skopeo), [jq](https://github.com/stedolan/jq) and [yq](https://github.com/mikefarah/yq).

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
