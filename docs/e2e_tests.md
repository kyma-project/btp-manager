---
title: E2E tests of btp-manager installation from OCI module image
---

## Overview

End-to-end (E2E) tests currently check if you can install and then uninstall btp-manager using OCI module image and Helm.
The flow is as follows:
1. create an OCI module image
2. push the image to the registry
3. create a Kubernetes cluster
4. wait for the btp-operator OCI module image to be available in the registry
5. wait for the btp-manager image to be available in the registry
6. fetch the btp-operator OCI module image
7. Helm install btp-manager from the chart
8. Verification if deployment is in Available state
9. Helm uninstall btp-manager 

### CI pipelines
The OCI module image is created by the Prow presubmit job named 'pull-btp-manager-module-build'. The actual execution is done by the `./hack/create_module_image.sh` script.
This script sets appropriate environment variables and invokes `make module-build`. In effect, the module is built, and the OCI module image is pushed to the registry. 
The registry URL and component name are predefined. 

> **NOTE:**
> The module image tag has the form 0.0.0-PR-<PR number> due to component description requirements imposed by the tooling used.
 
The tests are executed by Github Actions workflow (`e2e-test-k3s.yaml`). The Kubernetes cluster is created, and sources are checked out.
The workflow waits till the OCI module image is available for fetching.
The OCI module image is fetched from the registry by the `./testing/run_e2e_module_tests.sh` script. This script creates the required prerequisites, installs the Helm chart, and uninstalls it.

### Run E2E tests locally

For local tests, you can use the OCI module image from the official registry (that is, the module image created by the Prow presubmit job) 
or you can use the local Docker registry.
For example, to create an OCI module based on binary image from official registry (signed image) and push it to the local Docker registry, you can use the following command (adjusting the tag appropriately):

```shell
make module-build IMG=europe-docker.pkg.dev/kyma-project/dev/btp-manager:PR-176 MODULE_REGISTRY=localhost:5001/unsigned MODULE_VERSION=0.0.7-PR-176
```

Then you can run the E2E tests by issuing:
```shell
./testing/run_e2e_module_tests.sh localhost:5001/unsigned/component-descriptors/kyma.project.io/module/btp-operator:0.0.7-PR-176
```

If you want to use locally created (unsigned) binary image, stored in local docker registry, you need to build it first.
```shell
make module-image IMG_REGISTRY=localhost:5001 IMG=localhost:5001/btp-manager-local:PR-176
```

Then you create locally OCI module image referencing binary image you just created and run the tests.
```shell
make module-build IMG=localhost:5001/btp-manager-local:PR-176 MODULE_REGISTRY=localhost:5001/unsigned MODULE_VERSION=0.0.8-PR-176
./testing/run_e2e_module_tests.sh localhost:5001/unsigned/component-descriptors/kyma.project.io/module/btp-operator:0.0.8-PR-176
```

