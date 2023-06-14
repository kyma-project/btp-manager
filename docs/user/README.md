---
title: BTP Operator module
---

## Overview

Within the BTP Operator module, [BTP Manager](../../README.md) installs [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator/blob/main/README.md).

## BTP Manager

BTP Manager is an operator based on the [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework. It extends Kubernetes API by providing [BtpOperator](https://github.com/kyma-project/btp-manager/blob/main/config/crd/bases/operator.kyma-project.io_btpoperators.yaml) CRD which allows you to manage SAP BTP Service Operator resource through CR. 

## SAP BTP Service Operator

SAP BTP Service Operator allows you to connect SAP BTP services to your cluster and then manage them using Kubernetes-native tools.

## Enable BTP Operator module

To enable the BTP Operator module, follow the steps described in the [Install BTP Operator](01-10-installation.md) document.

## How BTP Operator module works

BTP Manager provisions, updates, and deprovisions SAP BTP Service Operator along with its resources, ServiceInstances, and ServiceBindings. SAP BTP Service Operator manages SAP BTP services in your cluster.

Read [BTP Manager operations](../contributor/02-10-operations.md) and the [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator) documentation to learn more.

## Contact and support

For contact and support details, go to {link}.

## Read more

This directory contains the end-user documentation of the BTP Operator module.  

For general information on BTP Manager, see the overarching [documentation](../../README.md), and for more details on the BTP Operator module, read the following documents:

- [Install BTP Operator](01-10-installation.md)
- [Use BTP Manager to manage SAP BTP Service Operator](02-10-usage.md)
