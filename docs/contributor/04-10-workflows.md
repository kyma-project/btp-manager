# GitHub Actions Workflows

## Auto Update Chart and Resources

The goal of the workflow is to update the chart (`module-chart/chart`) to the newest version, render the resource templates from the newest chart, and create a PR with the changes. The workflow works on two shell scripts:

- `make-module-chart.sh` - downloads the chart from the [upstream](https://github.com/SAP/sap-btp-service-operator), from the release tagged as `latest` and places it in the `module-chart/chart`. 
	
- `make-module-resources.sh` - uses Helm to render Kubernetes resources templates. As a base, it takes a chart from the `module-chart/chart` and values to [override](../../module-chart/overrides.yaml). After Helm finishes templating with the applied overrides, the generated resources are put into `module-resources/apply`. The resources used in the previous version but not used in the new version are placed under `module-resource/delete`.
During the process of iterating over the `sap-btp-service-operator` resources, the script also keeps track of the GVKs to generate RBAC rules in [`btpoperator_controller.go`](https://github.com/kyma-project/btp-manager/blob/5a8420347c6a526f158fde7c41c3842eb54e2fda/controllers/btpoperator_controller.go#L135-L146) which feeds into RBAC `ClusterRole` in the [`role.yaml`](https://github.com/kyma-project/btp-manager/blob/5a8420347c6a526f158fde7c41c3842eb54e2fda/config/rbac/role.yaml#L1) resource
kept in sync with `make manifests` just like any standard [kubebuilder operator](https://book-v2.book.kubebuilder.io/reference/markers/rbac.html). The script excludes all resources with a Helm hook "pre-delete" as it is not necessary to apply it. Additionally, all excluded resources are added to the `module-resources/excluded` folder, where you can see what was excluded.
 
Both scripts are run from the workflow but can also be triggered manually from the developer's computer. They are placed under `hack/update/`.

To keep `module-chart/chart` in sync with the [upstream](https://github.com/SAP/sap-btp-service-operator), you must not apply any manual changes there.

## Release Workflow

See [BTP Manager Release Pipeline](03-10-release.md) to learn more about the release workflow.

## E2E Tests Workflow 

This workflow is triggered by pull requests (PRs) on the `main` branch. It uses the DEV artifact registry, tags the binary image and OCI module image with the PR number, and calls the reusable [workflow](/.github/workflows/run-e2e-tests-reusable.yaml). 

## Unit Tests Workflow

This workflow is triggered by PRs on the `main` branch. Then it calls the reusable [workflow](/.github/workflows/run-unit-tests-reusable.yaml).

## Markdown links check workflow

This [workflow](/.github/workflows/markdown-link-check.yaml) is triggered daily at midnight and by each PR on the `main` branch. It checks for dead links in the repository.

## Reusable workflows

There are reusable workflows created. Anyone with access to a reusable workflow can call it from another workflow.

### E2E Tests

This [workflow](/.github/workflows/run-e2e-tests-reusable.yaml) runs the E2E tests on the k3s cluster. 
You pass the following parameters from the calling workflow:

| Parameter name  | Required | Description                                                          |
| ------------- | ------------- |----------------------------------------------------------------------|
| **image-repo**  | yes  | binary image registry reference                                      |
| **module-repo**  | yes  | OCI module image registry reference                                  |
| **image-tag**  | yes  | binary image tag                                                     |
| **module-tag**  | yes  | OCI module image tag                                                 |
| **skip-templates**  | no  | wait for images only, skip other artifacts                           |
| **last-k3s-versions**  | no  | number of most recent k3s versions to be used for tests, default = `1` |


The workflow:
- fetches the **last-k3s-versions** tag versions of k3s releases 
- prepares the **last-k3s-versions** k3s clusters with the Docker registries using the list of versions from the previous step
- waits for the artifacts to be ready in the registry
- runs the E2E tests on the clusters
- waits for all tests to finish


### Unit Tests

This [workflow](/.github/workflows/run-unit-tests-reusable.yaml) runs the unit tests.
No parameters are passed from the calling workflow (callee).

The workflow:
- checks out code and sets up the cache
- sets up the Go environment
- invokes `make test`
