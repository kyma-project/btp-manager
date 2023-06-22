# BTP Manager

## Overview

BTP Manager is an operator for [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator) based on the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator](https://github.com/kyma-project/btp-manager/blob/main/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) CRD which allows you to manage SAP BTP Service Operator resource through CR. 

## Installation

To enable the BTP Operator module from the latest release, you must install BTP Manager and SAP BTP Service Operator.

### Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster, or [k3d](https://k3d.io) for local installation

> **CAUTION** You also need the `kyma-system` Namespace. If you don't have it in your cluster, use the following command to create it:
> ```bash
> kubectl create namespace kyma-system
> ```

### Steps
 
1. To install BTP Manager, use the following commands:

    ```bash
    kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/rendered.yaml
    ```
    > **TIP:** Use the same command to upgrade the module to the latest version.

<br>

 2. To install SAP BTP Service Operator, apply the sample BtpOperator CR:

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/kyma-project/btp-manager/main/config/samples/operator_v1alpha1_btpoperator.yaml
    ```

For more installation options, read the [Install and uninstall BTP Manager](./docs/contributor/01-10-installation.md) document.

## Usage

Use BTP Manager to install or uninstall SAP BTP Service Operator. To find out how to do it, read the [Usage](docs/user/02-10-usage.md) document.

## Read more

If you want to provide new features for BTP Manager, visit the [`contributor`](docs/contributor) folder. You will find detailed information on BTP Manager's:

- [configuration](01-20-configuration.md)
- [operations](docs/contributor/02-10-operations.md)
- [release pipeline](docs/contributor/03-10-release.md)
- [Github Actions workflows](docs/contributor/04-10-workflows.md)
- [unit tests](docs/contributor/05-10-testing.md)
- [E2E tests](docs/contributor/05-20-e2e_tests.md)
- [certification management](docs/contributor/06-10-certs.md)

Visit the [`user`](docs/user) folder if you want to know more about [BTP Operator](docs/user/README.md), and [how to use the module](docs/user/02-10-usage.md).
