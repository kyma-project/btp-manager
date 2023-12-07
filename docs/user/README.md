# SAP BTP Operator Module

## Overview

Within the SAP BTP Operator module, [BTP Manager](https://github.com/kyma-project/btp-manager) installs the [SAP BTP service operator](https://github.com/SAP/sap-btp-service-operator/blob/main/README.md).

### BTP Manager

BTP Manager is an operator based on the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator CustomResourceDefinition](https://github.com/kyma-project/btp-manager/blob/main/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) (CRD) which allows you to manage the SAP BTP service operator resource through custom resource (CR). 

### SAP BTP Service Operator

The SAP BTP service operator allows you to connect SAP BTP services to your cluster and then manage them using Kubernetes-native tools.

## How SAP BTP Operator Module Works

BTP Manager provisions, updates, and deprovisions the SAP BTP service operator along with its resources, ServiceInstances, and ServiceBindings. The SAP BTP service operator manages SAP BTP services in your cluster.

Read [BTP Manager Operations](../contributor/02-10-operations.md) and the [SAP BTP service operator](https://github.com/SAP/sap-btp-service-operator) documentation to learn more.

## Read More

This directory contains the end user documentation of the SAP BTP Operator module.  
For the module's usage details, read the [Use BTP Manager to Manage SAP BTP Service Operator](02-10-usage.md) document.
