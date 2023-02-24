---
title: E2E tests of btp-manager installation from OCI module image
---

## Overview

End-to-end (E2E) tests currently check if you can install and then uninstall btp-manager using OCI module image and Helm.
The flow is as follows:
- create OCI module image
- push the image to the registry
- create a Kubernetes cluster
- wait for the OCI module image to be available in the registry
- fetch the OCI module image
- Helm install the btp-manager chart
- helm uninstall btp-manager 

### CI pipelines
The OCI module image is created by prow presubmit job named 'pull-btp-manager-module-build'. Actual execution is done by `./hack/create_module_image.sh` script.
This script sets appropriate environment variables and invokes `make module-build`. In effect the module is built and OCI module image is pushed to the registry. 
Registry url and component name are predefined. 

> **NOTE:**
> The image tag has form 0.0.0-PR-<PR number> due to component description requirements imposed by used tooling.
 
Test are executed by Github Actions workflow (`e2e-test-k3s.yaml`). The k3s cluster is created, sources are checked out.
The workflow waits till the OCI module image is available for fetching.
The OCI module image is fetched from the registry by the `./testing/run_e2e_module_tests.sh` script. This script creates the required prerequisites, installs the Helm chart, and uninstalls it.

### Run E2E tests locally

For local tests, you can use the OCI module image from the official registry (that is, the module image created by the Prow presubmit job) or you can use the local Docker registry.
For example, to create an OCI module based on current sources and push it to the local Docker registry, you can use the following command (adjusting the tag appropriately):
```shell
make module-build IMG=component-descriptors/kyma.project.io/module/btp-operator:0.0.5-PR-176 MODULE_REGISTRY=localhost:5001/unsigned MODULE_VERSION=0.0.5-PR-176
```

Then you can run the E2E tests by issuing:
```shell
./testing/run_e2e_module_tests.sh localhost:5001/unsigned/component-descriptors/kyma.project.io/module/btp-operator:0.0.5-PR-176
```
