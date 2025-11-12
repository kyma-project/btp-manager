[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/btp-manager)](https://api.reuse.software/info/github.com/kyma-project/btp-manager)

# BTP Manager

## Overview

BTP Manager is an operator for the [SAP BTP service operator](https://github.com/SAP/sap-btp-service-operator) based on the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator CustomResourceDefinition](/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) (CRD), which allows you to manage the SAP BTP service operator resource through custom resource (CR). BTP Manager and the SAP BTP service operator constitute the SAP BTP Operator module.

## Installation

To enable the SAP BTP Operator module from the latest release, you must install BTP Manager. For installation instructions, see [Install the SAP BTP Operator module](./docs/user/03-05-install-module.md).

## Usage

Use SAP BTP Operator to create SAP BTP services in your Kyma cluster. To find out how to do it, see the tutorial [Create an SAP BTP Service Instance in Your Kyma Cluster](./docs/user/tutorials/04-40-create-service-in-cluster.md).

## Uninstallation

To uninstall SAP BTP Operator, run the following commands:
```sh
kubectl delete -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-operator.yaml
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
* [Create the `sap-btp-manager` Secret](./docs/user/03-00-create-btp-manager-secret.md)
* [Environment Variables](./docs/user/03-01-environment-variables.md)
* [Install the SAP BTP Operator Module](./docs/user/03-05-install-module.md)
* [Preconfigured Credentials and Access](./docs/user/03-10-preconfigured-secret.md)
* [Working with Multiple Subaccounts](./docs//user/03-20-multitenancy.md)
* [Instance-Level Mapping](./docs/user/03-21-instance-level-mapping.md)
* [Namespace-Level Mapping](./docs/user/03-22-namespace-level-mapping.md)
* [Create  Service Instances and Service Bindings](./docs//user/03-30-create-instances-and-bindings.md)
* [Update Service Instances](./docs/user/03-31-update-service-instances.md)
* [Delete Service Bindigs and Service Instances](./docs/user/03-32-delete-bindings-and-instances.md)
* [Rotate Service Binding](./docs//user/03-40-service-binding-rotation.md)
* [Formats of Service Binding Secrets](./docs//user/03-50-formatting-service-binding-secret.md)
* [Pass Parameters](./docs/user/03-60-pass-parameters.md)
* [Resources](./docs/user/resources/README.md)
  * [SAP BTP Operator Custom Resource](./docs/user/resources/02-10-sap-btp-operator-cr.md)
  * [Service Instance Custom Resource](./docs/user/resources/02-20-service-instance-cr.md)
  * [Service Binding Custom Resource](./docs/user/resources/02-30-service-binding-cr.md)
* [Tutorials](./docs/user/tutorials/README.md)
  * [Create an SAP BTP Service Instance in Your Kyma Cluster](./docs/user/tutorials/04-40-create-service-in-cluster.md)
* [Troubleshooting](./docs/user/troubleshooting/README.md)
  * [You Cannot Delete Leftover Service Instances and Bindings](./docs/user/troubleshooting/05-01-leftover-resources.md)
  * [Resource CR Missing from the Cluster](./docs/user/troubleshooting/05-02-resource-cr-missing-from-cluster.md)

## Contributing
<!--- mandatory section - do not change this! --->

See the [Contributing](CONTRIBUTING.md) guidelines.

## Code of Conduct
<!--- mandatory section - do not change this! --->

See the [Code of Conduct](CODE_OF_CONDUCT.md) document.

## Licensing
<!--- mandatory section - do not change this! --->

See the [license](./LICENSE) file.