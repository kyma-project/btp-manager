# BTP Manager Release Pipeline

## Overview

The BTP Manager release pipeline creates proper artifacts:
 - btp-manager Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/unsigned%2Fcomponent-descriptors%2Fkyma.project.io%2Fmodule%2Fbtp-operator)
 - `btp-manager.yaml`, `btp-btp-operator-default-cr.yaml`

## Run the Pipeline

### Create a Release

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
3. If you chose in step 1.vi to bump the security scanner config, the GitHub action creates a PR with a new security scanner config that includes the new GitHub tag version.
4. A code owner approves the PR. 
5. The GitHub action creates a GitHub tag and draft release with the provided name.
6. The GitHub action asynchronously initiates unit tests and Image Builder.
7. The Image Builder uploads the binary image.
8. The GitHub action asynchronously initiates stress tests jobs, performance tests jobs, and E2E tests jobs upon the Image Builder job success status. E2E upgrade tests run only with real credentials for Service Manager.
9. The GitHub action runs E2E tests in parallel on the k3s clusters for the most recent k3s versions and with the specified credentials. The number of the most recent k3s versions to be used is defined in the **vars.LAST_K3S_VERSIONS** GitHub variable. 
10. If the unit tests, stress tests, and E2E tests are completed successfully and you have chosen to publish in step 1.vii, the GitHub action publishes the release.


### Replace an Existing Release

To regenerate an existing release, perform the following steps:

1. Delete the GitHub release.
2. Delete the GitHub tag.
3. Run the [**Create release**](#create-a-release) pipeline. 
