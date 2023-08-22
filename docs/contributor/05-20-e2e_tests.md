# E2E tests of BTP Manager installation from OCI module image

## Overview

The end-to-end (E2E) tests check if you can install, upgrade and uninstall BTP Manager using an OCI module image.
There are two tests:
- `e2e-tests` for checking installation and uninstallation of a given BTP Manager version
- `e2e-upgrade-tests` for checking BTP Manager upgradability from one version to another

The flows of the tests are similar. The upgrade tests contain extra steps for checking whether BTP Manager works as expected after upgrading to a new version. You can see the differences between the tests in the descriptions of the tests' flows below.

#### E2E tests for installation and uninstallation flow
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
13. Try to uninstall BTP Operator without the `force delete` label.
14. Verify if the deprovisioning safety measures work.
15. Add the `force delete` label to BtpOperator CR.
16. Verify if BTP Operator, ServiceInstance CRD and ServiceBinding CRD were deleted.
17. Uninstall BTP Manager. 

#### E2E tests for upgradability flow:
1. Create an OCI module image.
2. Push the image to the registry.
3. Create a Kubernetes cluster.
4. Wait for the new btp-operator OCI module image to be available in the registry.
5. Wait for the new btp-manager image to be available in the registry.
6. Download the latest btp-operator OCI module image.
7. Install the latest release of BTP Manager using `kubectl apply`.
8. Verify if deployment is in the `Available` state.
9. Install BTP Operator.
10. Verify if BTP Operator has the `Ready` status equal to `True`.
11. Create a Service Instance and Service Binding with real credentials.
12. Verify if the Service Instance and Service Binding have the `Ready` status `True`.
13. Download the new btp-operator OCI module image.
14. Upgrade BTP Manager to the new version using `kubectl apply`.
15. Verify if deployment is in the `Available` state.
16. Verify if the existing Service Instance and Service Binding have the `Ready` status `True`.
17. Create a new Service Binding with real credentials.
18. Verify if the new Service Binding has the `Ready` status `True`.
19. Try to uninstall BTP Operator without the `force delete` label.
20. Verify if the deprovisioning safety measures work.
21. Add the `force delete` label to BtpOperator CR.
22. Verify if BTP Operator, ServiceInstance CRD and ServiceBinding CRD were deleted.
23. Uninstall BTP Manager.

### CI pipelines
The Prow presubmit job, `pull-btp-manager-module-build`, creates the OCI module image. The [`create_module_image.sh`](../../scripts/create_module_image.sh) script does the actual execution.
This script sets appropriate environment variables and invokes `make module-build`. In effect, the module is built, and the OCI module image is pushed to the registry. 
The registry URL and component name are predefined. 

> **NOTE:**
> For PR workflow runs, the module image tag has the form `0.0.0-PR-<PR number>` due to component description requirements imposed by the tooling used.
 
The GitHub Actions workflows execute the two tests:
- [`run-e2e-tests-reusable.yaml`](../../scripts/testing/run_e2e_module_tests.sh) 
-  [`run-e2e-upgrade-tests-reusable.yaml`](../../scripts/testing/run_e2e_module_upgrade_tests.sh) 
<br>

The Kubernetes cluster is created, and the sources are checked out.
The workflows wait till the OCI module image is available for fetching.
The scripts fetch the OCI module image from the registry. They create the required prerequisites, 
get the BTP Manager and BTP Operator installed or upgraded, validate expected statuses, and get BTP Operator and BTP Manager uninstalled.

### Run E2E tests locally on k3d cluster
> **NOTE:**
> Valid only for the installation and uninstallation e2e tests.

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
1. Creates the binary `btp-manager:${PR_NAME}` image, and pushes it to the k3d registry.
2. Creates the OCI module image `component-descriptors/kyma.project.io/module/btp-operator:0.0.0-${PR_NAME}`, and pushes the module to the k3d registry.
3. Downloads the btp-operator OCI module image from k3d registry.
4. Installs BTP Manager, BTP Operator, Service Instance, and Service Binding.
5. Verifies states of resources.
6. Uninstalls BTP Operator and BTP Manager.
