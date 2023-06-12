# BTP Manager

## Overview

BTP Manager is an operator for [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator) based on the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator](https://github.com/kyma-project/btp-manager/blob/main/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) CRD which allows you to manage SAP BTP Service Operator resource through CR. 

## Install BTP Manager

 You can install BTP Manager locally or in your cluster. For more information, read the [Install and uninstall BTP Manager](docs/contributor/01-10-installation.md) document.

## Usage

You use BTP Manager to install or uninstall SAP BTP Service Operator. To find out how to do it, read the [Usage](docs/user/02-10-usage.md) document.

## Read more

If you want to provide new features for BTP Operator, visit the [`contributor`](docs/contributor) folder. For details on BTP Manager's operations, releases, testing and more, read the following documents:

- [Install and uninstall BTP Manager](docs/contributor/01-10-installation.md)
- [BTP Manager operations](docs/contributor/02-10-operations.md)
- [BTP Manager release pipeline](docs/contributor/03-10-release.md)
- [GitHub Actions workflows](docs/contributor/04-10-workflows.md)
- [Run unit tests](docs/contributor/05-10-testing.md)
- [Run E2E tests](docs/contributor/05-20-e2e_tests.md)
- [Certification management](docs/contributor/06-10-certs.md)

If you want to use the BTP Operator module, visit the [`user`](docs/user) folder to find more information on the following topics:

- [BTP Operator module](docs/user/README.md)
- [Install BTP Operator](docs/user/01-10-installation.md)
- [Configuration](docs/user/01-20-configuration.md)
- [Use BTP Manager to manage SAP BTP Service Operator](docs/user/02-10-usage.md)
- [Troubleshooting guide](docs/user/03-10-troubleshooting.md)
