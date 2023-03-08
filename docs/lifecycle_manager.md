---
title: Run BTP Manager installation via Lifecycle Manager
---

## Overview

This is an experimental way of installing BTM Manager in the cluster. 

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [k3d](https://k3d.io)

## Quick-Start - Install script

Use the following command to run the BTP Manager via Lifecycle Manager. In a nutshell, that is what it does:
- downloads and uses Kyma CLI to provision the k3d cluster,
- deploys Lifecycle Manager,
- applies BTP Manager template.yaml provided by the user,
- enables BTP Operator module,
- displays BTP Manager and BTP Operator status

```shell
./hack/run_lifecycle_manager.sh https://github.com/kyma-project/btp-manager/releases/download/0.2.3/template.yaml
```

## Delete k3d cluster

```shell
k3d cluster delete kyma
```