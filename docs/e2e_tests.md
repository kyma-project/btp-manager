---
title: E2E tests of btp-manager installation from OCI module image
---

## Overview

End-to-end (E2E) tests currently check if you can install and uninstall BTP Manager using an OCI module image.
The flow is as follows:
1. Create an OCI module image.
2. Push the image to the registry.
3. Create a Kubernetes cluster.
4. Wait for the btp-operator OCI module image to be available in the registry.
5. Wait for the btp-manager image to be available in the registry.
6. Download the btp-operator OCI module image.
7. Install the BTP Manager using `kubectl apply`.
8. Verify if deployment is in the `Available` state.
9. Install BTP Operator.
10. Verify if BTP Operator has the `Ready` status equal to `True`.
11. Create a Service Instance and Service Binding with either real or dummy credentials. 
12. When real credentials are used, verify if the Service Instance and Service Binding have the `Ready` status `True`. If dummy credentials are used, verify if the `Ready` status for both of them is`NotProvisioned`.
13. Uninstall BTP Operator.
14. Uninstall BTP Manager. 

### CI pipelines
The Prow presubmit job, `pull-btp-manager-module-build`, creates the OCI module image. The `./scripts/create_module_image.sh` script does the actual execution.
This script sets appropriate environment variables and invokes `make module-build`. In effect, the module is built, and the OCI module image is pushed to the registry. 
The registry URL and component name are predefined. 

> **NOTE:**
> The module image tag has the form `0.0.0-PR-<PR number>` due to component description requirements imposed by the tooling used.
 
The GitHub Actions workflow executes the tests (`run-e2e-tests-reusable.yaml`). The Kubernetes cluster is created, and the sources are checked out.
The workflow waits till the OCI module image is available for fetching.
The `./scripts/testing/run_e2e_module_tests.sh` script fetches the OCI module image from the registry. This script creates the required prerequisites, 
gets the BTP Manager and BTP Operator installed, validates expected statuses and the gets BTP Operator and BTP Manager uninstalled.

### Run E2E tests locally on k3d cluster

For local tests, you can use the OCI module image from the official registry (that is, the module image created by the Prow presubmit job) 
or you can use the local registry.
The easiest way is to create a k3d cluster and a local registry by running the following command:

```shell
kyma provision k3d
```

The `k3d-kyma` cluster will be created along with the k3d registry `k3d-kyma-registry:5001`.

Now you can run E2E tests. Setting PR_NAME allows you to control the image tag.
If you want to tag the images with `PR-234`, run the following script:

```shell
PR_NAME=PR-234 ./scripts/testing/run_e2e_on_k3d.sh
```

The script:
1. creates the binary `btp-manager:${PR_NAME}` image, and pushes it to the k3d registry.
2. creates the OCI module image `component-descriptors/kyma.project.io/module/btp-operator:0.0.0-${PR_NAME}`, and pushes the module to the k3d registry.
3. downloads the btp-operator OCI module image from k3d registry
4. installs BTP Manager, BTP Operator, Service Instance, and Service Binding.
5. verifies states of resources.
6. uninstalls BTP Operator and BTP Manager.