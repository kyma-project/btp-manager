# E2E Tests of BTP Manager Installation

## Overview

The end-to-end (E2E) tests check if you can install, upgrade and uninstall BTP Manager.
There are two tests:
- `e2e-tests` for checking installation and uninstallation of a given BTP Manager version
- `e2e-upgrade-tests` for checking BTP Manager upgradability from one version to another

The flows of the tests are similar. The upgrade tests contain extra steps for checking whether BTP Manager works as expected after upgrading to a new version. You can see the differences between the tests in the descriptions of the tests' flows below.

#### E2E Tests for Installation and Uninstallation Flow
1. Create a Kubernetes cluster.
2. Wait for the BTP Manager image to be available in the registry.
3. Install the BTP Manager using `make deploy`.
4. Verify if deployment is in the `Available` state.
5. Install BtpOperator. 
6. Verify if BtpOperator has the `Ready` status equal to `True`. 
7. Create a ServiceInstance and ServiceBinding with either real or dummy credentials. 
8. Verify if the ServiceInstance and ServiceBinding have the `Ready` status set to:
     - `True` if you use real credentials.
     - `NotProvisioned` if you use dummy credentials.
9. Try to uninstall BtpOperator without the `force delete` label. 
10. Verify if the deprovisioning safety measures work. 
11. Add the `force delete` label to BtpOperator custom resource (CR). 
12. Verify if BtpOperator, ServiceInstance CustomResourceDefinition (CRD) and ServiceBinding CRD were deleted. 
13. Uninstall BTP Manager. 

#### E2E Tests for Upgradability Flow:
1. Create a Kubernetes cluster. 
2. Wait for the new BTP Manager image to be available in the registry.
3. Download the manifest for the latest release.
4. Install the latest release of BTP Manager using `kubectl apply`.
5. Verify if deployment is in the `Available` state.
6. Install BtpOperator.
7. Verify if BtpOperator has the `Ready` status equal to `True`.
8. Create a ServiceInstance and ServiceBinding with real credentials.
9. Verify if the ServiceInstance and ServiceBinding have the `Ready` status set to `True`.
10. Upgrade BTP Manager to the new version using `make deploy`.
11. Verify if deployment is in the `Available` state.
12. Verify if the existing ServiceInstance and ServiceBinding have the `Ready` status set to `True`.
13. Create a new ServiceBinding with real credentials.
14. Verify if the new ServiceBinding has the `Ready` status set to `True`.
15. Try to uninstall BtpOperator without the `force delete` label.
16. Verify if the deprovisioning safety measures work.
17. Add the `force delete` label to BtpOperator CR.
18. Verify if BtpOperator, ServiceInstance CRD and ServiceBinding CRD were deleted.
19. Uninstall BTP Manager.

### CI Pipelines
 
The GitHub Actions workflows execute the two tests:
- [`run-e2e-tests-reusable.yaml`](../../scripts/testing/run_e2e_module_tests.sh) 
-  [`run-e2e-upgrade-tests-reusable.yaml`](../../scripts/testing/run_e2e_module_upgrade_tests.sh) 
<br>

The Kubernetes cluster is created, and the sources are checked out.
The workflows wait till the binary image is available for fetching.
The scripts create the required prerequisites, get the BTP Manager and BtpOperator installed or upgraded, validate expected statuses, and get BtpOperator and BTP Manager uninstalled.

### Real Credentials Rotation

Real credentials used in the test are configured as repository secrets.
The following secrets are used and substituted in the `sap-btp-manager` Kyma Secret resource:
- SM_CLIENT_ID - Service Manager client ID, `data.clientid`
- SM_CLIENT_SECRET - Service Manager client secret, `data.clientsecret`
- SM_URL - Service Manager URL, `data.sm_url`
- SM_TOKEN_URL - Service Manager token URL, `data.tokenurl`  
All secrets should be base64 encoded. Caveat of the new line character at the end of the secret value.

The following bash command could be used to encode the secret:

```echo -n "secret" | base64``` 

Currently used values are taken from Service Binding `e2e-test-sm` created for the `e2e-test-sm` Service Manager instance in the `e2e-test-btp-manager` subaccount of the `kyma-gopher` global account on the Canary environment.
In case of credentials rotation, the secrets should be updated in the repository secrets, regardless of the location and naming of the Service Manager instance and Secret Binding used.

