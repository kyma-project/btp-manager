[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/btp-manager)](https://api.reuse.software/info/github.com/kyma-project/btp-manager)

# BTP Manager

## Overview

BTP Manager is an operator for [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator) based on the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator](/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) CRD which allows you to manage SAP BTP Service Operator resource through CR. 

## Installation

To enable the BTP Operator module from the latest release, you must install BTP Manager and SAP BTP Service Operator.

### Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster, or [k3d](https://k3d.io) for local installation
- [jq](https://github.com/stedolan/jq) 

>**CAUTION:** You also need the `kyma-system` Namespace. If you don't have it in your cluster, use the following command to create it:
> ```bash
> kubectl create namespace kyma-system
> ```

### Steps
 
1. To install BTP Manager, use the following commands:

    ```bash
    kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-manager.yaml
    ```
    > **TIP:** Use the same command to upgrade the module to the latest version.

<br>

 2. To install SAP BTP Service Operator, apply the sample BtpOperator CR:

    ```bash
    kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-operator-default-cr.yaml
    ```
3. To check the `BtpOperator` CR status, run the following command:
   ```sh
   kubectl get btpoperators btpoperator
   ```
   > **NOTE:** The CR is in the `Warning` state and the message is `Secret resource not found reason: MissingSecret`. To create a Secret, follow the instructions in [Use BTP Manager to manage SAP BTP Service Operator](./docs/user/02-10-usage.md#create-and-install-a-secret).

For more installation options, read the [Install and uninstall BTP Manager](./docs/contributor/01-10-installation.md) document.

## Usage

Use BTP Manager to deploy an SAP BTP service in your Kyma cluster. To find out how to do it, read the [usage](./docs/user/02-10-usage.md) document.

## Uninstallation

To uninstall SAP BTP Service Operator, run the following commands:
```sh
kubectl delete -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-operator-default-cr.yaml
kubectl delete -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-manager.yaml
```

## Read more

If you want to provide new features for BTP Manager, visit the [`contributor`](./docs/contributor) folder. You will find detailed information on BTP Manager's:

- [configuration](./docs/contributor/01-20-configuration.md)
- [operations](./docs/contributor/02-10-operations.md)
- [release pipeline](./docs/contributor/03-10-release.md)
- [GitHub Actions workflows](./docs/contributor/04-10-workflows.md)
- [unit tests](./docs/contributor/05-10-testing.md)
- [E2E tests](./docs/contributor/05-20-e2e_tests.md)
- [certification management](./docs/contributor/06-10-certs.md)
- [informer's cache](./docs/contributor/07-10-informer-cache.md)
- [metrics](./docs/contributor/08-10-metrics.md)

Visit the [`user`](./docs/user) folder if you want to know more about [BTP Operator](./docs/user/README.md), and [how to use the module](./docs/user/02-10-usage.md).
