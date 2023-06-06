# BTP Manager

## Overview

BTP Manager is an operator for [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator) based on the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator](https://github.com/kyma-project/btp-manager/blob/main/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) CRD which allows you to manage SAP BTP Service Operator resource through CR. 

## Install BTP Manager

 You can install BTP Manager locally or in your cluster. For more information, read the [Install and uninstall BTP Manager](docs/contributor/01-01-installation.md) document.

## Usage

You use BTP Manager to install or uninstall SAP BTP Service Operator. To find out how to do it, read the [Usage](docs/user/02-01-usage.md) document.

## Read more

Visit the [`contributor`](docs/contributor) folder to read more on the following topics:

- [Install and uninstall BTP Manager](docs/contributor/01-01-installation.md)
- [BTP Manager operations](docs/contributor/02-01-operations.md)
- [BTP Manager release pipeline](docs/contributor/03-01-release.md)
- [Run unit tests](docs/contributor/05-01-testing.md)
- [Run E2E tests](docs/contributor/05-02-e2e_tests.md)
- [GitHub Actions workflows](docs/contributor/04-01-workflows.md)
- [Certification management](docs/contributor/06-01-certs.md)

For information on the BTP Operator module, visit the [`user`](docs/user) and read the following documents:

- [BTP Operator Module](docs/user/README.md)
- [Configuration](docs/user/01-01-configuration.md)
- [Use BTP Manager to manage SAP BTP Service Operator](docs/user/02-01-usage.md)
- [Troubleshooting guide](docs/user/03-01-troubleshooting.md)
