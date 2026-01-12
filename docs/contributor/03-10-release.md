# SAP BTP Operator Module Release and Promotion

## Overview

SAP BTP Operator's release and promotion process includes the following stages:

 - BTP Manager release - the process of creating a new release of BTP Manager, which includes building and testing it, creating a GitHub tag, and publishing the release.
 - Module Version Submit - the process of submitting a new version of BTP Manager to the `module-manifests` repository, which includes creating a PR with the `module-config.yaml` file for the new version.
 - Promotion to Channel - the process of promoting a released version of BTP Manager to a specific channel, which includes creating a PR in the `module-manifests` repository with the `module-releases.yaml` file modified in the section for the specified channel.
 
## Scenarios

### Release Only

Executing only the release is useful when you want to create a new release of BTP Manager or the SAP BTP Operator module without submitting a new version to the `module-manifests` repository.
You only create proper artifacts (a GitHub tag, Docker images, and release notes), but do not open a PR in the `module-manifests` repository.
To execute only the release, use the **Create release** GitHub action.
A module version not submitted to the `module-manifests` repository is not available for the `kyma-modules` repository, and cannot be used in the Kyma installation.

> [!NOTE]
> To submit a new version of the SAP BTP Operator module to the `module-manifests` repository later, use the **Submit module version** GitHub action.

### Release and Submit

Execute the release and submit scenario when you want to create a new release of the SAP BTP Operator module and submit a new version to the `module-manifests` repository. You create proper artifacts (a GitHub tag, Docker images, release notes) and a PR in the `module-manifests` repository.
To execute the release and submit scenario, use the **Create release with version submit** GitHub action.

### Release, Submit, and Promote

Execute the release, submit, and promote scenario when you want to create a new release of the SAP BTP Operator module, submit a new version to the `module-manifests` repository, and promote the released version to a specific channel. You create proper artifacts (a GitHub tag, Docker images, release notes), and open two PRs in the `module-manifests` repository - one for the `module-config.yaml` file and one for the `module-releases.yaml` file.
Execute the release, submit, and promote scenario in one of the following ways:
   - Use the **Create release with version submit** GitHub action and then, the **Promote module to channel** GitHub action to promote the released version to a specific channel.
   - Use the following GitHub actions: first **Create release**, then **Submit module version**, and finally **Promote module to channel**.


## Create a Release

![Release diagram](../assets/release.drawio.svg)

To create a release, follow these steps:

1. Run the **Create release** GitHub action:  
   i.  Go to the **Actions** tab, and choose the **Create release** workflow, and next **Run workflow**.  
   ii. Provide a version, for example, 1.2.0.
   iii. Choose real or dummy credentials for Service Manager.
   iv. Choose whether to bump or not to bump the security scanner config.
   v. Choose whether you want to publish the release.
2. The GitHub action, defined in the [`create-release`](/.github/workflows/create-release.yaml) file, validates the release by checking if the GitHub tag already exists, if there are any old Docker images for that GitHub tag, and if merged PRs that are part of this release are labeled correctly. Additionally, it stops the release process if a feature has been added, but only the patch version number has been bumped up.
3. The GitHub action asynchronously initiates unit tests.
4. The Image Builder builds binary images.
5. The Image Builder uploads the binary images to the Docker registry.
6. The GitHub action initiates test jobs (stress tests, performance tests, upgrade tests, secret customization tests) using the built image. E2E upgrade tests run only with real credentials for SAP Service Manager. E2E tests are executed in parallel on the k3s clusters for the most recent k3s versions and with the specified credentials. The number of the most recent k3s versions to be used is defined in the **vars.LAST_K3S_VERSIONS** GitHub variable.
7. If in step "Run the **Create release** GitHub action", you chose to bump the security scanner config, the GitHub action creates a PR with a new security scanner config that includes the new GitHub tag version.
8. The GitHub action creates a GitHub tag and draft release with the provided name. The GitHub action also uploads module manifests in the `btp-manager.yaml` file and module's default custom resource (CR) in the `btp-operator.yaml` as GitHub release assets.
9. If you chose to publish the release in step "Run the **Create release** GitHub action", the GitHub action publishes the release.

