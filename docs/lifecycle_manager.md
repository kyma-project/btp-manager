---
title: Install BTP Manager using Lifecycle Manager
---

## Overview

This is an experimental way of installing BTP Manager in the cluster. 

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [k3d](https://k3d.io)

## Quick-Start - Install script

Use the following command to run the BTP Manager via Lifecycle Manager. 

```shell
./hack/run_lifecycle_manager.sh https://github.com/kyma-project/btp-manager/releases/latest/download/template.yaml
```

It results in:
- downloading and using Kyma CLI to provision the k3d cluster,
- deploying Lifecycle Manager,
- applying BTP Manager template.yaml provided by the user,
- enabling the BTP Operator module,
- displaying the BTP Manager and BTP Operator status.

## Delete k3d cluster

```shell
k3d cluster delete kyma
```