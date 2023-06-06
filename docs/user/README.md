---
title: BTP Operator Module
---


## Overview

Within the BTP Operator module, [BTP Manager](../../README.md) installs [SAP BTP Service Operator](https://github.com/SAP/sap-btp-service-operator).

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Kubernetes cluster (you can use [k3d](https://k3d.io)) 

## Enable BTP Operator module

To enable the BTP Operator module, use the following command:

```
kubectl apply -f https://github.com/kyma-project/btp-manager/releases/latest/download/btp-manager.yaml
```
Use the same command to upgrade the module to the latest version.

For more details on other installation options, read the [Install and uninstall BTP Manager](../contributor/01-01-installation.md) document.

## How BTP Operator module works

BTP Manager provisions, updates, and deprovisions SAP BTP Service Operator along with its resources, Service Instances, and Service Bindings. SAP BTP Service Operator manages SAP BTP services in your cluster.

Read [BTP Manager operations](../contributor/02-01-operations.md) to learn more. 

## Read more

This directory contains the end-user documentation of the BTP Operator module.  

For general information on BTP Manager, see the overarching [documentation](../../README.md), and for more details, read the following documents:

- [Configuration](01-01-configuration.md)
- [Use BTP Manager to manage SAP BTP Service Operator](02-01-usage.md)
- [Troubleshooting guide](03-01-troubleshooting.md)