##  Create Release With Version Submit

![Release diagram](../assets/release-with-version-submit.drawio.svg)

To create a release, follow these steps:

1. Run the **Create release** GitHub action:  
   i.  Go to the **Actions** tab, and choose the **Create release** workflow, and next  **Run workflow**.  
   ii. Provide a version, for example, 1.2.0  
   iii. Choose real or dummy credentials for Service Manager  
   iv. Choose whether to bump or not to bump the security scanner config  
   v. Choose whether you want to publish the release
2. The GitHub action, defined in the [`create-release`](/.github/workflows/create-release.yaml) file, validates the release by checking if the GitHub tag already exists, if there are any old Docker images for that GitHub tag, and if merged PRs that are part of this release are labeled correctly. Additionally, it stops the release process if a feature has been added, but only the patch version number has been bumped up.
3. The GitHub action asynchronously initiates unit tests.
4. The Image Builder builds binary images.
5. The Image Builder uploads the binary images to registry.
6. The GitHub action initiates test jobs (stress tests, performance tests, upgrade tests, secret customization tests) using the built image. E2E upgrade tests run only with real credentials for the Service Manager. E2E tests are executed in parallel on the k3s clusters for the most recent k3s versions and with the specified credentials. The number of the most recent k3s versions to be used is defined in the **vars.LAST_K3S_VERSIONS** GitHub variable.
7. If you chose to bump the security scanner config in step "Run the **Create release** GitHub action", the GitHub action creates a PR with a new security scanner config that includes the new GitHub tag version.
8. The GitHub action creates a GitHub tag and draft release with the provided name. The GitHub action also uploads module manifests in the `btp-manager.yaml` file and module's default custom resource (CR) in the `btp-operator.yaml` as GitHub release assets.
9. If you chose to publish in step "Run the **Create release** GitHub action", the GitHub action publishes the release.
10. In the `module-manifests` repository, the GitHub action creates a PR with `module-config.yaml` for the new version of the module. If the PR for the given version already exists, the GitHub action updates the existing PR with the new `module-config.yaml`.

> [!NOTE]
> The PR created in the `module-manifests` repository is not automatically merged. It requires a code owner's approval. Once the PR is merged, the Submission Pipeline is triggered, pushing the generated ModuleTemplate to the `/kyma/kyma-modules` repository.

## Submit Module Version

To submit a module version, follow these steps:
1. Run the **Submit module version** GitHub action:  
   i.   Go to the **Actions** tab, and choose the **Submit module version** workflow, and next  **Run workflow**.  
   ii. [Optional] Override the default version taken from the latest GitHub tag with a custom version. For example, 1.2.0.
2. In the `module-manifests` repository, the GitHub action creates a PR with `module-config.yaml` for the new version of the module. If the PR for the given version already exists, the GitHub action updates the existing PR with the new `module-config.yaml`.

> [!NOTE]
> The PR created in the `module-manifests` repository is not automatically merged. It requires a code owner's approval. Once the PR is merged, the Submission Pipeline is triggered, pushing the generated ModuleTemplate to the `/kyma/kyma-modules` repository.
   
## Promote Module to Channel

To promote a released version, follow these steps:

1. Run the **Promote module to channel** GitHub action:  
   i.  Go to the **Actions** tab, and choose the **Promote module to channel** workflow, and next  **Run workflow**.  
   ii. Provide a version, for example, 1.2.0  
   iii. Choose the regular or fast channel
2. The GitHub action, defined in the [`promote_module_to_channel`](/.github/workflows/promote_module_to_channel.yaml) file, validates the release by checking if the GitHub tag already exists.
3. The GitHub action creates a PR in the `module-manifests` repository with the `module-releases.yaml` file modified in the section for the specified channel.
4. A code owner approves the PR.
5. Once the PR is merged, the Submission Pipeline is triggered, which generates a ModuleReleaseMeta CR and pushes it to the `/kyma/kyma-modules` repository.

