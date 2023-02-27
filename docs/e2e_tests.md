---
title: E2E tests of btp-manager installation from OCI module image
---

## Overview

End-to-end (E2E) tests currently check if you can install and then uninstall btp-manager using OCI module image and Helm.
The flow is as follows:
- create an OCI module image
- push the image to the registry
- create a Kubernetes cluster
- wait for the btp-operator OCI module image to be available in the registry
- wait for the btp-manager image to be available in the registry
- fetch the btp-operator OCI module image
- Helm install btp-manager from the chart
- Verification if deployment is in Available state
- Helm uninstall btp-manager 

### CI pipelines
The OCI module image is created by prow presubmit job named 'pull-btp-manager-module-build'. Actual execution is done by `./hack/create_module_image.sh` script.
This script sets appropriate environment variables and invokes `make module-build`. In effect the module is built and OCI module image is pushed to the registry. 
Registry url and component name are predefined. 

> **NOTE:**
> The module image tag has form 0.0.0-PR-<PR number> due to component description requirements imposed by tooling used.
 
Test are executed by Github Actions workflow (`e2e-test-k3s.yaml`). The k3s cluster is created, sources are checked out.
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

