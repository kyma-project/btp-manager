# BTP Manager Release and Promotion to Channel

## Overview

The BTP Manager release pipeline creates proper artifacts:
 - btp-manager Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/unsigned%2Fcomponent-descriptors%2Fkyma.project.io%2Fmodule%2Fbtp-operator)
 - `btp-manager.yaml`, `btp-btp-operator-default-cr.yaml`

## Create a Release

![Release diagram](../assets/release.drawio.svg)

To create a release, follow these steps:

1. Run GitHub action **Create release**:  
   i.  Go to the **Actions** tab  
   ii. Click **Create release** workflow   
   iii. Click  **Run workflow** on the right  
   iv. Provide a version, for example, 1.2.0  
   v. Choose real or dummy credentials for Service Manager  
   vi. Choose whether to bump or not to bump the security scanner config  
   vii. Choose whether you want to publish the release
2. The GitHub action, defined in the [`create-release`](/.github/workflows/create-release.yaml) file, validates the release by checking if the GitHub tag already exists, if there are any old Docker images for that GitHub tag, and if merged PRs that are part of this release are labeled correctly. Additionally, it stops the release process if a feature has been added, but only the patch version number has been bumped up.
3. The GitHub action asynchronously initiates unit tests.
4. The Image Builder builds binary images.
5. The Image Builder uploads the binary images to registry.
6. The GitHub action initiates tests jobs (stress tests, performance tests, upgrade tests, secret customization tests) using built image. E2E upgrade tests run only with real credentials for Service Manager. E2E tests are executed in parallel on the k3s clusters for the most recent k3s versions and with the specified credentials. The number of the most recent k3s versions to be used is defined in the **vars.LAST_K3S_VERSIONS** GitHub variable.
7. If you chose in step 1.vi to bump the security scanner config, the GitHub action creates a PR with a new security scanner config that includes the new GitHub tag version.
8. The GitHub action creates a GitHub tag and draft release with the provided name.
9. If you have chosen to publish in step 1.vii, the GitHub action publishes the release.

##  Create Release With Version Submit

![Release diagram](../assets/release-with-version-submit.drawio.svg)

To create a release, follow these steps:

1. Run GitHub action **Create release**:  
   i.  Go to the **Actions** tab  
   ii. Click **Create release** workflow   
   iii. Click  **Run workflow** on the right  
   iv. Provide a version, for example, 1.2.0  
   v. Choose real or dummy credentials for Service Manager  
   vi. Choose whether to bump or not to bump the security scanner config  
   vii. Choose whether you want to publish the release
2. The GitHub action, defined in the [`create-release`](/.github/workflows/create-release.yaml) file, validates the release by checking if the GitHub tag already exists, if there are any old Docker images for that GitHub tag, and if merged PRs that are part of this release are labeled correctly. Additionally, it stops the release process if a feature has been added, but only the patch version number has been bumped up.
3. The GitHub action asynchronously initiates unit tests.
4. The Image Builder builds binary images.
5. The Image Builder uploads the binary images to registry.
6. The GitHub action initiates tests jobs (stress tests, performance tests, upgrade tests, secret customization tests) using built image. E2E upgrade tests run only with real credentials for Service Manager. E2E tests are executed in parallel on the k3s clusters for the most recent k3s versions and with the specified credentials. The number of the most recent k3s versions to be used is defined in the **vars.LAST_K3S_VERSIONS** GitHub variable.
7. If you chose in step 1.vi to bump the security scanner config, the GitHub action creates a PR with a new security scanner config that includes the new GitHub tag version.
8. The GitHub action creates a GitHub tag and draft release with the provided name.
9. If you have chosen to publish in step 1.vii, the GitHub action publishes the release.
10. The GitHub action creates in module-manifests repository a PR with module-config.yaml for the new version of the module. If the PR for the given version already exists, the GitHub action updates the existing PR with the new module-config.yaml.

[!NOTE]
The PR created in the module-manifests repository is not automatically merged. It requires a code owner approval. Once the PR is merged, the Submission Pipeline is triggered, and pushes the generated ModuleTemplate to the /kyma/kyma-modules repository.

## Submit Module Version

To submit a module version, follow these steps:
1. Run GitHub action **Submit module version**:  
   i.  Go to the **Actions** tab  
   ii. Click **Submit module version** workflow   
   iii. Click  **Run workflow** on the right  
   iv. Provide a version, for example, 1.2.0. By default, the version is taken from the latest GitHub tag, but you can override it with a custom version.
2. The GitHub action creates in module-manifests repository a PR with module-config.yaml for the new version of the module. If the PR for the given version already exists, the GitHub action updates the existing PR with the new module-config.yaml.

[!NOTE]
The PR created in the module-manifests repository is not automatically merged. It requires a code owner approval. Once the PR is merged, the Submission Pipeline is triggered, and pushes the generated ModuleTemplate to the /kyma/kyma-modules repository.
   
## Promote Module to Channel

To promote a released version, follow these steps:

1. Run GitHub action **Promote module to channel**:  
   i.  Go to the **Actions** tab  
   ii. Click **Promote module to channel** workflow   
   iii. Click  **Run workflow** on the right  
   iv. Provide a version, for example, 1.2.0  
   v. Choose regular or fast channel
2. The GitHub action, defined in the [`promote_module_to_channel`](/.github/workflows/promote_module_to_channel.yaml) file, validates the release by checking if the GitHub tag already exists.
3. The GitHub actions creates a PR in the `module-manifests` repository with the `module-releases.yaml` file modified in the section for the specified channel.
4. A code owner approves the PR.
5. Once PR is merged Submission Pipeline is triggered that generates a ModuleReleaseMeta CR, and pushes it to the /kyma/kyma-modules repository.

