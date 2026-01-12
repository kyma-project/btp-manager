# GitHub Actions Workflows

## Auto Update Chart and Resources

The goal of the workflow is to update the chart (`module-chart/chart`) to the newest version, upload the operand's container images to the Kyma container images repository (`europe-docker.pkg.dev/kyma-project/prod`), render the resource templates from the newest chart, and create a PR with the changes. The created pull request requires a manual approval. All GitHub checks are triggered on the pull request. The workflow works on the following four shell scripts:

- `make-module-chart.sh` - downloads the chart from the [upstream](https://github.com/SAP/sap-btp-service-operator), from the release tagged as `latest` and places it in the `module-chart/chart`. 

- `update-external-images.sh` - updates `external-images.yaml`, which lists the operand's container images for uploading to the Kyma container images repository. The script extracts images from `module-chart/chart/values.yaml` by searching for the **repository** and **tag** fields.

After both scripts run, the workflow creates a pull request that is auto-merged into the `main` branch. Only a limited subset of GitHub checks is executed. The merge commit triggers the [sync-external-images](../../.github/workflows/sync-external-images.yaml) workflow, which runs the [image-syncer](https://github.com/kyma-project/test-infra/tree/main/cmd/image-syncer) tool to upload the operand's container images into the Kyma container images repository.
	
- `make-module-resources.sh` - uses Helm to render Kubernetes resources templates. As a base, it takes a chart from the `module-chart/chart` and values to [override](../../module-chart/overrides.yaml). After Helm finishes templating with the applied overrides, the generated resources are put into `module-resources/apply`. The script removes all container images for the operator from the chart's [deployment manifest](../../module-resources/apply/deployment.yml). The operator sets the container images dynamically by reading [environment variables with container images](../../config/manager/set_external_images.yaml). The resources used in the previous version but not used in the new version are placed under `module-resource/delete`.
During the process of iterating over the `sap-btp-service-operator` resources, the script also keeps track of the GVKs to generate RBAC rules in [`btpoperator_controller.go`](https://github.com/kyma-project/btp-manager/blob/5a8420347c6a526f158fde7c41c3842eb54e2fda/controllers/btpoperator_controller.go#L135-L146), which feeds into RBAC **ClusterRole** in the [`role.yaml`](https://github.com/kyma-project/btp-manager/blob/5a8420347c6a526f158fde7c41c3842eb54e2fda/config/rbac/role.yaml#L1) resource kept in sync with `make manifests` just like any standard [kubebuilder operator](https://book-v2.book.kubebuilder.io/reference/markers/rbac.html). The script excludes all resources with a Helm hook "pre-delete" as it is not necessary to apply it. Additionally, all excluded resources are added to the `module-resources/excluded` folder, where you can see what was excluded.

- `add_module_managed_images_envs.sh` - sets container images environment variables in [`set_external_images.yaml`](../../config/manager/set_external_images.yaml) based on `external-images.yaml`. The script has strict definitions of environment variables and container images and requires manual adjustments when new container images appear in the upstream chart.
 
All these scripts can be triggered manually from the developer's computer.

To keep `module-chart/chart` in sync with the [upstream](https://github.com/SAP/sap-btp-service-operator), you must not apply any manual changes there.

## Release Workflow

See [BTP Manager Release Pipeline](03-10-release.md) to learn more about the release workflow.

## E2E Tests Workflow 

This workflow uses the DEV artifact registry, tags the binary image and OCI module image with the PR number, and calls the [`run-e2e-tests-reusable`](/.github/workflows/run-e2e-tests-reusable.yaml) workflow.

## Unit Tests Workflow

This workflow calls the [`run-e2e-tests-reusable`](/.github/workflows/run-unit-tests-reusable.yaml) workflow.

## Markdown Links Check Workflow

The [`markdown-link-check`](/.github/workflows/markdown-link-check.yaml) workflow is triggered daily at midnight and by each PR on the `main` branch. It checks for dead links in the repository.

## Govulncheck Workflow

The [`run-govulncheck`](/.github/workflows/run-govulncheck.yaml) workflow runs the Govulncheck.

## Auto Merge Workflow

The [`auto-merge`](/.github/workflows/auto-merge.yaml) workflow enables the auto-merge functionality on a PR that is not a draft.

## All Checks Passed Workflow

The [`pr-checks`](/.github/workflows/pr-checks.yaml) workflow checks if all jobs, except those excluded in the workflow configuration, have passed.

## Module Catalog Protection Workflow

This [`module-catalog-protection`](/.github/workflows/module-catalog-protection.yaml) workflow ensures that changes to module catalog directories (`module-chart/` and `module-resources/`) are properly restricted. The workflow is triggered when a pull request contains changes to these protected directories.

### Protection Rules

1. Only the `kyma-gopher-bot` user can make changes to these directories.
2. Other users can override the protection by adding the `module-catalog-override` label to their PR.

### Workflow Behavior

- If the PR author is `kyma-gopher-bot`, the workflow passes.
- If the PR contains the `module-catalog-override` label, the workflow passes.
- If neither condition is met, the workflow fails with a detailed error message explaining how to resolve the issue.

## E2E BTP Manager Secret Customization Test Workflow

The [`run-e2e-sap-btp-manager-secret-customization-test`](/.github/workflows/run-e2e-sap-btp-manager-secret-customization-test.yaml) workflow runs the E2E BTP Manager secret customization tests by calling the [`run-e2e-sap-btp-manager-secret-customization-test-reusable`](/.github/workflows/run-e2e-sap-btp-manager-secret-customization-test-reusable.yaml) workflow.

## Upload Release Logs as Assets Workflow

The [`upload-release-logs`](/.github/workflows/upload-release-logs.yml) workflow uploads the logs from the release workflow as assets to the corresponding GitHub release. It is triggered on every published release event.

The workflow performs the following steps:

1. Checks out the repository
2. Waits for the "Create and promote release" workflow to finish if it is still in progress
3. Downloads logs from all attempts of the "Create and promote release" workflow for the current release
4. Uploads the downloaded logs as assets to the current GitHub release

## Reusable Workflows

There are reusable workflows created. Anyone with access to a reusable workflow can call it from another workflow.

### E2E Tests

The [`run-e2e-tests-reusable`](/.github/workflows/run-e2e-tests-reusable.yaml) workflow runs the E2E tests on the k3s cluster. 
You pass the following parameters from the calling workflow:

| Parameter name        | Required | Description                                                            |
|-----------------------|----------|------------------------------------------------------------------------|
| **image-repo**        | yes      | Binary image registry reference                                        |
| **image-tag**         | yes      | Binary image tag                                                       |
| **last-k3s-versions** | no       | Number of most recent k3s versions to be used for tests, default = `1` |


The workflow performs the following actions:
- Fetches the **last-k3s-versions** tag versions of k3s releases 
- Prepares the **last-k3s-versions** k3s clusters with the Docker registries using the list of versions from the previous step
- Waits for the binary image to be ready in the registry
- Runs the E2E tests on the clusters
- Waits for all tests to finish

### Unit Tests

The [`run-unit-tests-reusable`](/.github/workflows/run-unit-tests-reusable.yaml) workflow runs the unit tests.
No parameters are passed from the calling workflow (callee).

The workflow performs the following actions:
- Checks out code and sets up the cache
- Sets up the Go environment
- Invokes `make test`

### E2E BTP Manager Secret Customization Test

The [`run-e2e-sap-btp-manager-secret-customization-test-reusable`](/.github/workflows/run-e2e-sap-btp-manager-secret-customization-test-reusable.yaml) workflow runs the E2E BTP Manager secret customization test on the k3s cluster.
The following parameters are required from the calling workflow:

| Parameter name     | Required | Description                     |
|--------------------|----------|---------------------------------|
| **image-registry** | yes      | Binary image registry reference |
| **image-tag**      | yes      | Binary image tag                |

The workflow performs the following actions:
- Prepares the k3s cluster with the Docker registry
- Waits for the binary image to be ready in the registry
- Installs the module
- Runs the E2E BTP Manager secret customization test on the cluster

### Performance Tests

The [`run-performance-tests-reusable`](/.github/workflows/run-performance-tests-reusable.yaml) workflow runs performance tests on the k3s cluster. The following parameters are required from the calling workflow:

| Parameter name       | Required | Description                     |
|----------------------|----------|---------------------------------|
| **image-repo**       | yes      | Binary image registry reference |
| **image-tag**        | yes      | Binary image tag                |
| **credentials-mode** | yes      | Specifies whether to use real or dummy credentials |

The workflow performs the following actions for all jobs:
- Prepares the k3s cluster with the Docker registry
- Waits for the binary image to be ready in the registry
- Installs the module
<!--Why the change in formatting below?-->
<details>
<summary>Frequent Secret Update Test</summary>

- **Purpose**: Evaluates the system's response time and reconciliation success rate when the `sap-btp-manager` Secret is updated frequently.
- **Steps**:
    - Patches the `sap-btp-manager` Secret in a loop to simulate frequent updates.
    - Fetches metrics from `btp-manager-controller-manager` to measure average reconciliation time, reconciliation errors, and other reconciliation statistics.
- **The test fails in the following conditions**:
    - Average reconciliation time exceeds the defined threshold.
    - Reconciliation errors are detected.

</details>

<details>
<summary>Reconciliation After Secret Deletion Test</summary>

- **Purpose**: Measures the reconciliation performance of BTP Manager when the `sap-btp-manager` Secret is repeatedly deleted and reapplied.
- **Steps**:
    - Deletes and reapplies the `sap-btp-manager` Secret in a loop to simulate different BtpOperator statuses.
    - Fetches metrics from `btp-manager-controller-manager` to measure average and maximum reconciliation time, and counts the number of reconciliation errors.
- **The test fails in the following conditions**:
    - Average reconciliation time exceeds the defined threshold.
    - Reconciliation errors are detected.

</details>

<details>
<summary>Reconciliation After Crash Test</summary>

- **Purpose**: Tests the system's recovery and reconciliation performance after scaling down and scaling up the `btp-manager-controller-manager` deployment.
- **Steps**:
    - Scales down the `btp-manager-controller-manager` deployment to simulate a crash.
    - Deletes Secrets and ConfigMaps managed by BTP Manager to simulate missing resources.
    - Scales up the `btp-manager-controller-manager` deployment and waits for reconciliation.
    - Measures the time taken for reconciliation and verifies the recreation of managed resources.
- **The test fails in the following conditions**:
    - Reconciliation process does not complete within the expected time.
    - Managed resources are not recreated successfully.

</details>

<details>
<summary>Installation Duration Test</summary>

- **Purpose**: Measures the time taken to install and uninstall BTP Manager and BtpOperator, and the duration of certificate generation.
- **Steps**:
    - Installs BTP Manager and measures the installation duration.
    - Applies BtpOperator and measures the time taken to reach the `Ready` state.
    - Deletes and regenerates certificates to measure the duration of certificate regeneration.
    - Deletes BtpOperator and BTP Manager, measuring the time taken for each operation.
- **The test fails in the following conditions**:
    - Installation or uninstallation of BTP Manager or BtpOperator exceeds the expected duration.
    - Certificate regeneration process does not complete within the expected time.

</details>
