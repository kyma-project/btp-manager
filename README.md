[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/btp-manager)](https://api.reuse.software/info/github.com/kyma-project/btp-manager)

# BTP Manager

## Overview

BTP Manager is an operator for the [SAP BTP service operator](https://github.com/SAP/sap-btp-service-operator) based on the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator CustomResourceDefinition](/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) (CRD), which allows you to manage the SAP BTP service operator resource through custom resource (CR). 

## Installation

To enable the SAP BTP Operator module from the latest release, you must install BTP Manager and the SAP BTP service operator.

### Prerequisites

* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* Kubernetes cluster, or [k3d](https://k3d.io) for local installation
* [jq](https://github.com/stedolan/jq)
* BTP Manager Secret created. See [Create a BTP Manager Secret](./docs/user/tutorials/04-10-create-btp-manager-secret.md).
* `sap-btp-manager` Secret
* `kyma-system` namespace. If you don't have it in your cluster, use the following command to create it:
    ```bash
    kubectl create namespace kyma-system
    ```

### Steps
 
1. To install BTP Manager, use the following commands:

    ```bash
    kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-manager.yaml
    ```
    > **TIP:** Use the same command to upgrade the module to the latest version.

<br>

 2. To install the SAP BTP service operator, apply the sample BtpOperator CR:

    ```bash
    kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-operator-default-cr.yaml
    ```
3. To check the BtpOperator CR status, run the following command:
   ```sh
   kubectl get btpoperators btpoperator
   ```
   > **NOTE:**
   > If the BtpOperator CR is in the `Warning` state and the message is `Secret resource not found reason: MissingSecret`, you must create the Secret. See the instructions in [Create a BTP Manager Secret](./docs/user/tutorials/04-10-create-btp-manager-secret.md).

For more installation options, read the [Install and Uninstall BTP Manager](./docs/contributor/01-10-installation.md) document.

## Usage

Use BTP Manager to deploy an SAP BTP service in your Kyma cluster. To find out how to do it, see the [tutorials](./docs/user/tutorials/README.md).

## Uninstallation

To uninstall the SAP BTP service operator, run the following commands:
```sh
kubectl delete -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-operator-default-cr.yaml
kubectl delete -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-manager.yaml
```

## Read More

If you want to provide new features for BTP Manager, visit the [`contributor`](./docs/contributor) folder. You will find detailed information on BTP Manager's:

* [configuration](./docs/contributor/01-20-configuration.md)
* [operations](./docs/contributor/02-10-operations.md)
* [release pipeline](./docs/contributor/03-10-release.md)
* [GitHub Actions workflows](./docs/contributor/04-10-workflows.md)
* [unit tests](./docs/contributor/05-10-testing.md)
* [E2E tests](./docs/contributor/05-20-e2e_tests.md)
* [certification management](./docs/contributor/06-10-certs.md)
* [informer's cache](./docs/contributor/07-10-informer-cache.md)
* [metrics](./docs/contributor/08-10-metrics.md)

In the [`user`](./docs/user) folder, you will find the following documents:
* [SAP BTP Operator Module](./docs/user/README.md)
* [Preconfigured Credentials and Access](./docs/user/03-10-preconfigured-secret.md)
* [Working with Multiple Subaccounts](./docs//user/03-30-multitenancy.md)
* [Management of the Service Instances and Service Bindings Lifecycle](./docs//user/03-40-management-of-service-instances-and-bindings.md)
* [Service Binding Rotation](./docs//user/03-50-service-binding-rotation.md)
* [Formats of Service Binding Secrets](./docs//user/03-60-formatting-service-binding-secret.md)
* [Resources](./docs/user/resources/README.md)
  * [SAP BTP Operator Custom Resource](./docs/user/resources/02-10-sap-btp-operator-cr.md)
  * [Service Instance Custom Resource](./docs/user/resources/02-20-service-instance-cr.md)
  * [Service Binding Custom Resource](./docs/user/resources/02-30-service-binding-cr.md)
* [Tutorials](./docs/user/tutorials/README.md)
  * [Create a BTP Manager Secret](./docs/user/tutorials/04-10-create-btp-manager-secret.md)
  * [Create a SAP BTP Service Operator Secret](./docs/user/tutorials/04-20-create-btp-service-operator-secret.md)
  * [Install a Secret](./docs/user/tutorials/04-30-install-secret.md)
  * [Create an SAP BTP Service Instance in Your Kyma Cluster](./docs/user/tutorials/04-40-create-service-in-cluster.md)
  * [Create a Service Instance with a Custom Secret](./docs/user/tutorials/04-50-create-service-instance-with-custom-secret.md)

## Contributing
<!--- mandatory section - do not change this! --->

See the [Contributing](CONTRIBUTING.md) guidelines.

## Code of Conduct
<!--- mandatory section - do not change this! --->

See the [Code of Conduct](CODE_OF_CONDUCT.md) document.

## Licensing
<!--- mandatory section - do not change this! --->

See the [license](./LICENSE) file.